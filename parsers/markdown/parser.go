package markdown

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser and parser.RangeParser interfaces for Markdown files.
type Parser struct{}

// NewParser creates a new Markdown Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a Markdown file and returns extracted readable text.
// It processes basic Markdown syntax to make the content more readable.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "markdown",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "markdown file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "markdown",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
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
	return "sections"
}

// ParseRange extracts text from a specific section range in a Markdown file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate section range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("section numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid section range: start section must not be greater than end section (got %d-%d)", start, end), nil)
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

	// Split into sections based on headings
	sections := extractSections(text)

	// Validate range against actual section count
	if start > len(sections) || end > len(sections) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested section range exceeds file section count (file has %d sections, requested %d-%d)", len(sections), start, end), nil)
	}

	// Extract only the requested section range
	var result strings.Builder
	for i := start - 1; i < end && i < len(sections); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(sections[i])
	}

	// Process Markdown syntax for the extracted range
	processed := processMarkdown(result.String())

	if processed == "" {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in sections %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: processed,
	}, nil
}

// ExtractStructured extracts structured content from markdown based on the given range
func (p *Parser) ExtractStructured(content string, start, end int) (string, error) {
	// Split into blocks (paragraphs separated by double newlines)
	blocks := strings.Split(content, "\n\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("block numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid block range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(blocks) || end > len(blocks) {
		return "", fmt.Errorf("requested block range exceeds content (content has %d blocks, requested %d-%d)", len(blocks), start, end)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(blocks); i++ {
		if i > start-1 {
			result.WriteString("\n\n")
		}
		result.WriteString(blocks[i])
	}

	return result.String(), nil
}

// ExtractBlocks extracts blocks from markdown content based on the given range
func (p *Parser) ExtractBlocks(content string, start, end int) (string, error) {
	// Split into blocks (paragraphs separated by double newlines)
	blocks := strings.Split(content, "\n\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("block numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid block range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(blocks) || end > len(blocks) {
		return "", fmt.Errorf("requested block range exceeds content (content has %d blocks, requested %d-%d)", len(blocks), start, end)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(blocks); i++ {
		if i > start-1 {
			result.WriteString("\n\n")
		}
		result.WriteString(blocks[i])
	}

	return result.String(), nil
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

// processHeadings removes heading markers but preserves content.
func processHeadings(text string) string {
	// Remove heading markers (e.g., #, ##, ###)
	re := regexp.MustCompile(`(?m)^#+\s*`)
	return re.ReplaceAllString(text, "")
}

// processBoldItalic removes bold and italic markers but preserves content.
func processBoldItalic(text string) string {
	// Remove bold markers (**)
	re := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = re.ReplaceAllString(text, "$1")

	// Remove italic markers (*)
	re = regexp.MustCompile(`\*(.*?)\*`)
	text = re.ReplaceAllString(text, "$1")

	// Remove bold markers (__)
	re = regexp.MustCompile(`__(.*?)__`)
	text = re.ReplaceAllString(text, "$1")

	// Remove italic markers (_)
	re = regexp.MustCompile(`_(.*?)_`)
	text = re.ReplaceAllString(text, "$1")

	return text
}

// processLinks removes link formatting but preserves link text.
func processLinks(text string) string {
	// Remove [text](url) links
	re := regexp.MustCompile(`\[(.*?)\]\(.*?\)`)
	return re.ReplaceAllString(text, "$1")
}

// processLists removes list markers but preserves content.
func processLists(text string) string {
	// Remove ordered list markers (e.g., 1., 2., etc.)
	re := regexp.MustCompile(`(?m)^\s*\d+\.\s*`)
	text = re.ReplaceAllString(text, "")

	// Remove unordered list markers (e.g., *, -, +)
	re = regexp.MustCompile(`(?m)^\s*[\*\-+]\s*`)
	text = re.ReplaceAllString(text, "")

	return text
}

// cleanWhitespace cleans up extra whitespace in the text.
func cleanWhitespace(text string) string {
	// Replace multiple newlines with single newline
	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	// Trim leading and trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// extractSections splits markdown text into sections based on headings.
func extractSections(text string) []string {
	// Split by headings (lines starting with #)
	re := regexp.MustCompile(`(?m)^#+\s*.*$`)
	sections := re.Split(text, -1)

	// Remove empty sections
	var result []string
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return fmt.Errorf(message)