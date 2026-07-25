package yaml

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upendra7470/clip/internal/parser"
)

func TestExtractStructured(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Full extraction",
			content:  "key1: value1\nkey2: value2\nkey3: value3",
			start:    1,
			end:      3,
			expected: "key1: value1\nkey2: value2\nkey3: value3",
		},
		{
			name:     "Structured range extraction",
			content:  "key1: value1\nkey2: value2\nkey3: value3",
			start:    2,
			end:      2,
			expected: "key2: value2",
		},
		{
			name:     "Nested YAML extraction",
			content:  "key1:\n  nested1: value1\n  nested2: value2\nkey2: value3",
			start:    1,
			end:      2,
			expected: "nested1: value1\nnested2: value2",
		},
		{
			name:     "Array extraction",
			content:  "items:\n  - item1\n  - item2\n  - item3",
			start:    1,
			end:      2,
			expected: "item1\nitem2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			result, err := parser.ExtractStructured(tt.content, tt.start, tt.end)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "YAML"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestGetRangeUnit(t *testing.T) {
	p := &Parser{}
	want := string(parser.Values)

	if got := p.GetRangeUnit(); got != want {
		t.Errorf("GetRangeUnit() = %q, want %q", got, want)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.yaml")

	// Create YAML with multiple values
	content := []byte(`name: Sai
age: 19
city: Hyderabad
country: India`)
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
	expectedContent := "19"
	if !strings.Contains(result.Text, expectedContent) {
		t.Errorf("ParseRange() result missing expected content: got %q, want to contain %q", result.Text, expectedContent)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.yaml")

	// Create YAML file
	content := []byte(`name: Sai
age: 19`)
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

	// Test range exceeding value count
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding value count, got nil")
	}
}
