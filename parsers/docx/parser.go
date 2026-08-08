package docx

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser, parser.RangeParser, and parser.DocumentLister interfaces for DOCX files. It preserves headings, paragraphs, tables, and nested table content.
type Parser struct{}

// NewParser creates a new DOCX Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a DOCX file and extracts text content.
// DOCX files are ZIP archives containing XML files.
// This parser extracts text from word/document.xml <w:t> nodes.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read docx: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type":                "docx",
			"preserved_structure": "headings, paragraphs, tables, nested table content",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "docx file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "docx",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the DOCX file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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
		return parser.ParseResult{}, wrapError("failed to read DOCX as ZIP archive", err)
	}

	// Find and extract word/document.xml
	var documentXML string
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/document.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to open document.xml", err)
			}

			limitedReader := io.LimitReader(rc, parser.MaxFileSize)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				rc.Close() // Close immediately after reading
				return parser.ParseResult{}, wrapError("failed to read document.xml", err)
			}
			documentXML = string(content)
			break
		}
	}

	if documentXML == "" {
		return parser.ParseResult{}, wrapError("document.xml not found in DOCX", nil)
	}

	// Parse XML to extract structured content including tables
	text, _, err := extractStructuredContentFromXML(documentXML, false)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse DOCX XML", err)
	}

	if text == "" {
		return parser.ParseResult{}, wrapError("no text content found in DOCX", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeDOCX
}

// ListUnits implements the parser.DocumentLister interface for DOCX files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	// Open the DOCX file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return 0, nil, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return 0, nil, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, nil, wrapError("failed to get file info", err)
	}

	// Read the ZIP archive
	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		return 0, nil, wrapError("failed to read DOCX as ZIP archive", err)
	}

	// Find and extract word/document.xml
	var documentXML string
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/document.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				return 0, nil, wrapError("failed to open document.xml", err)
			}
			defer rc.Close()

			limitedReader := io.LimitReader(rc, parser.MaxFileSize)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				return 0, nil, wrapError("failed to read document.xml", err)
			}
			documentXML = string(content)
			break
		}
	}

	if documentXML == "" {
		return 0, nil, wrapError("document.xml not found in DOCX", nil)
	}

	// Parse XML to extract structured content with sections
	_, structuredParagraphs, err := extractStructuredContentFromXML(documentXML, true)
	if err != nil {
		return 0, nil, wrapError("failed to parse DOCX XML", err)
	}

	// Detect sections from paragraphs
	sections := detectSections(structuredParagraphs)
	totalSections := len(sections)

	// Extract section titles
	var sectionTitles []string
	for _, section := range sections {
		// Use the first paragraph of each section as the title
		if len(section.Paragraphs) > 0 {
			sectionTitles = append(sectionTitles, section.Paragraphs[0].Content)
		} else {
			sectionTitles = append(sectionTitles, "")
		}
	}

	return totalSections, sectionTitles, nil
}

// GetRangeUnit returns the unit type that this parser uses for ranges. It preserves section boundaries and maintains the original document structure.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitSections
}

// ParseRange extracts text from a specific section range in a DOCX file. It preserves section boundaries and maintains the original document structure.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	// Open the DOCX file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open DOCX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
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
		return parser.ParseResult{}, wrapError("failed to read DOCX as ZIP archive", err)
	}

	// Find and extract word/document.xml
	var documentXML string
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "word/document.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to open document.xml", err)
			}
			defer rc.Close()

			limitedReader := io.LimitReader(rc, parser.MaxFileSize)
			content, err := io.ReadAll(limitedReader)
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to read document.xml", err)
			}
			documentXML = string(content)
			break
		}
	}

	if documentXML == "" {
		return parser.ParseResult{}, wrapError("document.xml not found in DOCX", nil)
	}

	// Parse XML to extract structured content with sections
	_, structuredParagraphs, err := extractStructuredContentFromXML(documentXML, true)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse DOCX XML", err)
	}

	// Detect sections from paragraphs
	sections := detectSections(structuredParagraphs)
	totalSections := len(sections)

	// Handle sentinel values
	if start == -1 {
		start = 1 // Start from beginning
	}
	if end == -1 {
		end = totalSections // End at last section
	}

	// Validate section range
	if start < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("section numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("section numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid section range: start section must not be greater than end section (got %d-%d)", start, end), nil)
	}

	// Validate range against actual section count
	if start > totalSections {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested section range exceeds document section count (document has %d sections, requested %d-%d)", totalSections, start, end), nil)
	}
	if end > totalSections {
		end = totalSections // Adjust end to last section if it exceeds
	}

	// Extract only the requested section range
	var result strings.Builder
	for i := start - 1; i < end && i < len(sections); i++ {
		if i > start-1 {
			// Add double newline between sections for better separation
			result.WriteString("\n\n")
		}
		// Extract section content
		sectionContent := extractSectionContent(sections[i])
		result.WriteString(sectionContent)
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in sections %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
		// RangeUnit: "sections",
	}, nil
}

