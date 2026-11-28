package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/upendra7470/clip/internal/security"
)

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
