package wal

import (
	"time"
)

type Options struct {
	LogDir         string
	MaxLogFileSize uint32
	MaxSegmentSize uint32
	EnableSync     bool
	SyncInterval   time.Duration
}

func DefaultConfig() *Options {
	return &Options{
		LogDir:         "./wal_data/",
		MaxLogFileSize: 16 * 1024 * 1024, // 16MB
		MaxSegmentSize: 64 * 1024,        // 64KB
		EnableSync:     false,
		SyncInterval:   5 * time.Second,
	}
}
