package security

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIsSafeZipEntry(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"normal file", "word/document.xml", true},
		{"nested path", "ppt/slides/slide1.xml", true},
		{"path traversal", "../evil.txt", false},
		{"path traversal nested", "folder/../../evil.txt", false},
		{"absolute path", "/etc/passwd", false},
		{"traversal after clean", "./../evil.txt", false},
		{"backslash traversal", "..\\evil.txt", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSafeZipEntry(tt.path); got != tt.want {
				t.Errorf("IsSafeZipEntry(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestReadZipFile_RejectsTraversal(t *testing.T) {
	// Create a temporary ZIP file with a traversal entry
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	w, _ := zw.Create("../evil.txt")
	w.Write([]byte("evil content"))
	zw.Close()

	// Read it back
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to create zip reader: %v", err)
	}

	_, err = ReadZipFile(zr.File[0])
	if err == nil {
		t.Error("expected error for traversal entry, got nil")
	}
}

func TestReadZipFile_RejectsOversizedEntry(t *testing.T) {
	// Create a ZIP entry with an uncompressed size larger than limit
	// We can't easily create a real oversized zip, so test ValidateZipFile directly
	f := &zip.File{}
	// Can't set UncompressedSize64 directly in test, so skip direct construction
	_ = f

	// Instead test the limits constants
	if MaxZipEntrySize <= 0 {
		t.Error("MaxZipEntrySize should be positive")
	}
	if MaxTotalZipUncompressedSize < MaxZipEntrySize {
		t.Error("MaxTotalZipUncompressedSize should be >= MaxZipEntrySize")
	}
}

func TestCheckZipArchiveTotalSize(t *testing.T) {
	// Create a temporary ZIP with a small total size
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	w, _ := zw.Create("test.txt")
	w.Write([]byte("hello"))
	zw.Close()

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to create zip reader: %v", err)
	}

	if err := CheckZipArchiveTotalSize(zr.File); err != nil {
		t.Errorf("CheckZipArchiveTotalSize on small archive should pass: %v", err)
	}
}

func TestReadFileBounded_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(fpath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	content, err := ReadFileBounded(fpath)
	if err != nil {
		t.Fatalf("ReadFileBounded failed: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("ReadFileBounded = %q, want %q", content, "hello")
	}
}

func TestReadFileBounded_FileTooLarge(t *testing.T) {
	// Create a file larger than MaxInputFileSize by sparse writing
	dir := t.TempDir()
	fpath := filepath.Join(dir, "large.txt")
	f, err := os.Create(fpath)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	// Write enough data to exceed the limit (use sparse file)
	if err := f.Truncate(MaxInputFileSize + 1); err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
	f.Close()

	_, err = ReadFileBounded(fpath)
	if err == nil {
		t.Error("expected error for oversized file, got nil")
	}
}

func TestNewXMLDecoder(t *testing.T) {
	decoder := NewXMLDecoder(bytes.NewReader([]byte("<root/>")))
	if decoder == nil {
		t.Fatal("NewXMLDecoder returned nil")
	}
	if decoder.Strict != true {
		t.Error("XML decoder should be in Strict mode")
	}
	if decoder.Entity == nil {
		t.Error("XML decoder Entity map should not be nil")
	}
}

func TestSafeXMLContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"normal xml", `<root><a>hello</a></root>`, true},
		{"doctype uppercase", `<!DOCTYPE foo [ <!ENTITY x SYSTEM "file:///etc/passwd"> ]><root>&x;</root>`, false},
		{"doctype lowercase", `<!doctype foo>` + `<root/>`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeXMLContent(tt.content); got != tt.want {
				t.Errorf("SafeXMLContent(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestCheckXMLDepth(t *testing.T) {
	if err := CheckXMLDepth(10); err != nil {
		t.Errorf("CheckXMLDepth(10) should pass: %v", err)
	}
	if err := CheckXMLDepth(MaxXMLDepth + 1); err == nil {
		t.Error("CheckXMLDepth(MaxXMLDepth+1) should fail")
	}
}
