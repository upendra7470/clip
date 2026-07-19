package pdf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/upendra7470/clip/internal/parser"
)

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "PDF"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.pdf",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "failed to open PDF") {
		t.Errorf("Parse() error = %q, want to contain 'failed to open PDF'", err.Error())
	}
}

func TestParseInvalidPDF(t *testing.T) {
	// Create a file with invalid PDF content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.pdf")

	// Write invalid PDF content
	invalidPDF := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header, not PDF
	err := os.WriteFile(filePath, invalidPDF, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid PDF test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid PDF, got nil")
	}

	if !containsError(err.Error(), "failed to parse PDF") {
		t.Errorf("Parse() error = %q, want to contain 'failed to parse PDF'", err.Error())
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.pdf",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "failed to open PDF") {
		t.Errorf("Error message = %q, want to contain 'failed to open PDF'", err.Error())
	}
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

// containsSubstring checks if a string contains a substring.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
