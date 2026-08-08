package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// JSONParserError represents an error that occurs during JSON parsing.
type JSONParserError struct {
	message string
	cause   error
}

func (e *JSONParserError) Error() string {
	if e.message == "" {
		return "JSON parser error"
	}
	return e.message
}

func (e *JSONParserError) Unwrap() error {
	return e.cause
}

// NewParser creates a new JSON Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for JSON files.
type Parser struct{}

// Parse reads a JSON file and extracts readable text representation.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read json: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "json",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "json file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "json",
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
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		return parser.ParseResult{}, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
	}

	// Check if file is empty
	if len(content) == 0 {
		return parser.ParseResult{}, wrapError("empty JSON file", nil)
	}

	// Validate JSON syntax and preserve number precision using UseNumber
	var jsonData interface{}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.UseNumber()
	if err := decoder.Decode(&jsonData); err != nil {
		return parser.ParseResult{}, wrapError("invalid JSON syntax", err)
	}

	// Extract readable text from JSON
	text := extractTextFromJSON(jsonData)

	if text == "" {
		return parser.ParseResult{}, wrapError("no readable content found in JSON", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeJSON
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitEntries
}

// ParseRange extracts text from a specific line range in a JSON file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate line range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid line range: start line must not be greater than end line (got %d-%d)", start, end), nil)
	}

	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		return parser.ParseResult{}, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
	}
	// Split into lines
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

	// For range extraction, return the raw text content without JSON validation
	// since partial JSON extracts are expected and valid for the use case
	text := result.String()
	if text == "" {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no content found in lines %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// extractTextFromJSON extracts readable text from JSON data structure
func extractTextFromJSON(data interface{}) string {
	var result strings.Builder

	switch v := data.(type) {
	case map[string]interface{}:
		extractFromObject(v, &result)
	case []interface{}:
		extractFromArray(v, &result)
	default:
		// Handle primitive values
		if s, ok := v.(string); ok {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(s)
		} else if num, ok := v.(json.Number); ok {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(num.String())
		} else if num, ok := v.(float64); ok {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			// Handle numbers (JSON numbers become float64)
			if num == float64(int(num)) {
				fmt.Fprintf(&result, "%d", int(num))
			} else {
				fmt.Fprintf(&result, "%f", num)
			}
		} else if b, ok := v.(bool); ok {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			fmt.Fprintf(&result, "%t", b)
		} else if v == nil {
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString("null")
		}
	}

	return result.String()
}

// extractFromObject extracts text from JSON object (values only, no keys)
func extractFromObject(obj map[string]interface{}, result *strings.Builder) {
	for _, value := range obj {
		switch v := value.(type) {
		case string:
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(v)
		case json.Number:
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(v.String())
		case float64:
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			// Handle numbers
			if v == float64(int(v)) {
				fmt.Fprintf(result, "%d", int(v))
			} else {
				fmt.Fprintf(result, "%f", v)
			}
		case bool:
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			fmt.Fprintf(result, "%t", v)
		case nil:
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString("null")
		case map[string]interface{}:
			// Nested object - recurse
			extractFromObject(v, result)
		case []interface{}:
			// Array - handle each element
			extractFromArray(v, result)
		default:
			if result.Len() > 0 {
				result.WriteString("\n")
			}
			result.WriteString(fmt.Sprintf("%v", v))
		}
	}
}

// extractFromArray extracts text from JSON array
func extractFromArray(arr []interface{}, result *strings.Builder) {
	for i, item := range arr {
		// Add newline if result already has content and this is not the first element
		if result.Len() > 0 && i == 0 {
			result.WriteString("\n")
		} else if i > 0 {
			result.WriteString("\n")
		}

		switch v := item.(type) {
		case string:
			result.WriteString(v)
		case json.Number:
			result.WriteString(v.String())
		case float64:
			// Handle numbers
			if v == float64(int(v)) {
				fmt.Fprintf(result, "%d", int(v))
			} else {
				fmt.Fprintf(result, "%f", v)
			}
		case bool:
			fmt.Fprintf(result, "%t", v)
		case nil:
			result.WriteString("null")
		case map[string]interface{}:
			// Nested object in array
			extractFromObject(v, result)
		case []interface{}:
			// Nested array
			extractFromArray(v, result)
		default:
			result.WriteString(fmt.Sprintf("%v", v))
		}
	}
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &JSONParserError{
			message: message,
			cause:   nil,
		}
	}
	return &JSONParserError{
		message: message,
		cause:   err,
	}
}
