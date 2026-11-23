package csv

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// CSVParserError represents an error that occurs during CSV parsing.
type CSVParserError struct {
	message string
	cause   error
}

func (e *CSVParserError) Error() string {
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
	if e.message == "" {
		return "CSV parser error"
	}
	return e.message
}

func (e *CSVParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser, parser.RangeParser, and parser.DocumentLister interfaces for CSV files.
type Parser struct{}

// NewParser creates a new CSV Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a CSV file and extracts readable text representation.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read csv: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "csv",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "csv file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "csv",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	file, err := os.Open(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, fmt.Errorf("file %s does not exist: %w", req.File, err)
		}
		return parser.ParseResult{}, fmt.Errorf("error reading file %s: %w", req.File, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, fmt.Errorf("error reading file %s: %w", req.File, err)
	}
	if info.Size() > parser.MaxFileSize {
		return parser.ParseResult{}, fmt.Errorf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize)
	}

	csvReader := csv.NewReader(file)
	var result strings.Builder
	written := 0
	for {
		if err := ctx.Err(); err != nil {
			return parser.ParseResult{}, err
		}
		record, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return parser.ParseResult{}, wrapError("invalid CSV format", readErr)
		}
		if written > 0 {
			result.WriteString("\n")
		}
		result.WriteString(strings.Join(record, ", "))
		written++
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in CSV", nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeCSV
}

// ListUnits implements the parser.DocumentLister interface for CSV files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	file, err := os.Open(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil, fmt.Errorf("file %s does not exist: %w", req.File, err)
		}
		return 0, nil, fmt.Errorf("error reading file %s: %w", req.File, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, nil, fmt.Errorf("error reading file %s: %w", req.File, err)
	}
	if info.Size() > parser.MaxFileSize {
		return 0, nil, fmt.Errorf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize)
	}

	reader := csv.NewReader(file)
	rowCount := 0
	for {
		if err := ctx.Err(); err != nil {
			return 0, nil, err
		}
		_, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, nil, wrapError("failed to parse CSV", readErr)
		}
		rowCount++
	}

	return rowCount, nil, nil
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitRows
}

// ParseRange extracts text from a specific row range in a CSV file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	// Validate row range
	if start < 1 || (end != -1 && end < 1) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("row numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid row range: start row must not be greater than end row (got %d-%d)", start, end), nil)
	}

	file, err := os.Open(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, fmt.Errorf("file %s does not exist: %w", req.File, err)
		}
		return parser.ParseResult{}, fmt.Errorf("error reading file %s: %w", req.File, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, fmt.Errorf("error reading file %s: %w", req.File, err)
	}
	if info.Size() > parser.MaxFileSize {
		return parser.ParseResult{}, fmt.Errorf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize)
	}

	// Stream rows and stop once the requested end is reached. This keeps a
	// small range fast even when the CSV contains millions of rows.
	csvReader := csv.NewReader(file)
	csvReader.LazyQuotes = true
	var result strings.Builder
	rowNumber := 0
	written := 0
	for {
		if err := ctx.Err(); err != nil {
			return parser.ParseResult{}, err
		}
		record, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return parser.ParseResult{}, wrapError("invalid CSV format", readErr)
		}
		rowNumber++
		if rowNumber < start {
			continue
		}
		if end != -1 && rowNumber > end {
			break
		}
		if written > 0 {
			result.WriteString("\n")
		}
		result.WriteString(strings.Join(record, ", "))
		written++
		if end != -1 && rowNumber == end {
			break
		}
	}

	if rowNumber < start || (end != -1 && rowNumber < end) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested row range exceeds CSV row count (file has %d rows, requested %d-%d)", rowNumber, start, end), nil)
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in rows %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &CSVParserError{
			message: message,
			cause:   nil,
		}
	}
	return &CSVParserError{
		message: message,
		cause:   err,
	}
}