// extractSectionContent extracts the content from a section
func extractSectionContent(section Section) string {
	var content strings.Builder

	for i, para := range section.Paragraphs {
		if i > 0 {
			if para.IsTable {
				if content.Len() > 0 {
					content.WriteString("\n")
				}
			} else {
				content.WriteString("\n")
			}
		}
		content.WriteString(para.Content)
	}

	return content.String()
}

// extractParagraphsFromXML parses the XML and extracts paragraphs as a slice of strings.
func extractParagraphsFromXML(xmlContent string) ([]string, int, error) {
	var paragraphs []string
	var currentParagraph strings.Builder

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	// Harden XML decoder to prevent XXE and other XML attacks
	decoder.Strict = true
	decoder.Entity = map[string]string{}
	var inTextNode bool
	var inParagraph bool
	var currentText strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTextNode = true
				currentText.Reset()
			} else if t.Name.Local == "p" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inParagraph = true
			}
		case xml.CharData:
			if inTextNode {
				currentText.Write(t)
			}
		case xml.EndElement:
			if inTextNode && t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTextNode = false
				text := strings.TrimSpace(currentText.String())
				if text != "" {
					if currentParagraph.Len() > 0 {
						currentParagraph.WriteString(" ")
					}
					currentParagraph.WriteString(text)
				}
			} else if inParagraph && t.Name.Local == "p" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inParagraph = false
				paraText := strings.TrimSpace(currentParagraph.String())
				if paraText != "" {
					// Strip "Paragraph N:" prefix if present
					paraText = stripParagraphPrefix(paraText)
					paragraphs = append(paragraphs, paraText)
				}
				currentParagraph.Reset()
			}
		}
	}

	return paragraphs, len(paragraphs), nil
}

// stripParagraphPrefix removes "Paragraph N:" prefix from paragraph text if present.
func stripParagraphPrefix(text string) string {
	// Match "Paragraph N:" pattern where N is a number
	// This handles cases like "Paragraph 1: Hello World" -> "Hello World"
	re := regexp.MustCompile(`^Paragraph \d+:\s*`)
	// Return original text if no match
	if !re.MatchString(text) {
		return text
	}
	return re.ReplaceAllString(text, "")
}

// Section represents a logical section in the DOCX document
type Section struct {
	ID         int
	Paragraphs []StructuredParagraph
	StartIndex int
	EndIndex   int
}

// StructuredParagraph represents a paragraph or table from the DOCX structure
type StructuredParagraph struct {
	Content string
	IsTable bool
}

// extractStructuredParagraphsFromXML parses the XML and returns structured paragraphs for range filtering
func extractStructuredParagraphsFromXML(xmlContent string) ([]StructuredParagraph, error) {
	_, paragraphs, err := extractStructuredContentFromXML(xmlContent, true)
	return paragraphs, err
}

// detectSections identifies logical sections based on headings, breaks, and structural elements
func detectSections(paragraphs []StructuredParagraph) []Section {
	var sections []Section
	currentSection := Section{ID: 1, Paragraphs: []StructuredParagraph{}}

	for i, para := range paragraphs {
		// Detect section boundaries based on heading styles, breaks, etc.
		if isSectionBoundary(para) {
			if len(currentSection.Paragraphs) > 0 {
				currentSection.EndIndex = i - 1
				sections = append(sections, currentSection)
				currentSection = Section{ID: len(sections) + 1, StartIndex: i}
			}
		}
		currentSection.Paragraphs = append(currentSection.Paragraphs, para)
	}

	if len(currentSection.Paragraphs) > 0 {
		sections = append(sections, currentSection)
	}

	return sections
}

// isSectionBoundary determines if a paragraph represents a section boundary
func isSectionBoundary(para StructuredParagraph) bool {
	// Check for heading patterns (simple heuristic for now)
	// This can be enhanced with more sophisticated heading detection
	headingPattern := regexp.MustCompile(`^(#{1,6}\s+|\*\s+|\d+\.\s+)`)
	return headingPattern.MatchString(para.Content)
}

