package wal

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	wal_pb "wal/proto"

	pb "google.golang.org/protobuf/proto"
)

const segmentPrefix = "segment-"

// WALConfig holds the configuration for the Write Ahead Log
// This method opens the WAL file for writing and returns a pointer to the WriteAheadLog struct
func Open(config *WALConfig) (*WriteAheadLog, error) {
	fileNamePrefix := config.LogDir + segmentPrefix
	wal := &WriteAheadLog{
		logFileNamePrefix: fileNamePrefix,
		lastSequenceNo:    1,
		maxLogFileSize:    config.MaxLogFileSize,
		maxSegmentSize:    config.MaxSegmentSize,
		currentSegmentNo:  1,
		syncDelay:         time.NewTicker(config.SyncDelay),
	}

	err := wal.OpenExistingOrCreateSegment(config.LogDir)
	if err != nil {
		return nil, err
	}

	return wal, nil
}

// Write data to the log file
// Create WAL_DATA struct and marshal it to bytes
func (wal *WriteAheadLog) Write(data []byte) error {
	wal.locker.Lock()
	defer wal.locker.Unlock()
	wal.lastSequenceNo++
	walData := &wal_pb.WAL_DATA{
		LogSeqNo: wal.lastSequenceNo,
		Data:     data,
		Checksum: crc32.ChecksumIEEE(append(data, byte(wal.lastSequenceNo))),
	}
	bytesWalData, err := pb.Marshal(walData)
	if err != nil {
		return err
	}
	size := uint32(len(bytesWalData))
	if err := binary.Write(wal.bufWriter, binary.LittleEndian, size); err != nil {
		return err
	}
	if _, err := wal.bufWriter.Write(bytesWalData); err != nil {
		return err
	}
	return nil
}

func (wal *WriteAheadLog) Close() error {
	wal.locker.Lock()
	defer wal.locker.Unlock()

	if err := wal.Sync(); err != nil {
		return err
	}
	return wal.file.Close()
}

func (wal *WriteAheadLog) Sync() error {
	if err := wal.bufWriter.Flush(); err != nil {
		return err
	}
	return wal.file.Sync()

}

// Check if the directory is empty
func checkEmptyDir(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// OpenExistingOrCreateSegment checks if the directory is empty
// If it is empty, it creates a new segment file
// If it is not empty, it opens the last segment file for writing
func (wal *WriteAheadLog) OpenExistingOrCreateSegment(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	isEmptyDir, err := checkEmptyDir(dirPath)
	if err != nil {
		return err
	}
	if isEmptyDir {
		// Create a new segment if the directory is empty
		err := wal.createNewSegment()
		if err != nil {
			return err
		}
	} else {
		// Open the existing segment
		err := wal.openExistingSegment()
		if err != nil {
			return err
		}
	}
	return nil
}

// Create a file with the prefix and segment no
// It creates a new segment file with the name "segment-<segmentID>"
func (wal *WriteAheadLog) createNewSegment() error {
	fileName := wal.logFileNamePrefix + strconv.Itoa(wal.currentSegmentNo)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	wal.file = file
	wal.bufWriter = bufio.NewWriter(file)
	return nil
}

// Open the last segment file for writing
// It assumes that the segment files are named in the format "segment-<segmentID>"
// and by sorting the files, it can find the last segment file
// It opens the last segment file for writing and sets the currentSegmentNo to the last segment ID
// It also seeks to the end of the file to append new data
func (wal *WriteAheadLog) openExistingSegment() error {
	// Get the list of log files in the directory, using the prefix
	logFiles, err := filepath.Glob(wal.logFileNamePrefix + "*")
	if err != nil {
		return err
	}
	sort.Strings(logFiles)
	lastFileName := logFiles[len(logFiles)-1]
	// Open the last segment file for writing
	file, err := os.OpenFile(lastFileName, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	// Extract the segment ID from the file name
	s := strings.Split(lastFileName, "-")
	fmt.Println(s)
	lastSegmentNo, err := strconv.Atoi(s[2])
	if err != nil {
		return err
	}
	// Go to the end of the file
	file.Seek(0, io.SeekEnd)
	wal.file = file
	wal.bufWriter = bufio.NewWriter(file)
	wal.currentSegmentNo = lastSegmentNo
	return nil
}
