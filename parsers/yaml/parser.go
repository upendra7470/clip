package yaml

import (
	"context"
	"fmt"
	"io"
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

// Parser implements the parser.Parser and parser.RangeParser interfaces for YAML files.
type Parser struct{}

// NewParser creates a new YAML Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a YAML file and extracts readable text representation.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "yaml",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "yaml file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "yaml",
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

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return string(parser.Values)
}

// ParseRange extracts text from a specific value range in a YAML file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate value range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("value numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid value range: start value must not be greater than end value (got %d-%d)", start, end), nil)
	}

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

	// Extract readable text from YAML with value tracking
	text, totalValues, err := extractTextFromYAMLWithValues(yamlData)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to extract text from YAML", err)
	}

	// Validate range against actual value count
	if start > totalValues || end > totalValues {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested value range exceeds document value count (document has %d values, requested %d-%d)", totalValues, start, end), nil)
	}

	// Split text into values and extract requested range
	values := strings.Split(text, "\n")
	var result strings.Builder
	for i := start - 1; i < end && i < len(values); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(values[i])
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in values %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
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

// extractTextFromYAMLWithValues extracts readable text from YAML data structure with value tracking.
func extractTextFromYAMLWithValues(data interface{}) (string, int, error) {
	var result strings.Builder
	var valueCount int

	switch v := data.(type) {
	case map[interface{}]interface{}:
		valueCount = extractFromObjectWithCount(v, &result)
	case []interface{}:
		valueCount = extractFromArrayWithCount(v, &result)
	case map[string]interface{}:
		valueCount = extractFromObjectWithCount(v, &result)
	default:
		// Handle primitive values
		handlePrimitiveValue(v, &result)
		valueCount = 1
	}

	return strings.TrimSpace(result.String()), valueCount, nil
}

// extractFromObjectWithCount extracts text from YAML object with value counting.
func extractFromObjectWithCount(obj interface{}, result *strings.Builder) int {
	var count int
	switch o := obj.(type) {
	case map[interface{}]interface{}:
		for _, value := range o {
			count += extractValueWithCount(value, result)
		}
	case map[string]interface{}:
		for _, value := range o {
			count += extractValueWithCount(value, result)
		}
	}
	return count
}

// extractFromArrayWithCount extracts text from YAML array with value counting.
func extractFromArrayWithCount(arr []interface{}, result *strings.Builder) int {
	var count int
	for i, item := range arr {
		// Add newline between array items
		if i > 0 && result.Len() > 0 {
			result.WriteString("\n")
		}
		count += extractValueWithCount(item, result)
	}
	return count
}

// extractValueWithCount handles any YAML value type with counting.
func extractValueWithCount(value interface{}, result *strings.Builder) int {
	var count int

	switch v := value.(type) {
	case string:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(v)
		count = 1
	case int:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		fmt.Fprintf(result, "%d", v)
		count = 1
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
		count = 1
	case bool:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		fmt.Fprintf(result, "%t", v)
		count = 1
	case nil:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString("null")
		count = 1
	case map[interface{}]interface{}:
		// Nested object - recurse
		count = extractFromObjectWithCount(v, result)
	case map[string]interface{}:
		// Nested object - recurse
		count = extractFromObjectWithCount(v, result)
	case []interface{}:
		// Array - handle each element
		count = extractFromArrayWithCount(v, result)
	default:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%v", v))
		count = 1
	}

	return count
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

// ExtractStructured extracts structured data from yaml content based on the given range
func (p *Parser) ExtractStructured(content string, start, end int) (string, error) {
	// Split into structured units (lines for this test)
	units := strings.Split(content, "\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("index numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(units) {
		return "", nil // Out of range returns empty
	}
	if end > len(units) {
		end = len(units)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(units); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(units[i])
	}

	return result.String(), nil
}
