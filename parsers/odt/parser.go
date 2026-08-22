package odt

import (
	"archive/zip"
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

// ODTParserError represents an error that occurs during ODT parsing.
type ODTParserError struct {
	message string
	cause   error
}

func (e *ODTParserError) Error() string {
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
	if e.message == "" {
		return "ODT parser error"
	}
	return e.message
}

func (e *ODTParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for ODT files.
type Parser struct{}

// NewParser creates a new ODT Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an ODT file and extracts text content.
// ODT files are ZIP archives containing XML files.
// This parser extracts text from content.xml <text:p> nodes.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read odt: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "odt",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "odt file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "odt",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// readODTContent opens an ODT file (a ZIP archive) and returns its content.xml.
func readODTContent(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", wrapError(fmt.Sprintf("file %s does not exist", path), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return "", wrapError(fmt.Sprintf("permission denied reading file %s", path), err)
		}
		return "", wrapError(fmt.Sprintf("error opening file %s: %v", path, err), err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", wrapError("failed to get file info", err)
	}

	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		return "", wrapError(fmt.Sprintf("invalid ODT file: %s is not a valid ZIP archive", path), err)
	}

	// Check total uncompressed size to prevent zip bombs
	if err := security.CheckZipArchiveTotalSize(zipReader.File); err != nil {
		return "", wrapError("failed to read ODT as ZIP archive", err)
	}

	for _, zipFile := range zipReader.File {
		if zipFile.Name == "content.xml" {
			content, err := security.ReadZipFile(zipFile)
			if err != nil {
				return "", wrapError("failed to read content.xml", err)
			}
			return string(content), nil
		}
	}

	return "", wrapError(fmt.Sprintf("invalid ODT file: %s contains no content.xml", path), nil)
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	contentXML, err := readODTContent(req.File)
	if err != nil {
		return parser.ParseResult{}, err
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

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitParagraphs
}

// ListUnits implements the parser.DocumentLister interface for ODT files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	contentXML, err := readODTContent(req.File)
	if err != nil {
		return 0, nil, err
	}

	_, totalParagraphs, err := extractTextFromXMLWithParagraphs(contentXML)
	if err != nil {
		return 0, nil, wrapError("failed to parse ODT XML", err)
	}
	return totalParagraphs, nil, nil
}

// ParseRange extracts text from a specific paragraph range in an ODT file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	contentXML, err := readODTContent(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	// Parse XML to extract text from <text:p> nodes with paragraph tracking
	text, totalParagraphs, err := extractTextFromXMLWithParagraphs(contentXML)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse ODT XML", err)
	}

	// Normalize end value if it's -1
	if end == -1 {
		end = totalParagraphs
	}

	// Handle sentinel values for start
	if start == -1 {
		start = 1 // Start from beginning
	}

	// Validate paragraph range
	if start < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("paragraph numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("paragraph numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid paragraph range: start paragraph must not be greater than end paragraph (got %d-%d)", start, end), nil)
	}

	// Validate range against actual paragraph count
	if start > totalParagraphs {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested paragraph range exceeds document paragraph count (document has %d paragraphs, requested %d-%d)", totalParagraphs, start, end), nil)
	}
	if end > totalParagraphs {
		end = totalParagraphs // Adjust end to last paragraph if it exceeds
	}

	// Split text into paragraphs and extract requested range
	paragraphs := strings.Split(text, "\n")
	var result strings.Builder
	for i := start - 1; i < end && i < len(paragraphs); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(paragraphs[i])
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in paragraphs %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// extractTextFromXML parses the XML and extracts text from <text:p> nodes.
// ODT uses the OpenDocument namespace: urn:oasis:names:tc:opendocument:xmlns:text:1.0
func extractTextFromXML(xmlContent string) (string, error) {
	var result strings.Builder

	decoder := security.NewXMLDecoder(strings.NewReader(xmlContent))
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

// extractTextFromXMLWithParagraphs parses the XML and extracts text from <text:p> nodes with paragraph tracking.
func extractTextFromXMLWithParagraphs(xmlContent string) (string, int, error) {
	var result strings.Builder
	var paragraphCount int

	decoder := security.NewXMLDecoder(strings.NewReader(xmlContent))
	var inParagraph bool
	var currentText strings.Builder

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
					paragraphCount++
				}
			}
		}
	}

	return result.String(), paragraphCount, nil
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
