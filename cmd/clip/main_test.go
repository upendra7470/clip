package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/upendra7470/clip/internal/application"
	"github.com/upendra7470/clip/internal/clipboard"
	clipoutput "github.com/upendra7470/clip/internal/output"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/registry"
	"github.com/upendra7470/clip/internal/screencapture"
	"github.com/upendra7470/clip/internal/security"
	"github.com/upendra7470/clip/parsers/ocr"
)

type screenOCRRunner struct{ text string }

func (runner screenOCRRunner) Recognize(context.Context, string) (string, error) {
	return runner.text, nil
}

func TestListReturnsFailureWhenInspectionFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--list", path}, &stdout, &stderr); code == 0 {
		t.Fatalf("Run() returned success, stderr = %q", stderr.String())
	}
}

func TestListRejectsOversizedFileBeforeReading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.html")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(security.MaxInputFileSize + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--list", path}, &stdout, &stderr); code == 0 {
		t.Fatalf("Run() accepted oversized inspection input")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("exceeds maximum allowed")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestStructuredJSONOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("hello v2"), 0600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--format", "json", "--stdout", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	var payload struct {
		Version   string `json:"version"`
		Documents []struct {
			Text string `json:"text"`
		} `json:"documents"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Version != "2.0" || len(payload.Documents) != 1 || payload.Documents[0].Text != "hello v2" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestRichClipboardPublishesHTMLAndPlainText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(path, []byte("# Hello\n\nRich **copy**"), 0600); err != nil {
		t.Fatal(err)
	}
	testClipboard := clipboard.NewTestClipboard()
	clipboard.SetTestClipboard(testClipboard)
	t.Cleanup(func() { clipboard.SetTestClipboard(nil) })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--rich", "--format", "markdown", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(testClipboard.GetCopiedText(), "Rich copy") {
		t.Fatalf("plain clipboard = %q", testClipboard.GetCopiedText())
	}
	if !strings.Contains(testClipboard.GetCopiedHTML(), "<p>Rich copy</p>") {
		t.Fatalf("HTML clipboard = %q", testClipboard.GetCopiedHTML())
	}
}

func TestRichClipboardRejectsFileDestination(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--rich", "--output", "out.html", "notes.md"}, &stdout, &stderr); code == 0 {
		t.Fatal("Run() accepted --rich with --output")
	}
	if !strings.Contains(stderr.String(), "--rich writes to the clipboard") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestBatchFilesRespectsRecursiveFlag(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "top.txt"), []byte("top"), 0600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "deep.md"), []byte("deep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "not-a-document.desktop"), []byte("[Desktop Entry]"), 0600); err != nil {
		t.Fatal(err)
	}

	flat, err := batchFiles(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(flat) != 1 {
		t.Fatalf("flat batch = %#v", flat)
	}
	recursive, err := batchFiles(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(recursive) != 2 {
		t.Fatalf("recursive batch = %#v", recursive)
	}
}

func TestCompletionScript(t *testing.T) {
	script, err := completionScript("powershell")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(script), []byte("--smart")) {
		t.Fatalf("completion missing v2 options: %s", script)
	}
	if !bytes.Contains([]byte(script), []byte("--screen")) {
		t.Fatalf("completion missing screen OCR: %s", script)
	}
	if !bytes.Contains([]byte(script), []byte("--rich")) || !bytes.Contains([]byte(script), []byte("--make-searchable")) {
		t.Fatalf("completion missing v2.3 options: %s", script)
	}
	if !bytes.Contains([]byte(script), []byte("--watch")) {
		t.Fatalf("completion missing v2.4 options: %s", script)
	}
}

func TestExplicitPDFRejectsNonPDF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--pdf", path}, &stdout, &stderr); code == 0 {
		t.Fatalf("Run() accepted text as PDF")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("--pdf requires a PDF")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRejectsConflictingExplicitModes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--image", "--pdf", "anything.pdf"}, &stdout, &stderr); code == 0 {
		t.Fatal("Run() accepted conflicting modes")
	}
}

func TestRejectsInvalidLayout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--layout", "visual", "anything.pdf"}, &stdout, &stderr); code == 0 {
		t.Fatal("Run() accepted invalid layout")
	}
}

func TestScreenRejectsFileModes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--screen", "--pdf"}, &stdout, &stderr); code == 0 {
		t.Fatal("Run() accepted --screen with --pdf")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("--screen cannot be combined")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestScreenRejectsFilename(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--screen", "document.png"}, &stdout, &stderr); code == 0 {
		t.Fatal("Run() accepted a filename with --screen")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("does not accept a filename")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestWatchRejectsFilename(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--watch", "document.png"}, &stdout, &stderr); code == 0 {
		t.Fatal("Run() accepted a filename with --watch")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("--watch does not accept a filename")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestWatchRejectsFileModes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--watch", "--pdf"}, &stdout, &stderr); code == 0 {
		t.Fatal("Run() accepted --watch with --pdf")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("--watch cannot be combined")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunScreenCaptureExtractsAndDeletesTemporaryImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "screen.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	img.Set(0, 0, color.White)
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	previousCapture := captureScreen
	captureScreen = func(context.Context) (screencapture.Result, error) {
		return screencapture.Result{Path: path, Width: 20, Height: 10}, nil
	}
	t.Cleanup(func() { captureScreen = previousCapture })

	reg := registry.New()
	imageParser := ocr.NewParserWithRunner(screenOCRRunner{text: "hello from screen"})
	if err := reg.Register(imageParser.FileType(), imageParser); err != nil {
		t.Fatal(err)
	}
	app := application.New(reg, &realClipboard{})
	var stdout, stderr bytes.Buffer
	code := runScreenCapture(context.Background(), app, &realClipboard{}, clipoutput.Text, 0, "", true, false,
		parser.Selection{Layout: "plain"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runScreenCapture() = %d, stderr = %q", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("hello from screen")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Captured 20 × 10")) || !bytes.Contains(stderr.Bytes(), []byte("Recognized 3 words")) {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary screenshot still exists: %v", err)
	}
}
