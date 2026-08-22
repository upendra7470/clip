package rtf

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
	want := "RTF"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.rtf",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.rtf does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.rtf does not exist'", err.Error())
	}
}

func TestParseEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.rtf")

	// Create empty file
	err := os.WriteFile(filePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty RTF test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty file, got nil")
	}

	if !containsError(err.Error(), "empty RTF file") {
		t.Errorf("Parse() error = %q, want to contain 'empty RTF file'", err.Error())
	}
}

func TestParseInvalidRTF(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.rtf")

	// Create file with invalid RTF
	invalidContent := []byte(`Invalid RTF content without proper structure`)
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid RTF test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid RTF, got nil")
	}

	if !containsError(err.Error(), "invalid RTF syntax") {
		t.Errorf("Parse() error = %q, want to contain 'invalid RTF syntax'", err.Error())
	}
}

func TestParseSimpleRTF(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.rtf")

	// Create simple RTF
	content := []byte(`{\rtf1\ansi\deff0
{\fonttbl{\f0 Arial;}}
\f0\fs24
This is a simple RTF document.

It has multiple paragraphs.
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple RTF test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is extracted
	if !strings.Contains(result.Text, "This is a simple RTF document") || !strings.Contains(result.Text, "It has multiple paragraphs") {
		t.Errorf("Parse() result missing expected text content: %q", result.Text)
	}
}

func TestParseRTFWithUnicode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.rtf")

	// Create RTF with Unicode content
	content := []byte(`{\rtf1\ansi\deff0
{\fonttbl{\f0 Arial;}}
\f0\fs24
Hello 世界! 🌍

Привет мир!
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode RTF test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that Unicode content is preserved
	if !strings.Contains(result.Text, "Hello 世界! 🌍") {
		t.Errorf("Parse() result missing Unicode content: %q", result.Text)
	}
	if !strings.Contains(result.Text, "Привет мир!") {
		t.Errorf("Parse() result missing Unicode content: %q", result.Text)
	}
}

func TestParseRTFWithFormatting(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "formatting.rtf")

	// Create RTF with formatting
	content := []byte(`{\rtf1\ansi\deff0
{\fonttbl{\f0 Arial;}}
\f0\fs24
Plain text. {\b Bold text.} {\i Italic text.}
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create formatting RTF test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is extracted (formatting codes should be removed)
	if !strings.Contains(result.Text, "Plain text") || !strings.Contains(result.Text, "Bold text") || !strings.Contains(result.Text, "Italic text") {
		t.Errorf("Parse() result missing expected text content: %q", result.Text)
	}
	// Formatting codes should not be in the text content
	if strings.Contains(result.Text, "{") || strings.Contains(result.Text, "}") || strings.Contains(result.Text, "\\b") || strings.Contains(result.Text, "\\i") {
		t.Errorf("Parse() result should not contain formatting codes: %q", result.Text)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.rtf")

	// Create RTF with multiple paragraphs
	content := []byte(`{\rtf1\ansi\deff0
{\fonttbl{\f0 Arial;}}
\f0\fs24
First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range test RTF file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test parsing specific paragraph range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}

	// Check that the result contains expected content from paragraphs 2-3
	if !strings.Contains(result.Text, "Second paragraph") || !strings.Contains(result.Text, "Third paragraph") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.rtf")

	// Create RTF file
	content := []byte(`{\rtf1\ansi\deff0
{\fonttbl{\f0 Arial;}}
\f0\fs24
First paragraph.

Second paragraph.
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid RTF file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test invalid range (start > end)
	_, err = p.ParseRange(context.Background(), req, 3, 2)
	if err == nil {
		t.Fatal("ParseRange() expected error for invalid range, got nil")
	}

	// Test range exceeding file paragraphs
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding file paragraphs, got nil")
	}
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "open_ended.rtf")

	// Create RTF with multiple paragraphs
	content := []byte(`{\rtf1\ansi\deff0
{\fonttbl{\f0 Arial;}}
\f0\fs24
First paragraph.

Second paragraph.

Third paragraph.

Fourth paragraph.
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create open-ended test RTF file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test open-ended range (start from paragraph 2 to end)
	result, err := p.ParseRange(context.Background(), req, 2, -1)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error for open-ended range: %v", err)
	}

	// Check that the result contains expected content from paragraph 2 to end
	if !strings.Contains(result.Text, "Second paragraph") || !strings.Contains(result.Text, "Fourth paragraph") {
		t.Errorf("ParseRange() result missing expected content for open-ended range: got %q", result.Text)
	}
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
