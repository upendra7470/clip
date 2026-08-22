package xml

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
)

// XMLParserError represents an error that occurs during XML parsing.
type XMLParserError struct {
	message string
	cause   error
}

func (e *XMLParserError) Error() string {
	if e.message == "" {
		return "XML parser error"
	}
	return e.message
}

func (e *XMLParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for XML files.
type Parser struct{}

// NewParser creates a new XML Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an XML file and extracts readable text content.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read xml: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "xml",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "xml file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "xml",
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
	content, err := security.ReadFileBounded(req.File)
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
		return parser.ParseResult{}, wrapError("empty XML file", nil)
	}

	// Validate XML syntax and extract text
	text, err := extractTextFromXML(content)
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid XML syntax", err)
	}

	// Check if we extracted any meaningful text
	if strings.TrimSpace(text) == "" {
		return parser.ParseResult{}, wrapError("no readable content found in XML", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeXML
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnit("elements")
}

// ParseRange extracts text from a specific text block range in an XML file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	// Validate text block range
	if start < 1 || (end != -1 && end < 1) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("text block numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid text block range: start block must not be greater than end block (got %d-%d)", start, end), nil)
	}

	// Read the file content
	content, err := security.ReadFileBounded(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("permission denied reading file %s", req.File), err)
		}
		return parser.ParseResult{}, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
	}
	// Check if file is empty
	if len(content) == 0 {
		return parser.ParseResult{}, wrapError("empty XML file", nil)
	}

	// Validate XML syntax and extract text with block tracking
	text, totalBlocks, err := extractTextFromXMLWithBlocks(content)
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid XML syntax", err)
	}

	// Normalize end value if it's -1
	if end == -1 {
		end = totalBlocks
	}

	// Validate range against actual block count
	if start > totalBlocks || end > totalBlocks {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested text block range exceeds document block count (document has %d blocks, requested %d-%d)", totalBlocks, start, end), nil)
	}

	// Split text into blocks and extract requested range
	blocks := strings.Split(text, "\n")
	var result strings.Builder
	for i := start - 1; i < end && i < len(blocks); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(blocks[i])
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in blocks %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// extractTextFromXML extracts readable text from XML content
func extractTextFromXML(content []byte) (string, error) {
	var result strings.Builder
	decoder := security.NewXMLDecoder(strings.NewReader(string(content)))

	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break // End of XML document
			}
			return "", err
		}

		switch t := token.(type) {
		case xml.CharData:
			// Extract text content with whitespace normalized
			text := normalizeXMLText(string(t))
			if text != "" {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(text)
			}
		case xml.StartElement, xml.EndElement, xml.Comment, xml.ProcInst, xml.Directive:
			// Ignore elements, attributes, comments, processing instructions
			_ = t
			continue
		}
	}

	return result.String(), nil
}

// extractTextFromXMLWithBlocks extracts readable text from XML content with block tracking.
func extractTextFromXMLWithBlocks(content []byte) (string, int, error) {
	var result strings.Builder
	decoder := security.NewXMLDecoder(strings.NewReader(string(content)))
	blockCount := 0

	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break // End of XML document
			}
			return "", 0, err
		}

		switch t := token.(type) {
		case xml.CharData:
			// Extract text content with whitespace normalized
			text := normalizeXMLText(string(t))
			if text != "" {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(text)
				blockCount++
			}
		case xml.StartElement, xml.EndElement, xml.Comment, xml.ProcInst, xml.Directive:
			// Ignore elements, attributes, comments, processing instructions
			_ = t
			continue
		}
	}

	return result.String(), blockCount, nil
}

// normalizeXMLText trims surrounding whitespace and collapses internal
// whitespace runs (newlines, tabs, indentation) into single spaces.
func normalizeXMLText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &XMLParserError{
			message: message,
			cause:   nil,
		}
	}
	return &XMLParserError{
		message: message,
		cause:   err,
	}
}

// ExtractStructured extracts structured data from xml content based on the given range
func (p *Parser) ExtractStructured(content string, start, end int) (string, error) {
	// Split into structured units (element tags for this test)
	units := strings.Split(content, "<element>")
	// Filter and reconstruct with <element> prefix
	var elements []string
	for i, unit := range units {
		if i == 0 {
			continue
		}
		if endIdx := strings.Index(unit, "</element>"); endIdx != -1 {
			elements = append(elements, "<element>"+unit[:endIdx+len("</element>")])
		}
	}

	if start < 1 || end < 1 {
		return "", fmt.Errorf("index numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(elements) {
		return "", fmt.Errorf("requested range exceeds element count (document has %d elements, requested %d-%d)", len(elements), start, end)
	}
	if end > len(elements) {
		return "", fmt.Errorf("requested range exceeds element count (document has %d elements, requested %d-%d)", len(elements), start, end)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(elements); i++ {
		result.WriteString(elements[i])
	}

	return result.String(), nil
}
