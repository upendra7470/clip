package yaml

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
	want := "YAML"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.yaml",
	}

	_, err := p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "file nonexistent.yaml does not exist") {
		t.Errorf("Parse() error = %q, want to contain 'file nonexistent.yaml does not exist'", err.Error())
	}
}

func TestParseEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "empty.yaml")

	// Create empty file
	err := os.WriteFile(filePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty YAML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for empty file, got nil")
	}

	if !containsError(err.Error(), "empty YAML file") {
		t.Errorf("Parse() error = %q, want to contain 'empty YAML file'", err.Error())
	}
}

func TestParseInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.yaml")

	// Create file with invalid YAML
	invalidContent := []byte(`name: "Sai"
  age: 19`) // Incorrect indentation
	err := os.WriteFile(filePath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid YAML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.ParseWithContext(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid YAML, got nil")
	}

	if !containsError(err.Error(), "invalid YAML syntax") {
		t.Errorf("Parse() error = %q, want to contain 'invalid YAML syntax'", err.Error())
	}
}

func TestParseSimpleYAML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "simple.yaml")

	// Create simple YAML
	content := []byte(`name: "Sai"
age: 19
city: "Hyderabad"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create simple YAML test file: %v", err)
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
	if !strings.Contains(result.Text, "Sai") || !strings.Contains(result.Text, "19") || !strings.Contains(result.Text, "Hyderabad") {
		t.Errorf("Parse() result missing expected values: %q", result.Text)
	}

	// Check that keys are NOT present
	if strings.Contains(result.Text, "name:") || strings.Contains(result.Text, "age:") || strings.Contains(result.Text, "city:") {
		t.Errorf("Parse() result should not contain keys: %q", result.Text)
	}
}

func TestParseNestedYAML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nested.yaml")

	// Create nested YAML
	content := []byte(`name: "Sai"
age: 19
address:
  city: "Hyderabad"
  country: "India"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create nested YAML test file: %v", err)
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
	if !strings.Contains(result.Text, "Sai") || !strings.Contains(result.Text, "19") || !strings.Contains(result.Text, "Hyderabad") || !strings.Contains(result.Text, "India") {
		t.Errorf("Parse() result missing expected values: %q", result.Text)
	}

	// Check that keys are NOT present
	if strings.Contains(result.Text, "name:") || strings.Contains(result.Text, "age:") || strings.Contains(result.Text, "city:") || strings.Contains(result.Text, "country:") {
		t.Errorf("Parse() result should not contain keys: %q", result.Text)
	}
}

func TestParseYAMLArray(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "array.yaml")

	// Create YAML array
	content := []byte(`students:
  - name: "Sai"
    age: 19
  - name: "Ravi"
    age: 20`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create array YAML test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	result, err := p.ParseWithContext(context.Background(), req)

	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	// Check that array values are present (no keys)
	if !strings.Contains(result.Text, "Sai") || !strings.Contains(result.Text, "19") || !strings.Contains(result.Text, "Ravi") || !strings.Contains(result.Text, "20") {
		t.Errorf("Parse() result missing expected array values: %q", result.Text)
	}

	// Check that keys are NOT present
	if strings.Contains(result.Text, "name:") || strings.Contains(result.Text, "age:") || strings.Contains(result.Text, "students:") {
		t.Errorf("Parse() result should not contain keys: %q", result.Text)
	}
}

func TestParseYAMLWithUnicode(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unicode.yaml")

	// Create YAML with Unicode content
	content := []byte(`name: "Alice"
message: "Hello 世界! 🌍"
greeting: "Привет мир!"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode YAML test file: %v", err)
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

func TestParseYAMLWithNumbers(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "numbers.yaml")

	// Create YAML with various number types
	content := []byte(`age: 19
price: 29.99
quantity: 0
temperature: -5.5`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create numbers YAML test file: %v", err)
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
	if !strings.Contains(result.Text, "19") || !strings.Contains(result.Text, "29.99") || !strings.Contains(result.Text, "0") || !strings.Contains(result.Text, "-5.5") {
		t.Errorf("Parse() result missing expected number values: %q", result.Text)
	}

	// Check that keys are NOT present
	if strings.Contains(result.Text, "age:") || strings.Contains(result.Text, "price:") || strings.Contains(result.Text, "quantity:") || strings.Contains(result.Text, "temperature:") {
		t.Errorf("Parse() result should not contain keys: %q", result.Text)
	}
}

func TestParseYAMLWithBooleanAndNull(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "booleans.yaml")

	// Create YAML with boolean and null values
	content := []byte(`active: true
verified: false
optional: null`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create boolean YAML test file: %v", err)
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
	if !strings.Contains(result.Text, "true") || !strings.Contains(result.Text, "false") || !strings.Contains(result.Text, "null") {
		t.Errorf("Parse() result missing expected boolean/null values: %q", result.Text)
	}

	// Check that keys are NOT present
	if strings.Contains(result.Text, "active:") || strings.Contains(result.Text, "verified:") || strings.Contains(result.Text, "optional:") {
		t.Errorf("Parse() result should not contain keys: %q", result.Text)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.yaml")

	// Create YAML with multiple values
	content := []byte(`value1: "First Value"
value2: "Second Value"
value3: "Third Value"
value4: "Fourth Value"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range test YAML file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test parsing specific value range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}

	// Check that the result contains expected content from values 2-3
	if !strings.Contains(result.Text, "Second Value") || !strings.Contains(result.Text, "Third Value") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.yaml")

	// Create YAML file
	content := []byte(`value1: "First Value"
value2: "Second Value"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid YAML file: %v", err)
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

	// Test range exceeding file values
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding file values, got nil")
	}
}

// Test for open-ended range support
func TestParseRangeOpenEnded(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "open_ended.yaml")

	// Create YAML with multiple values
	content := []byte(`value1: "First Value"
value2: "Second Value"
value3: "Third Value"
value4: "Fourth Value"`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create open-ended test YAML file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test open-ended range (start from value 2 to end)
	result, err := p.ParseRange(context.Background(), req, 2, -1)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error for open-ended range: %v", err)
	}

	// Check that the result contains expected content from value 2 to end
	if !strings.Contains(result.Text, "Second Value") || !strings.Contains(result.Text, "Fourth Value") {
		t.Errorf("ParseRange() result missing expected content for open-ended range: got %q", result.Text)
	}
}

// containsError checks if a string contains a substring.
func containsError(s, substr string) bool {
	return strings.Contains(s, substr)
}
