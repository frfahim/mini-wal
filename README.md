# Mini-WAL 🗂️

A high-performance Write-Ahead Log (WAL) implementation in Go with Protocol Buffers serialization.

[![Go Version](https://img.shields.io/badge/Go-1.23.1+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](LICENSE)
[![Build System](https://img.shields.io/badge/Build-Bazel-green.svg)](https://bazel.build)

## 🎯 Overview

Mini-WAL is a lightweight, thread-safe Write-Ahead Log implementation designed for applications that need reliable data persistence with ACID properties. It provides:

- **Durability**: All writes are persisted to disk with configurable sync intervals
- **Consistency**: CRC32 checksums ensure data integrity
- **Atomicity**: Each log entry is written atomically with sequence numbers
- **Performance**: Buffered writes with automatic segment rotation
- **Concurrency**: Thread-safe operations with mutex protection
- **Recovery**: Automatic recovery from existing WAL files on startup

## 🏗️ Architecture

### Core Components

```
mini-wal/
├── internal/           # Core WAL implementation
│   ├── wal.go         # Main WAL operations (Open, Write, Read, Close)
│   ├── segments.go    # Segment management and rotation
│   ├── config.go      # Configuration and options
│   ├── types.go       # Data structures and types
│   └── const.go       # Constants and defaults
├── proto/             # Protocol Buffer definitions
│   ├── wal.proto      # WAL data structure definition
│   └── pb_utils.go    # Protobuf utilities
├── examples/          # Usage examples
│   └── main.go        # Basic usage demonstration
└── cmd/               # Command-line tools (if any)
```

### Data Format

Each WAL entry is serialized using Protocol Buffers with the following structure:

```protobuf
message WAL_DATA {
  uint64 logSeqNo = 1;        // Monotonic sequence number
  bytes data = 2;             // User data payload
  uint32 checksum = 3;        // CRC32 checksum for integrity
  optional bool isCheckpoint = 4;  // Checkpoint marker
}
```

### Storage Layout

```
wal_data/
├── segment-1           # First segment file
├── segment-2           # Second segment file (after rotation)
└── segment-N           # Additional segments...
```

Each segment file contains:
- **Size Prefix** (4 bytes): Length of the protobuf message
- **Protobuf Data**: Serialized WAL_DATA message
- **Repeats**: Multiple entries per segment until size limit

## 🚀 Quick Start

### Prerequisites

- **Go 1.23.1+**
- **Bazel** (for building)
- **Protocol Buffers** (for code generation)

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/frfahim/mini-wal.git
   cd mini-wal
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Generate Protocol Buffer code:**
   ```bash
   make build-proto
   # OR manually:
   # protoc --proto_path=proto --go_out=proto --go_opt=paths=source_relative proto/wal.proto
   ```

4. **Build with Bazel:**
   ```bash
   bazel build //...
   ```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "time"
    wal "wal/internal"
)

func main() {
    // Configure WAL options
    config := &wal.Options{
        LogDir:         "./my_wal_data/",
        MaxLogFileSize: 16 * 1024 * 1024, // 16MB per segment
        EnableSync:     true,
        SyncInterval:   5 * time.Second,
    }

    // Open WAL
    w, err := wal.Open(config)
    if err != nil {
        log.Fatalf("Failed to open WAL: %v", err)
    }
    defer w.Close()

    // Write data
    data := []byte("Hello, WAL!")
    if err := w.Write(data); err != nil {
        log.Fatalf("Failed to write: %v", err)
    }

    // Write checkpoint entry
    checkpointData := []byte("Important checkpoint")
    if err := w.WriteWithCheckpoint(checkpointData); err != nil {
        log.Fatalf("Failed to write checkpoint: %v", err)
    }

    // Manually sync to disk
    if err := w.Sync(); err != nil {
        log.Fatalf("Failed to sync: %v", err)
    }

    // Read all entries
    entries, err := w.ReadAll()
    if err != nil {
        log.Fatalf("Failed to read: %v", err)
    }

    for _, entry := range entries {
        fmt.Printf("Seq: %d, Data: %s, Checkpoint: %v\n", 
            entry.GetLogSeqNo(), 
            string(entry.GetData()), 
            entry.GetIsCheckpoint())
    }
}
```

## ⚙️ Configuration

### Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `LogDir` | `string` | `"./wal_data/"` | Directory to store WAL segments |
| `MaxLogFileSize` | `int32` | `16MB` | Maximum size per segment file |
| `maxSegments` | `int` | `5` | Maximum number of segments |
| `EnableSync` | `bool` | `false` | Enable automatic periodic sync |
| `SyncInterval` | `time.Duration` | `5s` | Interval between automatic syncs |

### Example Configurations

**High Performance (Minimal Sync):**
```go
config := &wal.Options{
    LogDir:         "./fast_wal/",
    MaxLogFileSize: 64 * 1024 * 1024, // 64MB
    EnableSync:     false,             // Manual sync only
}
```

**High Durability (Frequent Sync):**
```go
config := &wal.Options{
    LogDir:       "./durable_wal/",
    EnableSync:   true,
    SyncInterval: 1 * time.Second,     // Sync every second
}
```

**Small Files (Frequent Rotation):**
```go
config := &wal.Options{
    LogDir:         "./small_wal/",
    MaxLogFileSize: 1 * 1024 * 1024,   // 1MB segments
    EnableSync:     true,
}
```

## 🧪 Development & Testing

### Running Tests

**All tests:**
```bash
bazel test //...
```

**Specific test:**
```bash
bazel test //internal:wal_test --test_filter="TestWriteAndRead"
```

**With verbose output:**
```bash
bazel test //internal:wal_test --test_output=all
```

**Force re-run (no cache):**
```bash
bazel test //... --nocache_test_results
```

### Running Examples

```bash
# Build and run example
bazel run //examples:main

# Or with Go directly
go run examples/main.go
```

### Test Coverage

The test suite includes comprehensive coverage for:

- ✅ **Basic Operations**: Open, Write, Read, Close
- ✅ **Concurrency**: Multi-threaded writes with proper synchronization
- ✅ **Segment Rotation**: Automatic file rotation when size limits are reached
- ✅ **Data Integrity**: CRC32 checksum validation
- ✅ **Recovery**: WAL recovery from existing files
- ✅ **Error Handling**: Various error conditions and edge cases
- ✅ **Performance**: Large data sets and high-throughput scenarios
- ✅ **Checkpoints**: Checkpoint marking and reading

### Benchmarking

Create performance tests to measure throughput:

```bash
# Run performance tests (if available)
bazel test //internal:wal_test --test_filter="TestPerformance" --test_timeout=300
```

## 🔍 API Reference

### Core Functions

#### `Open(config *Options) (*WriteAheadLog, error)`
Opens a new WAL instance with the given configuration. Creates the log directory if it doesn't exist and recovers from existing segments.

#### `Write(data []byte) error`
Writes data to the WAL with automatic sequence numbering and CRC32 checksum.

#### `WriteWithCheckpoint(data []byte) error`
Writes data with a checkpoint marker, useful for marking important state transitions.

#### `ReadAll() ([]*wal_pb.WAL_DATA, error)`
Reads all entries from all segments in sequence order.

#### `Sync() error`
Forces a sync of buffered data to disk.

#### `Close() error`
Safely closes the WAL, ensuring all data is synced and resources are released.

### WAL Entry Structure

```go
type WAL_DATA struct {
    LogSeqNo     uint64  // Monotonic sequence number
    Data         []byte  // Your application data
    Checksum     uint32  // CRC32 integrity check
    IsCheckpoint *bool   // Optional checkpoint flag
}
```

## 🏗️ Build System

This project uses **Bazel** as the primary build system for reproducible builds and dependency management.

### Build Targets

```bash
# Build everything
bazel build //...

# Build specific targets
bazel build //internal:wal_lib
bazel build //examples:main
bazel build //proto:wal_go_proto

# Run tests
bazel test //internal:wal_test

# Clean build artifacts
bazel clean
```

### Alternative: Go Build

For simpler development:

```bash
# Build example
go build -o bin/example examples/main.go

# Run tests
go test ./internal/...

# Generate protobuf (requires protoc)
make build-proto
```

### Code Style

- Follow standard Go conventions
- Add tests for new features
- Update documentation as needed
- Use `gofmt` for formatting

## 📋 Roadmap

- [ ] **Compression**: Add optional compression for segments
- [ ] **Encryption**: Support for encrypted WAL files
- [ ] **CLI Tools**: Command-line utilities for WAL inspection
- [ ] **LSM Tree Integration**: Support for LSM tree-based storage backends
- [ ] More coming soon...

## 📄 License

This project is licensed under the **GNU General Public License v3.0** - see the [LICENSE](LICENSE) file for details.

---

**Built with ❤️ in Go**