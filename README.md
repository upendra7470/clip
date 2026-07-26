# Clip

A fast, cross-platform command-line utility that extracts text from documents and copies it directly to the user's clipboard using one consistent command regardless of document type.

## Problem Statement

Working with documents across different formats (PDF, DOCX, images, etc.) often requires multiple tools and manual copy-pasting. Clip aims to simplify this workflow with a unified interface.

## Vision

Clip will become the go-to tool for developers, researchers, and professionals who need to quickly extract text from various document formats without switching between different applications.

## Features

### Current Features (Phase 1)
- Basic CLI structure
- Help and version flags
- Clean project foundation

### Planned Features
- Support for multiple document formats (PDF, DOCX, images with OCR, etc.)
- Direct clipboard copying
- Text extraction with formatting preservation
- Batch processing
- Custom output formatting
- Plugin system for additional formats

## Installation

### From Source

```bash
git clone https://github.com/upendra7470/clip.git
cd clip
go build -o clip ./cmd/clip
sudo cp clip /usr/local/bin/clip
```

### Install to System

```bash
go install github.com/upendra7470/clip/cmd/clip@latest
# Verify installation
clip --version
```

# Verify installation
clip --version

## Troubleshooting

### PATH Issues

If the `clip` command is not recognized after installation, ensure that your PATH environment variable includes the directory where the binary was installed. For Go installations, this is typically `$GOPATH/bin`.

```bash
# Check if the binary is in your PATH
echo $PATH | grep $GOPATH/bin

# If not, add it to your PATH
```

## Build Instructions

```bash
# Build the CLI
go build -o clip ./cmd/clip

# Run tests
go test ./...

# Install to system
sudo cp clip /usr/local/bin/clip
```

## Usage

```bash
# Show help
clip --help

# Show version
clip --version

# Extract text from a file
clip report.pdf

# Extract text from a file with a range
clip report.pdf 5-10
clip "The Brain.docx" 1-3
clip presentation.pptx 2-5
clip spreadsheet.xlsx 2-10
```

### Future Usage Examples

```bash
# Extract text from a PDF and copy to clipboard
clip extract document.pdf

# Extract text from multiple files
clip extract file1.pdf file2.docx

# Extract text and save to file
clip extract document.pdf --output output.txt

# Extract text with specific format
clip extract document.pdf --format plain
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin feature/your-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/upendra7470/clip.git
cd clip

# Build and test
go build -o clip ./cmd/clip
go test ./...
```

### Code Quality

- Follow idiomatic Go
- Keep files small and focused
- Write clear, maintainable code
- Add tests for new functionality
- Keep the codebase clean and organized