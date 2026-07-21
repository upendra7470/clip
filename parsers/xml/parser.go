package xml

import (
	"context"
	"encoding/xml"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
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

// Parser implements the parser.Parser interface for XML files.
type Parser struct{}

// Parse reads an XML file and extracts readable text content.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open XML file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open XML file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open XML file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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

// extractTextFromXML extracts readable text from XML content
func extractTextFromXML(content []byte) (string, error) {
	var result strings.Builder
	decoder := xml.NewDecoder(strings.NewReader(string(content)))

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
			// Extract text content, trim excessive whitespace but preserve meaningful content
			text := strings.TrimSpace(string(t))
			if text != "" {
				if result.Len() > 0 {
					result.WriteString("\n")
				}
				result.WriteString(text)
			}
		case xml.StartElement, xml.EndElement, xml.Comment, xml.ProcInst, xml.Directive:
			// Ignore elements, attributes, comments, processing instructions
			continue
		}
	}

	return result.String(), nil
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
