package wal

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"time"

	wal_pb "wal/proto"

	pb "google.golang.org/protobuf/proto"
)

func initConfig(userConfig *Options) *Options {
	config := DefaultConfig()

	// Override default values with user-provided values
	if userConfig != nil {
		if userConfig.LogDir != "" {
			config.LogDir = userConfig.LogDir
		}
		if userConfig.MaxLogFileSize != 0 {
			config.MaxLogFileSize = userConfig.MaxLogFileSize
		}
		if userConfig.maxSegments != 0 {
			config.maxSegments = userConfig.maxSegments
		}
		if userConfig.SyncInterval != 0 {
			config.SyncInterval = userConfig.SyncInterval
		}
		if userConfig.EnableSync != config.EnableSync {
			config.EnableSync = userConfig.EnableSync
		}
	}
	return config
}

// WALConfig holds the configuration for the Write Ahead Log
// This method opens the WAL file for writing and returns a pointer to the WriteAheadLog struct
func Open(config *Options) (*WriteAheadLog, error) {
	// config is optional, it will get the default if not provided
	config = initConfig(config)
	fileNamePrefix := config.LogDir + segmentPrefix
	ctx, cancel := context.WithCancel(context.Background())
	wal := &WriteAheadLog{
		logFileNamePrefix: fileNamePrefix,
		lastSeqNo:         0,
		maxLogFileSize:    config.MaxLogFileSize,
		maxSegments:       config.maxSegments,
		currentSegmentNo:  1,
		syncDelay:         time.NewTicker(config.SyncInterval),
		syncInterval:      config.SyncInterval,
		ctx:               ctx,
		cancel:            cancel,
	}

	err := wal.openExistingOrCreateSegment(config.LogDir)
	if err != nil {
		return nil, err
	}
	if wal.lastSeqNo, err = wal.getLastSeqNo(); err != nil {
		return nil, fmt.Errorf("failed to get last sequence number: %w", err)
	}
	go wal.keepSyncing()

	return wal, nil
}

func (wal *WriteAheadLog) Write(data []byte) error {
	return wal.writeEntry(data, false)
}

func (wal *WriteAheadLog) WriteWithCheckpoint(data []byte) error {
	return wal.writeEntry(data, true)
}

// Write data to the log file
// Create WAL_DATA struct and marshal it to bytes
func (wal *WriteAheadLog) writeEntry(data []byte, isCheckpoint bool) error {
	wal.locker.Lock()
	defer wal.locker.Unlock()

	if wal.file == nil || wal.ctx.Err() != nil {
		return fmt.Errorf("WAL is closed, cannot write data")
	}

	if wal.checkRotateLog(data) {
		if err := wal.Sync(); err != nil {
			return fmt.Errorf("Couldn't rotate log, error in syncing %v", err)
		}
		if err := wal.rotateLog(); err != nil {
			return err
		}
	}

	wal.lastSeqNo++
	entry := &wal_pb.WAL_DATA{
		LogSeqNo: wal.lastSeqNo,
		Data:     data,
		Checksum: crc32.ChecksumIEEE(append(data, byte(wal.lastSeqNo))),
	}

	if isCheckpoint {
		if err := wal.Sync(); err != nil {
			return fmt.Errorf("Couldn't create checkpoint, error in syncing %v", err)
		}
		entry.IsCheckpoint = &isCheckpoint
	}
	return wal.WriteIntoBuffer(entry)
}

// WriteIntoBuffer writes the WAL_DATA into the buffer writer
// It marshals the WAL_DATA to bytes, writes the size of the data first, then
func (wal *WriteAheadLog) WriteIntoBuffer(entry *wal_pb.WAL_DATA) error {
	bytesWalData, err := pb.Marshal(entry)
	if err != nil {
		return err
	}
	// protobuf data length is written as 4 bytes in little-endian format 32 bits = 4 * 8 bits
	size := uint32(len(bytesWalData))
	// Protobuf messages are variable lenght encoding and have no built-in separator
	// So we write the size of the message first, then the message itself. means next N bytes are the data
	if err := binary.Write(wal.bufWriter, binary.LittleEndian, size); err != nil {
		return err
	}
	if _, err := wal.bufWriter.Write(bytesWalData); err != nil {
		return err
	}
	return nil
}

func (wal *WriteAheadLog) ReadAll() ([]*wal_pb.WAL_DATA, error) {
	entries, error := wal.readAllEntries(false)
	return entries, error
}

func (wal *WriteAheadLog) ReadFromCheckPoint() ([]*wal_pb.WAL_DATA, error) {
	entries, error := wal.readAllEntries(true)
	return entries, error
}

func (wal *WriteAheadLog) readAllEntries(fromCheckpoint bool) ([]*wal_pb.WAL_DATA, error) {
	// checkpointLogSeqNo := uint64(0)
	walFile, err := os.Open(wal.file.Name())
	if err != nil {
		return nil, err
	}
	defer walFile.Close()

	entries := []*wal_pb.WAL_DATA{}

	for {
		var size uint32
		if err := binary.Read(walFile, binary.LittleEndian, &size); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		data := make([]byte, size)
		_, err := walFile.Read(data)
		if err != nil {
			return nil, err
		}
		entry := &wal_pb.WAL_DATA{}
		if err := pb.Unmarshal(data, entry); err != nil {
			return nil, err
		}
		if crc32.ChecksumIEEE(append(entry.GetData(), byte(entry.GetLogSeqNo()))) != entry.GetChecksum() {
			return nil, fmt.Errorf("CRC mismatch for entry with seq no %d", entry.GetLogSeqNo())
		}
		if fromCheckpoint && entry.GetIsCheckpoint() {
			entries = entries[:0]
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (wal *WriteAheadLog) Sync() error {
	if err := wal.bufWriter.Flush(); err != nil {
		return err
	}
	if err := wal.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync WAL file: %w", err)
	}
	return nil
}

func (wal *WriteAheadLog) keepSyncing() {
	for {
		select {
		case <-wal.ctx.Done():
			return
		case <-wal.syncDelay.C:
			wal.locker.Lock()
			err := wal.Sync()
			wal.locker.Unlock()
			if err != nil {
				// Log the error
				log.Printf("failed to sync WAL: %v", err)
			}
		}
	}
}

func (wal *WriteAheadLog) resetTimer() {
	// Stop the ticker to reset the sync delay
	wal.syncDelay.Stop()
	// Reset the ticker to the sync interval
	wal.syncDelay.Reset(wal.syncInterval)
}

func (wal *WriteAheadLog) Close() error {
	// Cancel the context to stop any ongoing operations
	wal.cancel()

	if err := wal.Sync(); err != nil {
		return err
	}
	wal.resetTimer()
	err := wal.file.Close()
	wal.file = nil
	return err
}
