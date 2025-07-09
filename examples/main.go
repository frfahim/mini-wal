package main

import (
	"fmt"
	"log"

	wal "wal/internal"
)

func main() {
	// Create test directory
	testDir := "/Users/fahim/work/mini-wal/examples_data/"
	// os.MkdirAll(testDir, 0755)

	configs := wal.DefaultConfig()
	configs.LogDir = testDir

	// Open the WAL
	openedWal, err := wal.Open(configs)
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

	// Read content of a file
	openedWal.ReadAll()

	// You could add more verification here, like reading back the data

	fmt.Println("\nTest completed successfully!")
}
