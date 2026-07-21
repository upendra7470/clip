package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/upendra7470/clip/internal/parser"
)

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "DOCX"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.docx",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "Could not open DOCX file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open DOCX file'", err.Error())
	}
}

func TestParseInvalidDOCX(t *testing.T) {
	// Create a file with invalid DOCX content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.docx")

	// Write invalid content (not a ZIP file)
	invalidContent := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid DOCX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid DOCX, got nil")
	}

	if !containsError(err.Error(), "failed to read DOCX as ZIP archive") {
		t.Errorf("Parse() error = %q, want to contain 'failed to read DOCX as ZIP archive'", err.Error())
	}
}

func TestParseDOCXWithoutDocumentXML(t *testing.T) {
	// Create a ZIP file without word/document.xml
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "no_document.docx")

	// Create ZIP with other files but no document.xml
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)

	// Add a different file
	f, _ := writer.Create("_rels/.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`))

	writer.Close()

	err := os.WriteFile(filePath, buf.Bytes(), 0644)
	if err != nil {
		t.Fatalf("Failed to create DOCX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing document.xml, got nil")
	}

	if !containsError(err.Error(), "document.xml not found in DOCX") {
		t.Errorf("Parse() error = %q, want to contain 'document.xml not found in DOCX'", err.Error())
	}
}

func TestParseEmptyDOCX(t *testing.T) {
	// Create a minimal DOCX with empty document.xml
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.docx")

	// Create minimal DOCX structure
	docxContent := createMinimalDOCX("")
	err := os.WriteFile(filePath, docxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty DOCX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty DOCX, got nil")
	}

	if !containsError(err.Error(), "no text content found in DOCX") {
		t.Errorf("Parse() error = %q, want to contain 'no text content found in DOCX'", err.Error())
	}
}

func TestParseSimpleDOCX(t *testing.T) {
	// Create a simple DOCX with text content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.docx")

	// Create DOCX with simple text
	content := "Hello, World! This is a test."
	docxContent := createMinimalDOCX(content)
	err := os.WriteFile(filePath, docxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple DOCX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if result.Text != content {
		t.Errorf("Parse() result = %q, want %q", result.Text, content)
	}
}

func TestParseMultiParagraphDOCX(t *testing.T) {
	// Create a DOCX with multiple paragraphs
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "multi.docx")

	// Create DOCX with multiple paragraphs
	content := "First paragraph. Second paragraph. Third paragraph."
	docxContent := createMinimalDOCX(content)
	err := os.WriteFile(filePath, docxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create multi-paragraph DOCX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if result.Text != content {
		t.Errorf("Parse() result = %q, want %q", result.Text, content)
	}
}

func TestParseUnicodeDOCX(t *testing.T) {
	// Create a DOCX with Unicode content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.docx")

	// Create DOCX with Unicode text
	content := "Hello 世界! 🌍 Привет мир!"
	docxContent := createMinimalDOCX(content)
	err := os.WriteFile(filePath, docxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode DOCX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	if result.Text != content {
		t.Errorf("Parse() result = %q, want %q", result.Text, content)
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.docx",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "Could not open DOCX file") {
		t.Errorf("Error message = %q, want to contain 'Could not open DOCX file'", err.Error())
	}
}

// createMinimalDOCX creates a minimal valid DOCX file as a ZIP archive.
// It contains the required structure with word/document.xml containing the given text.
func createMinimalDOCX(text string) []byte {
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)

	// Add [Content_Types].xml
	f, _ := writer.Create("[Content_Types].xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	// Add _rels/.rels
	f, _ = writer.Create("_rels/.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	// Add word/_rels/document.xml.rels
	f, _ = writer.Create("word/_rels/document.xml.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`))

	// Add word/document.xml with the text content
	f, _ = writer.Create("word/document.xml")
	f.Write([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>%s</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`, text)))

	writer.Close()
	return buf.Bytes()
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
