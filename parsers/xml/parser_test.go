package xml

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
	want := "XML"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.xml",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.xml does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.xml does not exist'", err.Error())
	}
}

func TestParseEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.xml")

	// Create empty file
	err := os.WriteFile(filePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty file, got nil")
	}

	if !containsError(err.Error(), "empty XML file") {
		t.Errorf("Parse() error = %q, want to contain 'empty XML file'", err.Error())
	}
}

func TestParseInvalidXML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.xml")

	// Create file with invalid XML
	invalidContent := []byte(`<root><item>Test</root>`) // Unclosed item tag
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid XML, got nil")
	}

	if !containsError(err.Error(), "invalid XML syntax") {
		t.Errorf("Parse() error = %q, want to contain 'invalid XML syntax'", err.Error())
	}
}

func TestParseSimpleXML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.xml")

	// Create simple XML
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item>Test Item 1</item>
  <item>Test Item 2</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple XML test file: %v", err)
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
	if !strings.Contains(result.Text, "Test Item 1") || !strings.Contains(result.Text, "Test Item 2") {
		t.Errorf("Parse() result missing expected text content: %q", result.Text)
	}
}

func TestParseNestedXML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nested.xml")

	// Create nested XML
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <parent>
    <child>Child Content</child>
  </parent>
  <item>Direct Item</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create nested XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that nested text content is extracted
	if !strings.Contains(result.Text, "Child Content") || !strings.Contains(result.Text, "Direct Item") {
		t.Errorf("Parse() result missing expected nested text content: %q", result.Text)
	}
}

func TestParseXMLWithAttributes(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "attributes.xml")

	// Create XML with attributes
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item id="1">First Item</item>
  <item id="2">Second Item</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create attributes XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is extracted (attributes should be ignored)
	if !strings.Contains(result.Text, "First Item") || !strings.Contains(result.Text, "Second Item") {
		t.Errorf("Parse() result missing expected text content: %q", result.Text)
	}
	// Attributes should not be in the text content
	if strings.Contains(result.Text, "id=") {
		t.Errorf("Parse() result should not contain attributes: %q", result.Text)
	}
}

func TestParseXMLWithComments(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "comments.xml")

	// Create XML with comments
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <!-- This is a comment -->
  <item>Test Item</item>
  <!-- Another comment -->
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create comments XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is extracted (comments should be ignored)
	if !strings.Contains(result.Text, "Test Item") {
		t.Errorf("Parse() result missing expected text content: %q", result.Text)
	}
	// Comments should not be in the text content
	if strings.Contains(result.Text, "This is a comment") || strings.Contains(result.Text, "Another comment") {
		t.Errorf("Parse() result should not contain comments: %q", result.Text)
	}
}

func TestParseXMLWithUnicode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.xml")

	// Create XML with Unicode content
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item>Hello 世界! 🌍</item>
  <item>Привет мир!</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode XML test file: %v", err)
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

func TestParseXMLWithWhitespace(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "whitespace.xml")

	// Create XML with significant whitespace
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item>
    Text with
    multiple lines
  </item>
  <item>Single line</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create whitespace XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is extracted (whitespace should be trimmed)
	if !strings.Contains(result.Text, "Text with multiple lines") {
		t.Errorf("Parse() result missing expected text content: %q", result.Text)
	}
	if !strings.Contains(result.Text, "Single line") {
		t.Errorf("Parse() result missing expected text content: %q", result.Text)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.xml")

	// Create XML with multiple text blocks
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item>First Item</item>
  <item>Second Item</item>
  <item>Third Item</item>
  <item>Fourth Item</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range test XML file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test parsing specific text block range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}

	// Check that the result contains expected content from blocks 2-3
	if !strings.Contains(result.Text, "Second Item") || !strings.Contains(result.Text, "Third Item") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.xml")

	// Create XML file
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item>First Item</item>
  <item>Second Item</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid XML file: %v", err)
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

	// Test range exceeding file blocks
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding file blocks, got nil")
	}
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "open_ended.xml")

	// Create XML with multiple text blocks
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item>First Item</item>
  <item>Second Item</item>
  <item>Third Item</item>
  <item>Fourth Item</item>
</root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create open-ended test XML file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test open-ended range (start from block 2 to end)
	result, err := p.ParseRange(context.Background(), req, 2, -1)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error for open-ended range: %v", err)
	}

	// Check that the result contains expected content from block 2 to end
	if !strings.Contains(result.Text, "Second Item") || !strings.Contains(result.Text, "Fourth Item") {
		t.Errorf("ParseRange() result missing expected content for open-ended range: got %q", result.Text)
	}
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
