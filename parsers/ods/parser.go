package ods

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
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

// Parser implements the parser.Parser and parser.RangeParser interfaces for ODS files.
type Parser struct{}

// NewParser creates a new ODS Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an ODS file and extracts text content.
// ODS files are ZIP archives containing XML files.
// This parser extracts data from content.xml spreadsheet cells.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read ods: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "ods",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "ods file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "ods",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
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

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "rows"
}

// ParseRange extracts text from a specific row range in an ODS file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate row range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("row numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid row range: start row must not be greater than end row (got %d-%d)", start, end), nil)
	}

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

	// Parse XML to extract text from spreadsheet cells with row tracking
	text, totalRows, err := extractTextFromXMLWithRows(contentXML)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse ODS XML", err)
	}

	// Validate range against actual row count
	if start > totalRows || end > totalRows {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested row range exceeds document row count (document has %d rows, requested %d-%d)", totalRows, start, end), nil)
	}

	// Split text into rows and extract requested range
	rows := strings.Split(text, "\n")
	var result strings.Builder
	for i := start - 1; i < end && i < len(rows); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(rows[i])
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in rows %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
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

// extractTextFromXMLWithRows parses the XML and extracts text from spreadsheet cells with row tracking.
func extractTextFromXMLWithRows(xmlContent string) (string, int, error) {
	var result strings.Builder
	var rowCount int

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var inTable, inRow, inCell, inTextP bool
	var currentCell strings.Builder
	var rowText strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Check for table elements in the OpenDocument table namespace
			if t.Name.Local == "table" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inTable = true
			} else if inTable && t.Name.Local == "table-row" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inRow = true
				rowText.Reset()
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
					if rowText.Len() > 0 {
						rowText.WriteString("\t")
					}
					rowText.WriteString(cellText)
				}
			} else if inRow && t.Name.Local == "table-row" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inRow = false
				rowTextStr := strings.TrimSpace(rowText.String())
				if rowTextStr != "" {
					if result.Len() > 0 {
						result.WriteString("\n")
					}
					result.WriteString(rowTextStr)
					rowCount++
				}
			} else if inTable && t.Name.Local == "table" && t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
				inTable = false
			}
		}
	}

	return result.String(), rowCount, nil
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

	// Split into rows (separated by newlines)
	rows := strings.Split(content, "\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("row numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid row range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(rows) {
		return "", nil // Out of range returns empty
	}
	if end > len(rows) {
		end = len(rows)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(rows); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(rows[i])
	}

	return result.String(), nil
}
