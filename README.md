# Clip

A fast, cross-platform command-line utility that extracts text from documents and copies it directly to your clipboard — one consistent command, regardless of document type.

## Why Clip?

Working with documents across different formats (PDF, DOCX, XLSX, ...) usually means juggling multiple apps just to grab a snippet of text. Clip gives you a single command that opens the file, extracts the text you need — optionally from a specific range of pages, slides, rows, or sections — and puts it straight on your clipboard.

## Features

- **13 document formats** supported out of the box
- **Range extraction** — copy pages 5-10 of a PDF or slides 2-5 of a deck
- **Smart file resolution** — searches common locations when given only a filename
- **Safety hardened** — bounded file reads, zip-bomb protection, nesting-depth limits
- **Cross-platform** — Linux, macOS, and Windows
- **Zero configuration** — no config files required

## Supported Formats

| Format    | Range Unit |
|-----------|------------|
| PDF       | pages      |
| DOCX      | sections   |
| PPT / PPTX| slides     |
| ODT       | paragraphs |
| RTF       | paragraphs |
| TXT       | lines      |
| Markdown  | blocks     |
| CSV       | rows       |
| XLSX      | rows       |
| ODS       | rows       |
| JSON      | entries    |
| XML / HTML| entries / blocks |
| YAML      | values     |

## Installation

### From Source

```bash
git clone https://github.com/upendra7470/clip.git
cd clip
go build -o clip ./cmd/clip
sudo cp clip /usr/local/bin/clip   # or move it anywhere on your PATH
```

### Using go install

```bash
go install github.com/upendra7470/clip/cmd/clip@latest
```

Verify the installation:

```bash
clip --version
```

If `clip` is not recognized after installing with `go install`, make sure `$GOPATH/bin` (usually `$HOME/go/bin`) is in your `PATH`.

## Usage

```bash
# Extract everything and copy to clipboard
clip report.pdf

# Extract a range (pages 5-10) and copy to clipboard
clip report.pdf 5-10

# Filenames with spaces work when quoted; ranges are detected automatically
clip "The Brain.docx" 1-3
clip presentation.pptx 2-5
clip spreadsheet.xlsx 2-10

# Open-ended ranges: from value 2 to the end
clip data.yaml 2-

# Show help / version
clip --help
clip --version
```

Clip copies the extracted text to your clipboard by default so you can paste it anywhere immediately.

## Development

```bash
git clone https://github.com/upendra7470/clip.git
cd clip

make build    # build the CLI
make test     # run all tests
make fmt      # format code
make vet      # run go vet
make help     # list all commands
```

The project is plain Go with no code generation. Parsers live in `parsers/<format>/` and share safety utilities from `internal/security`.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines, including how to add a new format parser.

## Security

Clip processes untrusted documents, so the parsers enforce defensive limits (bounded reads, zip-bomb checks, XML token limits, RTF nesting-depth caps). If you discover a security vulnerability, please follow the process in [SECURITY.md](SECURITY.md) rather than opening a public issue.

## Roadmap

- OCR support for images
- Batch processing of multiple files
- Output to stdout / files (`--output`)
- Plugin system for additional formats

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) to get started.
