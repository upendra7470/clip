package rtf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// RTFParserError represents an error that occurs during RTF file parsing.
type RTFParserError struct {
	message string
	cause   error
}

func (e *RTFParserError) Error() string {
	if e.message == "" {
		return "RTF parser error"
	}
	return e.message
}

func (e *RTFParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for RTF files.
type Parser struct{}

// NewParser creates a new RTF Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse extracts plain text from an RTF file, ignoring formatting and control words.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read rtf: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": "rtf",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "rtf file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "rtf",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Check if file exists
	fileInfo, err := os.Stat(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError("Could not open RTF file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if errors.Is(err, os.ErrPermission) {
			return parser.ParseResult{}, wrapError("Could not open RTF file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open RTF file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Check if file is empty
	if fileInfo.Size() == 0 {
		return parser.ParseResult{}, wrapError("file is empty", nil)
	}

	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		return parser.ParseResult{}, wrapError("file cannot be read", err)
	}

	// Validate UTF-8
	if !isValidUTF8(content) {
		return parser.ParseResult{}, wrapError("invalid UTF-8 content", nil)
	}

	// Convert to string for processing
	rtfContent := string(content)

	// Extract plain text from RTF
	plainText, err := extractPlainText(rtfContent)
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid RTF format", err)
	}

	// Check if we got any text
	if len(plainText) == 0 {
		return parser.ParseResult{}, wrapError("no text content found", nil)
	}

	return parser.ParseResult{
		Text: plainText,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeRTF
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitParagraphs
}

// ParseRange extracts text from a specific paragraph range in an RTF file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate paragraph range
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("paragraph numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid paragraph range: start paragraph must not be greater than end paragraph (got %d-%d)", start, end), nil)
	}

	// Check if file exists
	fileInfo, err := os.Stat(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError("Could not open RTF file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if errors.Is(err, os.ErrPermission) {
			return parser.ParseResult{}, wrapError("Could not open RTF file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open RTF file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}

	// Check if file is empty
	if fileInfo.Size() == 0 {
		return parser.ParseResult{}, wrapError("file is empty", nil)
	}

	// Read the file content
	content, err := os.ReadFile(req.File)
	if err != nil {
		return parser.ParseResult{}, wrapError("file cannot be read", err)
	}

	// Validate UTF-8
	if !isValidUTF8(content) {
		return parser.ParseResult{}, wrapError("invalid UTF-8 content", nil)
	}

	// Convert to string for processing
	rtfContent := string(content)

	// Extract plain text from RTF with paragraph tracking
	plainText, totalParagraphs, err := extractPlainTextWithParagraphs(rtfContent)
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid RTF format", err)
	}

	// Validate range against actual paragraph count
	if start > totalParagraphs || end > totalParagraphs {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested paragraph range exceeds document paragraph count (document has %d paragraphs, requested %d-%d)", totalParagraphs, start, end), nil)
	}

	// Split text into paragraphs and extract requested range
	paragraphs := strings.Split(plainText, "\n")
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

// extractPlainText processes RTF content and extracts plain text.
// It handles RTF control words, groups, and escaped characters.
func extractPlainText(rtfContent string) (string, error) {
	var result strings.Builder
	inGroup := 0
	i := 0
	length := len(rtfContent)

	// Check for basic RTF structure
	if !strings.HasPrefix(rtfContent, "{\\rtf") && !strings.HasPrefix(rtfContent, "{\rtf") {
		return "", errors.New("missing RTF header")
	}

	for i < length {
		ch := rtfContent[i]

		switch {
		case ch == '{':
			// Start of a group
			inGroup++
			i++

		case ch == '}':
			// End of a group
			inGroup--
			if inGroup < 0 {
				return "", errors.New("unbalanced RTF groups")
			}
			i++

		case ch == '\\':
			// RTF control word or escaped character
			if i+1 >= length {
				i++ // Skip incomplete control word at end
				continue
			}

			nextChar := rtfContent[i+1]

			// Handle escaped characters
			if nextChar == '\'' {
				// Hexadecimal character (e.g., \'xx)
				if i+3 >= length {
					i += 2 // Skip incomplete hex sequence
					continue
				}
				hexStr := rtfContent[i+2 : i+4]
				hexBytes := []byte(hexStr)

				// Convert hex to decimal
				var charCode int
				for _, b := range hexBytes {
					charCode *= 16
					if b >= '0' && b <= '9' {
						charCode += int(b - '0')
					} else if b >= 'a' && b <= 'f' {
						charCode += int(b - 'a' + 10)
					} else if b >= 'A' && b <= 'F' {
						charCode += int(b - 'A' + 10)
					} else {
						charCode = 0
						break
					}
				}

				if charCode > 0 {
					result.WriteRune(rune(charCode))
				}
				i += 4
			} else if nextChar == 'u' {
				// Unicode character (e.g., \uN or \uN?)
				if i+2 >= length {
					i += 2 // Skip incomplete unicode sequence
					continue
				}
				unicodeChar := rtfContent[i+2]
				if unicodeChar >= '0' && unicodeChar <= '9' {
					// Simple unicode \uN
					result.WriteRune(rune(unicodeChar - '0'))
					i += 3
				} else if i+3 < length && rtfContent[i+3] == '?' {
					// Unicode with question mark \uN?
					result.WriteRune(rune(unicodeChar - '0'))
					i += 4
				} else {
					// Skip malformed unicode
					i += 2
				}
			} else if isAlpha(nextChar) {
				// Control word - skip it
				i += 2
				// Skip any digit parameter
				for i < length && isDigit(rtfContent[i]) {
					i++
				}
				// Skip space if present
				if i < length && rtfContent[i] == ' ' {
					i++
				}
			} else {
				// Escaped special character (like \\, \{, \})
				if nextChar == '\\' || nextChar == '{' || nextChar == '}' {
					result.WriteByte(nextChar)
				}
				i += 2
			}

		case ch == '\n' || ch == '\r':
			// Handle line breaks - convert to space to preserve word separation
			if result.Len() > 0 && result.String()[result.Len()-1] != ' ' {
				result.WriteByte(' ')
			}
			i++

		case ch == ' ':
			// Preserve spaces but avoid multiple consecutive spaces
			if result.Len() == 0 || result.String()[result.Len()-1] != ' ' {
				result.WriteByte(' ')
			}
			i++

		default:
			// Regular character - add to result
			if ch != '\t' && ch != 0 && ch != 0x0B { // Skip tabs, nulls, vertical tabs
				result.WriteByte(ch)
			}
			i++
		}
	}

	// Check if we have unbalanced groups at the end
	if inGroup > 0 {
		return "", errors.New("unbalanced RTF groups")
	}

	// Clean up the result
	text := result.String()

	// Replace multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")

	// Trim whitespace
	text = strings.TrimSpace(text)

	return text, nil
}

// isAlpha checks if a byte is an alphabetic character
func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isDigit checks if a byte is a digit
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// isValidUTF8 checks if the byte slice contains valid UTF-8.
func isValidUTF8(b []byte) bool {
	return utf8.Valid(b)
}

// extractPlainTextWithParagraphs processes RTF content and extracts plain text with paragraph tracking.
func extractPlainTextWithParagraphs(rtfContent string) (string, int, error) {
	var result strings.Builder
	inGroup := 0
	i := 0
	length := len(rtfContent)
	var paragraphCount int

	// Check for basic RTF structure
	if !strings.HasPrefix(rtfContent, "{\\rtf") && !strings.HasPrefix(rtfContent, "{\rtf") {
		return "", 0, errors.New("missing RTF header")
	}

	for i < length {
		ch := rtfContent[i]

		switch {
		case ch == '{':
			// Start of a group
			inGroup++
			i++

		case ch == '}':
			// End of a group
			inGroup--
			if inGroup < 0 {
				return "", 0, errors.New("unbalanced RTF groups")
			}
			i++

		case ch == '\\':
			// RTF control word or escaped character
			if i+1 >= length {
				i++ // Skip incomplete control word at end
				continue
			}

			nextChar := rtfContent[i+1]

			// Handle escaped characters
			if nextChar == '\'' {
				// Hexadecimal character (e.g., \'xx)
				if i+3 >= length {
					i += 2 // Skip incomplete hex sequence
					continue
				}
				hexStr := rtfContent[i+2 : i+4]
				hexBytes := []byte(hexStr)

				// Convert hex to decimal
				var charCode int
				for _, b := range hexBytes {
					charCode *= 16
					if b >= '0' && b <= '9' {
						charCode += int(b - '0')
					} else if b >= 'a' && b <= 'f' {
						charCode += int(b - 'a' + 10)
					} else if b >= 'A' && b <= 'F' {
						charCode += int(b - 'A' + 10)
					} else {
						charCode = 0
						break
					}
				}

				if charCode > 0 {
					result.WriteRune(rune(charCode))
				}
				i += 4
			} else if nextChar == 'u' {
				// Unicode character (e.g., \uN or \uN?)
				if i+2 >= length {
					i += 2 // Skip incomplete unicode sequence
					continue
				}
				unicodeChar := rtfContent[i+2]
				if unicodeChar >= '0' && unicodeChar <= '9' {
					// Simple unicode \uN
					result.WriteRune(rune(unicodeChar - '0'))
					i += 3
				} else if i+3 < length && rtfContent[i+3] == '?' {
					// Unicode with question mark \uN?
					result.WriteRune(rune(unicodeChar - '0'))
					i += 4
				} else {
					// Skip malformed unicode
					i += 2
				}
			} else if isAlpha(nextChar) {
				// Control word - skip it
				i += 2
				// Skip any digit parameter
				for i < length && isDigit(rtfContent[i]) {
					i++
				}
				// Skip space if present
				if i < length && rtfContent[i] == ' ' {
					i++
				}
			} else {
				// Escaped special character (like \\, \{, \})
				if nextChar == '\\' || nextChar == '{' || nextChar == '}' {
					result.WriteByte(nextChar)
				}
				i += 2
			}

		case ch == '\n' || ch == '\r':
			// Handle line breaks - convert to space to preserve word separation
			if result.Len() > 0 && result.String()[result.Len()-1] != ' ' {
				result.WriteByte(' ')
			}
			i++

		case ch == ' ':
			// Preserve spaces but avoid multiple consecutive spaces
			if result.Len() == 0 || result.String()[result.Len()-1] != ' ' {
				result.WriteByte(' ')
			}
			i++

		default:
			// Regular character - add to result
			if ch != '\t' && ch != 0 && ch != 0x0B { // Skip tabs, nulls, vertical tabs
				result.WriteByte(ch)
			}
			i++
		}
	}

	// Check if we have unbalanced groups at the end
	if inGroup > 0 {
		return "", 0, errors.New("unbalanced RTF groups")
	}

	// Clean up the result
	text := result.String()

	// Replace multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")

	// Trim whitespace
	text = strings.TrimSpace(text)

	// Count paragraphs (lines separated by newlines)
	paragraphs := strings.Split(text, "\n")
	paragraphCount = len(paragraphs)

	return text, paragraphCount, nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &RTFParserError{
			message: message,
			cause:   nil,
		}
	}
	return &RTFParserError{
		message: message,
		cause:   err,
	}
}

// ExtractParagraphs extracts paragraphs from rtf content based on the given range
func (p *Parser) ExtractParagraphs(content string, start, end int) (string, error) {
	// Split into paragraphs (separated by double newlines)
	paragraphs := strings.Split(content, "\n\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("paragraph numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid paragraph range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(paragraphs) {
		return "", nil // Out of range returns empty
	}
	if end > len(paragraphs) {
		end = len(paragraphs)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(paragraphs); i++ {
		if i > start-1 {
			result.WriteString("\n\n")
		}
		result.WriteString(paragraphs[i])
	}

	return result.String(), nil
}
