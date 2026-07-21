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

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "Could not open XML file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open XML file'", err.Error())
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

	_, err = p.Parse(context.Background(), req)

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
	invalidContent := []byte(`<person><name>Sai</person>`) // Unclosed tag
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

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
	content := []byte(`<person>
    <name>Sai</name>
    <age>19</age>
    <city>Hyderabad</city>
</person>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that all text content is present
	expectedTexts := []string{"Sai", "19", "Hyderabad"}
	for _, text := range expectedTexts {
		if !strings.Contains(result.Text, text) {
			t.Errorf("Parse() result missing expected text %q: %q", text, result.Text)
		}
	}
}

func TestParseNestedXML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nested.xml")

	// Create nested XML
	content := []byte(`<users>
    <user>
        <name>Sai</name>
        <contact>
            <email>sai@example.com</email>
            <phone>+1234567890</phone>
        </contact>
    </user>
    <user>
        <name>Ravi</name>
        <contact>
            <email>ravi@example.com</email>
            <phone>+0987654321</phone>
        </contact>
    </user>
</users>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create nested XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that nested content is present
	expectedTexts := []string{"Sai", "sai@example.com", "Ravi", "ravi@example.com"}
	for _, text := range expectedTexts {
		if !strings.Contains(result.Text, text) {
			t.Errorf("Parse() result missing expected text %q: %q", text, result.Text)
		}
	}
}

func TestParseXMLWithAttributes(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "attributes.xml")

	// Create XML with attributes (should be ignored)
	content := []byte(`<person id="1" status="active">
    <name lang="en">Sai</name>
    <age unit="years">19</age>
</person>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create attributes XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is present but attributes are ignored
	if !strings.Contains(result.Text, "Sai") {
		t.Errorf("Parse() result missing 'Sai': %q", result.Text)
	}
	if !strings.Contains(result.Text, "19") {
		t.Errorf("Parse() result missing '19': %q", result.Text)
	}
	// Attributes should not appear in output
	if strings.Contains(result.Text, "id=") || strings.Contains(result.Text, "status=") {
		t.Errorf("Parse() result should not contain attributes: %q", result.Text)
	}
}

func TestParseXMLWithNamespaces(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "namespaces.xml")

	// Create XML with namespaces (should be ignored)
	content := []byte(`<ns:person xmlns:ns="http://example.com/ns">
    <ns:name>Sai</ns:name>
    <ns:age>19</ns:age>
</ns:person>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create namespaces XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is present
	if !strings.Contains(result.Text, "Sai") {
		t.Errorf("Parse() result missing 'Sai': %q", result.Text)
	}
	if !strings.Contains(result.Text, "19") {
		t.Errorf("Parse() result missing '19': %q", result.Text)
	}
}

func TestParseXMLWithComments(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "comments.xml")

	// Create XML with comments (should be ignored)
	content := []byte(`<person>
    <!-- This is a comment -->
    <name>Sai</name>
    <!-- Another comment -->
    <age>19</age>
</person>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create comments XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is present but comments are ignored
	if !strings.Contains(result.Text, "Sai") {
		t.Errorf("Parse() result missing 'Sai': %q", result.Text)
	}
	if !strings.Contains(result.Text, "19") {
		t.Errorf("Parse() result missing '19': %q", result.Text)
	}
	// Comments should not appear in output
	if strings.Contains(result.Text, "This is a comment") {
		t.Errorf("Parse() result should not contain comments: %q", result.Text)
	}
}

func TestParseUnicodeXML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.xml")

	// Create XML with Unicode content
	content := []byte(`<person>
    <name>Alice</name>
    <message>Hello 世界! 🌍</message>
    <greeting>Привет мир!</greeting>
</person>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

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

func TestParseWhitespaceHandling(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "whitespace.xml")

	// Create XML with various whitespace scenarios
	content := []byte(`<person>
    <name>
        Sai
    </name>
    <age>19</age>
    <city>
        Hyderabad
    </city>
</person>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create whitespace XML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that text content is present and properly trimmed
	if !strings.Contains(result.Text, "Sai") {
		t.Errorf("Parse() result missing 'Sai': %q", result.Text)
	}
	if !strings.Contains(result.Text, "Hyderabad") {
		t.Errorf("Parse() result missing 'Hyderabad': %q", result.Text)
	}

	// Should not have excessive whitespace
	if strings.Contains(result.Text, "\n\n\n") {
		t.Errorf("Parse() result has excessive whitespace: %q", result.Text)
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.xml",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "Could not open XML file") {
		t.Errorf("Error message = %q, want to contain 'Could not open XML file'", err.Error())
	}
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
