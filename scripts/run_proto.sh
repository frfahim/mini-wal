#!/bin/bash
set -e

PROTO_DIR="$(dirname "$0")/../proto"
# Ensure the output directory exists, we can automate the pb directory based on the proto file
OUT_DIR="$PROTO_DIR/wal_pb"
PROTO_FILE="$PROTO_DIR/wal.proto"

# Check for protoc installation
if ! command -v protoc >/dev/null 2>&1; then
  echo "Error: protoc is not installed. Please install Protocol Buffers compiler." >&2
  exit 1
fi

# Check for protoc-gen-go installation
if ! command -v protoc-gen-go >/dev/null 2>&1; then
  echo "Error: protoc-gen-go is not installed. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" >&2
  exit 1
fi

# Generate Go code from proto
protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$OUT_DIR" \
  --go_opt=paths=source_relative \
  "$PROTO_FILE"

echo "Go protobuf code generated in $OUT_DIR"
