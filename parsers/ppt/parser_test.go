package ppt

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
	want := "PPT"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.ppt",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.ppt does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.ppt does not exist'", err.Error())
	}
}

func TestParseInvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.ppt")

	// Create invalid PPT file
	invalidContent := []byte("This is not a valid PPT file")
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid PPT test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid PPT file, got nil")
	}

	if !containsError(err.Error(), "invalid PPT file") {
		t.Errorf("Parse() error = %q, want to contain 'invalid PPT file'", err.Error())
	}
}

func TestParseEmptyContent(t *testing.T) {
	// Create empty PPT file
	// This would require creating a valid PPT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping empty content test - requires complex PPT structure")
}
func TestParseSimplePPT(t *testing.T) {
	// Create simple PPT file
	// This would require creating a valid PPT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping simple PPT test - requires complex PPT structure")
}
func TestParseRange(t *testing.T) {
	// Create PPT file with multiple slides
	// This would require creating a valid PPT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range test - requires complex PPT structure")
}
func TestParseRangeInvalid(t *testing.T) {
	// Create PPT file
	// This would require creating a valid PPT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range invalid test - requires complex PPT structure")
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	// Create PPT file with multiple slides
	// This would require creating a valid PPT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping open-ended range test - requires complex PPT structure")
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
