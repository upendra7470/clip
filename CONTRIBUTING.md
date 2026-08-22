# Contributing to Clip

Thank you for considering a contribution! This document explains how to set up your environment and submit changes.

## Development Setup

```bash
git clone https://github.com/upendra7470/clip.git
cd clip
go build ./...   # verify everything compiles
make test        # run the test suite
```

Requirements: Go 1.26+ (see `go.mod`).

## Project Layout

```
cmd/clip/            CLI entry point
internal/parser/     Shared parser interfaces and range parsing
internal/security/   Safety limits (bounded reads, zip-bomb checks, etc.)
internal/registry/   File-type registry
parsers/<format>/    One package per supported format
tests/               Integration tests
```

## How to Add a New Format Parser

1. Create a new package under `parsers/<format>/`.
2. Implement the interfaces from `internal/parser` (`Parser`, plus `RangeParser` / `DocumentLister` if applicable). Look at `parsers/csv/parser.go` for a simple reference implementation.
3. Enforce the shared safety limits from `internal/security` (max file size, bounded reads).
4. Register the parser in `cmd/clip/main.go`.
5. Add unit tests in the same package covering: missing file, empty file, invalid content, successful extraction, and range extraction (including open-ended ranges like `2-`).

## Code Style

- Follow idiomatic Go; run `make fmt` before committing.
- Run `go vet ./...` and fix any findings.
- Keep files small and focused.
- Add or update tests for any behavior change. PRs with failing tests will not be merged.
- Do not add comments that merely restate the code.

## Commit Messages

Use short, descriptive messages in the imperative mood:

```
Fix xlsx error messages for missing files
Add ODS range validation
```

## Submitting Changes

1. Fork the repository and create a feature branch (`git checkout -b feature/my-feature`).
2. Make your changes with tests.
3. Ensure `make test`, `gofmt -l .`, and `go vet ./...` all pass locally.
4. Push to your fork and open a Pull Request describing what changed and why.

The CI pipeline builds and tests every PR on Linux, macOS, and Windows — please keep it green.

## Reporting Bugs

Please use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include:

- Your OS and Go version
- The command you ran
- The exact output or error message
- A sample file that reproduces the issue (if possible)

For security vulnerabilities, see [SECURITY.md](SECURITY.md) instead.
