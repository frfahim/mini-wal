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
	wal_pb "wal/proto"

	proto "google.golang.org/protobuf/proto"
)

// Check if the directory is empty
func checkEmptyDir(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// openExistingOrCreateSegment checks if the directory is empty
// If it is empty, it creates a new segment file
// If it is not empty, it opens the last segment file for writing
func (wal *WriteAheadLog) openExistingOrCreateSegment(dirPath string) error {
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
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to the end of segment: %w", err)
	}
	wal.file = file
	wal.bufWriter = bufio.NewWriter(file)
	wal.currentSegmentNo = lastSegmentNo
	return nil
}

func (wal *WriteAheadLog) getLastSeqNo() (uint64, error) {
	// Get the last entry in the current segment
	lastEntry, err := wal.getLastEntryInSegment()
	if err != nil {
		return 0, err
	}
	if lastEntry == nil {
		return 0, nil // No entries in the segment
	}
	return lastEntry.GetLogSeqNo(), nil
}

func (wal *WriteAheadLog) getLastEntryInSegment() (*wal_pb.WAL_DATA, error) {
	var lastEntry *wal_pb.WAL_DATA
	// Read the last entry from the current segment
	for {
		var size uint32
		if err := binary.Read(wal.file, binary.LittleEndian, &size); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		data := make([]byte, size)
		_, err := wal.file.Read(data)
		if err != nil {
			return nil, err
		}
		entry, err := UnmarshalAndValidateEntry(data)
		if err != nil {
			return lastEntry, err
		}
		lastEntry = entry
	}
	return lastEntry, nil
}

func UnmarshalAndValidateEntry(data []byte) (*wal_pb.WAL_DATA, error) {
	entry := &wal_pb.WAL_DATA{}
	if err := proto.Unmarshal(data, entry); err != nil {
		return nil, err
	}
	if !verifyChecksum(entry) {
		return nil, fmt.Errorf("invalid checksum for entry with seq no %d", entry.GetLogSeqNo())
	}
	return entry, nil
}

func verifyChecksum(entry *wal_pb.WAL_DATA) bool {
	return entry.GetChecksum() == crc32.ChecksumIEEE(entry.GetData())
}
