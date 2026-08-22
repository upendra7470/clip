# Clip Makefile
# Provides build, test, and installation commands for the Clip CLI tool

BIN_DIR := $(HOME)/.local/bin

.PHONY: all build install uninstall clean test fmt vet help

# Default target
all: build

# Build the Clip binary
build:
	@echo "Building Clip..."
	@go build -o clip ./cmd/clip
	@echo "Built: ./clip"

# Install Clip to ~/.local/bin
install: build
	@echo "Installing Clip to $(BIN_DIR)..."
	@mkdir -p $(BIN_DIR)
	@cp clip $(BIN_DIR)/clip
	@rm -f clip
	@echo "Installed: $(BIN_DIR)/clip"
	@echo "Make sure $(BIN_DIR) is on your PATH."

# Remove the installed binary
uninstall:
	@echo "Uninstalling Clip..."
	@rm -f $(BIN_DIR)/clip
	@echo "Uninstalled Clip"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f clip
	@echo "Cleaned"

# Run all tests
test:
	@echo "Running tests..."
	@go test ./...
	@echo "Tests completed"

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@echo "Code formatted"

# Run go vet
vet:
	@go vet ./...
	@echo "Vet passed"

# Show help
help:
	@echo "Clip Makefile Commands:"
	@echo ""
	@echo "  make build      - Build the Clip binary"
	@echo "  make install    - Install Clip to $$HOME/.local/bin"
	@echo "  make uninstall  - Remove the installed binary"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make test       - Run all tests"
	@echo "  make fmt        - Format Go code"
	@echo "  make vet        - Run go vet"
	@echo "  make help       - Show this help message"
