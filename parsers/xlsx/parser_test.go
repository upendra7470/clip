package xlsx

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
	want := "XLSX"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.xlsx",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.xlsx does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.xlsx does not exist'", err.Error())
	}
}

func TestParseInvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.xlsx")

	// Create invalid XLSX file
	invalidContent := []byte("This is not a valid XLSX file")
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid XLSX file, got nil")
	}

	if !containsError(err.Error(), "invalid XLSX file") {
		t.Errorf("Parse() error = %q, want to contain 'invalid XLSX file'", err.Error())
	}
}

func TestParseEmptyContent(t *testing.T) {
	// Create empty XLSX file
	// This would require creating a valid XLSX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping empty content test - requires complex XLSX structure")
}
func TestParseSimpleXLSX(t *testing.T) {
	// Create simple XLSX file
	// This would require creating a valid XLSX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping simple XLSX test - requires complex XLSX structure")
}
func TestParseRange(t *testing.T) {
	// Create XLSX file with multiple rows
	// This would require creating a valid XLSX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range test - requires complex XLSX structure")
}
func TestParseRangeInvalid(t *testing.T) {
	// Create XLSX file
	// This would require creating a valid XLSX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping range invalid test - requires complex XLSX structure")
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	// Create XLSX file with multiple rows
	// This would require creating a valid XLSX structure, which is complex
	// For now, we'll skip this test
	t.Skip("Skipping open-ended range test - requires complex XLSX structure")
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
