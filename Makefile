.PHONY: all build test lint gen proto init clean help

# Binary output directory
BIN_DIR := bin

# Source command directory
CMD_DIR := cmd

# Get all subdirectories in cmd/
CMDS := $(notdir $(wildcard $(CMD_DIR)/*))

# Default target
all: gen lint test build

# Initialize environment
init:
	@echo "Installing dependencies..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go get -tool github.com/mazrean/kessoku/cmd/kessoku
	@go get -tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
	@go get -tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
	@go get -tool google.golang.org/protobuf/cmd/protoc-gen-go
	@go get -tool google.golang.org/grpc/cmd/protoc-gen-go-grpc

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@protoc \
		--plugin=protoc-gen-go=$$(go tool -n protoc-gen-go) \
		--plugin=protoc-gen-go-grpc=$$(go tool -n protoc-gen-go-grpc) \
		--plugin=protoc-gen-grpc-gateway=$$(go tool -n protoc-gen-grpc-gateway) \
		--go_out=internal/pb --go_opt=paths=source_relative \
		--go-grpc_out=internal/pb --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=internal/pb --grpc-gateway_opt=paths=source_relative \
		-I proto \
		-I /opt/homebrew/include \
		proto/post/v1/post.proto

# Generate code
gen: proto
	@echo "Generating code..."
	@go generate ./...

# Build all binaries from cmd/
build:
	@echo "Building binaries..."
	@mkdir -p $(BIN_DIR)
	@for cmd in $(CMDS); do \
		echo "  Building $$cmd..."; \
		go build -o $(BIN_DIR)/$$cmd ./$(CMD_DIR)/$$cmd; \
	done

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Lint and format code
lint:
	@echo "Formatting code..."
	@go fmt ./...
	@go vet ./...
	@if command -v goimports >/dev/null 2>&1; then \
		echo "Running goimports..."; \
		goimports -w -local arktie.org $$(find . -type f -name '*.go' -not -path './ent/*' -not -path './**/mock_*' -not -path './**/kessoku_band.go'); \
	else \
		echo "goimports not found, skipping..."; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)

# Help message
help:
	@echo "Makefile targets:"
	@echo "  all     - Run gen, lint, test and build (default)"
	@echo "  init    - Install development tools (goimports)"
	@echo "  gen     - Run go generate ./..."
	@echo "  build   - Build all binaries in $(CMD_DIR)/ to $(BIN_DIR)/"
	@echo "  test    - Run all tests"
	@echo "  lint    - Format code with gofmt and goimports (if available)"
	@echo "  clean   - Remove built binaries"
	@echo "  help    - Show this help message"
