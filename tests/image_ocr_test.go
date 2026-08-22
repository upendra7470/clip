package tests

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// tesseractInstalled reports whether a real OCR engine is available.
// The image-OCR CLI tests below require it and skip otherwise, keeping
// CI deterministic on machines without Tesseract.
func tesseractInstalled() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

// renderTextPNG renders text as a boldened, upscaled PNG that Tesseract
// recognizes reliably. Returns the encoded PNG bytes.
func renderTextPNG(t *testing.T, text string) []byte {
	t.Helper()

	face := basicfont.Face7x13
	w := face.Width*len(text) + 20
	h := face.Height + 20
	const scale = 6

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

	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		t.Fatalf("failed to encode PNG fixture: %v", err)
	}
	return buf.Bytes()
}

func writeFixture(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return path
}

// TestCLIImageOCR runs the full pipeline (detect -> parser -> OCR ->
// clipboard) on an image containing text.
func TestCLIImageOCR(t *testing.T) {
	if !tesseractInstalled() {
		t.Skip("tesseract is not installed; skipping image OCR CLI integration test")
	}

	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	testFile := writeFixture(t, "car.png", renderTextPNG(t, "HYUNDAI TS09AB1234"))

	cmd := exec.Command(clipPath, testFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip failed on image file: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(string(output), "Copied to clipboard") {
		t.Errorf("expected success output, got: %s", output)
	}
}

// TestCLIImageWithoutText verifies the no-text case produces a clean,
// user-friendly error instead of unexpected behavior.
func TestCLIImageWithoutText(t *testing.T) {
	if !tesseractInstalled() {
		t.Skip("tesseract is not installed; skipping image OCR CLI integration test")
	}

	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	blank := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw.Draw(blank, blank.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, blank); err != nil {
		t.Fatalf("failed to encode fixture: %v", err)
	}
	testFile := writeFixture(t, "blank.png", buf.Bytes())

	cmd := exec.Command(clipPath, testFile)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for image without text")
	}
	if !strings.Contains(strings.ToLower(string(output)), "no text detected") {
		t.Errorf("expected clean no-text message, got: %s", output)
	}
}

// TestCLIUnsupportedImageExtension checks extension-level rejection stays intact.
func TestCLIUnsupportedImageExtension(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// .bmp is intentionally not supported by detect.
	testFile := writeFixture(t, "photo.bmp", []byte("BM-fake"))

	cmd := exec.Command(clipPath, testFile)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for unsupported extension, got success")
	}
	if !strings.Contains(strings.ToLower(string(output)), "unsupported") {
		t.Errorf("expected unsupported-extension error, got: %s", output)
	}
}
