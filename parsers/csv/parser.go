package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser and parser.RangeParser interfaces for CSV files.
type Parser struct{}

// Parse reads a CSV file and extracts text content.
// It uses the standard library encoding/csv package for parsing.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the CSV file
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Create CSV reader
	csvReader := csv.NewReader(file)

	// Read all records
	records, err := csvReader.ReadAll()
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse CSV", err)
	}

	if len(records) == 0 {
		return parser.ParseResult{}, wrapError("no content found in CSV", nil)
	}

	// Convert CSV data to text format
	var result strings.Builder
	for i, record := range records {
		if i > 0 {
			result.WriteString("\n")
		}
		// Join fields with commas (preserve CSV structure)
		result.WriteString(strings.Join(record, ", "))
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

	// Open the CSV file
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open CSV file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Create CSV reader
	csvReader := csv.NewReader(file)

	// Read all records
	records, err := csvReader.ReadAll()
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse CSV", err)
	}

	if len(records) == 0 {
		return parser.ParseResult{}, wrapError("no content found in CSV", nil)
	}

	// Validate range against actual row count
	if start > len(records) || end > len(records) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested row range exceeds CSV row count (CSV has %d rows, requested %d-%d)", len(records), start, end), nil)
	}

	// Convert CSV data to text format for the requested row range
	var result strings.Builder
	for i := start - 1; i < end && i < len(records); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		// Join fields with commas (preserve CSV structure)
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
