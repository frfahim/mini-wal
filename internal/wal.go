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

// WALConfig holds the configuration for the Write Ahead Log
// This method opens the WAL file for writing and returns a pointer to the WriteAheadLog struct
func Open(config *Options) (*WriteAheadLog, error) {
	// config is optional, it will get the default if not provided
	if config == nil {
		config = DefaultConfig()
	}
	fileNamePrefix := config.LogDir + segmentPrefix
	ctx, cancel := context.WithCancel(context.Background())
	wal := &WriteAheadLog{
		logFileNamePrefix: fileNamePrefix,
		lastSeqNo:         0,
		maxLogFileSize:    config.MaxLogFileSize,
		maxSegmentSize:    config.MaxSegmentSize,
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

// Write data to the log file
// Create WAL_DATA struct and marshal it to bytes
func (wal *WriteAheadLog) Write(data []byte) error {
	wal.locker.Lock()
	defer wal.locker.Unlock()

	wal.lastSeqNo++
	walData := &wal_pb.WAL_DATA{
		LogSeqNo: wal.lastSeqNo,
		Data:     data,
		Checksum: crc32.ChecksumIEEE(append(data, byte(wal.lastSeqNo))),
	}
	return wal.WriteIntoBuffer(walData)
}

func (wal *WriteAheadLog) WriteWithCheckpoint(data []byte) error {
	wal.locker.Lock()
	defer wal.locker.Unlock()

	wal.lastSeqNo++
	isCheckpoint := true
	walData := &wal_pb.WAL_DATA{
		LogSeqNo:     wal.lastSeqNo,
		Data:         data,
		Checksum:     crc32.ChecksumIEEE(append(data, byte(wal.lastSeqNo))),
		IsCheckpoint: &isCheckpoint,
	}
	return wal.WriteIntoBuffer(walData)
}

// WriteIntoBuffer writes the WAL_DATA into the buffer writer
// It marshals the WAL_DATA to bytes, writes the size of the data first, then
func (wal *WriteAheadLog) WriteIntoBuffer(walData *wal_pb.WAL_DATA) error {
	bytesWalData, err := pb.Marshal(walData)
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
		fmt.Println(size)
		_, err := walFile.Read(data)
		// fmt.Println(n1)
		// n, err := io.ReadFull(walFile, data)
		// fmt.Println(n)
		if err != nil {
			return nil, err
		}
		entry := &wal_pb.WAL_DATA{}
		if err := pb.Unmarshal(data, entry); err != nil {
			return nil, err
		}
		if crc32.ChecksumIEEE(entry.GetData()) != entry.GetChecksum() {
			return nil, fmt.Errorf("CRC mismatch for entry with seq no %d", entry.GetLogSeqNo())
		}
		fmt.Println(entry)
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
	return wal.file.Close()
}