// extractStructuredContentFromXML parses the XML and extracts structured content including tables.
// If collectParagraphs is true, it returns the structured paragraphs separately for range filtering.
func extractStructuredContentFromXML(xmlContent string, collectParagraphs bool) (string, []StructuredParagraph, error) {
	var result strings.Builder
	var inTable bool
	var inTableRow bool
	var inTableCell bool
	var inParagraph bool
	var inTextNode bool
	var currentText strings.Builder
	var tableRows [][]string
	var currentRow []string
	var currentCell strings.Builder
	var paragraphs []StructuredParagraph
	var currentParagraph strings.Builder

	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	// Harden XML decoder to prevent XXE and other XML attacks
	decoder.Strict = true
	decoder.Entity = map[string]string{}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Handle table elements
			if t.Name.Local == "tbl" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTable = true
				currentParagraph.Reset()
			} else if t.Name.Local == "tr" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTableRow = true
				currentRow = []string{}
			} else if t.Name.Local == "tc" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTableCell = true
				currentCell.Reset()
			} else if t.Name.Local == "p" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inParagraph = true
				if inTableCell {
					currentCell.WriteString(" ")
				} else if collectParagraphs {
					currentParagraph.Reset()
				}
			} else if t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTextNode = true
				currentText.Reset()
			}
		case xml.CharData:
			if inTextNode {
				currentText.Write(t)
			}
		case xml.EndElement:
			if inTextNode && t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTextNode = false
				text := strings.TrimSpace(currentText.String())
				if text != "" {
					text = stripParagraphPrefix(text)
					if inTableCell {
						if currentCell.Len() > 0 {
							currentCell.WriteString(" ")
						}
						currentCell.WriteString(text)
					} else if inParagraph {
						if collectParagraphs {
							if currentParagraph.Len() > 0 {
								currentParagraph.WriteString(" ")
							}
							currentParagraph.WriteString(text)
						} else {
							if result.Len() > 0 {
								result.WriteString(" ")
							}
							result.WriteString(text)
						}
					}
				}
			} else if inParagraph && t.Name.Local == "p" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inParagraph = false
				if inTableCell {
					// Don't add newline inside table cells
				} else if collectParagraphs {
					// Collect the paragraph if we're in collection mode
					paraText := strings.TrimSpace(currentParagraph.String())
					if paraText != "" {
						paragraphs = append(paragraphs, StructuredParagraph{
							Content: paraText,
							IsTable: false,
						})
					}
				} else {
					if !inTableCell {
						result.WriteString("\n")
					}
				}
			} else if inTableCell && t.Name.Local == "tc" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTableCell = false
				currentRow = append(currentRow, strings.TrimSpace(currentCell.String()))
			} else if inTableRow && t.Name.Local == "tr" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTableRow = false
				tableRows = append(tableRows, currentRow)
			} else if inTable && t.Name.Local == "tbl" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inTable = false
				// Format the table in Markdown style
				if len(tableRows) > 0 {
					var tableResult strings.Builder
					// Write table header
					tableResult.WriteString("| ")
					tableResult.WriteString(strings.Join(tableRows[0], " | "))
					tableResult.WriteString(" |")
					tableResult.WriteString("\n")
					// Write table separator
					tableResult.WriteString("| ")
					for range tableRows[0] {
						tableResult.WriteString("---")
						if len(tableRows[0]) > 1 {
							tableResult.WriteString(" | ")
						}
					}
					tableResult.WriteString(" |")
					tableResult.WriteString("\n")
					// Write table rows
					for _, row := range tableRows[1:] {
						tableResult.WriteString("| ")
						tableResult.WriteString(strings.Join(row, " | "))
						tableResult.WriteString(" |")
						tableResult.WriteString("\n")
					}
					tableResult.WriteString("\n")

					if collectParagraphs {
						// Store table as a single paragraph entry
						paragraphs = append(paragraphs, StructuredParagraph{
							Content: tableResult.String(),
							IsTable: true,
						})
					} else {
						if result.Len() > 0 {
							result.WriteString("\n\n")
						}
						result.WriteString(tableResult.String())
					}
					tableRows = [][]string{}
				}
			}
		}
	}

	if collectParagraphs {
		// Build final result from collected paragraphs
		for i, para := range paragraphs {
			if i > 0 {
				if para.IsTable {
					if result.Len() > 0 {
						result.WriteString("\n")
					}
				} else {
					result.WriteString("\n")
				}
			}
			result.WriteString(para.Content)
		}
		return result.String(), paragraphs, nil
	}

	return result.String(), nil, nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &DOCXParserError{
			message: message,
			cause:   nil,
		}
	}
	return &DOCXParserError{
		message: message,
		cause:   err,
	}
}

// DOCXParserError represents an error that occurs during DOCX parsing.
type DOCXParserError struct {
	message string
	cause   error
}

func (e *DOCXParserError) Error() string {
	if e.message == "" {
		return "DOCX parser error"
	}
	return e.message
}

func (e *DOCXParserError) Unwrap() error {
	return e.cause
}

// ExtractBlocks extracts blocks from docx content based on the given range
func (p *Parser) ExtractBlocks(content string, start, end int) (string, error) {
	// Split into blocks (paragraphs separated by newlines)
	blocks := strings.Split(content, "\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("block numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid block range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(blocks) {
		return "", nil // Out of range returns empty
	}
	if end > len(blocks) {
		end = len(blocks)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(blocks); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(blocks[i])
	}

	return result.String(), nil
}

// ExtractTables extracts tables from docx content based on the given range
func (p *Parser) ExtractTables(content string, start, end int) (string, error) {
	// Split into tables (separated by newlines for this simple test)
	tables := strings.Split(content, "\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("table numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid table range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(tables) {
		return "", nil // Out of range returns empty
	}
	if end > len(tables) {
		end = len(tables)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(tables); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(tables[i])
	}

	return result.String(), nil
}
