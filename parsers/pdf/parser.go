package pdf

import (
	"context"
	"os"

	"github.com/ledongthuc/pdf"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser interface for PDF files.
type Parser struct{}

// Parse reads a PDF file and extracts text content.
// It uses the github.com/ledongthuc/pdf library for text extraction.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the PDF file
	file, err := os.Open(req.File)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to open PDF", err)
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

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypePDF
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