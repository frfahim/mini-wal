package wal

import (
	"bufio"
	"context"
	"os"
	"sync"
	"time"
)

type WriteAheadLog struct {
	logFileNamePrefix string
	file              *os.File           // current segment file
	bufWriter         *bufio.Writer      // buffered writer for the file
	currentSegmentNo  int                // current segment number
	lastSeqNo         uint64             // last sequence number written to the log
	locker            sync.Mutex         // Mutex to protect concurrent writes
	syncInterval      time.Duration      // Interval for periodic sync
	syncDelay         *time.Ticker       // Timer for periodic sync
	maxLogFileSize    int32              // maximum log file size
	maxSegments       int                // maximum segment size
	ctx               context.Context    // context for cancellation
	cancel            context.CancelFunc // function to cancel the context
}
