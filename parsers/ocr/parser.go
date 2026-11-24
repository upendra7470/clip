package ocr

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode"

	_ "golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
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
	return RecognizeSmart(ctx, file, false)
}

// RecognizeSmart performs OCR with optional contrast normalization and
// high-quality upscaling for small scans.
func RecognizeSmart(ctx context.Context, file string, smart bool) (string, error) {
	prepared, cleanup, err := prepareImage(file, smart)
	if err != nil {
		return "", err
	}
	defer cleanup()
	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()
	if smart {
		return recognizeBestOrientation(ocrCtx, bundledRunner{}, prepared)
	}
	return bundledRunner{}.Recognize(ocrCtx, prepared)
}

func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	prepared, cleanup, err := prepareImage(req.File, req.Selection.Smart)
	if err != nil {
		return parser.ParseResult{}, err
	}
	defer cleanup()

	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()

	var text string
	if req.Selection.Smart {
		text, err = recognizeBestOrientation(ocrCtx, p.runner, prepared)
	} else {
		text, err = p.runner.Recognize(ocrCtx, prepared)
	}
	if err != nil {
		return parser.ParseResult{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return parser.ParseResult{}, fmt.Errorf("OCR found no readable text in image")
	}
	return parser.ParseResult{Text: text}, nil
}

func recognizeBestOrientation(ctx context.Context, runner Runner, path string) (string, error) {
	bestText, err := runner.Recognize(ctx, path)
	if err != nil {
		return "", err
	}
	bestScore := textScore(bestText)
	for _, degrees := range []int{90, 180, 270} {
		rotated, cleanup, err := rotatedImage(path, degrees)
		if err != nil {
			return "", err
		}
		candidate, recognizeErr := runner.Recognize(ctx, rotated)
		cleanup()
		if recognizeErr != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			continue
		}
		if score := textScore(candidate); score > bestScore {
			bestText, bestScore = candidate, score
		}
	}
	return bestText, nil
}

func textScore(text string) int {
	score := 0
	for _, value := range text {
		if unicode.IsLetter(value) || unicode.IsDigit(value) {
			score++
		}
	}
	return score
}

func rotatedImage(path string, degrees int) (string, func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return "", func() {}, fmt.Errorf("open image for orientation detection: %w", err)
	}
	source, _, err := image.Decode(file)
	file.Close()
	if err != nil {
		return "", func() {}, fmt.Errorf("decode image for orientation detection: %w", err)
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	var target *image.Gray
	if degrees == 90 || degrees == 270 {
		target = image.NewGray(image.Rect(0, 0, height, width))
	} else {
		target = image.NewGray(image.Rect(0, 0, width, height))
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := color.GrayModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray)
			switch degrees {
			case 90:
				target.Set(height-1-y, x, value)
			case 180:
				target.Set(width-1-x, height-1-y, value)
			case 270:
				target.Set(y, width-1-x, value)
			}
		}
	}
	temp, err := os.CreateTemp("", fmt.Sprintf("clip-ocr-%d-*.png", degrees))
	if err != nil {
		return "", func() {}, fmt.Errorf("prepare OCR orientation: %w", err)
	}
	path = temp.Name()
	cleanup := func() { os.Remove(path) }
	if err := png.Encode(temp, target); err != nil {
		temp.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("encode OCR orientation: %w", err)
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("finish OCR orientation: %w", err)
	}
	return path, cleanup, nil
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

func prepareImage(path string, smart bool) (string, func(), error) {
	format, err := validateImage(path)
	if err != nil {
		return "", func() {}, err
	}
	if !smart && (format == "png" || format == "jpeg" || format == "webp") {
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
	if smart {
		const maxSmartPixels = 25_000_000
		bounds := imageData.Bounds()
		if uint64(bounds.Dx())*uint64(bounds.Dy()) > maxSmartPixels {
			return "", func() {}, fmt.Errorf("smart OCR image exceeds the %d-pixel preprocessing limit", maxSmartPixels)
		}
		imageData = enhanceForOCR(imageData)
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

func enhanceForOCR(source image.Image) image.Image {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	gray := image.NewGray(image.Rect(0, 0, width, height))
	var histogram [256]int
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := color.GrayModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray).Y
			gray.SetGray(x, y, color.Gray{Y: value})
			histogram[value]++
		}
	}
	low, high := percentileBounds(histogram, width*height)
	if high > low {
		for index, value := range gray.Pix {
			switch {
			case int(value) <= low:
				gray.Pix[index] = 0
			case int(value) >= high:
				gray.Pix[index] = 255
			default:
				gray.Pix[index] = uint8((int(value) - low) * 255 / (high - low))
			}
		}
	}
	if width >= 1400 {
		return gray
	}
	scale := 2
	if width*scale > 2400 || height*scale > 2400 {
		return gray
	}
	upscaled := image.NewGray(image.Rect(0, 0, width*scale, height*scale))
	xdraw.CatmullRom.Scale(upscaled, upscaled.Bounds(), gray, gray.Bounds(), xdraw.Over, nil)
	return upscaled
}

func percentileBounds(histogram [256]int, pixels int) (int, int) {
	cutoff := pixels / 100
	seen := 0
	low := 0
	for low < 255 {
		seen += histogram[low]
		if seen > cutoff {
			break
		}
		low++
	}
	seen = 0
	high := 255
	for high > 0 {
		seen += histogram[high]
		if seen > cutoff {
			break
		}
		high--
	}
	return low, high
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
