package ppt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/richardlehane/mscfb"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// PPTParserError represents an error that occurs during PPT parsing.
type PPTParserError struct {
	message string
	cause   error
}

func (e *PPTParserError) Error() string {
	if e.message == "" {
		return "PPT parser error"
	}
	return e.message
}

func (e *PPTParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser interface for PPT files.
type Parser struct{}

// Parse reads a PPT file and extracts text content from slides.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the PPT file
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open PPT file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open PPT file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open PPT file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to get file info", err)
	}

	// Check if file is empty
	if fileInfo.Size() == 0 {
		return parser.ParseResult{}, wrapError("empty PPT file", nil)
	}

	// Read the file content
	fileContent, err := os.ReadFile(req.File)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to read PPT file", err)
	}

	// Create a reader from the content
	reader := bytes.NewReader(fileContent)

	// Parse the OLE2 compound document
	ole, err := mscfb.New(reader)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse PPT as OLE2 document", err)
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
			content, err := io.ReadAll(entry)
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

			content, err := io.ReadAll(entry)
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
