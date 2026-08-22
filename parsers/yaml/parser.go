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
	"github.com/upendra7470/clip/internal/security"
	"gopkg.in/yaml.v3"
)

// YAMLParserError represents an error that occurs during YAML parsing.
type YAMLParserError struct {
	message string
	cause   error
}

func (e *YAMLParserError) Error() string {
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
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
	values, err := readYAMLValues(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	if len(values) == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in YAML", nil)
	}

	return parser.ParseResult{
		Text: strings.Join(values, "\n"),
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

// ListUnits implements the parser.DocumentLister interface for YAML files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	values, err := readYAMLValues(req.File)
	if err != nil {
		return 0, nil, err
	}
	return len(values), nil, nil
}

// ParseRange extracts text from a specific value range in a YAML file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate value range up front; -1 means "to end" when used as the end bound.
	if start < 1 || (end != -1 && end < 1) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("value numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid value range: start value must not be greater than end value (got %d-%d)", start, end), nil)
	}

	values, err := readYAMLValues(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	totalValues := len(values)

	// Normalize open-ended range
	if end == -1 {
		end = totalValues
	}

	// Validate range against actual value count
	if start > totalValues || end > totalValues {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested value range exceeds document value count (document has %d values, requested %d-%d)", totalValues, start, end), nil)
	}

	selected := values[start-1 : end]
	if len(selected) == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in values %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: strings.Join(selected, "\n"),
	}, nil
}

// readYAMLValues reads a YAML file and returns its scalar leaf values in document order.
func readYAMLValues(path string) ([]string, error) {
	content, err := security.ReadFileBounded(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, wrapError(fmt.Sprintf("file %s does not exist", path), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, wrapError(fmt.Sprintf("permission denied reading file %s", path), err)
		}
		return nil, wrapError(fmt.Sprintf("error reading file %s: %v", path, err), err)
	}

	// Check if file is empty
	if len(content) == 0 {
		return nil, wrapError("empty YAML file", nil)
	}

	// Parse YAML using yaml.Node to preserve document order and raw scalar text
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, wrapError("invalid YAML syntax", err)
	}

	var values []string
	collectLeafValues(&doc, &values)

	return values, nil
}

// collectLeafValues walks a yaml.Node tree in document order and appends
// the raw text of every scalar leaf value. Keys are intentionally omitted
// so that only human-readable values are extracted.
func collectLeafValues(node *yaml.Node, values *[]string) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			collectLeafValues(child, values)
		}
	case yaml.MappingNode:
		// Mapping nodes alternate key, value, key, value, ...
		for i := 0; i+1 < len(node.Content); i += 2 {
			collectLeafValues(node.Content[i+1], values)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			collectLeafValues(child, values)
		}
	case yaml.ScalarNode:
		// node.Value holds the resolved scalar text ("null" for nulls,
		// "-5.5" for negative floats) without quotes or keys.
		*values = append(*values, node.Value)
	}
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	return &YAMLParserError{
		message: message,
		cause:   err,
	}
}
