package pdf

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/upendra7470/clip/internal/filetype"
	clipLayout "github.com/upendra7470/clip/internal/layout"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/parsers/ocr"
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
		return p.ocrPages(ctx, req.File, 1, -1, req.Selection)
	}

	mode, err := clipLayout.ParseMode(req.Selection.Layout)
	if err != nil {
		return parser.ParseResult{}, err
	}
	// Extract each page independently so mixed PDFs can use native text on
	// digital pages and OCR only on scanned pages.
	var pages []string
	numPages := pdfReader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText := renderPage(page.Content().Text, mode)
		if pageText == "" {
			ocrResult, ocrErr := p.ocrPages(ctx, req.File, i, i, req.Selection)
			if ocrErr != nil {
				continue
			}
			pageText = ocrResult.Text
		}
		if pageText != "" {
			pages = append(pages, pageText)
		}
	}

	if len(pages) == 0 {
		return p.ocrPages(ctx, req.File, 1, numPages, req.Selection)
	}

	return parser.ParseResult{
		Text: strings.Join(pages, "\n\n"),
	}, nil
}

// ParseRange extracts text from a specific page range in a PDF file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate page range first (before file operations)
	if start < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("page numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("page numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
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
		return p.ocrPages(ctx, req.File, start, end, req.Selection)
	}

	// Validate page range against actual document size
	numPages := pdfReader.NumPage()
	if end == -1 {
		end = numPages
	}
	if start > numPages || end > numPages {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested page range exceeds document page count (document has %d pages, requested %d-%d)", numPages, start, end), nil)
	}

	mode, err := clipLayout.ParseMode(req.Selection.Layout)
	if err != nil {
		return parser.ParseResult{}, err
	}
	// Extract text from the specified page range, OCRing scanned pages in
	// place instead of flattening the entire range.
	var pages []string
	for i := start; i <= end; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText := renderPage(page.Content().Text, mode)
		if pageText == "" {
			ocrResult, ocrErr := p.ocrPages(ctx, req.File, i, i, req.Selection)
			if ocrErr != nil {
				continue
			}
			pageText = ocrResult.Text
		}
		if pageText != "" {
			pages = append(pages, pageText)
		}
	}

	if len(pages) == 0 {
		return p.ocrPages(ctx, req.File, start, end, req.Selection)
	}

	return parser.ParseResult{
		Text: strings.Join(pages, "\n\n"),
	}, nil
}

func renderPage(fragments []pdf.Text, mode clipLayout.Mode) string {
	tokens := make([]clipLayout.Token, 0, len(fragments))
	for _, fragment := range fragments {
		height := fragment.FontSize
		if height <= 0 {
			height = 1
		}
		tokens = append(tokens, clipLayout.Token{
			Text: fragment.S,
			X:    fragment.X,
			// PDF coordinates increase bottom-to-top and Y is a baseline.
			// Centering around the negated baseline groups mixed font sizes.
			Y:      -fragment.Y - height/2,
			Width:  fragment.W,
			Height: height,
		})
	}
	return clipLayout.Render(tokens, mode)
}

func (p *Parser) ocrPages(ctx context.Context, path string, start, end int, selection parser.Selection) (parser.ParseResult, error) {
	tempDir, err := os.MkdirTemp("", "clip-pdf-ocr-*")
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to prepare PDF OCR", err)
	}
	defer os.RemoveAll(tempDir)

	selectedPages := []string{fmt.Sprint(start)}
	if end != -1 {
		selectedPages[0] = fmt.Sprintf("%d-%d", start, end)
	} else {
		selectedPages[0] = fmt.Sprintf("%d-", start)
	}
	conf := model.NewStatelessConfiguration()
	if err := api.ExtractImagesFile(ctx, path, tempDir, selectedPages, conf); err != nil {
		return parser.ParseResult{}, wrapError("failed to extract scanned PDF images for OCR", err)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to find extracted PDF images", err)
	}
	images := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			images = append(images, filepath.Join(tempDir, entry.Name()))
		}
	}
	sort.Strings(images)
	if len(images) == 0 {
		return parser.ParseResult{}, wrapError("PDF contains no embedded text or extractable scanned images", nil)
	}

	var result strings.Builder
	for index, imagePath := range images {
		text, err := ocr.RecognizeSmartLayout(ctx, imagePath, selection.Smart, selection.Layout)
		if err != nil {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("failed to OCR extracted PDF image %d", index+1), err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if result.Len() > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(text)
	}
	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("OCR found no readable text in PDF pages %d-%d", start, end), nil)
	}
	return parser.ParseResult{Text: result.String()}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypePDF
}

// ListUnits implements the parser.DocumentLister interface for PDF files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
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

	fileInfo, err := file.Stat()
	if err != nil {
		return 0, nil, wrapError("failed to get file info", err)
	}

	pdfReader, err := pdf.NewReader(file, fileInfo.Size())
	if err != nil {
		return 0, nil, wrapError("failed to parse PDF", err)
	}

	numPages := pdfReader.NumPage()
	pageTitles := make([]string, numPages)
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		// Use the first line of page text as the title, if any.
		texts := page.Content().Text
		if len(texts) > 0 {
			title := strings.TrimSpace(texts[0].S)
			if len(title) > 80 {
				title = title[:80] + "…"
			}
			pageTitles[i-1] = title
		}
	}

	return numPages, pageTitles, nil
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
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
	if e.message == "" {
		return "PDF parser error"
	}
	return e.message
}

func (e *PDFParserError) Unwrap() error {
	return e.cause
}
