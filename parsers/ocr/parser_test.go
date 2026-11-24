package ocr

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/upendra7470/clip/internal/parser"
)

type fakeRunner struct {
	text string
	err  error
}

type sequenceRunner struct {
	outputs []string
	calls   int
}

func (r *sequenceRunner) Recognize(context.Context, string) (string, error) {
	output := r.outputs[r.calls]
	r.calls++
	return output, nil
}

func (f fakeRunner) Recognize(context.Context, string) (string, error) {
	return f.text, f.err
}

func TestSmartOCRChoosesBestOrientation(t *testing.T) {
	path := testImage(t, 20, 10)
	runner := &sequenceRunner{outputs: []string{"x", "short", "correct orientation text", "no"}}
	p := NewParserWithRunner(runner)
	result, err := p.ParseWithContext(context.Background(), parser.ParseRequest{
		File:      path,
		Selection: parser.Selection{Smart: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "correct orientation text" {
		t.Fatalf("Text = %q", result.Text)
	}
	if runner.calls != 4 {
		t.Fatalf("OCR calls = %d, want 4", runner.calls)
	}
}

func TestOCRCopiesRecognizedText(t *testing.T) {
	path := testImage(t, 20, 10)
	p := NewParserWithRunner(fakeRunner{text: "  Hello from OCR\n"})

	result, err := p.ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err != nil {
		t.Fatalf("ParseWithContext() error = %v", err)
	}
	if result.Text != "Hello from OCR" {
		t.Fatalf("Text = %q", result.Text)
	}
}

func TestOCRRejectsInvalidImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.png")
	if err := os.WriteFile(path, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := NewParserWithRunner(fakeRunner{text: "ignored"}).
		ParseWithContext(context.Background(), parser.ParseRequest{File: path})
	if err == nil {
		t.Fatal("expected invalid image error")
	}
}

func testImage(t *testing.T, width, height int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scan.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.Black)
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
