package ocr

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/ocrbundle"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
)

type Runner interface {
	Recognize(ctx context.Context, file string) (string, error)
}

type bundledRunner struct{}

func (bundledRunner) Recognize(ctx context.Context, file string) (string, error) {
	paths, err := ocrbundle.Resolve()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, paths.Engine,
		"--detect-model", paths.DetectionModel,
		"--rec-model", paths.RecognitionModel,
		file,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("built-in OCR failed: %s", detail)
	}
	return string(output), nil
}

type Parser struct {
	runner Runner
}

func NewParser() *Parser {
	return &Parser{runner: bundledRunner{}}
}

func NewParserWithRunner(runner Runner) *Parser {
	p := NewParser()
	p.runner = runner
	return p
}

func Available() (string, bool) {
	paths, err := ocrbundle.Resolve()
	return paths.Engine, err == nil
}

func Recognize(ctx context.Context, file string) (string, error) {
	prepared, cleanup, err := prepareImage(file)
	if err != nil {
		return "", err
	}
	defer cleanup()
	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()
	return bundledRunner{}.Recognize(ocrCtx, prepared)
}

func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	prepared, cleanup, err := prepareImage(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}
	defer cleanup()

	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()

	text, err := p.runner.Recognize(ocrCtx, prepared)
	if err != nil {
		return parser.ParseResult{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return parser.ParseResult{}, fmt.Errorf("OCR found no readable text in image")
	}
	return parser.ParseResult{Text: text}, nil
}

func validateImage(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open image: %w", err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return "", fmt.Errorf("invalid or unsupported image: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return "", fmt.Errorf("invalid image dimensions")
	}
	if config.Width > security.MaxImageDimension || config.Height > security.MaxImageDimension {
		return "", fmt.Errorf("image dimensions %dx%d exceed the %d-pixel limit", config.Width, config.Height, security.MaxImageDimension)
	}
	if uint64(config.Width)*uint64(config.Height) > security.MaxImagePixels {
		return "", fmt.Errorf("image contains too many pixels for safe OCR")
	}
	if format == "" {
		return "", fmt.Errorf("unsupported image format")
	}
	return format, nil
}

func prepareImage(path string) (string, func(), error) {
	format, err := validateImage(path)
	if err != nil {
		return "", func() {}, err
	}
	if format == "png" || format == "jpeg" || format == "webp" {
		return path, func() {}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", func() {}, fmt.Errorf("open image for conversion: %w", err)
	}
	imageData, _, err := image.Decode(file)
	file.Close()
	if err != nil {
		return "", func() {}, fmt.Errorf("decode image for OCR: %w", err)
	}
	temp, err := os.CreateTemp("", "clip-ocr-*.png")
	if err != nil {
		return "", func() {}, fmt.Errorf("prepare image for OCR: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() { os.Remove(tempPath) }
	if err := png.Encode(temp, imageData); err != nil {
		temp.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("convert image for OCR: %w", err)
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("finish image conversion: %w", err)
	}
	return tempPath, cleanup, nil
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
