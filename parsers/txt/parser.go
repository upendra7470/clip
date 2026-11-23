package txt

import (
	"bufio"
	"context"
	"errors"
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
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
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
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
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
	limitedReader := io.LimitReader(file, parser.MaxFileSize)
	content, err := io.ReadAll(limitedReader)
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
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		return 0, nil, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}
	if len(content) > parser.MaxFileSize {
		return 0, nil, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
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
	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	file, err := os.Open(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		return parser.ParseResult{}, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}
	if info.Size() > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
	}

	// Handle sentinel values
	if start == -1 {
		start = 1 // Start from beginning
	}

	// Validate line range
	if start < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid line range: start line must not be greater than end line (got %d-%d)", start, end), nil)
	}

	// Stream lines and stop when a bounded range is complete.
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), parser.MaxFileSize)
	var result strings.Builder
	lineNumber := 0
	written := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return parser.ParseResult{}, err
		}
		lineNumber++
		if lineNumber < start {
			continue
		}
		if end != -1 && lineNumber > end {
			break
		}
		if !isValidUTF8(scanner.Bytes()) {
			return parser.ParseResult{}, wrapError("invalid UTF-8", nil)
		}
		if written > 0 {
			result.WriteString("\n")
		}
		result.Write(scanner.Bytes())
		written++
		if end != -1 && lineNumber == end {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return parser.ParseResult{}, wrapError("error reading text file", err)
	}
	if lineNumber < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested line range exceeds file line count (file has %d lines, requested %d-%d)", lineNumber, start, end), nil)
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
