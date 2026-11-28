package searchablepdf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/upendra7470/clip/internal/parser"
)

type fakeExtractor struct{ text string }

func (f fakeExtractor) ExtractText(context.Context, string, *parser.Range, parser.Selection) (parser.ParseResult, error) {
	return parser.ParseResult{Text: f.text}, nil
}

func TestWatermarkTextSanitizesAndEscapesPlaceholders(t *testing.T) {
	got := watermarkText("  50%\ncomplete\x00  ")
	if got != "50%% complete" {
		t.Fatalf("watermarkText() = %q", got)
	}
}

func TestWatermarkTextLimitsPageSize(t *testing.T) {
	got := watermarkText(strings.Repeat("x", maxPageTextRunes+100))
	if len([]rune(got)) != maxPageTextRunes {
		t.Fatalf("watermarkText() length = %d", len([]rune(got)))
	}
}

func TestCreateAddsSearchableTextAndPreservesValidPDF(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "scan.pdf")
	output := filepath.Join(directory, "scan.searchable.pdf")
	if err := os.WriteFile(input, minimalBlankPDF(), 0600); err != nil {
		t.Fatal(err)
	}

	pages, err := Create(context.Background(), fakeExtractor{text: "hello searchable world"}, input, output, parser.Selection{})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 1 {
		t.Fatalf("Create() pages = %d, want 1", pages)
	}
	if err := api.ValidateFile(context.Background(), output, model.NewStatelessConfiguration(), nil); err != nil {
		t.Fatalf("generated PDF is invalid: %v", err)
	}
	inputData, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	outputData, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(outputData) <= len(inputData) || !bytes.Contains(outputData, []byte("/Artifact")) {
		t.Fatal("generated PDF did not contain an OCR text-layer object")
	}
}

func TestCreateRefusesToOverwrite(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "scan.pdf")
	if err := os.WriteFile(input, minimalBlankPDF(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(context.Background(), fakeExtractor{text: "text"}, input, input, parser.Selection{}); err == nil {
		t.Fatal("Create() overwrote its input")
	}
}

func minimalBlankPDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>",
		"<< /Length 0 >>\nstream\n\nendstream",
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return pdf.Bytes()
}
