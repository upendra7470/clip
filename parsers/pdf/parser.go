package pdf

import (
	"context"
	"fmt"
	"os"

	"github.com/ledongthuc/pdf"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser and parser.RangeParser interfaces for PDF files.
type Parser struct{}

// Parse reads a PDF file and extracts text content.
// It uses the github.com/ledongthuc/pdf library for text extraction.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
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

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "pages"
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
