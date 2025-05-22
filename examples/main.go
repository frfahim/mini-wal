package main

import (
	"fmt"
	"log"
	"os"
	"time"

	wal "wal/internal"
)

func main() {
	// Create test directory
	testDir := "/Users/fahim/work/mini-wal/examples_data/"
	os.MkdirAll(testDir, 0755)

	// Initialize WAL configuration
	config := &wal.WALConfig{
		LogDir:         testDir,
		MaxLogFileSize: 1024 * 1024, // 1MB
		MaxSegmentSize: 64 * 1024,   // 64KB
		SyncDelay:      time.Second * 5,
	}

	// Open the WAL
	openedWal, err := wal.Open(config)
	if err != nil {
		log.Fatalf("Failed to open WAL: %v", err)
	}
	fmt.Println("WAL opened successfully")

	// Write some test data
	testData := []string{
		"This is test message 10",
		"This is test message 20",
		"This is a longer test message to verify everything works with larger data blocks",
	}

	for i, data := range testData {
		err := openedWal.Write([]byte(data))
		if err != nil {
			log.Fatalf("Failed to write data: %v", err)
		}
		fmt.Printf("Successfully wrote test data %d: %s\n", i+1, data)
	}
	openedWal.Close()

	// Verify the files were created
	files, err := os.ReadDir(testDir)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	fmt.Printf("\nFiles created in %s:\n", testDir)
	for _, file := range files {
		info, _ := file.Info()
		fmt.Printf("- %s (size: %d bytes)\n", file.Name(), info.Size())
	}

	// You could add more verification here, like reading back the data

	fmt.Println("\nTest completed successfully!")
}
