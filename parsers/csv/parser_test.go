package csv

import (
	"context"
	"os"
	"path/filepath"
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

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "Could not open CSV file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open CSV file'", err.Error())
	}
}

func TestParseEmptyCSV(t *testing.T) {
	// Create an empty CSV file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.csv")

	// Write empty CSV content
	err := os.WriteFile(filePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty CSV, got nil")
	}

	if !containsError(err.Error(), "no content found in CSV") {
		t.Errorf("Parse() error = %q, want to contain 'no content found in CSV'", err.Error())
	}
}

func TestParseSimpleCSV(t *testing.T) {
	// Create a simple CSV file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.csv")

	// Write simple CSV content
	csvContent := "Name,Age,City\nJohn,25,New York\nJane,30,San Francisco"
	err := os.WriteFile(filePath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create simple CSV test file: %v", err)
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

func TestParseCSVWithQuotes(t *testing.T) {
	// Create a CSV file with quoted fields
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "quoted.csv")

	// Write CSV content with quotes
	csvContent := `"Name","Age","City"
"John Doe",25,"New York, NY"
"Jane Smith",30,"San Francisco, CA"`
	err := os.WriteFile(filePath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create quoted CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// The CSV parser automatically removes quotes and handles commas within quoted fields
	expected := "Name, Age, City\nJohn Doe, 25, New York, NY\nJane Smith, 30, San Francisco, CA"
	if result.Text != expected {
		t.Errorf("Parse() result = %q, want %q", result.Text, expected)
	}
}

func TestParseCSVWithUnicode(t *testing.T) {
	// Create a CSV file with Unicode content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.csv")

	// Write CSV content with Unicode
	csvContent := "Name,Message\nAlice,Hello 世界! 🌍\nBob,Привет мир!"
	err := os.WriteFile(filePath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	expected := "Name, Message\nAlice, Hello 世界! 🌍\nBob, Привет мир!"
	if result.Text != expected {
		t.Errorf("Parse() result = %q, want %q", result.Text, expected)
	}
}

func TestParseCSVWithEmptyFields(t *testing.T) {
	// Create a CSV file with empty fields
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty_fields.csv")

	// Write CSV content with empty fields
	csvContent := "Name,Age,City\nJohn,,New York\n,30,San Francisco\nBob,25,"
	err := os.WriteFile(filePath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty fields CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.Parse(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	expected := "Name, Age, City\nJohn, , New York\n, 30, San Francisco\nBob, 25, "
	if result.Text != expected {
		t.Errorf("Parse() result = %q, want %q", result.Text, expected)
	}
}

func TestParseInvalidCSV(t *testing.T) {
	// Create a file with invalid CSV content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.csv")

	// Write invalid CSV content (unclosed quotes)
	csvContent := `"Name,Age,City
"John,25,New York
"Jane,30,San Francisco`
	err := os.WriteFile(filePath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid CSV test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid CSV, got nil")
	}

	if !containsError(err.Error(), "failed to parse CSV") {
		t.Errorf("Parse() error = %q, want to contain 'failed to parse CSV'", err.Error())
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.csv",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "Could not open CSV file") {
		t.Errorf("Error message = %q, want to contain 'Could not open CSV file'", err.Error())
	}
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
