package wal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempWalDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "wal-test-")
	dir2 := t.TempDir()
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir2
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

func TestWriteAndReadEntries(t *testing.T) {
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

func TestReadWriteWithCheckpoint(t *testing.T) {
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
	entries, err := wal.ReadFromCheckPoint()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entries, got %d", len(entries))
	}

	// Second entry should be a checkpoint
	if !entries[0].GetIsCheckpoint() {
		t.Errorf("Entry 1 should be a checkpoint")
	}

	if !bytes.Equal(entries[0].GetData(), checkpointData) {
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

func TestWALSyncing(t *testing.T) {
	dir := tempWalDir(t)
	syncDelay := 100 * time.Millisecond
	wal, _ := Open(&Options{LogDir: dir + "/", SyncInterval: syncDelay})
	// Write 3 entries
	testData := make([][]byte, 3)
	for i := 0; i < 3; i++ {
		testData[i] = []byte(fmt.Sprintf("Synced data-%d", i))
		if err := wal.Write(testData[i]); err != nil {
			t.Fatalf("Write failed at entry %d: %v", i, err)
		}
	}
	if err := wal.Sync(); err != nil {
		t.Fatalf("Sync Failed")
	}
	entries, _ := wal.ReadAll()
	if len(entries) != 3 {
		t.Errorf("All data didn't synced, Got: %d", len(entries))
	}
	testData_2 := make([][]byte, 2)
	for i := 0; i < 2; i++ {
		testData_2[i] = []byte(fmt.Sprintf("Synced data-%d with delay", i))
		if err := wal.Write(testData_2[i]); err != nil {
			t.Fatalf("Write failed at entry %d: %v", i, err)
		}
	}
	entries_2, _ := wal.ReadAll()
	if len(entries_2) != 3 {
		t.Errorf("Expected synced data 3 but Got: %d", len(entries))
	}
	// it will sync automatially with in the syncDelay
	time.Sleep(syncDelay)
	entries_3, _ := wal.ReadAll()
	if len(entries_3) != 5 {
		t.Errorf("Expected synced data 5 but Got: %d", len(entries_3))
	}
}

func TestChecksumValidation(t *testing.T) {
	dir := tempWalDir(t)
	wal, _ := Open(&Options{LogDir: dir + "/"})
	if err := wal.Write([]byte("valid data")); err != nil {
		t.Fatalf("Failed to write %v", err)
	}
	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close %v", err)
	}

	fileName := filepath.Join(dir+"/", segmentPrefix+"1")
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("Failed to open the file")
	}
	if _, err := file.Write([]byte("Corrupted data")); err != nil {
		t.Fatalf("Couldn't write into open file")
	}
	file.Close()

	wal, _ = Open(&Options{LogDir: dir + "/"})
	defer wal.Close()

	// Reading should either fail or return only valid entries
	entries, err := wal.ReadAll()
	// the err variable won't be nil because the file is corrupted
	if err == nil {
		if len(entries) != 1 {
			t.Errorf("Expected 1 entries but got %d", len(entries))
		} else {
			t.Fatalf("Couldn't parse the error:- %v", err)
		}
	}
}
