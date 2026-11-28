package ocr

import (
	"bytes"
	"context"
	"encoding/json"
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
	clipLayout "github.com/upendra7470/clip/internal/layout"
	"github.com/upendra7470/clip/internal/ocrbundle"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
)

type Runner interface {
	Recognize(ctx context.Context, file string) (string, error)
}

type layoutRunner interface {
	RecognizeLayout(ctx context.Context, file string, mode clipLayout.Mode) (string, error)
}

type bundledRunner struct{}

func (bundledRunner) Recognize(ctx context.Context, file string) (string, error) {
	return runBundled(ctx, file, false)
}

func (bundledRunner) RecognizeLayout(ctx context.Context, file string, mode clipLayout.Mode) (string, error) {
	if mode == clipLayout.Plain {
		return runBundled(ctx, file, false)
	}
	output, err := runBundled(ctx, file, true)
	if err != nil {
		return "", err
	}
	tokens, err := decodeLayout([]byte(output))
	if err != nil {
		return "", fmt.Errorf("decode built-in OCR layout: %w", err)
	}
	return clipLayout.Render(tokens, mode), nil
}

func runBundled(ctx context.Context, file string, structured bool) (string, error) {
	paths, err := ocrbundle.Resolve()
	if err != nil {
		return "", err
	}

	args := []string{
		"--detect-model", paths.DetectionModel,
		"--rec-model", paths.RecognitionModel,
	}
	if structured {
		args = append(args, "--json")
	}
	args = append(args, file)
	// #nosec G204 -- the executable is the embedded, integrity-checked OCR engine; arguments are passed without a shell.
	cmd := exec.CommandContext(ctx, paths.Engine, args...)
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

type ocrJSON struct {
	Paragraphs []struct {
		Lines []struct {
			Text     string       `json:"text"`
			Vertices [][2]float64 `json:"vertices"`
			Words    []struct {
				Text     string       `json:"text"`
				Vertices [][2]float64 `json:"vertices"`
			} `json:"words"`
		} `json:"lines"`
	} `json:"paragraphs"`
}

func decodeLayout(data []byte) ([]clipLayout.Token, error) {
	var document ocrJSON
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	var tokens []clipLayout.Token
	for _, paragraph := range document.Paragraphs {
		for _, line := range paragraph.Lines {
			if len(line.Words) == 0 {
				if token, ok := positionedToken(line.Text, line.Vertices); ok {
					tokens = append(tokens, token)
				}
				continue
			}
			for _, word := range line.Words {
				if token, ok := positionedToken(word.Text, word.Vertices); ok {
					tokens = append(tokens, token)
				}
			}
		}
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("OCR found no positioned text")
	}
	return tokens, nil
}

func positionedToken(text string, vertices [][2]float64) (clipLayout.Token, bool) {
	if strings.TrimSpace(text) == "" || len(vertices) == 0 {
		return clipLayout.Token{}, false
	}
	minX, maxX := vertices[0][0], vertices[0][0]
	minY, maxY := vertices[0][1], vertices[0][1]
	for _, vertex := range vertices[1:] {
		if vertex[0] < minX {
			minX = vertex[0]
		}
		if vertex[0] > maxX {
			maxX = vertex[0]
		}
		if vertex[1] < minY {
			minY = vertex[1]
		}
		if vertex[1] > maxY {
			maxY = vertex[1]
		}
	}
	return clipLayout.Token{Text: text, X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}, true
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
	return RecognizeSmartLayout(ctx, file, smart, "auto")
}

// RecognizeSmartLayout performs OCR and reconstructs text using word geometry.
func RecognizeSmartLayout(ctx context.Context, file string, smart bool, layoutValue string) (string, error) {
	return RecognizeEnhancedLayout(ctx, file, smart, false, layoutValue)
}

// RecognizeEnhancedLayout performs optional multi-pass preprocessing and OCR
// result selection in addition to smart layout reconstruction.
func RecognizeEnhancedLayout(ctx context.Context, file string, smart, enhance bool, layoutValue string) (string, error) {
	if enhance {
		variants, cleanup, err := prepareEnhancedVariants(file)
		if err != nil {
			return "", err
		}
		defer cleanup()
		ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
		defer cancel()
		mode, err := clipLayout.ParseMode(layoutValue)
		if err != nil {
			return "", err
		}
		return recognizeEnhanced(ocrCtx, bundledRunner{}, variants, mode)
	}
	prepared, cleanup, err := prepareImage(file, smart)
	if err != nil {
		return "", err
	}
	defer cleanup()
	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()
	mode, err := clipLayout.ParseMode(layoutValue)
	if err != nil {
		return "", err
	}
	if smart {
		return recognizeBestOrientationLayout(ocrCtx, bundledRunner{}, prepared, mode)
	}
	return recognizeWithLayout(ocrCtx, bundledRunner{}, prepared, mode)
}

func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	ocrCtx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()
	mode, err := clipLayout.ParseMode(req.Selection.Layout)
	if err != nil {
		return parser.ParseResult{}, err
	}

	var text string
	if req.Selection.Enhance {
		variants, cleanup, prepareErr := prepareEnhancedVariants(req.File)
		if prepareErr != nil {
			return parser.ParseResult{}, prepareErr
		}
		defer cleanup()
		text, err = recognizeEnhanced(ocrCtx, p.runner, variants, mode)
	} else {
		prepared, cleanup, prepareErr := prepareImage(req.File, req.Selection.Smart)
		if prepareErr != nil {
			return parser.ParseResult{}, prepareErr
		}
		defer cleanup()
		if req.Selection.Smart {
			text, err = recognizeBestOrientationLayout(ocrCtx, p.runner, prepared, mode)
		} else {
			text, err = recognizeWithLayout(ocrCtx, p.runner, prepared, mode)
		}
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

func recognizeWithLayout(ctx context.Context, runner Runner, path string, mode clipLayout.Mode) (string, error) {
	if structured, ok := runner.(layoutRunner); ok {
		return structured.RecognizeLayout(ctx, path, mode)
	}
	return runner.Recognize(ctx, path)
}

func recognizeBestOrientationLayout(ctx context.Context, runner Runner, path string, mode clipLayout.Mode) (string, error) {
	bestText, err := recognizeWithLayout(ctx, runner, path, mode)
	if err != nil {
		return "", err
	}
	bestScore := textScore(bestText)
	for _, degrees := range []int{90, 180, 270} {
		rotated, cleanup, err := rotatedImage(path, degrees)
		if err != nil {
			return "", err
		}
		candidate, recognizeErr := recognizeWithLayout(ctx, runner, rotated, mode)
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
	// #nosec G304 -- Clip intentionally opens the local image path selected by the user.
	file, err := os.Open(path)
	if err != nil {
		return "", func() {}, fmt.Errorf("open image for orientation detection: %w", err)
	}
	source, _, err := image.Decode(file)
	_ = file.Close()
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
	cleanup := func() { _ = os.Remove(path) }
	if err := png.Encode(temp, target); err != nil {
		_ = temp.Close()
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
	// #nosec G304 -- Clip intentionally opens the local image path selected by the user.
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

	// #nosec G304 -- Clip intentionally opens the local image path selected by the user.
	file, err := os.Open(path)
	if err != nil {
		return "", func() {}, fmt.Errorf("open image for conversion: %w", err)
	}
	imageData, _, err := image.Decode(file)
	_ = file.Close()
	if err != nil {
		return "", func() {}, fmt.Errorf("decode image for OCR: %w", err)
	}
	if smart {
		const maxSmartPixels = 25_000_000
		bounds := imageData.Bounds()
		width, height := bounds.Dx(), bounds.Dy()
		if width <= 0 || height <= 0 || width > maxSmartPixels/height {
			return "", func() {}, fmt.Errorf("smart OCR image exceeds the %d-pixel preprocessing limit", maxSmartPixels)
		}
		imageData = enhanceForOCR(imageData)
	}
	temp, err := os.CreateTemp("", "clip-ocr-*.png")
	if err != nil {
		return "", func() {}, fmt.Errorf("prepare image for OCR: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() { _ = os.Remove(tempPath) }
	if err := png.Encode(temp, imageData); err != nil {
		_ = temp.Close()
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
				scaled := (int(value) - low) * 255 / (high - low)
				// #nosec G115 -- the surrounding branches prove scaled is in [0, 255].
				gray.Pix[index] = uint8(scaled)
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
