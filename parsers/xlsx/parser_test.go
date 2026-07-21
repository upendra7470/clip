package xlsx

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
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

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "Could not open XLSX file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open XLSX file'", err.Error())
	}
}

func TestParseInvalidXLSX(t *testing.T) {
	// Create a file with invalid XLSX content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.xlsx")

	// Write invalid content (not a ZIP file)
	invalidContent := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid XLSX, got nil")
	}

	if !containsError(err.Error(), "failed to read XLSX as ZIP archive") {
		t.Errorf("Parse() error = %q, want to contain 'failed to read XLSX as ZIP archive'", err.Error())
	}
}

func TestParseXLSXWithoutWorksheets(t *testing.T) {
	// Create a ZIP file without worksheets
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "no_worksheets.xlsx")

	// Create minimal XLSX structure without worksheets
	xlsxContent := createXLSXWithoutWorksheets()
	err := os.WriteFile(filePath, xlsxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing worksheets, got nil")
	}

	if !containsError(err.Error(), "no worksheets found in XLSX") {
		t.Errorf("Parse() error = %q, want to contain 'no worksheets found in XLSX'", err.Error())
	}
}

func TestParseEmptyXLSX(t *testing.T) {
	// Create a minimal XLSX with empty worksheet
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.xlsx")

	// Create XLSX with empty data
	xlsxContent := createMinimalXLSX([][]string{})
	err := os.WriteFile(filePath, xlsxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty XLSX, got nil")
	}

	if !containsError(err.Error(), "no content found in XLSX") {
		t.Errorf("Parse() error = %q, want to contain 'no content found in XLSX'", err.Error())
	}
}

func TestParseSimpleXLSX(t *testing.T) {
	// Create a simple XLSX with text content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.xlsx")

	// Create XLSX with simple data
	data := [][]string{
		{"Name", "Age", "City"},
		{"John", "25", "New York"},
		{"Jane", "30", "San Francisco"},
	}
	xlsxContent := createMinimalXLSX(data)
	err := os.WriteFile(filePath, xlsxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	expected := "Name, Age, City\nJohn, 25, New York\nJane, 30, San Francisco"
	if result.Text != expected {
		t.Errorf("Parse() result = %q, want %q", result.Text, expected)
	}
}

func TestParseXLSXWithSharedStrings(t *testing.T) {
	// Create an XLSX with shared strings
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "shared.xlsx")

	// Create XLSX with data that will use shared strings
	data := [][]string{
		{"Name", "Status", "City"},
		{"John", "Active", "New York"},
		{"Jane", "Active", "San Francisco"},
		{"Bob", "Inactive", "Chicago"},
	}
	xlsxContent := createMinimalXLSX(data)
	err := os.WriteFile(filePath, xlsxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create shared strings XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Should contain all the data
	if !strings.Contains(result.Text, "Name") ||
		!strings.Contains(result.Text, "Active") ||
		!strings.Contains(result.Text, "New York") {
		t.Errorf("Parse() result = %q, missing expected content", result.Text)
	}
}

func TestParseXLSXWithUnicode(t *testing.T) {
	// Create an XLSX with Unicode content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.xlsx")

	// Create XLSX with Unicode data
	data := [][]string{
		{"Name", "Message"},
		{"Alice", "Hello 世界! 🌍"},
		{"Bob", "Привет мир!"},
	}
	xlsxContent := createMinimalXLSX(data)
	err := os.WriteFile(filePath, xlsxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Should preserve Unicode content
	if !strings.Contains(result.Text, "Hello 世界! 🌍") ||
		!strings.Contains(result.Text, "Привет мир!") {
		t.Errorf("Parse() result = %q, missing Unicode content", result.Text)
	}
}

func TestParseXLSXWithEmptyCells(t *testing.T) {
	// Create an XLSX with empty cells
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty_cells.xlsx")

	// Create XLSX with empty cells
	data := [][]string{
		{"Name", "Age", "City"},
		{"John", "", "New York"},
		{"", "30", "San Francisco"},
		{"Bob", "25", ""},
	}
	xlsxContent := createMinimalXLSX(data)
	err := os.WriteFile(filePath, xlsxContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty cells XLSX test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Should handle empty cells
	lines := strings.Split(result.Text, "\n")
	if len(lines) != 4 {
		t.Errorf("Parse() result should have 4 lines, got %d: %q", len(lines), result.Text)
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.xlsx",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "Could not open XLSX file") {
		t.Errorf("Error message = %q, want to contain 'Could not open XLSX file'", err.Error())
	}
}

// createMinimalXLSX creates a minimal valid XLSX file as a ZIP archive.
// It contains the required structure with xl/sharedStrings.xml and xl/worksheets/sheet1.xml.
func createMinimalXLSX(data [][]string) []byte {
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)

	// Add [Content_Types].xml
	f, _ := writer.Create("[Content_Types].xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
	<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
	<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`))

	// Add _rels/.rels
	f, _ = writer.Create("_rels/.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`))

	// Add xl/_rels/workbook.xml.rels
	f, _ = writer.Create("xl/_rels/workbook.xml.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
	<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`))

	// Add xl/workbook.xml
	f, _ = writer.Create("xl/workbook.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<sheets>
		<sheet name="Sheet1" sheetId="1" r:id="rId1"/>
	</sheets>
</workbook>`))

	// Add xl/sharedStrings.xml
	f, _ = writer.Create("xl/sharedStrings.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="0" uniqueCount="0"/>`))

	// Add xl/worksheets/sheet1.xml with the data
	f, _ = writer.Create("xl/worksheets/sheet1.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<sheetData>`))

	// Add rows
	for i, row := range data {
		f.Write([]byte(`<row>`))
		for j, cell := range row {
			cellRef := fmt.Sprintf("%c%d", 'A'+j, i+1) // A1, B1, C1, etc.
			if cell == "" {
				f.Write([]byte(fmt.Sprintf(`<c r="%s"/>`, cellRef)))
			} else {
				f.Write([]byte(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, cellRef, cell)))
			}
		}
		f.Write([]byte(`</row>`))
	}

	f.Write([]byte(`</sheetData>
</worksheet>`))

	writer.Close()
	return buf.Bytes()
}

// createXLSXWithoutWorksheets creates an XLSX file without any worksheet files.
func createXLSXWithoutWorksheets() []byte {
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)

	// Add [Content_Types].xml (without worksheet references)
	f, _ := writer.Create("[Content_Types].xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
	<Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>`))

	// Add _rels/.rels
	f, _ = writer.Create("_rels/.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`))

	// Add xl/_rels/workbook.xml.rels (without worksheet references)
	f, _ = writer.Create("xl/_rels/workbook.xml.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>`))

	// Add xl/workbook.xml (without sheets)
	f, _ = writer.Create("xl/workbook.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<sheets>
	</sheets>
</workbook>`))

	// Add xl/sharedStrings.xml
	f, _ = writer.Create("xl/sharedStrings.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="0" uniqueCount="0"/>`))

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
