package docx

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upendra7470/clip/internal/parser"
)

func TestParseRange(t *testing.T) {
	// Create a temporary test DOCX file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.docx")
	createTestDOCX(t, testFile, "Paragraph 1: Hello World\nParagraph 2: This is a test\nParagraph 3: For range extraction\nParagraph 4: With multiple paragraphs\nParagraph 5: End of document")

	// Test requesting 2-4 from a document with at least 4 paragraphs
	docxParser := &Parser{}
	req := parser.ParseRequest{
		File: testFile,
	}
	result, err := docxParser.ParseRange(context.Background(), req, 2, 4)
	if err != nil {
		t.Fatalf("ParseRange failed: %v", err)
	}
	assert.NoError(t, err)

	// Regression tests for the DOCX range extraction bug fix
	assert.NotContains(t, result.Text, "Warning:", "ParseResult.Text must not contain warning messages")
	assert.NotContains(t, result.Text, "Paragraph 2:", "ParseResult.Text must not contain 'Paragraph 2:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 3:", "ParseResult.Text must not contain 'Paragraph 3:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 4:", "ParseResult.Text must not contain 'Paragraph 4:' prefix")

	// Assert that the result contains only the actual document content
	assert.Contains(t, result.Text, "This is a test", "ParseResult.Text must contain actual content from paragraph 2")
	assert.Contains(t, result.Text, "For range extraction", "ParseResult.Text must contain actual content from paragraph 3")
	assert.Contains(t, result.Text, "With multiple paragraphs", "ParseResult.Text must contain actual content from paragraph 4")

	// Test requesting a range that exceeds document length
	result, err = docxParser.ParseRange(context.Background(), req, 4, 10)
	assert.NoError(t, err)
	assert.NotContains(t, result.Text, "Warning:", "ParseResult.Text must not contain warning messages even when range is adjusted")
	assert.Contains(t, result.Text, "With multiple paragraphs")
	assert.Contains(t, result.Text, "End of document")

	// Test clipboard content contains only extracted document content
	result, err = docxParser.ParseRange(context.Background(), req, 1, 3)
	assert.NoError(t, err)
	assert.NotContains(t, result.Text, "Warning:", "ParseResult.Text must not contain warning messages")
	assert.NotContains(t, result.Text, "Paragraph 1:", "ParseResult.Text must not contain 'Paragraph 1:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 2:", "ParseResult.Text must not contain 'Paragraph 2:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 3:", "ParseResult.Text must not contain 'Paragraph 3:' prefix")
	assert.Contains(t, result.Text, "Hello World")
	assert.Contains(t, result.Text, "This is a test")
	assert.Contains(t, result.Text, "For range extraction")
	assert.NotContains(t, result.Text, "End of document")
}

func TestParseRangeWithRealisticDOCX(t *testing.T) {
	// Test with the realistic DOCX fixture
	docxParser := &Parser{}
	req := parser.ParseRequest{
		File: "../../test_realistic.docx",
	}

	// Test full document extraction
	result, err := docxParser.Parse(context.Background(), req)
	assert.NoError(t, err)

	// Verify full extraction returns actual content
	assert.Contains(t, result.Text, "This is the first paragraph of the document")
	assert.Contains(t, result.Text, "This is the second paragraph")
	assert.Contains(t, result.Text, "Third paragraph here")
	assert.Contains(t, result.Text, "This is the first paragraph after the table")
	assert.Contains(t, result.Text, "Second paragraph after the table")
	assert.Contains(t, result.Text, "Final paragraph of the document")

	// Verify table structure is preserved
	assert.Contains(t, result.Text, "| Name | Age | Occupation |")
	assert.Contains(t, result.Text, "| --- | --- | --- |")
	assert.Contains(t, result.Text, "| John Doe | 30 | Software Engineer |")
	assert.Contains(t, result.Text, "| Jane Smith | 25 | Data Scientist |")

	// Test range 1-3 extraction
	result, err = docxParser.ParseRange(context.Background(), req, 1, 3)
	assert.NoError(t, err)

	// Verify range output contains no artificial warning
	assert.NotContains(t, result.Text, "Warning:")

	// Verify range output contains no artificial "Paragraph N:" prefixes
	assert.NotContains(t, result.Text, "Paragraph 1:")
	assert.NotContains(t, result.Text, "Paragraph 2:")
	assert.NotContains(t, result.Text, "Paragraph 3:")

	// Verify range 1-3 returns actual content
	assert.Contains(t, result.Text, "This is the first paragraph of the document")
	assert.Contains(t, result.Text, "This is the second paragraph")
	assert.Contains(t, result.Text, "Third paragraph here")

	// Test range that includes the table
	result, err = docxParser.ParseRange(context.Background(), req, 3, 6)
	assert.NoError(t, err)

	// Verify table remains structured
	assert.Contains(t, result.Text, "Name")
	assert.Contains(t, result.Text, "Age")
	assert.Contains(t, result.Text, "Occupation")

	// Test range outside document length
	result, err = docxParser.ParseRange(context.Background(), req, 16, 20)
	assert.Error(t, err)

	// Verify error handling for out-of-range requests
	assert.Contains(t, err.Error(), "requested paragraph range exceeds document paragraph count")
	assert.NotContains(t, result.Text, "Warning:")
}

