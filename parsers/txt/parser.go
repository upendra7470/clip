package txt

import (
	"context"
	"os"
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

// Parser implements the parser.Parser interface for plain text files.
type Parser struct{}

// Parse reads the entire content of a text file and returns it unchanged.
// It ignores any selection criteria and returns the complete file content.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the file
	file, err := os.Open(req.File)
	if err != nil {
		return parser.ParseResult{}, wrapError("file cannot be opened", err)
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
