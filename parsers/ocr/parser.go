package ocr

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
)

type Runner interface {
	Recognize(ctx context.Context, file, language string, pageSegMode int) (string, error)
}

type tesseractRunner struct{}

func (tesseractRunner) Recognize(ctx context.Context, file, language string, pageSegMode int) (string, error) {
	path, err := exec.LookPath("tesseract")
	if err != nil {
		return "", fmt.Errorf("OCR engine unavailable: install Tesseract, then run clip again")
	}

	cmd := exec.CommandContext(ctx, path, file, "stdout", "-l", language, "--psm", strconv.Itoa(pageSegMode))
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("Tesseract OCR failed: %s", detail)
	}
	return string(output), nil
}

type Parser struct {
	Language    string
	PageSegMode int
	runner      Runner
}

func NewParser(language string, pageSegMode int) *Parser {
	if language == "" {
		language = "eng"
	}
	if pageSegMode < 0 || pageSegMode > 13 {
		pageSegMode = 3
	}
	return &Parser{
		Language:    language,
		PageSegMode: pageSegMode,
		runner:      tesseractRunner{},
	}
}

func NewParserWithRunner(language string, pageSegMode int, runner Runner) *Parser {
	p := NewParser(language, pageSegMode)
	p.runner = runner
	return p
}

func Available() (string, bool) {
	path, err := exec.LookPath("tesseract")
	return path, err == nil
}

func PDFRendererAvailable() (string, bool) {
	path, err := exec.LookPath("pdftoppm")
	return path, err == nil
}

func Recognize(ctx context.Context, file, language string, pageSegMode int) (string, error) {
	if err := validateImage(file); err != nil {
		return "", err
	}
	if language == "" {
		language = "eng"
	}
	if pageSegMode < 0 || pageSegMode > 13 {
		pageSegMode = 3
	}
	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()
	return tesseractRunner{}.Recognize(ocrCtx, file, language, pageSegMode)
}

func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	if err := validateImage(req.File); err != nil {
		return parser.ParseResult{}, err
	}

	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()

	text, err := p.runner.Recognize(ocrCtx, req.File, p.Language, p.PageSegMode)
	if err != nil {
		return parser.ParseResult{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return parser.ParseResult{}, fmt.Errorf("OCR found no readable text in image")
	}
	return parser.ParseResult{Text: text}, nil
}

func validateImage(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open image: %w", err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("invalid or unsupported image: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return fmt.Errorf("invalid image dimensions")
	}
	if config.Width > security.MaxImageDimension || config.Height > security.MaxImageDimension {
		return fmt.Errorf("image dimensions %dx%d exceed the %d-pixel limit", config.Width, config.Height, security.MaxImageDimension)
	}
	if uint64(config.Width)*uint64(config.Height) > security.MaxImagePixels {
		return fmt.Errorf("image contains too many pixels for safe OCR")
	}
	if format == "" {
		return fmt.Errorf("unsupported image format")
	}
	return nil
}

func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeImage
}

func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("OCR requires a file path")
}

func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	result, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err != nil {
		return nil, err
	}
	return &parser.DocumentUnit{
		Text: result.Text,
		Meta: map[string]interface{}{"path": path, "type": "image", "ocr": true},
	}, nil
}

func (p *Parser) ParseDirectory(string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("OCR directory parsing is not supported")
}
