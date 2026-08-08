package html

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// HTMLParserError represents an error that occurs during HTML parsing.
type HTMLParserError struct {
	message string
	cause   error
}

func (e *HTMLParserError) Error() string {
	if e.message == "" {
		return "HTML parser error"
	}
	return e.message
}

func (e *HTMLParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for HTML files.
type Parser struct{}

// NewParser creates a new HTML Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an HTML file and extracts readable text content.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read html: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "html",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "html file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "html",
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
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError("file %s does not exist", fmt.Errorf("file %s does not exist", req.File))
		}
		return parser.ParseResult{}, wrapError("error reading file %s: %v", fmt.Errorf("error reading file %s: %v", req.File, err))
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError("file %s exceeds maximum allowed size of %d bytes", fmt.Errorf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize))
	}

	// Check if file is empty
	if len(content) == 0 {
		return parser.ParseResult{}, wrapError("empty HTML file", nil)
	}

	// Extract text from HTML
	text, err := extractTextFromHTML(string(content))
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to extract text from HTML", err)
	}

	// Check if we extracted any meaningful text
	if strings.TrimSpace(text) == "" {
		return parser.ParseResult{}, wrapError("no readable content found in HTML", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeHTML
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitSections
}

// ParseRange extracts text from a specific text block range in an HTML file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate text block range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("text block numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid text block range: start block must not be greater than end block (got %d-%d)", start, end), nil)
	}

	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError("file %s does not exist", fmt.Errorf("file %s does not exist", req.File))
		}
		return parser.ParseResult{}, wrapError("error reading file %s: %v", fmt.Errorf("error reading file %s: %v", req.File, err))
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError("file %s exceeds maximum allowed size of %d bytes", fmt.Errorf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize))
	}

	// Check if file is empty
	if len(content) == 0 {
		return parser.ParseResult{}, wrapError("empty HTML file", nil)
	}

	// Extract text from HTML and split into blocks
	text, err := extractTextFromHTML(string(content))
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to extract text from HTML", err)
	}

	// Check if we extracted any meaningful text
	if strings.TrimSpace(text) == "" {
		return parser.ParseResult{}, wrapError("no readable content found in HTML", nil)
	}

	// Split text into blocks (paragraphs separated by double newlines)
	blocks := strings.Split(text, "\n\n")
	var nonEmptyBlocks []string
	for _, block := range blocks {
		if strings.TrimSpace(block) != "" {
			nonEmptyBlocks = append(nonEmptyBlocks, block)
		}
	}

	// Validate range against actual block count
	if start > len(nonEmptyBlocks) || end > len(nonEmptyBlocks) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested text block range exceeds document block count (document has %d blocks, requested %d-%d)", len(nonEmptyBlocks), start, end), nil)
	}

	// Extract only the requested text block range
	var result strings.Builder
	for i := start - 1; i < end && i < len(nonEmptyBlocks); i++ {
		if i > start-1 {
			result.WriteString("\n\n")
		}
		result.WriteString(nonEmptyBlocks[i])
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in blocks %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// extractTextFromHTML extracts readable text from HTML content
func extractTextFromHTML(html string) (string, error) {
	var result strings.Builder
	var inScript, inStyle, inComment bool
	var buffer strings.Builder
	var lastBlockElement string
	var needNewlineBeforeNextBlock bool

	for i := 0; i < len(html); i++ {
		char := html[i]

		// Handle script tags
		if !inScript && !inStyle && !inComment && i+6 < len(html) && strings.ToLower(html[i:i+7]) == "<script" {
			inScript = true
			i += 6 // skip past "<script"
			continue
		} else if inScript && i+8 < len(html) && strings.ToLower(html[i:i+9]) == "</script>" {
			inScript = false
			i += 8 // skip past "</script>"
			continue
		}

		// Handle style tags
		if !inScript && !inStyle && !inComment && i+5 < len(html) && strings.ToLower(html[i:i+6]) == "<style" {
			inStyle = true
			i += 5 // skip past "<style"
			continue
		} else if inStyle && i+7 < len(html) && strings.ToLower(html[i:i+8]) == "</style>" {
			inStyle = false
			i += 7 // skip past "</style>"
			continue
		}

		// Handle comments
		if !inScript && !inStyle && !inComment && i+3 < len(html) && html[i:i+4] == "<!--" {
			inComment = true
			i += 3 // skip past "<!--"
			continue
		} else if inComment && i+2 < len(html) && html[i:i+3] == "-->" {
			inComment = false
			i += 2 // skip past "-->"
			continue
		}

		// Skip content inside script, style, or comments
		if inScript || inStyle || inComment {
			continue
		}

		// Handle opening tag
		if char == '<' {
			// Check if this is a closing tag
			isClosingTag := i+1 < len(html) && html[i+1] == '/'

			// Extract tag name for tracking
			tagName := extractTagName(html, i)

			// Check if current buffer has actual text content
			currentBufferHasContent := strings.TrimSpace(buffer.String()) != ""

			// If we need a newline before the next block and this is an opening block element,
			// add the newline directly to the result (even if buffer is empty)
			if needNewlineBeforeNextBlock && !isClosingTag && isBlockElementByName(tagName) {
				result.WriteString("\n\n")
				needNewlineBeforeNextBlock = false
			}

			// Flush buffer if we have content (before updating lastBlockElement)
			if buffer.Len() > 0 {
				flushBuffer(&result, &buffer, false)
			}

			if !isClosingTag {
				// Opening tag - update last block element if it's a section element
				if isBlockElementByName(tagName) {
					lastBlockElement = tagName
				}
			} else {
				// Closing tag - set flag if this block had content
				if lastBlockElement == tagName && currentBufferHasContent {
					needNewlineBeforeNextBlock = true
				}
				// Clear last block element if it matches
				if lastBlockElement == tagName {
					lastBlockElement = ""
				}
			}
			// Skip until closing '>'
			for i < len(html) && html[i] != '>' {
				i++
			}
			continue
		}

		// Collect text content
		buffer.WriteByte(char)
	}

	// Flush any remaining content
	if buffer.Len() > 0 {
		flushBuffer(&result, &buffer, needNewlineBeforeNextBlock)
	}

	return result.String(), nil
}

// flushBuffer flushes the buffer to the result, handling whitespace appropriately
func flushBuffer(result *strings.Builder, buffer *strings.Builder, addNewline bool) {
	text := strings.TrimSpace(buffer.String())
	if text != "" || addNewline {
		if addNewline {
			// Only add newline if we're specifically requested to
			result.WriteString("\n")
		}
		if text != "" {
			if result.Len() > 0 && !addNewline && !strings.HasSuffix(result.String(), "\n") {
				// Only add space if we're not adding a newline and result doesn't already end with newline
				result.WriteString(" ")
			}
			result.WriteString(text)
		}
	}
	buffer.Reset()
}

// extractTagName extracts the tag name from a tag at the given position
func extractTagName(html string, pos int) string {
	// Determine if this is a closing tag
	isClosingTag := pos+1 < len(html) && html[pos+1] == '/'

	// Find the start of the tag name
	start := pos + 1 // skip "<"
	if isClosingTag {
		start++ // skip "/" for closing tags
	}

	// Skip whitespace
	for start < len(html) && (html[start] == ' ' || html[start] == '\t' || html[start] == '\n' || html[start] == '\r') {
		start++
	}
	if start >= len(html) {
		return ""
	}

	// Extract the tag name
	end := start
	for end < len(html) && html[end] != '>' && html[end] != ' ' && html[end] != '\t' && html[end] != '\n' && html[end] != '\r' {
		end++
	}

	if start >= end {
		return ""
	}

	return strings.ToLower(html[start:end])
}

// isBlockElementByName checks if the given tag name is a block-level element
func isBlockElementByName(tagName string) bool {
	if tagName == "" {
		return false
	}

	// List of common block-level elements
	blockElements := map[string]bool{
		"html":       true,
		"body":       true,
		"div":        true,
		"p":          true,
		"h1":         true,
		"h2":         true,
		"h3":         true,
		"h4":         true,
		"h5":         true,
		"h6":         true,
		"section":    true,
		"article":    true,
		"header":     true,
		"footer":     true,
		"nav":        true,
		"main":       true,
		"aside":      true,
		"ul":         true,
		"ol":         true,
		"li":         true,
		"table":      true,
		"tr":         true,
		"td":         true,
		"th":         true,
		"blockquote": true,
		"pre":        true,
		"figure":     true,
		"form":       true,
	}

	return blockElements[tagName]
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &HTMLParserError{
			message: message,
			cause:   nil,
		}
	}
	return &HTMLParserError{
		message: message,
		cause:   err,
	}
}

// ExtractBlocks extracts blocks from html content based on the given range
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
