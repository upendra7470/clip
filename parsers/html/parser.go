package html

import (
	"context"
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

// Parser implements the parser.Parser interface for HTML files.
type Parser struct{}

// Parse reads an HTML file and extracts readable text content.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open HTML file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open HTML file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open HTML file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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

			// Flush buffer if we have content (before updating lastBlockElement)
			if buffer.Len() > 0 {
				// Add newline if we need one before this block's content
				shouldAddNewline := needNewlineBeforeNextBlock && isBlockElementByName(tagName) && !isClosingTag
				flushBuffer(&result, &buffer, shouldAddNewline)
				needNewlineBeforeNextBlock = false // Reset after adding newline
			}

			if !isClosingTag {
				// Opening tag - update last block element if it's a block element
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

// isBlockElement checks if the element at the given position is a block-level element
func isBlockElement(html string, pos int) bool {
	// Find the start of the tag name
	start := pos + 2 // skip "</"
	for start < len(html) && (html[start] == ' ' || html[start] == '\t' || html[start] == '\n' || html[start] == '\r') {
		start++
	}
	if start >= len(html) {
		return false
	}

	// Extract the tag name
	end := start
	for end < len(html) && html[end] != '>' && html[end] != ' ' && html[end] != '\t' && html[end] != '\n' && html[end] != '\r' {
		end++
	}

	if start >= end {
		return false
	}

	tagName := strings.ToLower(html[start:end])

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
