<div align="center">

# Clip

### Any document. One command. Straight to your clipboard.

[![CI](https://github.com/upendra7470/clip/actions/workflows/ci.yml/badge.svg)](https://github.com/upendra7470/clip/actions/workflows/ci.yml)
[![Security](https://github.com/upendra7470/clip/actions/workflows/security.yml/badge.svg)](https://github.com/upendra7470/clip/actions/workflows/security.yml)
[![Release](https://img.shields.io/github/v/release/upendra7470/clip?display_name=tag)](https://github.com/upendra7470/clip/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-7c3aed.svg)](LICENSE)

**Windows · macOS · Linux** &nbsp;|&nbsp; **Offline OCR included** &nbsp;|&nbsp; **No runtime dependencies**

</div>

---

Clip is a fast, private command-line tool that extracts text from documents and images, then puts it on your clipboard. It understands the file automatically, can select only the pages or rows you need, and performs OCR locally—nothing is uploaded.

```console
$ clip report.pdf 5-10
✓ Found: report.pdf
✓ Extracted pages 5-10 successfully
✓ Copied to clipboard
```

## Install once

The official release is a single self-contained executable. It includes the OCR engine, recognition models, scanned-PDF support, and native clipboard integration.

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/upendra7470/clip/main/install.sh | sh
```

**Windows PowerShell**

```powershell
irm https://raw.githubusercontent.com/upendra7470/clip/main/install.ps1 | iex
```

That is the entire setup—no Tesseract, Poppler, Python, Java, Go, Homebrew, or Linux clipboard package is required. You can also download the single binary directly from [GitHub Releases](https://github.com/upendra7470/clip/releases/latest).

> macOS may ask you to confirm a downloaded command-line app the first time it runs. Clip supports Apple Silicon and Intel Macs; Linux supports x64 and ARM64; Windows supports x64.

## Use it

```sh
clip <file> [range]
```

```sh
clip contract.pdf              # every page
clip contract.pdf 4-9          # pages 4 through 9
clip "Q4 Review.pptx" 2-       # slide 2 to the end
clip customers.xlsx -20        # first 20 rows
clip receipt.png               # built-in offline OCR
clip scanned-document.pdf      # OCR fallback happens automatically
clip notes.txt 30-50           # only these lines
clip --list workbook.xlsx      # inspect available units
clip --ocr-status              # verify the bundled OCR engine
```

When only a filename is supplied, Clip can find it in the current directory, Downloads, Desktop, or Documents.

## Why Clip

- **One memorable command** for documents, spreadsheets, presentations, and images
- **Private by design**—all extraction and OCR happens on your computer
- **Truly portable**—a native binary for Windows, macOS, and Linux
- **Built-in OCR**—the engine and verified models ship inside the executable
- **Scanned PDF intelligence**—embedded text first, OCR automatically when needed
- **Precise ranges**—copy only the pages, slides, rows, lines, or sections you want
- **Fast on large files**—streaming and targeted extraction avoid unnecessary work
- **Defensive parsing**—bounded reads, archive limits, image limits, and OCR timeouts

## Supported formats

| Content | Extensions | Range unit |
|---|---|---|
| PDF | `.pdf` | pages |
| Word | `.docx` | sections |
| PowerPoint | `.ppt`, `.pptx` | slides |
| Excel / OpenDocument Sheets | `.xlsx`, `.ods` | rows |
| Text / Markdown | `.txt`, `.md` | lines / blocks |
| CSV | `.csv` | rows |
| OpenDocument Text / Rich Text | `.odt`, `.rtf` | paragraphs |
| JSON / XML / HTML / YAML | `.json`, `.xml`, `.html`, `.yaml`, `.yml` | entries / blocks / values |
| Images | `.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`, `.tif`, `.tiff`, `.webp` | offline OCR |

The current OCR model recognizes printed Latin-alphabet text. OCR quality depends on image resolution, contrast, orientation, and layout.

## How the self-contained build works

Official releases compile a native [ocrs](https://github.com/robertknight/ocrs) engine for each platform and embed it with pinned, checksum-verified models inside Clip. On first OCR use, those assets are unpacked into the user's private cache directory. PDF image extraction and clipboard communication are implemented in Go, so no helper application is needed.

| Platform | Release binary |
|---|---|
| Linux x64 | `clip_linux_amd64` |
| Linux ARM64 | `clip_linux_arm64` |
| macOS Intel | `clip_darwin_amd64` |
| macOS Apple Silicon | `clip_darwin_arm64` |
| Windows x64 | `clip_windows_amd64.exe` |

Checksums are published beside every release.

## Build and contribute

Building from source needs Go 1.26 or newer. Source builds intentionally use a placeholder OCR bundle; the release workflow builds and embeds the native engine and models.

```sh
git clone https://github.com/upendra7470/clip.git
cd clip
go test ./...
go build -o clip ./cmd/clip
```

See [CONTRIBUTING.md](CONTRIBUTING.md), the detailed [installation guide](INSTALLATION.md), [security policy](SECURITY.md), and [changelog](CHANGELOG.md).

## License

Clip is available under the [MIT License](LICENSE). Bundled OCR components retain their upstream MIT/Apache-2.0 notices.
