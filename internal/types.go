package wal

import (
	"bufio"
	"os"
	"sync"
	"time"
)

type WALConfig struct {
	LogDir         string
	MaxLogFileSize uint32
	MaxSegmentSize uint32
	enableSync     bool
	SyncDelay      time.Duration
}

type WriteAheadLog struct {
	logFileNamePrefix string
	file              *os.File
	locker            sync.Mutex
	lastSequenceNo    uint64
	bufWriter         *bufio.Writer
	syncDelay         *time.Ticker
	maxLogFileSize    uint32
	maxSegmentSize    uint32
	currentSegmentNo  int
}
