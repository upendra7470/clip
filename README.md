<div align="center">

# Clip

### Any document. One command. Straight to your clipboard.

Clip is a fast, private, cross-platform CLI that extracts useful text from documents, spreadsheets, presentations, images, and scanned PDFs.

[![CI](https://github.com/upendra7470/clip/actions/workflows/ci.yml/badge.svg)](https://github.com/upendra7470/clip/actions/workflows/ci.yml)
[![Security](https://github.com/upendra7470/clip/actions/workflows/security.yml/badge.svg)](https://github.com/upendra7470/clip/actions/workflows/security.yml)
[![Latest release](https://img.shields.io/github/v/release/upendra7470/clip?display_name=tag&sort=semver)](https://github.com/upendra7470/clip/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-7c3aed.svg)](LICENSE)

**Windows · macOS · Linux** &nbsp;|&nbsp; **Offline OCR** &nbsp;|&nbsp; **No runtime dependencies**

[Install](#install) · [Quick start](#quick-start) · [Features](#why-clip) · [Command reference](#command-reference) · [Formats](#supported-formats) · [Roadmap](#roadmap)

</div>

---

```console
$ clip report.pdf 5-10
✓ Found: report.pdf
✓ Extracted pages 5-10 successfully
✓ Copied to clipboard
```

Clip detects the file type automatically, understands document-specific ranges, and performs OCR locally. Your files and extracted text never leave your computer.

> [!IMPORTANT]
> The latest stable release is **v2.0.0**. The next capabilities are being reviewed in the [v2.1–v2.7 pull requests](#roadmap). Commands marked **Upcoming** below become available after those releases are merged and published.

## Why Clip

- **One memorable command** for documents, spreadsheets, presentations, images, and scans.
- **Private by design**—extraction and OCR run locally, with no upload or cloud API.
- **Self-contained releases**—OCR models, PDF processing, and native clipboard support ship in one executable.
- **Precise extraction**—copy only the pages, slides, rows, lines, sections, or entries you need.
- **Scanned-PDF intelligence**—combine native text and OCR page by page.
- **Structured output**—produce plain text, Markdown, JSON, JSONL, or Unicode-safe AI chunks.
- **Cross-platform**—native binaries for Windows, macOS, Linux x64, and Linux ARM64.
- **Defensive parsing**—bounded reads, archive limits, image limits, and OCR timeouts protect the host.

## Install

Official releases are single, self-contained executables. You do **not** need Tesseract, Poppler, Python, Java, Go, Homebrew, or a separate Linux clipboard utility.

### macOS and Linux

```sh
curl -fsSL https://raw.githubusercontent.com/upendra7470/clip/main/install.sh | sh
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/upendra7470/clip/main/install.ps1 | iex
```

Open a new terminal if `clip` is not immediately found, then verify the installation:

```sh
clip --version
clip --ocr-status
```

Prefer a manual installation? Download the correct binary and its checksum from [GitHub Releases](https://github.com/upendra7470/clip/releases/latest). See the [complete installation guide](INSTALLATION.md) for custom locations, `PATH` setup, checksum verification, and uninstall instructions.

## Quick start

The basic form is:

```sh
clip <file> [range]
```

```sh
# Documents and ranges
clip contract.pdf
clip contract.pdf 4-9
clip "Q4 Review.pptx" 2-
clip customers.xlsx -20
clip notes.txt 30-50

# Images and scanned PDFs
clip receipt.png
clip scanned-contract.pdf

# Structured or file output
clip --smart scan.png
clip --format markdown --stdout spreadsheet.xlsx
clip --format jsonl --chunks 1200 --output handbook.jsonl handbook.pdf
clip --recursive --output knowledge.md ./documents

# Inspection and setup
clip --list workbook.xlsx
clip --ocr-status
clip --completion zsh
```

When only a filename is supplied, Clip searches the current directory and common locations such as Downloads, Desktop, and Documents.

### Range syntax

Ranges use the natural unit for each format:

| Syntax | Meaning | Example |
|---|---|---|
| `5` | One unit | Page 5 |
| `5-10` | Inclusive range | Pages 5 through 10 |
| `5-` | From a unit to the end | Slide 5 onward |
| `-10` | From the beginning to a unit | First 10 rows |

## Command reference

### Stable in v2.0.0

| Option | Purpose |
|---|---|
| `--smart` | Improve OCR input and produce structure-preserving Markdown |
| `--format text\|markdown\|json\|jsonl` | Select the output representation |
| `--chunks N` | Split output into Unicode-safe chunks of approximately `N` characters |
| `--output <file>` | Write extracted content to a file instead of the clipboard |
| `--stdout` | Write extracted content to standard output |
| `--recursive` | Process supported files in subdirectories |
| `--lang <hint>` | Use `auto`, `latin`, `en`, `fr`, `de`, `es`, `it`, `pt`, or `nl` |
| `--list`, `--inspect` | Show document units such as pages, slides, rows, or sections |
| `--ocr-status` | Verify that the bundled OCR engine is ready |
| `--completion <shell>` | Generate completion for Bash, Zsh, Fish, or PowerShell |
| `--version` | Print the installed Clip version |
| `--help` | Display CLI help |

### Upcoming v2.1–v2.7

| Option | Purpose | Planned release |
|---|---|---|
| `--image <file>` | Run explicit layout-aware OCR on an image | v2.1 |
| `--pdf <file>` | Extract a PDF while preserving its logical layout | v2.1 |
| `--layout auto\|plain\|exact` | Choose paragraph-aware, simple, or spatial reconstruction | v2.1 |
| `--screen` | Select a screen region, OCR it offline, and copy the result | v2.2 |
| `--rich` | Copy safe formatted HTML and a plain-text fallback together | v2.3 |
| `--make-searchable <pdf>` | Preserve the PDF and add an invisible OCR text layer | v2.3 |
| `--watch` | Continuously OCR images copied to the clipboard | v2.4 |
| `--clipboard` | OCR the image currently on the clipboard once | v2.5 |
| `--auto` | Detect and process a clipboard image, local path, public HTTPS document, or text | v2.6 |
| `--enhance` | Run adaptive, multi-pass OCR for difficult, rotated, skewed, or low-contrast scans | v2.7 |

Preview examples from the open release pull requests:

```sh
clip --image receipt.png
clip --pdf --layout exact report.pdf
clip --screen
clip --screen --format markdown --stdout
clip --rich --pdf report.pdf
clip --make-searchable scanned.pdf
clip --make-searchable --output archive.pdf scanned.pdf
clip --watch
clip --watch --smart --rich
clip --clipboard
clip --clipboard --smart --stdout
clip --auto
clip --auto --smart --format jsonl --chunks 1200 --stdout
clip --enhance difficult-scan.png
clip --screen --enhance --layout exact
```

`--make-searchable` defaults to `<name>.searchable.pdf`, skips pages that already contain native text, and never overwrites an existing output file.

## Output workflows

Clipboard output remains the default. Clip can also feed terminals, scripts, knowledge bases, and AI pipelines without intermediate tools.

```sh
# Human-readable Markdown
clip --format markdown --stdout report.pdf

# One JSON document envelope
clip --format json --output report.json report.pdf

# Stream-friendly AI/RAG chunks
clip --format jsonl --chunks 1200 --stdout handbook.pdf

# Deterministic directory corpus
clip --smart --recursive --output corpus.md ./documents
```

Tabular parser output is converted to Markdown tables automatically. Directory batches are sorted deterministically, ignore unsupported files, and stop at a 1,000-file safety limit.

## Supported formats

| Content | Extensions | Range unit |
|---|---|---|
| PDF | `.pdf` | Pages |
| Microsoft Word | `.docx` | Sections |
| Microsoft PowerPoint | `.ppt`, `.pptx` | Slides |
| Excel / OpenDocument Sheets | `.xlsx`, `.ods` | Rows |
| Text / Markdown | `.txt`, `.md` | Lines / blocks |
| CSV | `.csv` | Rows |
| OpenDocument Text / Rich Text | `.odt`, `.rtf` | Paragraphs |
| JSON | `.json` | Entries |
| XML / HTML | `.xml`, `.html`, `.htm` | Entries / blocks |
| YAML | `.yaml`, `.yml` | Values |
| Images | `.png`, `.jpg`, `.jpeg`, `.gif`, `.bmp`, `.tif`, `.tiff`, `.webp` | Offline OCR |

The bundled model recognizes printed Latin-script text. OCR quality varies with resolution, contrast, orientation, typography, and layout. Additional writing systems require future model packs and are not claimed by the current release.

## Platform support

| Platform | Release binary | Notes |
|---|---|---|
| Windows x64 | `clip_windows_amd64.exe` | Native clipboard; no helper utility |
| macOS Apple Silicon | `clip_darwin_arm64` | Native clipboard; Screen Recording permission for upcoming screen OCR |
| macOS Intel | `clip_darwin_amd64` | Native clipboard; Screen Recording permission for upcoming screen OCR |
| Linux x64 | `clip_linux_amd64` | Native X11 and Wayland clipboard support |
| Linux ARM64 | `clip_linux_arm64` | Native X11 and Wayland clipboard support |

For upcoming `clip --screen`, macOS may request Screen Recording permission, Wayland displays its standard desktop-portal dialog, and Windows/X11 show a temporary selection overlay. Clip stores the capture with private permissions and removes it immediately after OCR.

## Privacy and security

Clip is designed for local and potentially untrusted documents:

- No document content, screenshots, OCR text, or telemetry is uploaded.
- Installers and release artifacts use SHA-256 verification.
- Official binaries include GitHub build-provenance attestations.
- File size, decompressed archive size, XML depth, image dimensions, pixel count, and OCR duration are bounded.
- Searchable-PDF output uses private temporary files and refuses accidental overwrites.
- Rich clipboard rendering omits raw HTML from extracted documents.
- Dependency vulnerabilities are checked before release.

Verify an official release with the GitHub CLI:

```sh
gh attestation verify ./clip_linux_amd64 --repo upendra7470/clip
```

Please report vulnerabilities privately according to the [security policy](SECURITY.md).

## How the self-contained build works

The release workflow compiles the pinned [ocrs](https://github.com/robertknight/ocrs) engine for each platform and embeds it with checksum-verified recognition models. On first OCR use, Clip extracts those assets into a private user cache. PDF processing and clipboard communication are implemented in the application, so no external converter, OCR service, or clipboard program is needed.

## Shell and file-manager integration

Generate shell completion without installing another package:

```sh
clip --completion bash
clip --completion zsh
clip --completion fish
clip --completion powershell
```

Optional Windows Explorer, macOS Finder, and Linux file-manager templates are available in [`integrations/`](integrations/).

## Clipboard Watch OCR

Upcoming in v2.4, `clip --watch` turns the clipboard into an always-ready OCR inbox:

```sh
clip --watch
```

Copy a screenshot or image from any application. Clip recognizes it locally, replaces the image with usable text, and waits for the next image. Press `Ctrl+C` to stop.

- Repeated clipboard events are deduplicated.
- Password-manager content marked as sensitive is never OCRed.
- Each temporary image uses private permissions and is deleted immediately.
- Invalid images are reported without stopping the watcher.
- `--smart`, `--layout`, `--format`, `--rich`, and `--stdout` remain available.

For a one-shot workflow, upcoming v2.5 adds:

```sh
clip --clipboard
```

Copy an image first, run the command, and Clip recognizes the current clipboard image once before exiting. This is ideal for shell aliases, launchers, shortcuts, and automation that should not leave a watcher running. Sensitive content is refused before its bytes are read.

## Universal Clipboard Input

Upcoming in v2.6, one command chooses the pipeline for you:

```sh
clip --auto
```

Copy any supported input first:

- an image is recognized with the local OCR engine;
- a local file or directory path is sent through Clip's document pipeline;
- a public HTTPS document is downloaded privately, extracted locally, and deleted immediately;
- ordinary text can be converted to Markdown, JSON, JSONL, rich clipboard content, or AI-ready chunks.

All existing output controls compose with universal input:

```sh
clip --auto --smart --rich
clip --auto --format json --stdout
clip --auto --format jsonl --chunks 1200 --output context.jsonl
```

Sensitive clipboard items are rejected before their contents are read. Remote input is HTTPS-only, rejects credentials and non-public network addresses, validates every redirect, applies the global 50 MiB limit, and uses a private temporary directory.

## OCR Intelligence

Upcoming in v2.7, `--enhance` gives difficult scans an adaptive, multi-pass recognition pipeline:

```sh
clip --enhance faded-receipt.jpg
clip --pdf --enhance --layout exact archive.pdf
clip --screen --enhance --rich
clip --auto --enhance --format json --stdout
```

Clip produces contrast-normalized and automatically binarized candidates, handles light text on dark backgrounds, estimates and corrects small skew, tests all four orientations, and scores the OCR results for readable words rather than accepting the first pass. The strongest result continues through the selected layout and output renderer.

Enhancement works with image files, scanned PDFs, screen capture, one-shot clipboard OCR, clipboard watch, searchable-PDF creation, and universal input. Processing stays local and temporary candidates are deleted after each operation.

## Roadmap

The next releases are intentionally kept as focused, independently reviewable pull requests:

| Version | Update | Pull request |
|---|---|---|
| v2.1 | Layout-aware image OCR and PDF extraction | [#17](https://github.com/upendra7470/clip/pull/17) |
| v2.2 | Instant offline screen-region OCR | [#18](https://github.com/upendra7470/clip/pull/18) |
| v2.3 | Rich clipboard copy and searchable-PDF export | [#19](https://github.com/upendra7470/clip/pull/19) |
| v2.4 | Continuous clipboard image OCR | [#20](https://github.com/upendra7470/clip/pull/20) |
| v2.5 | Instant one-shot clipboard OCR | [#21](https://github.com/upendra7470/clip/pull/21) |
| v2.6 | Universal, security-hardened clipboard input | [#22](https://github.com/upendra7470/clip/pull/22) |
| v2.7 | Adaptive multi-pass OCR intelligence | [#23](https://github.com/upendra7470/clip/pull/23) |

These changes are stacked and should be merged in version order. See the [v2 specification](docs/V2.md), [v2.1 design](docs/V2.1.md), [v2.2 design](docs/V2.2.md), [v2.4 design](docs/V2.4.md), [v2.5 design](docs/V2.5.md), [v2.6 design](docs/V2.6.md), [v2.7 design](docs/V2.7.md), and [changelog](CHANGELOG.md) for details.

## Build and contribute

Source builds require Go 1.26 or newer:

```sh
git clone https://github.com/upendra7470/clip.git
cd clip
go test ./...
go build -o clip ./cmd/clip
```

Source builds contain a placeholder OCR bundle. The release workflow compiles and embeds the native OCR engine and models, so use an official release when testing the zero-dependency OCR experience.

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) and the [Code of Conduct](CODE_OF_CONDUCT.md) before opening a pull request.

## License

Clip is available under the [MIT License](LICENSE). Bundled OCR components retain their upstream MIT and Apache-2.0 notices.
