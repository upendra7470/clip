package json

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
	want := "JSON"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.json",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "Could not open JSON file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open JSON file'", err.Error())
	}
}

func TestParseEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.json")

	// Create empty file
	err := os.WriteFile(filePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty file, got nil")
	}

	if !containsError(err.Error(), "empty JSON file") {
		t.Errorf("Parse() error = %q, want to contain 'empty JSON file'", err.Error())
	}
}

func TestParseInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.json")

	// Create file with invalid JSON
	invalidContent := []byte(`{ "name": "Sai", "age": 19, }`) // Trailing comma
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid JSON, got nil")
	}

	if !containsError(err.Error(), "invalid JSON syntax") {
		t.Errorf("Parse() error = %q, want to contain 'invalid JSON syntax'", err.Error())
	}
}

func TestParseMalformedJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "malformed.json")

	// Create file with malformed JSON
	malformedContent := []byte(`{ "name": "Sai" "age": 19 }`) // Missing comma
	err := os.WriteFile(filePath, malformedContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create malformed JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for malformed JSON, got nil")
	}

	if !containsError(err.Error(), "invalid JSON syntax") {
		t.Errorf("Parse() error = %q, want to contain 'invalid JSON syntax'", err.Error())
	}
}

func TestParseSimpleObject(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.json")

	// Create simple JSON object
	content := []byte(`{
  "name": "Sai",
  "age": 19,
  "city": "Hyderabad"
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that values are present (no keys)
	expectedValues := []string{"Sai", "19", "Hyderabad"}
	for _, value := range expectedValues {
		if !strings.Contains(result.Text, value) {
			t.Errorf("Parse() result missing expected value %q: %q", value, result.Text)
		}
	}

	// Check that keys are NOT present
	unexpectedKeys := []string{"name:", "age:", "city:"}
	for _, key := range unexpectedKeys {
		if strings.Contains(result.Text, key) {
			t.Errorf("Parse() result should not contain keys %q: %q", key, result.Text)
		}
	}
}

func TestParseNestedObject(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nested.json")

	// Create nested JSON object
	content := []byte(`{
  "name": "Sai",
  "age": 19,
  "address": {
    "city": "Hyderabad",
    "country": "India"
  }
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create nested JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that values are present (no keys)
	expectedValues := []string{"Sai", "19", "Hyderabad", "India"}
	for _, value := range expectedValues {
		if !strings.Contains(result.Text, value) {
			t.Errorf("Parse() result missing expected value %q: %q", value, result.Text)
		}
	}

	// Check that keys are NOT present
	unexpectedKeys := []string{"name:", "age:", "city:", "country:"}
	for _, key := range unexpectedKeys {
		if strings.Contains(result.Text, key) {
			t.Errorf("Parse() result should not contain keys %q: %q", key, result.Text)
		}
	}
}

func TestParseArray(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "array.json")

	// Create JSON array
	content := []byte(`[
  {
    "name": "Sai"
  },
  {
    "name": "Ravi"
  }
]`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create array JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that both names are present (values only, no keys)
	if !strings.Contains(result.Text, "Sai") {
		t.Errorf("Parse() result missing 'Sai': %q", result.Text)
	}
	if !strings.Contains(result.Text, "Ravi") {
		t.Errorf("Parse() result missing 'Ravi': %q", result.Text)
	}

	// Check that keys are NOT present
	if strings.Contains(result.Text, "name:") {
		t.Errorf("Parse() result should not contain keys: %q", result.Text)
	}
}

func TestParseUnicodeContent(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.json")

	// Create JSON with Unicode content
	content := []byte(`{
  "name": "Alice",
  "message": "Hello 世界! 🌍",
  "greeting": "Привет мир!"
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode JSON test file: %v", err)
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

func TestParseComplexUnicode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "complex_unicode.json")

	// Create JSON with complex Unicode content
	content := []byte(`{
  "title": "Complex Unicode Test",
  "description": "Testing various Unicode characters: \u0041\u030A (Å), \u03A9 (Ω), \uD83D\uDE00 (😀), \uD83D\uDC4D (👍)"
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create complex Unicode JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that complex Unicode content is preserved
	expectedContent := "Testing various Unicode characters: Å (Å), Ω (Ω), 😀 (😀), 👍 (👍)"
	if !strings.Contains(result.Text, expectedContent) {
		t.Errorf("Parse() result missing complex Unicode content: %q", result.Text)
	}
}

func TestParseBooleanAndNull(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "booleans.json")

	// Create JSON with boolean and null values
	content := []byte(`{
  "active": true,
  "verified": false,
  "optional": null
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create boolean JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that boolean and null values are present (values only, no keys)
	if !strings.Contains(result.Text, "true") {
		t.Errorf("Parse() result missing 'true': %q", result.Text)
	}
	if !strings.Contains(result.Text, "false") {
		t.Errorf("Parse() result missing 'false': %q", result.Text)
	}
	if !strings.Contains(result.Text, "null") {
		t.Errorf("Parse() result missing 'null': %q", result.Text)
	}

	// Check that keys are NOT present
	unexpectedKeys := []string{"active:", "verified:", "optional:"}
	for _, key := range unexpectedKeys {
		if strings.Contains(result.Text, key) {
			t.Errorf("Parse() result should not contain keys %q: %q", key, result.Text)
		}
	}
}

func TestParseEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "edge_cases.json")

	// Create JSON with edge cases
	content := []byte(`{
  "empty_string": "",
  "whitespace_string": "   ",
  "number": 0,
  "negative_number": -0,
  "large_number": 99999999999999999999999999999999999999999999999999999999999999999999999999999999999999999,
  "negative_large_number": -99999999999999999999999999999999999999999999999999999999999999999999999999999999999999999
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create edge cases JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that edge cases are handled correctly
	if !strings.Contains(result.Text, "") {
		t.Errorf("Parse() result missing empty string: %q", result.Text)
	}
	if !strings.Contains(result.Text, "   ") {
		t.Errorf("Parse() result missing whitespace string: %q", result.Text)
	}
	if !strings.Contains(result.Text, "0") {
		t.Errorf("Parse() result missing number: %q", result.Text)
	}
	if !strings.Contains(result.Text, "-0") {
		t.Errorf("Parse() result missing negative number: %q", result.Text)
	}
	if !strings.Contains(result.Text, "99999999999999999999999999999999999999999999999999999999999999999999999999999999999999999") {
		t.Errorf("Parse() result missing large number: %q", result.Text)
	}
	if !strings.Contains(result.Text, "-99999999999999999999999999999999999999999999999999999999999999999999999999999999999999999") {
		t.Errorf("Parse() result missing negative large number: %q", result.Text)
	}
}

func TestParseNumbers(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "numbers.json")

	// Create JSON with various number types
	content := []byte(`{
  "age": 19,
  "price": 29.99,
  "quantity": 0,
  "temperature": -5.5
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create numbers JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that numbers are present (values only, no keys)
	if !strings.Contains(result.Text, "19") {
		t.Errorf("Parse() result missing '19': %q", result.Text)
	}
	if !strings.Contains(result.Text, "29.99") {
		t.Errorf("Parse() result missing '29.99': %q", result.Text)
	}
	if !strings.Contains(result.Text, "0") {
		t.Errorf("Parse() result missing '0': %q", result.Text)
	}
	if !strings.Contains(result.Text, "-5.5") {
		t.Errorf("Parse() result missing '-5.5': %q", result.Text)
	}

	// Check that keys are NOT present
	unexpectedKeys := []string{"age:", "price:", "quantity:", "temperature:"}
	for _, key := range unexpectedKeys {
		if strings.Contains(result.Text, key) {
			t.Errorf("Parse() result should not contain keys %q: %q", key, result.Text)
		}
	}
}

func TestParseScientificNotation(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "scientific.json")

	// Create JSON with scientific notation
	content := []byte(`{
  "small": 1.23e-5,
  "large": 9.87e+10,
  "negative": -3.14e-2
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create scientific notation JSON test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that scientific notation values are present (values only, no keys)
	if !strings.Contains(result.Text, "1.23e-5") {
		t.Errorf("Parse() result missing '1.23e-5': %q", result.Text)
	}
	if !strings.Contains(result.Text, "9.87e+10") {
		t.Errorf("Parse() result missing '9.87e+10': %q", result.Text)
	}
	if !strings.Contains(result.Text, "-3.14e-2") {
		t.Errorf("Parse() result missing '-3.14e-2': %q", result.Text)
	}

	// Check that keys are NOT present
	unexpectedKeys := []string{"small:", "large:", "negative:"}
	for _, key := range unexpectedKeys {
		if strings.Contains(result.Text, key) {
			t.Errorf("Parse() result should not contain keys %q: %q", key, result.Text)
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.json",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "Could not open JSON file") {
		t.Errorf("Error message = %q, want to contain 'Could not open JSON file'", err.Error())
	}
}

func TestGetRangeUnit(t *testing.T) {
	p := &Parser{}
	want := parser.RangeUnit("entries")

	if got := p.GetRangeUnit(); got != want {
		t.Errorf("GetRangeUnit() = %q, want %q", got, want)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.json")

	// Create JSON with multiple entries
	content := []byte(`{
  "name": "Sai",
  "age": 19,
  "city": "Hyderabad",
  "country": "India"
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range test JSON file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test parsing specific line range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}

	// Check that the result contains expected content from lines 2-3
	expectedContent := `"age": 19,`
	if !strings.Contains(result.Text, expectedContent) {
		t.Errorf("ParseRange() result missing expected content: got %q, want to contain %q", result.Text, expectedContent)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.json")

	// Create JSON file
	content := []byte(`{
  "name": "Sai",
  "age": 19
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid JSON file: %v", err)
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

	// Test range exceeding file lines
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding file lines, got nil")
	}
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
