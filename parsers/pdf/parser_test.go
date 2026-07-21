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

	if !containsError(err.Error(), "Could not open PDF file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open PDF file'", err.Error())
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
	if !containsError(err.Error(), "Could not open PDF file") {
		t.Errorf("Error message = %q, want to contain 'Could not open PDF file'", err.Error())
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

// TestParseRange tests the ParseRange method for extracting specific page ranges
func TestParseRange(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.pdf", // Use nonexistent file to test validation logic
	}

	// Test invalid page range - reversed
	_, err := p.ParseRange(context.Background(), req, 10, 5)
	if err == nil {
		t.Error("ParseRange() expected error for reversed range, got nil")
	}
	// The validation should happen before file parsing, so we should get the range error
	if !containsError(err.Error(), "invalid page range: start page must not be greater than end page") {
		t.Errorf("ParseRange() error = %q, want to contain 'invalid page range'", err.Error())
	}

	// Test invalid page range - page 0
	_, err = p.ParseRange(context.Background(), req, 0, 5)
	if err == nil {
		t.Error("ParseRange() expected error for page 0, got nil")
	}
	if !containsError(err.Error(), "page numbers must start from 1") {
		t.Errorf("ParseRange() error = %q, want to contain 'page numbers must start from 1'", err.Error())
	}

	// Test page range exceeding document size - this will fail at file parsing, but that's ok for this test
	// The important thing is that the validation logic is tested above
}

// Test that RangeParser interface is implemented
func TestRangeParserInterface(t *testing.T) {
	p := &Parser{}
	_, ok := interface{}(p).(parser.RangeParser)
	if !ok {
		t.Error("PDF Parser should implement RangeParser interface")
	}
}
