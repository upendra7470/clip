package ods

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

// ODSParserError represents an error that occurs during ODS parsing.
type ODSParserError struct {
	message string
	cause   error
}

func (e *ODSParserError) Error() string {
	if e.message == "" {
		return "ODS parser error"
	}
	return e.message
}

func (e *ODSParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser interface for ODS files.
type Parser struct{}

// Parse reads an ODS file and extracts text content.
// ODS files are ZIP archives containing XML files.
// This parser extracts data from content.xml spreadsheet cells.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the ODS file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open ODS file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open ODS file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open ODS file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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
		return parser.ParseResult{}, wrapError("failed to read ODS as ZIP archive", err)
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
		return parser.ParseResult{}, wrapError("content.xml not found in ODS", nil)
	}

	// Parse XML to extract text from spreadsheet cells
	text, err := extractTextFromXML(contentXML)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse ODS XML", err)
	}

	if text == "" {
		return parser.ParseResult{}, wrapError("no text content found in ODS", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeODS
}

// extractTextFromXML parses the XML and extracts text from spreadsheet cells.
// ODS uses the OpenDocument namespace: urn:oasis:names:tc:opendocument:xmlns:table:1.0
func extractTextFromXML(xmlContent string) (string, error) {
	var result strings.Builder

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var inTable, inRow, inCell, inTextP bool
	var currentCell strings.Builder

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
			// Check for table elements in the OpenDocument table namespace
			if t.Name.Local == "table" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inTable = true
			} else if inTable && t.Name.Local == "table-row" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inRow = true
			} else if inRow && t.Name.Local == "table-cell" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inCell = true
				currentCell.Reset()
			} else if inCell && t.Name.Local == "p" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:text:1.0" {
				inTextP = true
			}
		case xml.CharData:
			if inTextP {
				currentCell.Write(t)
			}
		case xml.EndElement:
			if inTextP && t.Name.Local == "p" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:text:1.0" {
				inTextP = false
			} else if inCell && t.Name.Local == "table-cell" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inCell = false
				cellText := strings.TrimSpace(currentCell.String())
				if cellText != "" {
					if result.Len() > 0 {
						result.WriteString("\n")
					}
					result.WriteString(cellText)
				}
			} else if inRow && t.Name.Local == "table-row" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inRow = false
			} else if inTable && t.Name.Local == "table" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inTable = false
			}
		}
	}

	return result.String(), nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &ODSParserError{
			message: message,
			cause:   nil,
		}
	}
	return &ODSParserError{
		message: message,
		cause:   err,
	}
}
