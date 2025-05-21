PROTO_DIR=proto
PROTO_FILE=$(PROTO_DIR)/wal.proto
OUT_DIR=$(PROTO_DIR)

.PHONY: build-proto
build-proto:
	@command -v protoc >/dev/null 2>&1 || (echo "Error: protoc is not installed."; exit 1)
	@command -v protoc-gen-go >/dev/null 2>&1 || (echo "Error: protoc-gen-go is not installed. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; exit 1)
	protoc --proto_path=$(PROTO_DIR) --go_out=$(OUT_DIR) --go_opt=paths=source_relative $(PROTO_FILE)
	@echo "Go protobuf code generated in $(OUT_DIR)"