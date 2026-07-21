package tests

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/upendra7470/clip/internal/application"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/registry"
	"github.com/upendra7470/clip/parsers/txt"
)

func TestCLIBuilds(t *testing.T) {
	// Test that the CLI can be built
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}

	// Clean up
	os.Remove("clip")
}

func TestCLIHelpFlag(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test --help flag
	cmd = exec.Command("./clip", "--help")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help flag failed: %v\n%s", err, output)
	}

	// Check that help output contains expected text
	outputStr := string(output)
	if !contains(outputStr, "Clip - Universal Document Extractor") {
		t.Errorf("Help output should contain 'Clip - Universal Document Extractor', got: %s", outputStr)
	}
	if !contains(outputStr, "Usage:") {
		t.Errorf("Help output should contain 'Usage:', got: %s", outputStr)
	}
	if !contains(outputStr, "clip <filename>") {
		t.Errorf("Help output should contain 'clip <filename>', got: %s", outputStr)
	}
}

func TestCLIVersionFlag(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test --version flag
	cmd = exec.Command("./clip", "--version")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version flag failed: %v\n%s", err, output)
	}

	// Check that version output contains expected text
	outputStr := string(output)
	if !contains(outputStr, "Clip v1.0.0") {
		t.Errorf("Version output should contain 'Clip v1.0.0', got: %s", outputStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCLIFileResolution(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a test file
	testContent := "Hello, this is a test file for Clip CLI."
	testFile := "test_clip.txt"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test file resolution and extraction
	cmd = exec.Command("./clip", testFile)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("File extraction failed: %v\n%s", err, output)
	}

	// Check that output contains success messages
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted text successfully") {
		t.Errorf("Output should contain '✓ Extracted text successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLIErrorHandling(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test with non-existent file
	cmd = exec.Command("./clip", "nonexistent_file.pdf")
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Expected error for non-existent file, but got none")
	}

	// Check that error message is user-friendly
	outputStr := string(output)
	if !contains(outputStr, "file \"nonexistent_file.pdf\" not found") {
		t.Errorf("Error output should contain file not found message, got: %s", outputStr)
	}
	if !contains(outputStr, "search locations checked:") {
		t.Errorf("Error output should contain search locations, got: %s", outputStr)
	}
}

func TestCLINoFileProvided(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Expected no error when no file provided (should show help), but got: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test with no file provided - should show help and exit gracefully
	cmd = exec.Command("./clip")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Expected no error when no file provided (should show help), but got: %v\n%s", err, output)
	}

	// Check that help is shown
	outputStr := string(output)
	if !contains(outputStr, "Clip - Universal Document Extractor") {
		t.Errorf("Output should contain help message, got: %s", outputStr)
	}
	if !contains(outputStr, "Usage:") {
		t.Errorf("Output should contain usage information, got: %s", outputStr)
	}
}

