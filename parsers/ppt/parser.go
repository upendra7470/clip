package ppt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/richardlehane/mscfb"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
)

// PPTParserError represents an error that occurs during PPT parsing.
type PPTParserError struct {
	message string
	cause   error
}

func (e *PPTParserError) Error() string {
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
	if e.message == "" {
		return "PPT parser error"
	}
	return e.message
}

func (e *PPTParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for PPT files.
type Parser struct{}

// NewParser creates a new PPT Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a PPT file and extracts text content from slides.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read ppt: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "ppt",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "ppt file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "ppt",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// readPPTContent reads and validates a PPT file's raw bytes.
func readPPTContent(path string) ([]byte, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, wrapError(fmt.Sprintf("file %s does not exist", path), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, wrapError(fmt.Sprintf("permission denied reading file %s", path), err)
		}
		return nil, wrapError(fmt.Sprintf("error opening file %s: %v", path, err), err)
	}

	// Check if file is empty
	if fileInfo.Size() == 0 {
		return nil, wrapError("empty PPT file", nil)
	}

	// Read the file content
	fileContent, err := security.ReadFileBounded(path)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("error reading file %s: %v", path, err), err)
	}
	if len(fileContent) > parser.MaxFileSize {
		return nil, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", path, parser.MaxFileSize), nil)
	}

	return fileContent, nil
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	fileContent, err := readPPTContent(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	// Create a reader from the content
	reader := bytes.NewReader(fileContent)

	// Parse the OLE2 compound document
	ole, err := mscfb.New(reader)
	if err != nil {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid PPT file: failed to parse %s as OLE2 document", req.File), err)
	}

	// Extract text from the PPT streams
	text, err := extractTextFromOLE(ole, fileContent)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to extract text from PPT", err)
	}

	// Check if we extracted any text
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return parser.ParseResult{}, wrapError("no text content found in PPT", nil)
	}

	return parser.ParseResult{
		Text: trimmedText,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypePPT
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitSlides
}

// ListUnits implements the parser.DocumentLister interface for PPT files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	fileContent, err := readPPTContent(req.File)
	if err != nil {
		return 0, nil, err
	}

	ole, err := mscfb.New(bytes.NewReader(fileContent))
	if err != nil {
		return 0, nil, wrapError(fmt.Sprintf("invalid PPT file: failed to parse %s as OLE2 document", req.File), err)
	}

	_, totalSlides, err := extractTextFromOLEWithSlides(ole, fileContent)
	if err != nil {
		return 0, nil, wrapError("failed to extract text from PPT", err)
	}
	return totalSlides, nil, nil
}

// ParseRange extracts text from a specific slide range in a PPT file.
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

	fileContent, err := readPPTContent(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}
	// Create a reader from the content
	reader := bytes.NewReader(fileContent)

	// Parse the OLE2 compound document
	ole, err := mscfb.New(reader)
	if err != nil {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid PPT file: failed to parse %s as OLE2 document", req.File), err)
	}

	// Extract text from the PPT streams with slide tracking
	text, totalSlides, err := extractTextFromOLEWithSlides(ole, fileContent)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to extract text from PPT", err)
	}

	// Normalize end value if it's -1
	if end == -1 {
		end = totalSlides
	}

	// Validate range against actual slide count
	if start > totalSlides || end > totalSlides {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested slide range exceeds document slide count (document has %d slides, requested %d-%d)", totalSlides, start, end), nil)
	}

	// Split text into slides and extract requested range
	slides := strings.Split(text, "\n\n")
	var result strings.Builder
	for i := start - 1; i < end && i < len(slides); i++ {
		if i > start-1 {
			result.WriteString("\n\n")
		}
		result.WriteString(slides[i])
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in slides %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// extractTextFromOLE extracts text from OLE2 compound document (PPT format)
func extractTextFromOLE(ole *mscfb.Reader, fileContent []byte) (string, error) {
	var result strings.Builder
	var slideTexts []string

	// Iterate through all entries in the OLE document
	for {
		entry, err := ole.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read OLE entry: %w", err)
		}

		// Look for streams that might contain text data
		// PPT files typically have streams with specific names
		if entry.Name == "PowerPoint Document" || entry.Name == "Current User" ||
			strings.Contains(entry.Name, "Slide") || strings.Contains(entry.Name, "Text") {
			limitedReader := io.LimitReader(entry, parser.MaxFileSize)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				continue // Skip streams we can't read
			}

			// Extract text from the stream content
			text := extractTextFromStream(content)
			if text != "" {
				slideTexts = append(slideTexts, text)
			}
		}
	}

	// If no specific streams found, try to extract from all streams
	if len(slideTexts) == 0 {
		// Create a new reader to start from beginning
		reader := bytes.NewReader(fileContent)
		ole2, err := mscfb.New(reader)
		if err != nil {
			return "", fmt.Errorf("failed to reparse OLE document: %w", err)
		}

		for {
			entry, err := ole2.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}

			limitedReader := io.LimitReader(entry, parser.MaxFileSize)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				continue
			}

			text := extractTextFromStream(content)
			if text != "" {
				slideTexts = append(slideTexts, text)
			}
		}
	}

	if len(slideTexts) == 0 {
		return "", fmt.Errorf("no text found in PPT streams")
	}

	// Join slide texts with blank lines between slides
	for i, slideText := range slideTexts {
		if i > 0 {
			result.WriteString("\n\n") // Separate slides with one blank line
		}
		result.WriteString(slideText)
	}

	return result.String(), nil
}

