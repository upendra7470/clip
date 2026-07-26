package txt

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// TextParserError represents an error that occurs during text file parsing.
type TextParserError struct {
	message string
	cause   error
}

func (e *TextParserError) Error() string {
	if e.message == "" {
		return "text parser error"
	}
	return e.message
}

func (e *TextParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser, parser.RangeParser, and parser.DocumentLister interfaces for plain text files.
type Parser struct{}

// Parse implements the parser.Parser interface method for reading from io.Reader
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read text: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "text",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "text file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "text",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the file
	file, err := os.Open(req.File)
	if err != nil {
		return parser.ParseResult{}, wrapError("Could not open text file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return parser.ParseResult{}, wrapError("Could not open text file:\n"+req.File+"\n\nReason:\npermission denied", err)
	}

	// If selection criteria is specified, process it (though text parser ignores it)
	if req.Selection.Pages != "" || req.Selection.Range != "" || req.Selection.Query != "" {
		// Text parser ignores selection criteria and returns full content
	}

	// Return the file content as a string
	return parser.ParseResult{Text: string(content)}, nil
}

// NewParser creates a new TXT Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeTXT
}

// ListUnits implements the parser.DocumentLister interface for TXT files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return 0, nil, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return 0, nil, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")
	lineCount := len(lines)

	// Return line numbers as unit names
	var lineNumbers []string
	for i := 0; i < lineCount; i++ {
		lineNumbers = append(lineNumbers, fmt.Sprintf("Line %d", i+1))
	}

	return lineCount, lineNumbers, nil
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitLines
}

// ParseRange extracts text from a specific line range in a text file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate line range
	if start < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	// Allow end to be -1 (indicating "to end")
	if end != -1 && end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid line range: start line must not be greater than end line (got %d-%d)", start, end), nil)
	}

	// Read the entire file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Validate UTF-8
	if !isValidUTF8(content) {
		return parser.ParseResult{}, wrapError("invalid UTF-8", nil)
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")

	// Validate range against actual line count
	if start > len(lines) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested line range exceeds file line count (file has %d lines, requested %d-%d)", len(lines), start, end), nil)
	}

	// If end is -1, it means "to end"
	if end == -1 {
		end = len(lines)
	} else if end > len(lines) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested line range exceeds file line count (file has %d lines, requested %d-%d)", len(lines), start, end), nil)
	}

	// Extract only the requested line range
	var result strings.Builder
	for i := start - 1; i < end && i < len(lines); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(lines[i])
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in lines %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// isValidUTF8 checks if the byte slice contains valid UTF-8.
func isValidUTF8(b []byte) bool {
	return utf8.Valid(b)
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &TextParserError{
			message: message,
			cause:   nil,
		}
	}
	return &TextParserError{
		message: message,
		cause:   err,
	}
}

// ExtractLines extracts lines from txt content based on the given range
func (p *Parser) ExtractLines(content string, start, end int) (string, error) {
	// Split into lines
	lines := strings.Split(content, "\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("line numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid line range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(lines) {
		return "", nil // Out of range returns empty
	}
	if end > len(lines) {
		end = len(lines)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(lines); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(lines[i])
	}

	return result.String(), nil
}
