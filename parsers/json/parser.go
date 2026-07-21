package json

import (
	"context"
	"encoding/json"
	"fmt"
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

// Parser implements the parser.Parser and parser.RangeParser interfaces for JSON files.
type Parser struct{}

// Parse reads a JSON file and extracts readable text representation.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open JSON file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open JSON file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open JSON file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Check if file is empty
	if len(content) == 0 {
		return parser.ParseResult{}, wrapError("empty JSON file", nil)
	}

	// Validate JSON syntax
	var jsonData interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
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
func (p *Parser) GetRangeUnit() string {
	return "lines"
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
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open JSON file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open JSON file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open JSON file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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

	if text == "" {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no readable content found in lines %d-%d", start, end), nil)
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

	return strings.TrimSpace(result.String())
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
