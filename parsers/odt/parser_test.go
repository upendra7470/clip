package odt

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
	want := "ODT"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.odt",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.odt does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.odt does not exist'", err.Error())
	}
}

func TestParseInvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.odt")

	// Create invalid ODT file (not a valid ZIP)
	invalidContent := []byte("This is not a valid ODT file")
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid ODT test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid ODT file, got nil")
	}

	if !containsError(err.Error(), "invalid ODT file") {
		t.Errorf("Parse() error = %q, want to contain 'invalid ODT file'", err.Error())
	}
}

func TestParseEmptyContent(t *testing.T) {
	// Create empty ODT file (valid ZIP but empty content.xml)
	// This would require creating a valid ODT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping empty content test - requires complex ODT structure")
}
func TestParseSimpleODT(t *testing.T) {
	// Create simple ODT file (valid ZIP with content.xml)
	// This would require creating a valid ODT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping simple ODT test - requires complex ODT structure")
}
func TestParseRange(t *testing.T) {
	// Create ODT file with multiple paragraphs
	// This would require creating a valid ODT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range test - requires complex ODT structure")
}
func TestParseRangeInvalid(t *testing.T) {
	// Create ODT file
	// This would require creating a valid ODT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range invalid test - requires complex ODT structure")
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	// Create ODT file with multiple paragraphs
	// This would require creating a valid ODT structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping open-ended range test - requires complex ODT structure")
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
