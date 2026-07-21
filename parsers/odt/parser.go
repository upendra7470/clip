package odt

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// ODTParserError represents an error that occurs during ODT parsing.
type ODTParserError struct {
	message string
	cause   error
}

func (e *ODTParserError) Error() string {
	if e.message == "" {
		return "ODT parser error"
	}
	return e.message
}

func (e *ODTParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser interface for ODT files.
type Parser struct{}

// Parse reads an ODT file and extracts text content.
// ODT files are ZIP archives containing XML files.
// This parser extracts text from content.xml <text:p> nodes.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the ODT file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open ODT file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open ODT file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open ODT file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to get file info", err)
	}

	// Read the ZIP archive
	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to read ODT as ZIP archive", err)
	}

	// Find and extract content.xml
	var contentXML string
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "content.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to open content.xml", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to read content.xml", err)
			}
			contentXML = string(content)
			break
		}
	}

	if contentXML == "" {
		return parser.ParseResult{}, wrapError("content.xml not found in ODT", nil)
	}

	// Parse XML to extract text from <text:p> nodes
	text, err := extractTextFromXML(contentXML)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse ODT XML", err)
	}

	if text == "" {
		return parser.ParseResult{}, wrapError("no text content found in ODT", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeODT
}

// extractTextFromXML parses the XML and extracts text from <text:p> nodes.
// ODT uses the OpenDocument namespace: urn:oasis:names:tc:opendocument:xmlns:text:1.0
func extractTextFromXML(xmlContent string) (string, error) {
	var result strings.Builder

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var inParagraph bool
	var currentText strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Check for paragraph elements in the OpenDocument text namespace
			if t.Name.Local == "p" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:text:1.0" {
				inParagraph = true
				currentText.Reset()
			}
		case xml.CharData:
			if inParagraph {
				currentText.Write(t)
			}
		case xml.EndElement:
			if inParagraph && t.Name.Local == "p" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:text:1.0" {
				inParagraph = false
				text := strings.TrimSpace(currentText.String())
				if text != "" {
					if result.Len() > 0 {
						result.WriteString("\n")
					}
					result.WriteString(text)
				}
			}
		}
	}

	return result.String(), nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &ODTParserError{
			message: message,
			cause:   nil,
		}
	}
	return &ODTParserError{
		message: message,
		cause:   err,
	}
}
