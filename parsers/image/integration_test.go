package image

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/upendra7470/clip/internal/parser"
)

// These tests exercise the parser against a real Tesseract installation.
// They skip automatically when tesseract is not on PATH so that the normal
// test suite stays deterministic in environments without OCR tooling.
//
// To run them locally:
//
//	brew install tesseract          # macOS
//	sudo apt install tesseract-ocr  # Debian/Ubuntu
//	go test ./parsers/image/ -v

func tesseractAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath(TesseractBinary); err != nil {
		t.Skipf("%s is not installed; skipping real-OCR integration test", TesseractBinary)
	}
}

// renderTextImage draws text onto a white background with a boldened,
// upscaled basic font - a style Tesseract recognizes reliably.
func renderTextImage(t *testing.T, text string, scale int, encode func(image.Image) []byte) []byte {
	t.Helper()

	face := basicfont.Face7x13
	w := face.Width*len(text) + 20
	h := face.Height + 20
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	for _, off := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		d := &font.Drawer{
			Dst:  img,
			Src:  image.Black,
			Face: face,
			Dot:  fixed.P(10+off[0], 10+face.Ascent+off[1]),
		}
		d.DrawString(text)
	}

	scaled := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					scaled.Set(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}

	return encode(scaled)
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func encodeJPEG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return path
}

func TestIntegrationPNGWithText(t *testing.T) {
	tesseractAvailable(t)

	path := writeTemp(t, "car.png", renderTextImage(t, "HYUNDAI TS09AB1234", 6, encodePNG))
	p := NewParser()

	result, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err != nil {
		t.Fatalf("ParseWithContext() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "HYUNDAI") {
		t.Errorf("OCR result missing 'HYUNDAI': %q", result.Text)
	}
}

func TestIntegrationJPEGWithText(t *testing.T) {
	tesseractAvailable(t)

	path := writeTemp(t, "sale.jpg", renderTextImage(t, "SALE PRICE 850000", 6, encodeJPEG))
	p := NewParser()

	result, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err != nil {
		t.Fatalf("ParseWithContext() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "SALE") || !strings.Contains(result.Text, "PRICE") {
		t.Errorf("OCR result missing expected words: %q", result.Text)
	}
}

func TestIntegrationImageWithoutText(t *testing.T) {
	tesseractAvailable(t)

	// A blank white photo of an object contains no readable text;
	// OCR must report that cleanly rather than failing unexpectedly.
	path := writeTemp(t, "blank.png", encodePNG(image.NewRGBA(image.Rect(0, 0, 200, 200))))
	p := NewParser()

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected clean 'no text detected' error, got nil")
	}
	if !strings.Contains(err.Error(), "no text detected in image") {
		t.Errorf("error = %q, want to contain 'no text detected in image'", err.Error())
	}
}
