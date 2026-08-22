package csv

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
	want := "CSV"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.csv",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.csv does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.csv does not exist'", err.Error())
	}
}

func TestParseEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.csv")

	// Create empty file
	err := os.WriteFile(filePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty file, got nil")
	}

	if !containsError(err.Error(), "no readable content found in CSV") {
		t.Errorf("Parse() error = %q, want to contain 'no readable content found in CSV'", err.Error())
	}
}

func TestParseSimpleCSV(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.csv")

	// Create simple CSV
	content := []byte(`name,age,city
Sai,19,Hyderabad
Ravi,20,Bangalore`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that CSV content is present
	expectedValues := []string{"Sai", "19", "Hyderabad", "Ravi", "20", "Bangalore"}
	for _, value := range expectedValues {
		if !strings.Contains(result.Text, value) {
			t.Errorf("Parse() result missing expected value %q: %q", value, result.Text)
		}
	}
}

func TestParseCSVWithQuotes(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "quotes.csv")

	// Create CSV with quoted values
	content := []byte(`name,description
Sai,"A student from Hyderabad"
Ravi,"A student from Bangalore"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create quoted CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that quoted values are preserved
	if !strings.Contains(result.Text, "A student from Hyderabad") {
		t.Errorf("Parse() result missing quoted value: %q", result.Text)
	}
}

func TestParseCSVWithCommas(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "commas.csv")

	// Create CSV with commas in quoted values
	content := []byte(`name,address
Sai,"Hyderabad, Telangana"
Ravi,"Bangalore, Karnataka"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create commas CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that commas within quotes are preserved
	if !strings.Contains(result.Text, "Hyderabad, Telangana") {
		t.Errorf("Parse() result missing comma-preserved value: %q", result.Text)
	}
}

func TestParseCSVWithNewlines(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "newlines.csv")

	// Create CSV with newlines in quoted values
	content := []byte(`name,description
Sai,"A student
from Hyderabad"
Ravi,"A student
from Bangalore"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create newlines CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that newlines within quotes are handled
	if !strings.Contains(result.Text, "A student\nfrom Hyderabad") {
		t.Errorf("Parse() result missing newline-handled value: %q", result.Text)
	}
}

func TestParseCSVWithUnicode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.csv")

	// Create CSV with Unicode content
	content := []byte(`name,message
Alice,Hello 世界! 🌍
Bob,Привет мир!`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode CSV test file: %v", err)
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

func TestParseCSVWithEmptyFields(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.csv")

	// Create CSV with empty fields
	content := []byte(`name,age,city
Sai,19,
Ravi,,Bangalore`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty fields CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that empty fields are handled
	if !strings.Contains(result.Text, "Sai, 19") {
		t.Errorf("Parse() result missing empty field handling: %q", result.Text)
	}
}

func TestParseCSVWithNumbers(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "numbers.csv")

	// Create CSV with various number types
	content := []byte(`name,age,price,quantity
Item1,19,29.99,100
Item2,25,99.99,50`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create numbers CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that numbers are present
	if !strings.Contains(result.Text, "19") || !strings.Contains(result.Text, "29.99") {
		t.Errorf("Parse() result missing number values: %q", result.Text)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.csv")

	// Create CSV with multiple rows
	content := []byte(`name,age
Sai,19
Ravi,20
Alice,21
Bob,22`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range test CSV file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test parsing specific row range
	result, err := p.ParseRange(context.Background(), req, 3, 4)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}

	// Check that the result contains expected content from rows 3-4
	if !strings.Contains(result.Text, "Ravi") || !strings.Contains(result.Text, "Alice") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.csv")

	// Create CSV file
	content := []byte(`name,age
Sai,19
Ravi,20`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid CSV file: %v", err)
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

	// Test range exceeding file rows
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding file rows, got nil")
	}
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "open_ended.csv")

	// Create CSV with multiple rows
	content := []byte(`name,age
Sai,19
Ravi,20
Alice,21
Bob,22`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create open-ended test CSV file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test open-ended range (start from row 2 to end)
	result, err := p.ParseRange(context.Background(), req, 2, -1)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error for open-ended range: %v", err)
	}

	// Check that the result contains expected content from row 2 to end
	if !strings.Contains(result.Text, "Ravi") || !strings.Contains(result.Text, "Bob") {
		t.Errorf("ParseRange() result missing expected content for open-ended range: got %q", result.Text)
	}
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