func TestCLIFilenameWithSpaces(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a test file with spaces in the name
	testContent := "Hello, this is a test file with spaces in the name for Clip CLI."
	testFile := "test file with spaces.txt"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test file resolution and extraction with quoted filename
	cmd = exec.Command("./clip", testFile)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("File extraction with spaces failed: %v\n%s", err, output)
	}

	// Check that output contains success messages
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted text successfully") {
		t.Errorf("Output should contain '✓ Extracted text successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLIRangeExtraction(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a multi-line test file
	testContent := `Line 1: Hello World
Line 2: This is a test
Line 3: For range extraction
Line 4: With multiple lines
Line 5: To test the functionality`
	testFile := "test_range.txt"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test range extraction (lines 2-3)
	cmd = exec.Command("./clip", testFile, "2-3")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Range extraction failed: %v\n%s", err, output)
	}

	// Check that output contains success messages with correct range unit
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted lines 2-3 successfully") {
		t.Errorf("Output should contain '✓ Extracted lines 2-3 successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLISmartFilenameResolution(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a test file with spaces and mixed case - need to create a valid DOCX file
	testFile := "The Brain.docx"

	// Create a temporary directory for DOCX content
	tempDir, err := os.MkdirTemp("", "docx_content")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create word/document.xml
	wordDir := filepath.Join(tempDir, "word")
	err = os.MkdirAll(wordDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create word directory: %v", err)
	}

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>Hello, this is a test file for smart filename resolution.</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`

	err = os.WriteFile(filepath.Join(wordDir, "document.xml"), []byte(documentXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create document.xml: %v", err)
	}

	// Create the DOCX file as a ZIP archive
	zipFile, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create DOCX file: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)

	// Add document.xml to the ZIP
	xmlFile, err := os.Open(filepath.Join(wordDir, "document.xml"))
	if err != nil {
		zipWriter.Close()
		zipFile.Close()
		t.Fatalf("Failed to open document.xml: %v", err)
	}
	defer xmlFile.Close()

	xmlInfo, err := xmlFile.Stat()
	if err != nil {
		zipWriter.Close()
		zipFile.Close()
		t.Fatalf("Failed to get document.xml info: %v", err)
	}

	xmlHeader := &zip.FileHeader{
		Name:   "word/document.xml",
		Method: zip.Deflate,
	}
	xmlHeader.SetModTime(xmlInfo.ModTime())

	xmlWriter, err := zipWriter.CreateHeader(xmlHeader)
	if err != nil {
		zipWriter.Close()
		zipFile.Close()
		t.Fatalf("Failed to create ZIP entry: %v", err)
	}

	_, err = io.Copy(xmlWriter, xmlFile)
	if err != nil {
		zipWriter.Close()
		zipFile.Close()
		t.Fatalf("Failed to write document.xml to ZIP: %v", err)
	}

	// Close the ZIP file
	err = zipWriter.Close()
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to close ZIP writer: %v", err)
	}

	err = zipFile.Close()
	if err != nil {
		t.Fatalf("Failed to close ZIP file: %v", err)
	}

	defer os.Remove(testFile)

	// Test different variations of the filename
	testCases := []struct {
		name     string
		query    string
		expected bool
	}{
		{"exact match", "The Brain.docx", true},
		{"lowercase no spaces", "thebrain.docx", true},
		{"lowercase with spaces", "the brain.docx", true},
		{"uppercase no spaces", "THEBRAIN.DOCX", true},
		{"uppercase with spaces", "THE BRAIN.DOCX", true},
		{"mixed case no spaces", "TheBrain.docx", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("./clip", tc.query)
			output, err = cmd.CombinedOutput()
			if tc.expected && err != nil {
				t.Errorf("Expected success for %q but got error: %v\n%s", tc.query, err, output)
				return
			} else if !tc.expected && err == nil {
				t.Errorf("Expected error for %q but got success", tc.query)
				return
			}

			if tc.expected {
				outputStr := string(output)
				if !contains(outputStr, "✓ Found:") {
					t.Errorf("Output should contain '✓ Found:' for %q, got: %s", tc.query, outputStr)
				}
				if !contains(outputStr, "✓ Extracted text successfully") {
					t.Errorf("Output should contain '✓ Extracted text successfully' for %q, got: %s", tc.query, outputStr)
				}
			}
		})
	}
}

func TestCLIFilenameWithSpacesNoQuotes(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a test file with spaces in the name
	testContent := "Hello, this is a test file with spaces in the name for Clip CLI."
	testFile := "test file with spaces.txt"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test file resolution and extraction without quotes
	cmd = exec.Command("./clip", testFile)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("File extraction with spaces failed: %v\n%s", err, output)
	}

	// Check that output contains success messages
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted text successfully") {
		t.Errorf("Output should contain '✓ Extracted text successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLIRangeExtractionPDF(t *testing.T) {
	// Skip PDF test since it requires a real PDF file
	// In a real scenario, this would test with an actual PDF file
	t.Skip("PDF range extraction test skipped - requires actual PDF file")
}

func TestCLIRangeExtractionCSV(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a multi-row test CSV file
	testContent := `Name,Age,City
John,25,New York
Jane,30,Los Angeles
Bob,35,Chicago
Alice,28,Houston`
	testFile := "test_range.csv"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test range extraction (rows 2-3)
	cmd = exec.Command("./clip", testFile, "2-3")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CSV range extraction failed: %v\n%s", err, output)
	}

	// Check that output contains success messages with correct range unit
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted rows 2-3 successfully") {
		t.Errorf("Output should contain '✓ Extracted rows 2-3 successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLIRangeExtractionMarkdown(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a multi-line test Markdown file
	testContent := `# Heading 1

Content for line 1

## Heading 2

Content for line 2

## Heading 3

Content for line 3`
	testFile := "test_range.md"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test range extraction (lines 3-5)
	cmd = exec.Command("./clip", testFile, "3-5")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Markdown range extraction failed: %v\n%s", err, output)
	}

	// Check that output contains success messages with correct range unit
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted lines 3-5 successfully") {
		t.Errorf("Output should contain '✓ Extracted lines 3-5 successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLIRangeExtractionJSON(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a multi-line test JSON file
	testContent := `{
	"name": "John",
	"age": 30,
	"city": "New York",
	"hobbies": ["reading", "hiking", "coding"]
}`
	testFile := "test_range.json"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test range extraction (lines 2-4)
	cmd = exec.Command("./clip", testFile, "2-4")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("JSON range extraction failed: %v\n%s", err, output)
	}

	// Check that output contains success messages with correct range unit
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted lines 2-4 successfully") {
		t.Errorf("Output should contain '✓ Extracted lines 2-4 successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLISingleUnitRange(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a multi-line test file
	testContent := `Line 1: Hello World
Line 2: This is a test
Line 3: For single unit extraction
Line 4: With multiple lines
Line 5: To test the functionality`
	testFile := "test_single.txt"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test single unit extraction (line 3)
	cmd = exec.Command("./clip", testFile, "3")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Single unit range extraction failed: %v\n%s", err, output)
	}

	// Check that output contains success messages with correct range unit
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted lines 3-3 successfully") {
		t.Errorf("Output should contain '✓ Extracted lines 3-3 successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLIRangeExtractionNotEntireDocument(t *testing.T) {
	// Create a multi-line test file with unique content on each line
	testContent := `UNIQUE_LINE_1
UNIQUE_LINE_2
UNIQUE_LINE_3
UNIQUE_LINE_4
UNIQUE_LINE_5`
	testFile := "test_not_entire.txt"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test range extraction using application layer directly to verify actual extracted content
	// Import the necessary packages
	reg := registry.New()
	clipboard := &mockClipboard{}
	app := application.New(reg, clipboard)

	// Register the TXT parser
	txtParser := &txt.Parser{}
	if err := reg.Register(filetype.FileTypeTXT, txtParser); err != nil {
		t.Fatalf("Failed to register TXT parser: %v", err)
	}

	// Extract lines 2-4
	rangeObj := &parser.Range{Start: 2, End: 4}
	err = app.ExtractWithRange(context.Background(), testFile, rangeObj)
	if err != nil {
		t.Fatalf("Range extraction failed: %v", err)
	}

	// Verify the extracted content was correct
	extracted := clipboard.GetCopiedText()
	if extracted == "" {
		t.Fatal("No text was copied to clipboard")
	}

	// Check that the extracted content does NOT contain the entire document
	if contains(extracted, "UNIQUE_LINE_1") {
		t.Errorf("Range extraction should not include line 1, got: %s", extracted)
	}
	if contains(extracted, "UNIQUE_LINE_5") {
		t.Errorf("Range extraction should not include line 5, got: %s", extracted)
	}

	// Check that it contains the expected lines
	if !contains(extracted, "UNIQUE_LINE_2") {
		t.Errorf("Range extraction should include line 2, got: %s", extracted)
	}
	if !contains(extracted, "UNIQUE_LINE_3") {
		t.Errorf("Range extraction should include line 3, got: %s", extracted)
	}
	if !contains(extracted, "UNIQUE_LINE_4") {
		t.Errorf("Range extraction should include line 4, got: %s", extracted)
	}
}

// mockClipboard is a test implementation of application.Clipboard interface.
type mockClipboard struct {
	copiedText string
}

func (m *mockClipboard) Copy(text string) error {
	m.copiedText = text
	return nil
}

func (m *mockClipboard) GetCopiedText() string {
	return m.copiedText
}

func TestCLIInvalidRange(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a test file
	testContent := "Hello, this is a test file for Clip CLI."
	testFile := "test_invalid_range.txt"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test with invalid range (should show error)
	cmd = exec.Command("./clip", testFile, "0-5")
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Expected error for invalid range, but got none")
	}

	// Check that error message is user-friendly
	outputStr := string(output)
	if !contains(outputStr, "range values must start from 1") {
		t.Errorf("Error output should contain range validation message, got: %s", outputStr)
	}
}
