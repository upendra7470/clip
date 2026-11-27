# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Clip 2.0 smart workflow with `--smart` OCR cleanup and Markdown output
- Structured `text`, `markdown`, `json`, and `jsonl` rendering
- Unicode-safe `--chunks` output for AI and RAG pipelines
- Deterministic directory processing with optional recursion and safety limits
- Bash, Zsh, Fish, and PowerShell completion generation
- Optional Windows Explorer, macOS Finder, and Linux file-manager integration templates
- Automatic Markdown table rendering for tabular parser output
- Latin-script OCR language hints and automatic recognition
- Self-contained offline OCR for PNG, JPEG, GIF, BMP, TIFF, and WebP images
- Automatic scanned-PDF OCR fallback with pure-Go image extraction
- `--ocr-status` option
- One-command installers for Windows, macOS, and Linux
- Native X11, Wayland, macOS, and Windows clipboard integration without helper programs
- Image dimension, pixel-count, and OCR timeout safety limits

### Improved

- PPTX range extraction inflates only requested slides and uses numeric slide ordering
- TXT, CSV, and XLSX range extraction streams requested content
- PDF text assembly uses amortized linear string building
- Linux clipboard handling retries X11 when Wayland copying fails

### Fixed

- Markdown beginning with `[` is no longer misclassified as JSON
- HTML beginning with `<` is no longer misclassified as XML
- `--list` and `--inspect` return a failure status when inspection fails

## [1.0] - 2026-08-22

### Added

- Text extraction and clipboard copying for 13 formats: PDF, DOCX, PPT, PPTX, ODT, RTF, TXT, Markdown, CSV, XLSX, ODS, JSON, XML, HTML, YAML
- Range extraction with per-format units (pages, slides, rows, sections, values) including open-ended ranges (`2-`, `-10`)
- Smart file resolution that searches common locations for bare filenames
- `--help`, `--version` flags
- Safety hardening: bounded reads, zip-bomb protection, XML token limits, RTF nesting-depth limits
- Cross-platform CI (Linux, macOS, Windows)

### Fixed

- Release binaries report their real version (`clip --version`) via build-time ldflags injection instead of a hardcoded constant
