package ocr

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
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
	paths   []string
}

func (r *sequenceRunner) Recognize(_ context.Context, path string) (string, error) {
	r.paths = append(r.paths, path)
	output := r.outputs[r.calls]
	r.calls++
	return output, nil
}

func TestEnhancedOCRChoosesBestPassAndDeletesVariants(t *testing.T) {
	path := testImage(t, 40, 40)
	runner := &sequenceRunner{outputs: []string{
		"n0ise !!!",
		"clean invoice total 125.00",
		"side",
		"upside down",
		"other side",
	}}
	p := NewParserWithRunner(runner)
	result, err := p.ParseWithContext(context.Background(), parser.ParseRequest{
		File:      path,
		Selection: parser.Selection{Enhance: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "clean invoice total 125.00" {
		t.Fatalf("Text = %q", result.Text)
	}
	if runner.calls != 5 {
		t.Fatalf("OCR calls = %d, want 5", runner.calls)
	}
	for _, temporary := range runner.paths {
		if _, err := os.Stat(temporary); !os.IsNotExist(err) {
			t.Fatalf("OCR temporary file remains: %s", temporary)
		}
	}
}

func TestOtsuBinarizeNormalizesDarkBackground(t *testing.T) {
	image := image.NewGray(image.Rect(0, 0, 4, 1))
	image.Pix = []byte{10, 10, 240, 10}
	result := otsuBinarize(image)
	if result.Pix[2] != 0 {
		t.Fatalf("light text pixel = %d, want black", result.Pix[2])
	}
	if result.Pix[0] != 255 {
		t.Fatalf("dark background pixel = %d, want white", result.Pix[0])
	}
}

func TestTextQualityScoreRewardsReadableWords(t *testing.T) {
	readable := textQualityScore("Invoice total 125.00 due tomorrow")
	noise := textQualityScore("! | / ? �\x00")
	if readable <= noise {
		t.Fatalf("readable score %d <= noise score %d", readable, noise)
	}
}

func TestEstimateSkewFindsAndCorrectsTextLines(t *testing.T) {
	page := image.NewGray(image.Rect(0, 0, 240, 140))
	for index := range page.Pix {
		page.Pix[index] = 255
	}
	for _, y := range []int{30, 55, 80, 105} {
		for x := 25; x < 215; x++ {
			page.SetGray(x, y, color.Gray{Y: 0})
			page.SetGray(x, y+1, color.Gray{Y: 0})
		}
	}
	skewed := rotateSmall(page, 2)
	angle := estimateSkew(skewed)
	if math.Abs(math.Abs(angle)-2) > 0.6 {
		t.Fatalf("estimated angle = %.1f, want magnitude near 2", angle)
	}
	corrected := rotateSmall(skewed, angle)
	if residual := math.Abs(estimateSkew(corrected)); residual > 0.6 {
		t.Fatalf("residual angle = %.1f after correction", residual)
	}
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

func TestDecodeLayoutUsesOCRWordCoordinates(t *testing.T) {
	data := []byte(`{
		"paragraphs": [{"lines": [
			{"text":"Name Value","vertices":[[0,0],[100,0],[100,10],[0,10]],"words":[
				{"text":"Name","vertices":[[0,0],[20,0],[20,10],[0,10]]},
				{"text":"Value","vertices":[[80,0],[100,0],[100,10],[80,10]]}
			]}
		]}]
	}`)
	tokens, err := decodeLayout(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 2 || tokens[1].Text != "Value" || tokens[1].X != 80 || tokens[1].Width != 20 {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}

func TestDecodeLayoutRejectsEmptyOCR(t *testing.T) {
	if _, err := decodeLayout([]byte(`{"paragraphs":[]}`)); err == nil {
		t.Fatal("expected empty OCR error")
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
