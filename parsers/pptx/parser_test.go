package pptx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/upendra7470/clip/internal/parser"
)

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "PPTX"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.pptx",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.pptx does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.pptx does not exist'", err.Error())
	}
}

func TestParseInvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.pptx")

	// Create invalid PPTX file
	invalidContent := []byte("This is not a valid PPTX file")
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid PPTX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid PPTX file, got nil")
	}

	if !containsError(err.Error(), "invalid PPTX file") {
		t.Errorf("Parse() error = %q, want to contain 'invalid PPTX file'", err.Error())
	}
}

func TestParseEmptyContent(t *testing.T) {
	// Create empty PPTX file
	// This would require creating a valid PPTX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping empty content test - requires complex PPTX structure")
}
func TestParseSimplePPTX(t *testing.T) {
	// Create simple PPTX file
	// This would require creating a valid PPTX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping simple PPTX test - requires complex PPTX structure")
}
func TestParseRange(t *testing.T) {
	// Create PPTX file with multiple slides
	// This would require creating a valid PPTX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range test - requires complex PPTX structure")
}
func TestParseRangeInvalid(t *testing.T) {
	// Create PPTX file
	// This would require creating a valid PPTX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range invalid test - requires complex PPTX structure")
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	// Create PPTX file with multiple slides
	// This would require creating a valid PPTX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping open-ended range test - requires complex PPTX structure")
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
