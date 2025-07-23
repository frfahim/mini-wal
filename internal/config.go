package wal

import (
	"time"
)

type Options struct {
	LogDir         string
	MaxLogFileSize int32
	maxSegments    int
	EnableSync     bool
	SyncInterval   time.Duration
}

func DefaultConfig() *Options {
	return &Options{
		LogDir:         "./wal_data/",
		MaxLogFileSize: 16 * 1024 * 1024, // 16MB
		maxSegments:    5,
		EnableSync:     false,
		SyncInterval:   5 * time.Second,
	}
}
