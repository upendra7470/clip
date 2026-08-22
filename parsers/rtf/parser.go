package rtf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
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
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
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

// readRTFContent reads an RTF file and validates its basic structure.
func readRTFContent(path string) ([]byte, error) {
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
		return nil, wrapError("empty RTF file", nil)
	}

	content, err := security.ReadFileBounded(path)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("error reading file %s: %v", path, err), err)
	}

	return content, nil
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	content, err := readRTFContent(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
	}

	// Validate UTF-8
	if !isValidUTF8(content) {
		return parser.ParseResult{}, wrapError("invalid UTF-8 content", nil)
	}

	// Extract plain text from RTF
	plainText, err := extractPlainText(string(content))
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid RTF syntax", err)
	}

	// Check if we got any text
	if len(plainText) == 0 {
		return parser.ParseResult{}, wrapError("no text content found in RTF", nil)
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
	content, err := readRTFContent(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}
	if len(content) > parser.MaxFileSize {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", req.File, parser.MaxFileSize), nil)
	}

	// Validate UTF-8
	if !isValidUTF8(content) {
		return parser.ParseResult{}, wrapError("invalid UTF-8 content", nil)
	}

	// Extract plain text from RTF with paragraph tracking
	plainText, totalParagraphs, err := extractPlainTextWithParagraphs(string(content))
	if err != nil {
		return parser.ParseResult{}, wrapError("invalid RTF syntax", err)
	}

	// Handle sentinel values for start
	if start == -1 {
		start = 1 // Start from beginning
	}

	// Validate paragraph range
	if start < 1 || (end != -1 && end < 1) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("paragraph numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid paragraph range: start paragraph must not be greater than end paragraph (got %d-%d)", start, end), nil)
	}

	// Normalize end value if it's -1 (open-ended range)
	if end == -1 {
		end = totalParagraphs
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

// rtfDestinationWords lists control words that begin destination groups whose
// content is metadata rather than document text (e.g. font tables).
var rtfDestinationWords = map[string]bool{
	"fonttbl":           true,
	"colortbl":          true,
	"stylesheet":        true,
	"listtable":         true,
	"listoverridetable": true,
	"rsidtbl":           true,
	"xmlnstbl":          true,
	"thautob":           true,
	"latentstyles":      true,
	"datastore":         true,
	"generator":         true,
	"filetbl":           true,
	"revtbl":            true,
	"panose":            true,
	"pict":              true,
	"object":            true,
	"mmathPr":           true,
	"info":              true,
	"title":             true,
	"subject":           true,
	"author":            true,
	"keywords":          true,
	"doccomm":           true,
	"operator":          true,
	"category":          true,
	"comments":          true,
	"company":           true,
	"template":          true,
	"version":           true,
	"comment":           true,
	"footnote":          true,
	"annotation":        true,
	"header":            true,
	"footer":            true,
	"headerl":           true,
	"headerr":           true,
	"headerf":           true,
	"footerl":           true,
	"footerr":           true,
	"footerf":           true,
	"shpinst":           true,
	"shptxt":            true,
	"nonshppict":        true,
	"fldinst":           true,
	"private":           true,
	"do":                true,
	"objdata":           true,
	"objclass":          true,
	"objname":           true,
	"result":            true,
	"pnseclvl":          true,
	"passwordhash":      true,
	"expandedcolortbl":  true,
}

// tokenizeRTF walks RTF content and returns text where source newlines act as paragraph markers.
// It handles groups (skipping destination groups such as font tables), control words,
// escaped characters, and hex/unicode escapes.
func tokenizeRTF(rtfContent string) (string, error) {
	var result strings.Builder
	var lastByte byte
	var groupStack []bool // suppression flag per open group
	i := 0
	length := len(rtfContent)

	suppressed := func() bool {
		for _, s := range groupStack {
			if s {
				return true
			}
		}
		return false
	}

	writeByte := func(b byte) {
		if suppressed() {
			return
		}
		result.WriteByte(b)
		lastByte = b
	}

	// Check for basic RTF structure
	if !strings.HasPrefix(rtfContent, "{\\rtf") {
		return "", errors.New("missing RTF header")
	}

	for i < length {
		ch := rtfContent[i]

		switch {
		case ch == '{':
			// Start of a group
			groupStack = append(groupStack, false)
			if len(groupStack) > security.MaxRTFNestingDepth {
				return "", fmt.Errorf("RTF nesting depth exceeds maximum allowed depth of %d", security.MaxRTFNestingDepth)
			}
			i++

		case ch == '}':
			// End of a group
			if len(groupStack) == 0 {
				return "", errors.New("unbalanced RTF groups")
			}
			groupStack = groupStack[:len(groupStack)-1]
			i++

		case ch == '\\':
			// RTF control word or escaped character
			if i+1 >= length {
				i++ // Skip incomplete control word at end
				continue
			}

			nextChar := rtfContent[i+1]

			switch {
			case nextChar == '*':
				// Ignorable destination (\*) - suppress the enclosing group
				if len(groupStack) > 0 {
					groupStack[len(groupStack)-1] = true
				}
				i += 2

			case nextChar == '\'':
				// Hexadecimal character (e.g., \'xx)
				if i+4 > length {
					i += 2 // Skip incomplete hex sequence
					continue
				}
				if charCode, err := strconv.ParseUint(rtfContent[i+2:i+4], 16, 8); err == nil && charCode > 0 {
					writeByte(byte(charCode))
				}
				i += 4

			case nextChar == 'u':
				// Unicode character (e.g., \uN? where N is a signed decimal code point)
				j := i + 2
				negative := false
				if j < length && rtfContent[j] == '-' {
					negative = true
					j++
				}
				codeStart := j
				for j < length && isDigit(rtfContent[j]) {
					j++
				}
				if j == codeStart {
					// No digits - malformed, skip the escape
					i += 2
					continue
				}
				code, err := strconv.Atoi(rtfContent[codeStart:j])
				if err != nil {
					i += 2
					continue
				}
				if negative {
					code = -code
				}
				// RTF uses signed 16-bit values; negative codes wrap around
				if code < 0 {
					code += 65536
				}
				var runeBuf [utf8.UTFMax]byte
				n := utf8.EncodeRune(runeBuf[:], rune(code))
				for _, b := range runeBuf[:n] {
					writeByte(b)
				}
				// Skip one fallback character if present (standard \uc1 substitution)
				if j < length && rtfContent[j] != '\\' && rtfContent[j] != '{' && rtfContent[j] != '}' && rtfContent[j] != '\n' && rtfContent[j] != '\r' {
					j++
				}
				i = j

			case isAlpha(nextChar):
				// Control word - read the full word name
				wordStart := i + 1
				j := wordStart
				for j < length && isAlpha(rtfContent[j]) {
					j++
				}
				word := rtfContent[wordStart:j]
				// Skip any digit parameter
				for j < length && isDigit(rtfContent[j]) {
					j++
				}
				// Skip single delimiter space if present
				if j < length && rtfContent[j] == ' ' {
					j++
				}
				// Destination control words suppress their whole group's text
				if len(groupStack) > 0 && !groupStack[len(groupStack)-1] && rtfDestinationWords[word] {
					groupStack[len(groupStack)-1] = true
				}
				i = j

			default:
				// Escaped special character (like \\, \{, \})
				if nextChar == '\\' || nextChar == '{' || nextChar == '}' || nextChar == '~' {
					writeByte(nextChar)
				} else if nextChar == '\n' || nextChar == '\r' {
					// \<newline> acts as a paragraph break in RTF
					writeByte('\n')
				}
				i += 2
			}

		case ch == '\n' || ch == '\r':
			// Source newlines act as paragraph separators
			writeByte('\n')
			i++

		case ch == ' ':
			// Preserve spaces but avoid consecutive spaces or leading space after a break
			if lastByte != ' ' && lastByte != '\n' {
				writeByte(' ')
			}
			i++

		default:
			// Regular character - add to result
			if ch != '\t' && ch != 0 && ch != 0x0B { // Skip tabs, nulls, vertical tabs
				writeByte(ch)
			}
			i++
		}
	}

	// Check if we have unbalanced groups at the end
	if len(groupStack) > 0 {
		return "", errors.New("unbalanced RTF groups")
	}

	return result.String(), nil
}

// splitRTFParagraphs converts tokenized text into clean, non-empty paragraphs.
func splitRTFParagraphs(tokenized string) []string {
	lines := strings.Split(tokenized, "\n")
	paragraphs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			paragraphs = append(paragraphs, line)
		}
	}
	return paragraphs
}

// extractPlainText processes RTF content and extracts plain text.
func extractPlainText(rtfContent string) (string, error) {
	tokenized, err := tokenizeRTF(rtfContent)
	if err != nil {
		return "", err
	}
	return strings.Join(splitRTFParagraphs(tokenized), "\n"), nil
}

// extractPlainTextWithParagraphs processes RTF content and extracts plain text
// along with the number of paragraphs found.
func extractPlainTextWithParagraphs(rtfContent string) (string, int, error) {
	tokenized, err := tokenizeRTF(rtfContent)
	if err != nil {
		return "", 0, err
	}
	paragraphs := splitRTFParagraphs(tokenized)
	return strings.Join(paragraphs, "\n"), len(paragraphs), nil
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
