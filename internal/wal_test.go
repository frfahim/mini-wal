package wal

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"
)

func tempWalDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "wal-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestOpenAndCloseWAL(t *testing.T) {
	wal, err := Open(nil)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := wal.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestWriteAndReadSingleEntry(t *testing.T) {
	dir := tempWalDir(t)
	wal, _ := Open(&Options{LogDir: dir + "/"})
	// defer wal.Close()
	data := []byte("Add single entry")
	if err := wal.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	wal.Sync()
	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if !bytes.Equal(entries[0].GetData(), data) {
		t.Errorf("Read data mismatch: got %v, want %v", entries[0].GetData(), data)
	}
	// Check sequence number
	if entries[0].GetLogSeqNo() != 1 {
		t.Errorf("Expected sequence number 1, got %d", entries[0].GetLogSeqNo())
	}
}

func TestWriteAndReadMultipleEntries(t *testing.T) {
	dir := tempWalDir(t)
	wal, err := Open(&Options{LogDir: dir + "/"})
	defer wal.Close()

	// Write 10 entries
	testData := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		testData[i] = []byte(fmt.Sprintf("WAL Entry-%d", i))
		if err := wal.Write(testData[i]); err != nil {
			t.Fatalf("Write failed at entry %d: %v", i, err)
		}
	}
	if err := wal.Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 10 {
		t.Errorf("Expected 10 entries, got %d", len(entries))
	}

	// Verify data and sequence numbers
	for i, entry := range entries {
		if !bytes.Equal(entry.GetData(), testData[i]) {
			t.Errorf("Entry %d data mismatch: got %v, want %v", i, entry.GetData(), testData[i])
		}

		if entry.GetLogSeqNo() != uint64(i+1) {
			t.Errorf("Entry %d sequence number mismatch: got %d, want %d", i, entry.GetLogSeqNo(), i+1)
		}
	}
}

func TestWriteWithCheckpoint(t *testing.T) {
	dir := tempWalDir(t)
	wal, _ := Open(&Options{LogDir: dir + "/", SyncInterval: 10 * time.Millisecond})
	defer wal.Close()

	// Write regular entry
	if err := wal.Write([]byte("Regular entry 01")); err != nil {
		t.Fatalf("Write regular entry failed: %v", err)
	}

	// Write checkpoint entry
	checkpointData := []byte("checkpoint entry 01")
	if err := wal.WriteWithCheckpoint(checkpointData); err != nil {
		t.Fatalf("WriteWithCheckpoint failed: %v", err)
	}

	wal.Sync()
	entries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// First entry should not be a checkpoint
	if entries[0].GetIsCheckpoint() {
		t.Errorf("Entry 0 should not be a checkpoint")
	}

	// Second entry should be a checkpoint
	if !entries[1].GetIsCheckpoint() {
		t.Errorf("Entry 1 should be a checkpoint")
	}

	if !bytes.Equal(entries[1].GetData(), checkpointData) {
		t.Errorf("Checkpoint data mismatch: got %v, want %v", entries[1].GetData(), checkpointData)
	}
}

func TestWriteAfterClose(t *testing.T) {
	dir := tempWalDir(t)
	wal, _ := Open(&Options{LogDir: dir + "/", SyncInterval: 10 * time.Millisecond})

	wal.Write([]byte("Entry data before closing WAL"))
	// Close the WAL
	if err := wal.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	err := wal.Write([]byte("Entry data after closing WAL"))
	if err == nil {
		t.Errorf("Expected error when writing after close, got nil")
	}
}
