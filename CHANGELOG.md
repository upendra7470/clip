# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- OCR support for image files (PNG, JPEG, WebP): `clip image.png` extracts visible text with local Tesseract and copies it to the clipboard
- Image safety limits: header-only dimension/pixel checks guard against decompression bombs before OCR runs

## [1.0.0] - 2026-08-22

### Added

- Text extraction and clipboard copying for 13 formats: PDF, DOCX, PPT, PPTX, ODT, RTF, TXT, Markdown, CSV, XLSX, ODS, JSON, XML, HTML, YAML
- Range extraction with per-format units (pages, slides, rows, sections, values) including open-ended ranges (`2-`, `-10`)
- Smart file resolution that searches common locations for bare filenames
- `--help`, `--version` flags
- Safety hardening: bounded reads, zip-bomb protection, XML token limits, RTF nesting-depth limits
- Cross-platform CI (Linux, macOS, Windows)
