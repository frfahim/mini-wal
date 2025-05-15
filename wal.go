package wal

import (
	"bufio"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	wal_proto "github.com/frfahim/wal/proto"
	"github.com/frfahim/wal/proto/wal_pb"
	// "mvdan.cc/unparam/check"
)

const segmentPrefix = "segment-"

type WALConfig struct {
	LogDir         string
	MaxLogFileSize uint32
	MaxSegmentSize uint32
	enableSync     bool
	SyncDelay      time.Duration
}

type WriteAheadLog struct {
	// logDir           string
	logFileNamePrefix string
	file              *os.File
	locker            sync.Mutex
	lastSequenceNo    uint64
	bufWriter         *bufio.Writer
	syncDelay         *time.Ticker
	shouldSync        bool
	maxLogFileSize    uint32
	maxSegmentSize    uint32
	segmentCount      int
	currentSegmentID  int
}

func NewWAL(config *WALConfig) (*WriteAheadLog, error) {
	fileNamePrefix := config.LogDir + segmentPrefix
	wal := &WriteAheadLog{
		// logDir:           config.LogDir,
		logFileNamePrefix: fileNamePrefix,
		// locker:            sync.Mutex{},
		lastSequenceNo:   0,
		maxLogFileSize:   config.MaxLogFileSize,
		maxSegmentSize:   config.MaxSegmentSize,
		currentSegmentID: 0,
		segmentCount:     0,
		shouldSync:       true,
		syncDelay:        time.NewTicker(config.SyncDelay),
	}

	err := wal.OpenExistingOrCreateSegment(config.LogDir)
	if err != nil {
		return nil, err
	}

	return wal, nil
}

func (wal *WriteAheadLog) Write(data []byte) error {
	wal.locker.Lock()
	defer wal.locker.Unlock()
	wal.lastSequenceNo++
	walData := &wal_pb.WAL_DATA{
		LogSeqNo: wal.lastSequenceNo,
		Data:     data,
		Checksum: crc32.ChecksumIEEE(append(data, byte(wal.lastSequenceNo))),
	}
	bytesWalData := wal_proto.MarshalData(walData)
	size := uint32(len(bytesWalData))
	if err := binary.Write(wal.bufWriter, binary.LittleEndian, size); err != nil {
		return err
	}
	if _, err := wal.bufWriter.Write(bytesWalData); err != nil {
		return err
	}
	return nil
}

func checkEmptyDir(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

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

func (wal *WriteAheadLog) createNewSegment() error {
	fileName := wal.logFileNamePrefix + string(wal.currentSegmentID)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	wal.file = file
	wal.bufWriter = bufio.NewWriter(file)
	return nil
}

func (wal *WriteAheadLog) openExistingSegment() error {
	logFiles, err := filepath.Glob(wal.logFileNamePrefix + "*")
	if err != nil {
		return err
	}
	sort.Strings(logFiles)
	lastFileName := logFiles[len(logFiles)-1]
	file, err := os.OpenFile(lastFileName, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	s := strings.Split(lastFileName, "-")
	lastSegmentID, err := strconv.Atoi(s[1])
	if err != nil {
		return err
	}
	// Go to the end of the file
	file.Seek(0, io.SeekEnd)
	wal.file = file
	wal.currentSegmentID = lastSegmentID
	wal.bufWriter = bufio.NewWriter(file)
	wal.segmentCount = len(logFiles) - 1 // Calculate the segment count
	return nil
}
