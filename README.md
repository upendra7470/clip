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
```

### Install to System

```bash
go install github.com/upendra7470/clip/cmd/clip@latest
```

## Build Instructions

```bash
# Build the CLI
go build -o clip ./cmd/clip

# Run tests
go test ./...
```

## Usage

### Basic Usage

```bash
# Show help
clip --help
clip -h

# Show version
clip --version

# Extract text from a file
clip extract file.ext --range 1-10:lines

# Extract text from a file using pages
clip extract file.ext --range 1-10:pages

# Extract text from a file using slides
clip extract file.ext --range 1-10:slides

# Extract text from a file using blocks
clip extract file.ext --range 1-10:blocks

# Extract text from a file using rows
clip extract file.ext --range 1-10:rows

# Extract text from a file using sections
clip extract file.ext --range 1-10:sections

# Extract text from a file using entries
clip extract file.ext --range 1-10:entries

# Extract text from a file using values
clip extract file.ext --range 1-10:values
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