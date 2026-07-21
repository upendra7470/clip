package yaml

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"gopkg.in/yaml.v3"
)

// YAMLParserError represents an error that occurs during YAML parsing.
type YAMLParserError struct {
	message string
	cause   error
}

func (e *YAMLParserError) Error() string {
	if e.message == "" {
		return "YAML parser error"
	}
	return e.message
}

func (e *YAMLParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser interface for YAML files.
type Parser struct{}

// Parse reads a YAML file and extracts readable text representation.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open YAML file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open YAML file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open YAML file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Check if file is empty
	if len(content) == 0 {
		return parser.ParseResult{}, wrapError("empty YAML file", nil)
	}

	// Parse YAML content
	var yamlData interface{}
	if err := yaml.Unmarshal(content, &yamlData); err != nil {
		return parser.ParseResult{}, wrapError("invalid YAML syntax", err)
	}

	// Extract readable text from YAML
	text := extractTextFromYAML(yamlData)

	if text == "" {
		return parser.ParseResult{}, wrapError("no readable content found in YAML", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeYAML
}

// extractTextFromYAML extracts readable text from YAML data structure
func extractTextFromYAML(data interface{}) string {
	var result strings.Builder

	switch v := data.(type) {
	case map[interface{}]interface{}:
		extractFromObject(v, &result)
	case []interface{}:
		extractFromArray(v, &result)
	case map[string]interface{}:
		extractFromObject(v, &result)
	default:
		// Handle primitive values
		handlePrimitiveValue(v, &result)
	}

	return strings.TrimSpace(result.String())
}

// extractFromObject extracts text from YAML object (values only, no keys)
func extractFromObject(obj interface{}, result *strings.Builder) {
	switch o := obj.(type) {
	case map[interface{}]interface{}:
		for _, value := range o {
			extractValue(value, result)
		}
	case map[string]interface{}:
		for _, value := range o {
			extractValue(value, result)
		}
	}
}

// extractFromArray extracts text from YAML array
func extractFromArray(arr []interface{}, result *strings.Builder) {
	for i, item := range arr {
		// Add newline between array items
		if i > 0 && result.Len() > 0 {
			result.WriteString("\n")
		}
		extractValue(item, result)
	}
}

// extractValue handles any YAML value type
func extractValue(value interface{}, result *strings.Builder) {
	switch v := value.(type) {
	case string:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(v)
	case int:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		fmt.Fprintf(result, "%d", v)
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
	case map[interface{}]interface{}:
		// Nested object - recurse
		extractFromObject(v, result)
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

// handlePrimitiveValue handles primitive YAML values
func handlePrimitiveValue(value interface{}, result *strings.Builder) {
	if result.Len() > 0 {
		result.WriteString("\n")
	}

	switch v := value.(type) {
	case string:
		result.WriteString(v)
	case int:
		fmt.Fprintf(result, "%d", v)
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
	default:
		result.WriteString(fmt.Sprintf("%v", v))
	}
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &YAMLParserError{
			message: message,
			cause:   nil,
		}
	}
	return &YAMLParserError{
		message: message,
		cause:   err,
	}
}
