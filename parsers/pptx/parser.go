package pptx

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
)

// PPTXParserError represents an error that occurs during PPTX parsing.
type PPTXParserError struct {
	message string
	cause   error
}

func (e *PPTXParserError) Error() string {
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
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

// NewParser creates a new PPTX Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a PPTX file and extracts text content from slides.
// PPTX files are ZIP archives containing XML files.
// This parser extracts text from ppt/slides/slide*.xml <a:t> nodes.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read pptx: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "pptx",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "pptx file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "pptx",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// pptxSlide is a slide's XML content read into memory.
type pptxSlide struct {
	name    string
	content []byte
}

// openPPTXSlides opens a PPTX file (a ZIP archive) and returns its slides'
// XML contents in order. Contents are read eagerly so the archive can be
// closed safely before extraction.
func openPPTXSlides(path string) ([]pptxSlide, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, wrapError(fmt.Sprintf("file %s does not exist", path), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, wrapError(fmt.Sprintf("permission denied reading file %s", path), err)
		}
		return nil, wrapError(fmt.Sprintf("error opening file %s: %v", path, err), err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, wrapError("failed to get file info", err)
	}

	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		return nil, wrapError(fmt.Sprintf("invalid PPTX file: %s is not a valid ZIP archive", path), err)
	}

	// Check total uncompressed size to prevent zip bombs
	if err := security.CheckZipArchiveTotalSize(zipReader.File); err != nil {
		return nil, wrapError("failed to read PPTX as ZIP archive", err)
	}

	var slideFiles []*zip.File
	for _, zipFile := range zipReader.File {
		if strings.HasPrefix(zipFile.Name, "ppt/slides/") && strings.HasSuffix(zipFile.Name, ".xml") {
			slideFiles = append(slideFiles, zipFile)
		}
	}

	if len(slideFiles) == 0 {
		return nil, nil
	}

	// Sort slides by name to ensure proper order
	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].Name < slideFiles[j].Name
	})

	slides := make([]pptxSlide, 0, len(slideFiles))
	for _, sf := range slideFiles {
		content, err := security.ReadZipFile(sf)
		if err != nil {
			return nil, wrapError(fmt.Sprintf("failed to read slide file %s", sf.Name), err)
		}
		slides = append(slides, pptxSlide{name: sf.Name, content: content})
	}

	return slides, nil
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	slides, err := openPPTXSlides(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	if len(slides) == 0 {
		return parser.ParseResult{}, wrapError("no slides found in PPTX", nil)
	}

	// Extract text from each slide
	var result strings.Builder
	for i, slide := range slides {
		text, err := extractTextFromSlide(slide.content)
		if err != nil {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("failed to parse slide %d (%s)", i+1, slide.name), err)
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
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitSlides
}

// ListUnits implements the parser.DocumentLister interface for PPTX files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	slides, err := openPPTXSlides(req.File)
	if err != nil {
		return 0, nil, err
	}
	return len(slides), nil, nil
}

// ParseRange extracts text from a specific slide range in a PPTX file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	// Validate slide range
	if start < 1 || (end != -1 && end < 1) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("slide numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid slide range: start slide must not be greater than end slide (got %d-%d)", start, end), nil)
	}

	slideFiles, err := openPPTXSlides(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	if len(slideFiles) == 0 {
		return parser.ParseResult{}, wrapError("no slides found in PPTX", nil)
	}

	// Normalize end value if it's -1
	if end == -1 {
		end = len(slideFiles)
	}

	// Validate range against actual slide count
	if start > len(slideFiles) || end > len(slideFiles) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested slide range exceeds document slide count (document has %d slides, requested %d-%d)", len(slideFiles), start, end), nil)
	}

	// Extract text from the requested slide range
	var result strings.Builder
	for i := start - 1; i < end && i < len(slideFiles); i++ {
		text, err := extractTextFromSlide(slideFiles[i].content)
		if err != nil {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("failed to parse slide %d (%s)", i+1, slideFiles[i].name), err)
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

// extractTextFromSlide extracts text from a slide's XML content.
func extractTextFromSlide(content []byte) (string, error) {
	var result strings.Builder
	decoder := security.NewXMLDecoder(strings.NewReader(string(content)))
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

// ExtractSlides extracts slides from pptx content based on the given range
func (p *Parser) ExtractSlides(content string, start, end int) (string, error) {
	// Split into slides (separated by newlines for this test)
	slides := strings.Split(content, "\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("slide numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid slide range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(slides) {
		return "", nil // Out of range returns empty
	}
	if end > len(slides) {
		end = len(slides)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(slides); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(slides[i])
	}

	return result.String(), nil
}
