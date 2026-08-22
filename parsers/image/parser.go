// Package image implements the parser.Parser interface for image files
// (PNG, JPEG, WebP) by running local OCR and extracting visible text.
package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
	_ "golang.org/x/image/webp"
)

// Engine abstracts the local OCR engine so the parser can be tested
// without an actual engine installed.
type Engine interface {
	// Name returns a human-readable engine name for diagnostics.
	Name() string
	// ExtractText runs OCR on raw image bytes and returns detected text.
	ExtractText(ctx context.Context, data []byte) (string, error)
}

// ImageParserError represents an error that occurs during image parsing.
type ImageParserError struct {
	message string
	cause   error
}

func (e *ImageParserError) Error() string {
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
	if e.message == "" {
		return "image parser error"
	}
	return e.message
}

func (e *ImageParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser interface for image files using OCR.
type Parser struct {
	engine Engine

	// MaxSize overrides the maximum accepted image size in bytes.
	// Zero means parser.MaxFileSize is used. Intended for testing.
	MaxSize int64
}

// NewParser creates a new image Parser backed by the default OCR engine.
func NewParser() *Parser {
	return &Parser{engine: DefaultEngine}
}

// NewParserWithEngine creates an image Parser that uses the provided OCR
// engine. Intended for testing.
func NewParserWithEngine(engine Engine) *Parser {
	return &Parser{engine: engine}
}

// Parse reads image bytes and extracts readable text via OCR.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	data, err := security.ReadAllBounded(reader, parser.MaxFileSize)
	if err != nil {
		return nil, wrapError("failed to read image", err)
	}
	text, err := p.extractText(context.Background(), data)
	if err != nil {
		return nil, err
	}
	return &parser.DocumentUnit{
		Text: text,
		Meta: map[string]interface{}{
			"type": "image",
			"ocr":  true,
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "image file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": "image",
			"ocr":  true,
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context.
// It validates the image, runs OCR through the configured engine, and returns
// the extracted text.
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	content, err := security.ReadFileBounded(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return parser.ParseResult{}, wrapError(fmt.Sprintf("permission denied reading file %s", req.File), err)
		}
		return parser.ParseResult{}, wrapError(fmt.Sprintf("error reading file %s: %v", req.File, err), err)
	}

	text, err := p.extractText(ctx, content)
	if err != nil {
		return parser.ParseResult{}, err
	}

	return parser.ParseResult{
		Text: text,
	}, nil
}

// extractText validates the raw image bytes against size/dimension limits
// and runs it through the configured OCR engine.
func (p *Parser) extractText(ctx context.Context, content []byte) (string, error) {
	if len(content) == 0 {
		return "", wrapError("empty image file", nil)
	}
	maxSize := int64(parser.MaxFileSize)
	if p.MaxSize > 0 {
		maxSize = p.MaxSize
	}
	if int64(len(content)) > maxSize {
		return "", wrapError(fmt.Sprintf("image exceeds maximum allowed size of %d bytes", maxSize), nil)
	}
	if err := validateImageDimensions(content); err != nil {
		return "", err
	}

	if p.engine == nil {
		p.engine = DefaultEngine
	}
	text, err := p.engine.ExtractText(ctx, content)
	if err != nil {
		return "", err
	}

	// Normalize engine output: strip trailing separators and blank lines.
	text = strings.TrimRight(text, "\f\n\r\t ")
	if strings.TrimSpace(text) == "" {
		return "", wrapError("no text detected in image", nil)
	}

	return text, nil
}

// validateImageDimensions decodes only the image header (no full pixel
// decompression) and enforces the configured dimension limits. This guards
// against decompression bombs before the image reaches the OCR engine.
func validateImageDimensions(content []byte) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return wrapError("unsupported or corrupt image: could not decode header", err)
	}
	return checkImageDimensions(cfg.Width, cfg.Height)
}

// checkImageDimensions enforces the configured image dimension and pixel
// count limits. It is a pure function so limits can be tested directly.
func checkImageDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return wrapError(fmt.Sprintf("invalid image dimensions %dx%d", width, height), nil)
	}
	if width > security.MaxImageDimension || height > security.MaxImageDimension {
		return wrapError(fmt.Sprintf("image dimensions %dx%d exceed maximum allowed dimension of %d pixels per side", width, height, security.MaxImageDimension), nil)
	}
	if int64(width)*int64(height) > security.MaxImagePixels {
		return wrapError(fmt.Sprintf("image has %d pixels, exceeding maximum allowed %d pixels", int64(width)*int64(height), security.MaxImagePixels), nil)
	}
	return nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeImage
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
// Ranges operate on lines of extracted text.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitLines
}

// ParseRange extracts specific lines of OCR text from an image.
// The image is re-processed with OCR for the requested range.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Validate line range up front; -1 as end means "to end".
	if start < 1 || (end != -1 && end < 1) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("line numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid line range: start line must not be greater than end line (got %d-%d)", start, end), nil)
	}

	result, err := p.ParseWithContext(ctx, req)
	if err != nil {
		return parser.ParseResult{}, err
	}

	lines := strings.Split(result.Text, "\n")

	// Normalize open-ended range
	if end == -1 {
		end = len(lines)
	}

	if start > len(lines) || end > len(lines) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested line range exceeds extracted text line count (text has %d lines, requested %d-%d)", len(lines), start, end), nil)
	}

	var sb strings.Builder
	for i := start - 1; i < end; i++ {
		if i > start-1 {
			sb.WriteString("\n")
		}
		sb.WriteString(lines[i])
	}

	if sb.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in lines %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: sb.String(),
	}, nil
}

// ListUnits implements the parser.DocumentLister interface for image files.
// Units are the non-empty lines of extracted OCR text.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	result, err := p.ParseWithContext(ctx, req)
	if err != nil {
		return 0, nil, err
	}

	lineCount := 0
	for _, line := range strings.Split(result.Text, "\n") {
		if strings.TrimSpace(line) != "" {
			lineCount++
		}
	}
	return lineCount, nil, nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	return &ImageParserError{
		message: message,
		cause:   err,
	}
}