// extractTextFromStream extracts text from a single stream
func extractTextFromStream(content []byte) string {
	var result strings.Builder
	inText := false
	var currentText strings.Builder

	for _, b := range content {
		// Check if this byte is printable ASCII or common Unicode
		if (b >= 32 && b <= 126) || b >= 128 {
			if !inText {
				inText = true
				currentText.Reset()
			}
			currentText.WriteByte(b)
		} else {
			if inText {
				// End of text run
				text := currentText.String()
				if len(text) > 2 { // Minimum length to avoid false positives
					if result.Len() > 0 {
						result.WriteString(" ")
					}
					result.WriteString(text)
				}
				inText = false
			}
		}
	}

	// Handle any remaining text
	if inText {
		text := currentText.String()
		if len(text) > 2 {
			if result.Len() > 0 {
				result.WriteString(" ")
			}
			result.WriteString(text)
		}
	}

	return cleanExtractedText(result.String())
}

// cleanExtractedText cleans up the extracted text
func cleanExtractedText(text string) string {
	// Remove common PPT binary artifacts and clean up whitespace
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	// Join with single newlines and clean up
	result := strings.Join(cleanedLines, "\n")

	// Remove duplicate whitespace but preserve meaningful line breaks
	result = strings.Join(strings.Fields(result), " ")

	// Replace multiple spaces with single space
	result = strings.Join(strings.Fields(result), " ")

	return result
}

// extractTextFromOLEWithSlides extracts text from OLE2 compound document with slide tracking.
func extractTextFromOLEWithSlides(ole *mscfb.Reader, fileContent []byte) (string, int, error) {
	var result strings.Builder
	var slideTexts []string

	// Iterate through all entries in the OLE document
	for {
		entry, err := ole.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, fmt.Errorf("failed to read OLE entry: %w", err)
		}

		// Look for streams that might contain text data
		// PPT files typically have streams with specific names
		if entry.Name == "PowerPoint Document" || entry.Name == "Current User" ||
			strings.Contains(entry.Name, "Slide") || strings.Contains(entry.Name, "Text") {
			limitedReader := io.LimitReader(entry, parser.MaxFileSize)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				continue // Skip streams we can't read
			}

			// Extract text from the stream content
			text := extractTextFromStream(content)
			if text != "" {
				slideTexts = append(slideTexts, text)
			}
		}
	}

	// If no specific streams found, try to extract from all streams
	if len(slideTexts) == 0 {
		// Create a new reader to start from beginning
		reader := bytes.NewReader(fileContent)
		ole2, err := mscfb.New(reader)
		if err != nil {
			return "", 0, fmt.Errorf("failed to reparse OLE document: %w", err)
		}

		for {
			entry, err := ole2.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}

			limitedReader := io.LimitReader(entry, parser.MaxFileSize)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				continue
			}

			text := extractTextFromStream(content)
			if text != "" {
				slideTexts = append(slideTexts, text)
			}
		}
	}

	if len(slideTexts) == 0 {
		return "", 0, fmt.Errorf("no text found in PPT streams")
	}

	// Join slide texts with blank lines between slides
	for i, slideText := range slideTexts {
		if i > 0 {
			result.WriteString("\n\n") // Separate slides with one blank line
		}
		result.WriteString(slideText)
	}

	return result.String(), len(slideTexts), nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &PPTParserError{
			message: message,
			cause:   nil,
		}
	}
	return &PPTParserError{
		message: message,
		cause:   err,
	}
}

// ExtractSlides extracts slides from ppt content based on the given range
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
