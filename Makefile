# Clip Makefile
# Provides installation and build commands for the Clip CLI tool

.PHONY: all build install uninstall clean test

# Default target
all: build

# Build the Clip binary
build:
	@echo "Building Clip..."
	@go build -o clip ./cmd/clip
	@echo "✓ Built: ./clip"

# Install Clip globally
install:
	@echo "Installing Clip globally..."
	@go build -o clip ./cmd/clip
	@sudo mkdir -p /usr/local/bin
	@sudo cp clip /usr/local/bin/clip
	@sudo chmod +x /usr/local/bin/clip
	@rm clip
	@echo "✓ Installed: /usr/local/bin/clip"
	@echo "✓ You can now run 'clip' from anywhere"

# Uninstall Clip
uninstall:
	@echo "Uninstalling Clip..."
	@sudo rm -f /usr/local/bin/clip
	@echo "✓ Uninstalled Clip"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f clip
	@echo "✓ Cleaned"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...
	@echo "✓ Tests completed"

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@echo "✓ Code formatted"

# Show help
help:
	@echo "Clip Makefile Commands:"
	@echo ""
	@echo "  make build      - Build the Clip binary"
	@echo "  make install    - Install Clip globally to /usr/local/bin"
	@echo "  make uninstall  - Remove Clip from /usr/local/bin"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make test       - Run all tests"
	@echo "  make fmt        - Format Go code"
	@echo "  make help       - Show this help message"