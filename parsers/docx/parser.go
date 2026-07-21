package docx

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

// Parser implements the parser.Parser interface for DOCX files.
type Parser struct{}

// Parse reads a DOCX file and extracts text content.
// DOCX files are ZIP archives containing XML files.
// This parser extracts text from word/document.xml <w:t> nodes.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the DOCX file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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
		return parser.ParseResult{}, wrapError("failed to read DOCX as ZIP archive", err)
	}

	// Find and extract word/document.xml
	var documentXML string
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/document.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to open document.xml", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to read document.xml", err)
			}
			documentXML = string(content)
			break
		}
	}

	if documentXML == "" {
		return parser.ParseResult{}, wrapError("document.xml not found in DOCX", nil)
	}

	// Parse XML to extract text from <w:t> nodes
	text, err := extractTextFromXML(documentXML)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse DOCX XML", err)
	}

	if text == "" {
		return parser.ParseResult{}, wrapError("no text content found in DOCX", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeDOCX
}

// extractTextFromXML parses the XML and extracts text from <w:t> nodes.
func extractTextFromXML(xmlContent string) (string, error) {
	// Simple XML parsing to extract text from <w:t> nodes
	// We use a decoder to handle the XML properly
	var result strings.Builder

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var inTextNode bool
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
			if t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTextNode = true
				currentText.Reset()
			}
		case xml.CharData:
			if inTextNode {
				currentText.Write(t)
			}
		case xml.EndElement:
			if inTextNode && t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTextNode = false
				text := strings.TrimSpace(currentText.String())
				if text != "" {
					if result.Len() > 0 {
						result.WriteString(" ")
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
		return &DOCXParserError{
			message: message,
			cause:   nil,
		}
	}
	return &DOCXParserError{
		message: message,
		cause:   err,
	}
}

// DOCXParserError represents an error that occurs during DOCX parsing.
type DOCXParserError struct {
	message string
	cause   error
}

func (e *DOCXParserError) Error() string {
	if e.message == "" {
		return "DOCX parser error"
	}
	return e.message
}

func (e *DOCXParserError) Unwrap() error {
	return e.cause
}
