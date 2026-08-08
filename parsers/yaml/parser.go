package yaml

import (
	"context"
	"errors"
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
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
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
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitValues
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
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		return parser.ParseResult{}, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
	}
	if os.IsNotExist(err) {
		return parser.ParseResult{}, wrapError("Could not open YAML file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
	}
	if os.IsPermission(err) {
		return parser.ParseResult{}, wrapError("Could not open YAML file:\n"+req.File+"\n\nReason:\npermission denied", err)
	}
	// Check if file is empty
	if len(content) == 0 {
		return parser.ParseResult{}, wrapError("empty YAML file", nil)
	}

	// Parse YAML content using yaml.Node to preserve document order
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return parser.ParseResult{}, wrapError("invalid YAML syntax", err)
	}

	// Extract readable text from YAML with value tracking
	text, totalValues, err := extractTextFromYAMLWithValuesFromNode(&doc)
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

// extractTextFromYAML extracts readable text from YAML data structure.
// For objects, it outputs "key: value" lines for leaf values.
// For nested objects/arrays, it recurses into them.
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

// extractFromObject extracts text from YAML object.
// For leaf values, outputs "key: value".
// For nested objects/arrays, outputs "key:" and recurses.
func extractFromObject(obj interface{}, result *strings.Builder) {
	switch o := obj.(type) {
	case map[interface{}]interface{}:
		for key, value := range o {
			writeValueForKey(key, value, result)
		}
	case map[string]interface{}:
		for key, value := range o {
			writeValueForKey(key, value, result)
		}
	}
}

// writeValueForKey writes a key-value pair to the result.
// For leaf values, outputs "key: value".
// For nested objects/arrays, outputs "key:" and recurses.
func writeValueForKey(key interface{}, value interface{}, result *strings.Builder) {
	switch v := value.(type) {
	case map[interface{}]interface{}:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%v:", key))
		extractFromObject(v, result)
	case map[string]interface{}:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%v:", key))
		extractFromObject(v, result)
	case []interface{}:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%v:", key))
		extractFromArray(v, result)
	default:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%v: %v", key, value))
	}
}

// extractFromArray extracts text from YAML array.
func extractFromArray(arr []interface{}, result *strings.Builder) {
	for i, item := range arr {
		// Add newline between array items
		if i > 0 && result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("- %v", item))
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
// Uses yaml.Node to preserve document order.
func extractTextFromYAMLWithValues(data interface{}) (string, int, error) {
	// Convert the data back to YAML bytes and re-parse with yaml.Node for order preservation
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
		return "", 0, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var result strings.Builder
	var valueCount int
	extractValuesFromNode(&doc, &result, &valueCount)

	return strings.TrimSpace(result.String()), valueCount, nil
}

// extractTextFromYAMLWithValuesFromNode extracts values from a yaml.Node directly.
func extractTextFromYAMLWithValuesFromNode(doc *yaml.Node) (string, int, error) {
	var result strings.Builder
	var valueCount int
	extractValuesFromNode(doc, &result, &valueCount)
	return strings.TrimSpace(result.String()), valueCount, nil
}

// extractValuesFromNode extracts values from a yaml.Node tree in document order.
func extractValuesFromNode(node *yaml.Node, result *strings.Builder, count *int) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			extractValuesFromNode(child, result, count)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content)-1; i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			key := keyNode.Value
			// For leaf values, output "key: value"
			// For nested mappings/sequences, output "key:" and recurse
			if valueNode.Kind == yaml.ScalarNode {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(fmt.Sprintf("%s: %s", key, valueNode.Value))
				*count++
			} else {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(fmt.Sprintf("%s:", key))
				extractValuesFromNode(valueNode, result, count)
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if child.Kind == yaml.ScalarNode {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(fmt.Sprintf("- %s", child.Value))
				*count++
			} else {
				extractValuesFromNode(child, result, count)
			}
		}
	case yaml.ScalarNode:
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(node.Value)
		*count++
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

// ExtractStructured extracts structured data from yaml content based on the given range.
// It parses the YAML content and extracts leaf values (key: value pairs) in order.
func (p *Parser) ExtractStructured(content string, start, end int) (string, error) {
	// Parse the YAML content using yaml.Node to preserve document order
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Extract leaf values in document order
	var values []string
	extractLeafValuesFromNode(&doc, "", &values)

	if start < 1 || end < 1 {
		return "", fmt.Errorf("index numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(values) {
		return "", nil // Out of range returns empty
	}
	if end > len(values) {
		end = len(values)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(values); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(values[i])
	}

	return result.String(), nil
}

// extractLeafValuesFromNode extracts leaf values from a yaml.Node tree in document order.
// For mapping nodes, it outputs "key: value" for leaf values using the child key.
// For sequence nodes, it outputs items without any key prefix.
func extractLeafValuesFromNode(node *yaml.Node, prefix string, values *[]string) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// A document node has one child (the root content)
		for _, child := range node.Content {
			extractLeafValuesFromNode(child, prefix, values)
		}
	case yaml.MappingNode:
		// Mapping nodes have key-value pairs: [key1, value1, key2, value2, ...]
		for i := 0; i < len(node.Content)-1; i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			key := keyNode.Value
			// For nested mappings/sequences, recurse with the child key as prefix
			// For leaf values, use the child key as prefix
			extractLeafValuesFromNode(valueNode, key, values)
		}
	case yaml.SequenceNode:
		// Sequence nodes contain items - pass empty prefix so items don't get parent key
		for _, child := range node.Content {
			extractLeafValuesFromNode(child, "", values)
		}
	case yaml.ScalarNode:
		// Leaf value - format it with prefix if present
		if prefix != "" {
			*values = append(*values, fmt.Sprintf("%s: %s", prefix, node.Value))
		} else {
			*values = append(*values, node.Value)
		}
	}
}
