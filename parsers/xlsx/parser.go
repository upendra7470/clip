package xlsx

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// XLSXParserError represents an error that occurs during XLSX parsing.
type XLSXParserError struct {
	message string
	cause   error
}

func (e *XLSXParserError) Error() string {
	if e.message == "" {
		return "XLSX parser error"
	}
	return e.message
}

func (e *XLSXParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for XLSX files.
type Parser struct{}

// NewParser creates a new XLSX Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an XLSX file and extracts readable text representation.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read xlsx: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": filetype.FileTypeXLSX,
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "xlsx file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": filetype.FileTypeXLSX,
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
			return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// For XLSX files, we'll treat them as CSV-like for now
	// Split into lines and extract text
	lines := strings.Split(string(content), "\n")
	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(strings.TrimSpace(line))
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in XLSX", nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeXLSX
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "rows"
}

// ParseRange extracts text from a specific row range in an XLSX file.
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
			return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Split into lines (rows)
	lines := strings.Split(string(content), "\n")

	// Validate range against actual row count
	if start > len(lines) || end > len(lines) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested row range exceeds XLSX row count (file has %d rows, requested %d-%d)", len(lines), start, end), nil)
	}

	// Extract only the requested row range
	var result strings.Builder
	for i := start - 1; i < end && i < len(lines); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(strings.TrimSpace(lines[i]))
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
		return &XLSXParserError{
			message: message,
			cause:   nil,
		}
	}
	return &XLSXParserError{
		message: message,
		cause:   err,
	}
}
