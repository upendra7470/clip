package pptx

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// PPTXParserError represents an error that occurs during PPTX parsing.
type PPTXParserError struct {
	message string
	cause   error
}

func (e *PPTXParserError) Error() string {
	if e.message == "" {
		return "PPTX parser error"
	}
	return e.message
}

func (e *PPTXParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for PPTX files.
type Parser struct{}

// Parse reads a PPTX file and extracts text content from slides.
// PPTX files are ZIP archives containing XML files.
// This parser extracts text from ppt/slides/slide*.xml <a:t> nodes.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the PPTX file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open PPTX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open PPTX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open PPTX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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
		return parser.ParseResult{}, wrapError("failed to read PPTX as ZIP archive", err)
	}

	// Find all slide files
	var slideFiles []*zip.File
	for _, zipFile := range zipReader.File {
		if strings.HasPrefix(zipFile.Name, "ppt/slides/") && strings.HasSuffix(zipFile.Name, ".xml") {
			slideFiles = append(slideFiles, zipFile)
		}
	}

	if len(slideFiles) == 0 {
		return parser.ParseResult{}, wrapError("no slides found in PPTX", nil)
	}

	// Sort slides by name to ensure proper order
	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].Name < slideFiles[j].Name
	})

	// Extract text from each slide
	var result strings.Builder
	for i, slideFile := range slideFiles {
		text, err := extractTextFromSlide(slideFile)
		if err != nil {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("failed to parse slide %d", i+1), err)
		}

		// Trim and add to result
		trimmedText := strings.TrimSpace(text)
		if trimmedText != "" {
			if result.Len() > 0 {
				result.WriteString("\n\n") // Separate slides with one blank line
			}
			result.WriteString(trimmedText)
		}
	}

	// Check if we extracted any text
	finalText := strings.TrimSpace(result.String())
	if finalText == "" {
		return parser.ParseResult{}, wrapError("no text content found in PPTX", nil)
	}

	return parser.ParseResult{
		Text: finalText,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypePPTX
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "slides"
}

// ParseRange extracts text from a specific slide range in a PPTX file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate slide range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("slide numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid slide range: start slide must not be greater than end slide (got %d-%d)", start, end), nil)
	}

	// Open the PPTX file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open PPTX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open PPTX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open PPTX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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
		return parser.ParseResult{}, wrapError("failed to read PPTX as ZIP archive", err)
	}

	// Find all slide files
	var slideFiles []*zip.File
	for _, zipFile := range zipReader.File {
		if strings.HasPrefix(zipFile.Name, "ppt/slides/") && strings.HasSuffix(zipFile.Name, ".xml") {
			slideFiles = append(slideFiles, zipFile)
		}
	}

	if len(slideFiles) == 0 {
		return parser.ParseResult{}, wrapError("no slides found in PPTX", nil)
	}

	// Sort slides by name to ensure proper order
	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].Name < slideFiles[j].Name
	})

	// Validate range against actual slide count
	if start > len(slideFiles) || end > len(slideFiles) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested slide range exceeds document slide count (document has %d slides, requested %d-%d)", len(slideFiles), start, end), nil)
	}

	// Extract text from the requested slide range
	var result strings.Builder
	for i := start - 1; i < end && i < len(slideFiles); i++ {
		text, err := extractTextFromSlide(slideFiles[i])
		if err != nil {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("failed to parse slide %d", i+1), err)
		}

		// Trim and add to result
		trimmedText := strings.TrimSpace(text)
		if trimmedText != "" {
			if result.Len() > 0 {
				result.WriteString("\n\n") // Separate slides with one blank line
			}
			result.WriteString(trimmedText)
		}
	}

	// Check if we extracted any text
	finalText := strings.TrimSpace(result.String())
	if finalText == "" {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in slides %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: finalText,
	}, nil
}

// extractTextFromSlide extracts text from a single slide XML file.
func extractTextFromSlide(slideFile *zip.File) (string, error) {
	rc, err := slideFile.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open slide file: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read slide content: %w", err)
	}

	var result strings.Builder
	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	var inTextNode bool
	var currentText strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to parse slide XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Check for <a:t> nodes (PowerPoint text nodes)
			// Handle both full namespace and "a" prefix
			if t.Name.Local == "t" && (t.Name.Space == "http://schemas.openxmlformats.org/drawingml/2006/main" || t.Name.Space == "a") {
				inTextNode = true
				currentText.Reset()
			}
		case xml.CharData:
			if inTextNode {
				// Trim whitespace within text nodes but preserve multiple spaces as single space
				trimmed := strings.Join(strings.Fields(string(t)), " ")
				if trimmed != "" {
					if currentText.Len() > 0 {
						currentText.WriteString(" ")
					}
					currentText.WriteString(trimmed)
				}
			}
		case xml.EndElement:
			if inTextNode && t.Name.Local == "t" && (t.Name.Space == "http://schemas.openxmlformats.org/drawingml/2006/main" || t.Name.Space == "a") {
				inTextNode = false
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
		return &PPTXParserError{
			message: message,
			cause:   nil,
		}
	}
	return &PPTXParserError{
		message: message,
		cause:   err,
	}
}
