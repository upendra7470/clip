package csv

import (
	"context"
	"encoding/csv"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser interface for CSV files.
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
