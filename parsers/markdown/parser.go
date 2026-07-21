package markdown

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser and parser.RangeParser interfaces for Markdown files.
type Parser struct{}

// Parse reads a Markdown file and returns extracted readable text.
// It processes basic Markdown syntax to make the content more readable.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open Markdown file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open Markdown file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open Markdown file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Convert to string
	text := string(content)

	// Process Markdown syntax
	processed := processMarkdown(text)

	return parser.ParseResult{
		Text: processed,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeMarkdown
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "lines"
}

// ParseRange extracts text from a specific line range in a Markdown file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate line range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid line range: start line must not be greater than end line (got %d-%d)", start, end), nil)
	}

	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open Markdown file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open Markdown file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open Markdown file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Convert to string
	text := string(content)

	// Split into lines
	lines := strings.Split(text, "\n")

	// Validate range against actual line count
	if start > len(lines) || end > len(lines) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested line range exceeds file line count (file has %d lines, requested %d-%d)", len(lines), start, end), nil)
	}

	// Extract only the requested line range
	var result strings.Builder
	for i := start - 1; i < end && i < len(lines); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(lines[i])
	}

	// Process Markdown syntax for the extracted range
	processed := processMarkdown(result.String())

	if processed == "" {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in lines %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: processed,
	}, nil
}

// processMarkdown processes basic Markdown syntax to extract readable text.
func processMarkdown(text string) string {
	result := text

	// Process in a specific order to avoid interference

	// 1. Remove code blocks but preserve content
	result = processCodeBlocks(result)

	// 2. Process headings
	result = processHeadings(result)

	// 3. Process bold and italic
	result = processBoldItalic(result)

	// 4. Process links
	result = processLinks(result)

	// 5. Process lists
	result = processLists(result)

	// 6. Clean up extra whitespace
	result = cleanWhitespace(result)

	return result
}

// processCodeBlocks removes code block fences but preserves content.
func processCodeBlocks(text string) string {
	// Remove ```code``` blocks
	re := regexp.MustCompile("(?s)```[^`]*```")
	return re.ReplaceAllString(text, "")
}

// processHeadings removes heading markers.
func processHeadings(text string) string {
	// Process line by line to handle multiline headings
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		// Remove #, ##, ###, etc. from headings
		re := regexp.MustCompile(`^#{1,6}\s+`)
		lines[i] = re.ReplaceAllString(line, "")
	}
	return strings.Join(lines, "\n")
}

// processBoldItalic removes bold and italic markers.
func processBoldItalic(text string) string {
	// Remove **bold** and *italic*
	re := regexp.MustCompile(`\*\*(.*?)\*\*|\*(.*?)\*`)
	return re.ReplaceAllString(text, "$1$2")
}

// processLinks extracts link text, removes URL.
func processLinks(text string) string {
	// Remove [text](url), keep text
	re := regexp.MustCompile(`\[(.*?)\]\(.*?\)`)
	return re.ReplaceAllString(text, "$1")
}

// processLists removes list markers.
func processLists(text string) string {
	// Process line by line to handle multiline lists
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		// Remove - *, +, or 1. from list items
		re := regexp.MustCompile(`^[\s]*[-+*][\s]+|^[\s]*\d+\.[\s]+`)
		lines[i] = re.ReplaceAllString(line, "")
	}
	return strings.Join(lines, "\n")
}

// cleanWhitespace cleans up extra whitespace.
func cleanWhitespace(text string) string {
	// Replace multiple newlines with single newline
	re := regexp.MustCompile(`\n{3,}`)
	return re.ReplaceAllString(text, "\n\n")
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &MarkdownParserError{
			message: message,
			cause:   nil,
		}
	}
	return &MarkdownParserError{
		message: message,
		cause:   err,
	}
}

// MarkdownParserError represents an error that occurs during Markdown parsing.
type MarkdownParserError struct {
	message string
	cause   error
}

func (e *MarkdownParserError) Error() string {
	if e.message == "" {
		return "markdown parser error"
	}
	return e.message
}

func (e *MarkdownParserError) Unwrap() error {
	return e.cause
}
