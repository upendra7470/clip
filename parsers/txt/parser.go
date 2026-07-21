package txt

import (
	"context"
	"fmt"
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

// Parser implements the parser.Parser and parser.RangeParser interfaces for plain text files.
type Parser struct{}

// Parse reads the entire content of a text file and returns it unchanged.
// It ignores any selection criteria and returns the complete file content.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the file
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open TXT file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Read the entire file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		return parser.ParseResult{}, wrapError("file cannot be read", err)
	}

	// Validate UTF-8
	if !isValidUTF8(content) {
		return parser.ParseResult{}, wrapError("invalid UTF-8", nil)
	}

	return parser.ParseResult{
		Text: string(content),
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeTXT
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "lines"
}

// ParseRange extracts text from a specific line range in a text file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate line range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
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
	if start > len(lines) || end > len(lines) {
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
