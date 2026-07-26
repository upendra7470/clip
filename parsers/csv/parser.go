package csv

import (
	"context"
	"encoding/csv"
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
	if e.message == "" {
		return "CSV parser error"
	}
	return e.message
}

func (e *CSVParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for CSV files.
type Parser struct{}

// NewParser creates a new CSV Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a CSV file and extracts readable text representation.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
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
	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Parse CSV content
	csvReader := csv.NewReader(strings.NewReader(string(content)))
	records, err := csvReader.ReadAll()
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid CSV format", err)
	}

	// Extract readable text from CSV
	var result strings.Builder
	for i, record := range records {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(strings.Join(record, ", "))
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

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "rows"
}

// ParseRange extracts text from a specific row range in a CSV file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate row range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("row numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid row range: start row must not be greater than end row (got %d-%d)", start, end), nil)
	}

	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Parse CSV content
	csvReader := csv.NewReader(strings.NewReader(string(content)))
	records, err := csvReader.ReadAll()
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid CSV format", err)
	}

	// Validate range against actual row count
	if start > len(records) || end > len(records) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested row range exceeds CSV row count (file has %d rows, requested %d-%d)", len(records), start, end), nil)
	}

	// Extract only the requested row range
	var result strings.Builder
	for i := start - 1; i < end && i < len(records); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(strings.Join(records[i], ", "))
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
