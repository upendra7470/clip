package image

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/registry"
	"github.com/upendra7470/clip/internal/security"
)

// fakeEngine is a deterministic OCR engine for tests.
type fakeEngine struct {
	text string
	err  error
}

func (f *fakeEngine) Name() string { return "fake" }

func (f *fakeEngine) ExtractText(ctx context.Context, data []byte) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

// mustPNG encodes w x h solid-color image as PNG bytes.
func mustPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	return buf.Bytes()
}

// writeTestFile writes data to a temp file and returns its path.
func writeTestFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return path
}

func TestFileType(t *testing.T) {
	p := &Parser{}
	if got := p.FileType(); got != filetype.FileTypeImage {
		t.Errorf("FileType() = %q, want %q", got, filetype.FileTypeImage)
	}
}

func TestParseSuccess(t *testing.T) {
	path := writeTestFile(t, "img.png", mustPNG(t, 10, 10))
	p := NewParserWithEngine(&fakeEngine{text: "HYUNDAI\nTS09AB1234\nSALE ₹8,50,000\n"})

	result, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err != nil {
		t.Fatalf("ParseWithContext() unexpected error: %v", err)
	}
	for _, want := range []string{"HYUNDAI", "TS09AB1234", "₹8,50,000"} {
		if !strings.Contains(result.Text, want) {
			t.Errorf("ParseWithContext() result missing %q: %q", want, result.Text)
		}
	}
}

func TestParseNoTextDetected(t *testing.T) {
	path := writeTestFile(t, "blank.png", mustPNG(t, 10, 10))
	p := NewParserWithEngine(&fakeEngine{text: "\f\n   \n"})

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected error for image without text, got nil")
	}
	if !containsError(err.Error(), "no text detected in image") {
		t.Errorf("error = %q, want to contain 'no text detected in image'", err.Error())
	}
}

func TestParseEngineFailure(t *testing.T) {
	path := writeTestFile(t, "img.png", mustPNG(t, 10, 10))
	engineErr := errors.New("engine crashed")
	p := NewParserWithEngine(&fakeEngine{err: engineErr})

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected engine failure error, got nil")
	}
	if !containsError(err.Error(), "engine crashed") {
		t.Errorf("error = %q, want it to preserve the root cause", err.Error())
	}
	if !errors.Is(err, engineErr) {
		t.Error("error should unwrap to the underlying engine error")
	}
}

func TestParseMissingFile(t *testing.T) {
	p := NewParserWithEngine(&fakeEngine{})
	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: "nonexistent.png"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !containsError(err.Error(), "file nonexistent.png does not exist") {
		t.Errorf("error = %q, want to contain 'file nonexistent.png does not exist'", err.Error())
	}
}

func TestParseEmptyFile(t *testing.T) {
	path := writeTestFile(t, "empty.png", []byte{})
	p := NewParserWithEngine(&fakeEngine{})

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
	if !containsError(err.Error(), "empty image file") {
		t.Errorf("error = %q, want to contain 'empty image file'", err.Error())
	}
}

func TestParseCorruptImage(t *testing.T) {
	path := writeTestFile(t, "corrupt.png", []byte("this is not an image at all"))
	p := NewParserWithEngine(&fakeEngine{})

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected error for corrupt image, got nil")
	}
	if !containsError(err.Error(), "unsupported or corrupt image") {
		t.Errorf("error = %q, want to contain 'unsupported or corrupt image'", err.Error())
	}
}

func TestParseUnsupportedFormat(t *testing.T) {
	// GIF magic; the gif decoder is intentionally not registered.
	path := writeTestFile(t, "anim.gif", []byte("GIF89a\x01\x00\x01\x00\x00\xff\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x00;"))
	p := NewParserWithEngine(&fakeEngine{})

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected error for unsupported image format, got nil")
	}
	if !containsError(err.Error(), "unsupported or corrupt image") {
		t.Errorf("error = %q, want to contain 'unsupported or corrupt image'", err.Error())
	}
}

func TestDimensionLimitExceeded(t *testing.T) {
	// Real PNG whose header claims a width beyond MaxImageDimension;
	// DecodeConfig only reads the header so this stays cheap.
	huge := mustPNG(t, security.MaxImageDimension+1, 1)
	path := writeTestFile(t, "huge.png", huge)
	p := NewParserWithEngine(&fakeEngine{})

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected error for oversized dimensions, got nil")
	}
	if !containsError(err.Error(), "exceed maximum allowed dimension") {
		t.Errorf("error = %q, want to contain 'exceed maximum allowed dimension'", err.Error())
	}
}

func TestPixelLimitExceeded(t *testing.T) {
	err := checkImageDimensions(security.MaxImageDimension, security.MaxImageDimension)
	if err == nil {
		t.Fatal("expected pixel-count limit error, got nil")
	}
	if !containsError(err.Error(), "exceeding maximum allowed") {
		t.Errorf("error = %q, want to contain 'exceeding maximum allowed'", err.Error())
	}
}

func TestInvalidDimensions(t *testing.T) {
	if err := checkImageDimensions(0, 10); err == nil {
		t.Error("expected error for zero width")
	}
	if err := checkImageDimensions(10, -1); err == nil {
		t.Error("expected error for negative height")
	}
	if err := checkImageDimensions(100, 100); err != nil {
		t.Errorf("unexpected error for normal dimensions: %v", err)
	}
}

func TestOversizedFileRejected(t *testing.T) {
	data := mustPNG(t, 10, 10)
	path := writeTestFile(t, "big.png", data)
	p := &Parser{
		engine:  &fakeEngine{},
		MaxSize: 8, // smaller than any real PNG
	}

	_, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected size limit error, got nil")
	}
	if !containsError(err.Error(), "exceeds maximum allowed size") {
		t.Errorf("error = %q, want to contain 'exceeds maximum allowed size'", err.Error())
	}
}

func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestRegistration(t *testing.T) {
	reg := registry.New()
	p := NewParser()
	if err := reg.Register(p.FileType(), p); err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}
	got, err := reg.Lookup(filetype.FileTypeImage)
	if err != nil {
		t.Fatalf("Lookup() unexpected error: %v", err)
	}
	if got.FileType() != filetype.FileTypeImage {
		t.Errorf("Lookup().FileType() = %q, want %q", got.FileType(), filetype.FileTypeImage)
	}
}

func TestTesseractMissingBinary(t *testing.T) {
	t.Setenv("PATH", "")
	e := &TesseractEngine{}

	if e.IsAvailable() {
		t.Skip("tesseract still resolvable with empty PATH")
	}

	_, err := e.ExtractText(context.Background(), mustPNG(t, 4, 4))
	if err == nil {
		t.Fatal("expected error when OCR binary is missing, got nil")
	}
	if !containsError(err.Error(), "not installed") {
		t.Errorf("error = %q, want to mention the engine is not installed", err.Error())
	}
	for _, hint := range []string{"brew install tesseract", "apt install", "winget"} {
		if !containsError(err.Error(), hint) {
			t.Errorf("error = %q, want installation hint containing %q", err.Error(), hint)
		}
	}
}

func TestTesseractEngineName(t *testing.T) {
	e := &TesseractEngine{}
	if e.Name() != TesseractBinary {
		t.Errorf("Name() = %q, want %q", e.Name(), TesseractBinary)
	}
}
