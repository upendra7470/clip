<div align="center">

# Clip

**Copy text from any document to your clipboard with one command.**

[![CI](https://github.com/upendra7470/clip/actions/workflows/ci.yml/badge.svg)](https://github.com/upendra7470/clip/actions/workflows/ci.yml)
[![Security Scan](https://github.com/upendra7470/clip/actions/workflows/security.yml/badge.svg)](https://github.com/upendra7470/clip/actions/workflows/security.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/upendra7470/clip.svg)](https://pkg.go.dev/github.com/upendra7470/clip)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

*Windows · macOS · Linux — no configuration required*

</div>

---

## Why Clip?

Extracting text from documents usually means opening heavyweight apps, selecting, copying, and cleaning up formatting along the way. **Clip** does it in a single step: it detects the file type, extracts the text (or just the range you want), and places it straight on your clipboard — ready to paste anywhere.

```bash
clip report.pdf          # whole document
clip report.pdf 5-10     # only pages 5-10
```

## Features

- **14 file formats** supported out of the box — see the [table](#supported-formats)
- **OCR for images** (PNG, JPEG, WebP) via local Tesseract — offline, private
- **Range extraction** with units that make sense per format (pages, slides, rows, sections, values)
- **Open-ended ranges** — `2-` for "from 2 to the end", `-5` for "the first 5"
- **Smart filename resolution** — searches your current directory, Downloads, Desktop, and Documents when given a bare filename
- **Safety hardened** against malicious documents: bounded reads, zip-bomb protection, XML token limits, RTF nesting caps
- **Cross-platform** — first-class support for Windows, macOS, and Linux
- **Zero configuration** — one binary, no config files

## Supported Formats

| Format | Extension(s) | Range Unit |
|--------|--------------|------------|
| PDF | `.pdf` | pages |
| Word | `.docx` | sections |
| PowerPoint | `.ppt`, `.pptx` | slides |
| OpenDocument Text | `.odt` | paragraphs |
| Rich Text | `.rtf` | paragraphs |
| Plain text | `.txt` | lines |
| Markdown | `.md` | blocks |
| CSV | `.csv` | rows |
| Excel | `.xlsx` | rows |
| OpenDocument Sheets | `.ods` | rows |
| JSON | `.json` | entries |
| XML / HTML | `.xml`, `.html` | entries / blocks |
| YAML | `.yaml`, `.yml` | values |
| Images (OCR) | `.png`, `.jpg`, `.jpeg`, `.webp` | lines |

## Requirements

- **None at runtime** — Clip is a single static binary.
- **Linux users:** a clipboard backend is required — `xclip` (X11) or `wl-copy` (Wayland). Install with `sudo apt install xclip`, `sudo dnf install xclip`, or your distribution's equivalent.
- **Image OCR:** requires a local [Tesseract](#ocr-image-text-extraction) installation; everything else works without it.
- **Go 1.26+** only if building from source.

## OCR (image text extraction)

Clip extracts visible text from images using **Tesseract**, running fully offline on your machine — no API keys, no cloud, no network access.

```bash
clip screenshot.png        # copy recognized text to the clipboard
clip receipt.jpg 1-3       # copy specific lines of recognized text
```

If an image contains no readable text (e.g. a photo of a car with no text), Clip reports `no text detected in image` instead of failing unexpectedly.

Install the OCR engine:

| Platform | Command |
|----------|---------|
| macOS | `brew install tesseract` |
| Debian / Ubuntu | `sudo apt install tesseract-ocr` |
| Fedora | `sudo dnf install tesseract` |
| Arch | `sudo pacman -S tesseract` |
| Windows | `winget install UB-Mannheim.TesseractOCR` |

Without Tesseract installed, image files produce a clear error explaining how to install it; all other formats work as usual.

## Installation

### Option 1 — Download a prebuilt binary (all platforms)

Grab the archive for your platform from the [**Releases**](https://github.com/upendra7470/clip/releases) page, then:

**macOS / Linux**

```bash
tar -xzf clip_*.tar.gz                  # extract
mkdir -p ~/.local/bin                   # ensure the dir exists
mv clip_* ~/.local/bin/clip             # move onto your PATH
chmod +x ~/.local/bin/clip
```

> On Apple Silicon download the `darwin_arm64` build; Intel Macs use `darwin_amd64`.

**Windows**

```powershell
Expand-Archive clip_*_windows_amd64.zip # extract
New-Item -ItemType Directory -Force "$HOME\.local\bin" | Out-Null
Move-Item clip_*_windows_amd64.exe "$HOME\.local\bin\clip.exe"
```

Then add `%USERPROFILE%\.local\bin` to your PATH (*Settings → System → About → Advanced system settings → Environment Variables*, or):

```powershell
[Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", "User") + ";$HOME\.local\bin", "User")
```

### Option 2 — Install with Go (all platforms)

If you have Go installed, this works identically on Windows, macOS, and Linux:

```bash
go install github.com/upendra7470/clip/cmd/clip@latest
```

The binary lands in your Go bin directory (`$(go env GOPATH)/bin`). Make sure it's on your PATH.

### Option 3 — Build from source (all platforms)

```bash
git clone https://github.com/upendra7470/clip.git
cd clip
go build -o clip ./cmd/clip        # on Windows use: go build -o clip.exe ./cmd/clip
```

On macOS/Linux you can also use the Makefile:

```bash
make build       # build into ./
make install     # install to ~/.local/bin
```

### Verify the installation

```bash
clip --version
# clip v1.0
```

<details>
<summary><b>PATH not set? (click to expand)</b></summary>

**macOS / Linux (bash/zsh)**

```bash
echo 'export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"' >> ~/.zshrc   # or ~/.bashrc
source ~/.zshrc
```

**Windows (PowerShell)**

```powershell
[Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", "User") + ";$HOME\go\bin", "User")
```
</details>

## Usage

```
clip <filename> [range]
```

| Command | What it does |
|---------|-------------|
| `clip document.pdf` | Extract everything → clipboard |
| `clip report.pdf 5-10` | Pages 5–10 → clipboard |
| `clip "Q4 Review.pptx" 2-5` | Slides 2–5 (quotes handle spaces) |
| `clip data.csv -3` | First 3 rows |
| `clip notes.txt 10-` | From line 10 to the end |
| `clip --list file.xlsx` | Show the document's range units |
| `clip --inspect file.docx` | Inspect document structure |
| `clip --help` / `--version` | Help / version info |

Clip prints what it did and confirms the copy:

```
$ clip notes.txt 1-2
✓ Found: notes.txt
✓ Extracted text successfully
✓ Copied to clipboard
```

### Uninstall

```bash
rm ~/.local/bin/clip        # macOS / Linux
Remove-Item "$HOME\.local\bin\clip.exe"   # Windows (PowerShell)
```

## Documentation

- [Installation guide](INSTALLATION.md) — detailed setup, PATH troubleshooting, smart resolution behavior
- [Contributing guide](CONTRIBUTING.md) — how to add a new format parser
- [Security policy](SECURITY.md) — how to report vulnerabilities
- [Changelog](CHANGELOG.md)

## Development

```bash
git clone https://github.com/upendra7470/clip.git
cd clip
make test    # run the full test suite
make vet     # static analysis
make fmt     # formatting
```

CI builds and tests every change on Ubuntu, macOS, and Windows; releases are automatically built for all major platforms when a `v*` tag is pushed.

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for details.