func TestParseRangeNoAdjustmentNoWarning(t *testing.T) {
	// Test that no warning is generated when requested range fits exactly
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.docx")
	createTestDOCX(t, testFile, "Paragraph 1: Hello\nParagraph 2: World\nParagraph 3: Test\nParagraph 4: Document")

	docxParser := &Parser{}
	req := parser.ParseRequest{
		File: testFile,
	}

	// Request range 2-4 which exactly matches the document's paragraph count
	result, err := docxParser.ParseRange(context.Background(), req, 2, 4)
	assert.NoError(t, err)

	// Should not contain any warning since no adjustment was needed
	assert.NotContains(t, result.Text, "Warning:", "No warning should be generated when range fits exactly")
	assert.NotContains(t, result.Text, "Paragraph 2:", "ParseResult.Text must not contain 'Paragraph 2:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 3:", "ParseResult.Text must not contain 'Paragraph 3:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 4:", "ParseResult.Text must not contain 'Paragraph 4:' prefix")
	assert.Contains(t, result.Text, "World")
	assert.Contains(t, result.Text, "Test")
	assert.Contains(t, result.Text, "Document")
}

func TestParseFullDocumentNoPrefixes(t *testing.T) {
	// Test that full document extraction also removes "Paragraph N:" prefixes
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.docx")
	createTestDOCX(t, testFile, "Paragraph 1: Hello\nParagraph 2: World\nParagraph 3: Test")

	docxParser := &Parser{}
	req := parser.ParseRequest{
		File: testFile,
	}

	// Test full document extraction
	result, err := docxParser.Parse(context.Background(), req)
	assert.NoError(t, err)

	// Should not contain any paragraph prefixes
	assert.NotContains(t, result.Text, "Paragraph 1:", "Full document extraction must not contain 'Paragraph 1:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 2:", "Full document extraction must not contain 'Paragraph 2:' prefix")
	assert.NotContains(t, result.Text, "Paragraph 3:", "Full document extraction must not contain 'Paragraph 3:' prefix")

	// Should contain the actual content
	assert.Contains(t, result.Text, "Hello")
	assert.Contains(t, result.Text, "World")
	assert.Contains(t, result.Text, "Test")
}

func createTestDOCX(t *testing.T, path string, content string) {
	// Create a DOCX file with the provided test content
	// Always create a new DOCX with the test content, don't use the fixture
	tempDir := t.TempDir()
	dst := filepath.Join(tempDir, "word", "document.xml")

	// Create directory structure
	err := os.MkdirAll(filepath.Dir(dst), 0755)
	assert.NoError(t, err)

	// Create the document.xml with test content
	testContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>`
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		testContent += `
		<w:p>
			<w:r>
				<w:t>` + line + `</w:t>
			</w:r>
		</w:p>`
	}
	testContent += `
	</w:body>
</w:document>`
	srcContent := []byte(testContent)

	// Write the document.xml to the temporary location
	err = os.WriteFile(dst, srcContent, 0644)
	assert.NoError(t, err)

	// Create the DOCX file (ZIP archive)
	zipFile, err := os.Create(path)
	assert.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add document.xml to the ZIP archive
	xmlFile, err := zipWriter.Create("word/document.xml")
	assert.NoError(t, err)

	_, err = xmlFile.Write(srcContent)
	assert.NoError(t, err)
}
