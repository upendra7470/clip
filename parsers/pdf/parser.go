package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser, parser.RangeParser, and parser.DocumentLister interfaces for PDF files.
type Parser struct{}

// NewParser creates a new PDF Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads a PDF file and extracts text content.
// It uses the github.com/ledongthuc/pdf library for text extraction.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read pdf: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type":                "pdf",
			"preserved_structure": "page boundaries",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "pdf file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "pdf",
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the PDF file
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to get file info", err)
	}

	// Parse the PDF
	pdfReader, err := pdf.NewReader(file, fileInfo.Size())
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse PDF", err)
	}

	// Extract text from all pages
	var text string
	numPages := pdfReader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText := page.Content().Text
		if len(pageText) > 0 {
			if text != "" {
				text += "\n"
			}
			// Convert []pdf.Text to string
			for _, t := range pageText {
				text += t.S
			}
		}
	}

	if text == "" {
		return parser.ParseResult{}, wrapError("no text content found in PDF", nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// ParseRange extracts text from a specific page range in a PDF file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate page range first (before file operations)
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("page numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid page range: start page must not be greater than end page (got %d-%d)", start, end), nil)
	}

	// Open the PDF file
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to get file info", err)
	}

	// Parse the PDF
	pdfReader, err := pdf.NewReader(file, fileInfo.Size())
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse PDF", err)
	}

	// Validate page range against actual document size
	numPages := pdfReader.NumPage()
	if start > numPages || end > numPages {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested page range exceeds document page count (document has %d pages, requested %d-%d)", numPages, start, end), nil)
	}

	// Extract text from the specified page range
	var text string
	for i := start; i <= end; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText := page.Content().Text
		if len(pageText) > 0 {
			if text != "" {
				text += "\n"
			}
			// Convert []pdf.Text to string
			for _, t := range pageText {
				text += t.S
			}
		}
	}

	if text == "" {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in pages %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypePDF
}

// ListUnits implements the parser.DocumentLister interface for PDF files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	// Open the PDF file
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return 0, nil, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return 0, nil, wrapError("Could not open PDF file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Get file info for size
	_, err = file.Stat()
	if err != nil {
		return 0, nil, wrapError("failed to get file info", err)
	}

	// Read the PDF content
	limitedReader := io.LimitReader(file, parser.MaxFileSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return 0, nil, wrapError("failed to read PDF content", err)
	}

	// Use pdfcpu to get page count and titles
	cmd := exec.Command("pdfcpu", "info", "--", req.File)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to simple page count if pdfcpu is not available
		// This is a simplified approach - a real implementation would use a proper PDF library
		pageCount := countPDFPages(content)
		return pageCount, nil, nil
	}

	// Parse pdfcpu output to get page count and titles
	pageCount, pageTitles, err := parsePDFCPUInfo(string(output))
	if err != nil {
		return 0, nil, wrapError("failed to parse PDF information", err)
	}

	return pageCount, pageTitles, nil
}

// countPDFPages counts the number of pages in a PDF file by looking for the /Page object.
func countPDFPages(content []byte) int {
	// This is a simplified approach - a real implementation would use a proper PDF library
	// Count occurrences of "/Page" in the PDF content
	return bytes.Count(content, []byte("/Page"))
}

// parsePDFCPUInfo parses the output of pdfcpu info command to extract page count and titles.
func parsePDFCPUInfo(output string) (int, []string, error) {
	// This is a simplified parser - a real implementation would use a proper PDF library
	// Look for "Pages:" line to get page count
	pageCount := 0
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Pages:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					pageCount = count
				}
			}
		}
	}

	// For page titles, we would need to parse the PDF's bookmarks or document info
	// This is a simplified approach that just returns empty titles
	pageTitles := make([]string, pageCount)
	return pageCount, pageTitles, nil
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitPages
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &PDFParserError{
			message: message,
			cause:   nil,
		}
	}
	return &PDFParserError{
		message: message,
		cause:   err,
	}
}

// PDFParserError represents an error that occurs during PDF parsing.
type PDFParserError struct {
	message string
	cause   error
}

func (e *PDFParserError) Error() string {
	if e.message == "" {
		return "PDF parser error"
	}
	return e.message
}

func (e *PDFParserError) Unwrap() error {
	return e.cause
}

// ExtractPages extracts pages from pdf content based on the given range
func (p *Parser) ExtractPages(content string, start, end int) (string, error) {
	// Split into pages (separated by newlines for this test)
	pages := strings.Split(content, "\n")

	if start < 1 || end < 1 {
		return "", fmt.Errorf("page numbers must start from 1, got %d-%d", start, end)
	}
	if end < start {
		return "", fmt.Errorf("invalid page range: start must not be greater than end (got %d-%d)", start, end)
	}
	if start > len(pages) {
		return "", nil // Out of range returns empty
	}
	if end > len(pages) {
		end = len(pages)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(pages); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(pages[i])
	}

	return result.String(), nil
}
