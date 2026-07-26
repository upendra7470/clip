package xml

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
			content:  "<element>Item 1</element><element>Item 2</element><element>Item 3</element>",
			start:    1,
			end:      3,
			expected: "<element>Item 1</element><element>Item 2</element><element>Item 3</element>",
		},
		{
			name:     "Structured range extraction",
			content:  "<element>Item 1</element><element>Item 2</element><element>Item 3</element>",
			start:    2,
			end:      2,
			expected: "<element>Item 2</element>",
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

func TestParseRangeWithElements(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Full element extraction",
			content:  "<root><element>Element 1</element><element>Element 2</element></root>",
			start:    1,
			end:      2,
			expected: "Element 1\nElement 2",
		},
		{
			name:     "Single element extraction",
			content:  "<root><element>Element 1</element><element>Element 2</element></root>",
			start:    1,
			end:      1,
			expected: "Element 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test.xml")
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test XML file: %v", err)
			}

			p := NewParser()
			req := parser.ParseRequest{
				File: filePath,
			}
			result, err := p.ParseRange(context.Background(), req, tt.start, tt.end)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.Text)
		})
	}
}

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "XML"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestGetRangeUnit(t *testing.T) {
	p := &Parser{}
	want := parser.RangeUnit("elements")

	if got := p.GetRangeUnit(); got != want {
		t.Errorf("GetRangeUnit() = %q, want %q", got, want)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.xml")

	// Create XML with multiple text blocks
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <item>First item content</item>
  <item>Second item content</item>
  <item>Third item content</item>
  <section>
    <title>Section Title</title>
    <description>Section description</description>
  </section>
  <data>Final data element</data>
</root>
`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range test XML file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test parsing specific text block range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}

	// Check that the result contains expected content from blocks 2-3
	expectedContent := "Second item content"
	if !strings.Contains(result.Text, expectedContent) {
		t.Errorf("ParseRange() result missing expected content: got %q, want to contain %q", result.Text, expectedContent)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.xml")

	// Create XML file
	content := []byte(`<?xml version="1.0"?><root><item>Block 1</item><item>Block 2</item></root>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid XML file: %v", err)
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

	// Test range exceeding block count
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding block count, got nil")
	}
}

func TestExtractTextFromXML(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected string
	}{
		{
			name:     "Simple XML",
			xml:      `<?xml version="1.0"?><root>Hello World</root>`,
			expected: "Hello World",
		},
		{
			name:     "Multiple elements",
			xml:      `<?xml version="1.0"?><root><item>First</item><item>Second</item></root>`,
			expected: "First\nSecond",
		},
		{
			name:     "Nested elements",
			xml:      `<?xml version="1.0"?><root><parent><child>Nested content</child></parent></root>`,
			expected: "Nested content",
		},
		{
			name:     "Mixed content",
			xml:      `<?xml version="1.0"?><root>Text before<child>Child text</child>Text after</root>`,
			expected: "Text before\nChild text\nText after",
		},
		{
			name:     "Attributes ignored",
			xml:      `<?xml version="1.0"?><root attr="value"><item>Content</item></root>`,
			expected: "Content",
		},
		{
			name:     "Comments ignored",
			xml:      `<?xml version="1.0"?><root><!-- comment --><item>Content</item></root>`,
			expected: "Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTextFromXML([]byte(tt.xml))
			assert.NoError(t, err)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestExtractTextFromXMLWithBlocks(t *testing.T) {
	tests := []struct {
		name           string
		xml            string
		expectedText   string
		expectedBlocks int
	}{
		{
			name:           "Simple XML with blocks",
			xml:            `<?xml version="1.0"?><root>Hello World</root>`,
			expectedText:   "Hello World",
			expectedBlocks: 1,
		},
		{
			name:           "Multiple elements create multiple blocks",
			xml:            `<?xml version="1.0"?><root><item>First</item><item>Second</item></root>`,
			expectedText:   "First\nSecond",
			expectedBlocks: 2,
		},
		{
			name:           "Nested elements",
			xml:            `<?xml version="1.0"?><root><parent><child>Nested</child></parent></root>`,
			expectedText:   "Nested",
			expectedBlocks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, blockCount, err := extractTextFromXMLWithBlocks([]byte(tt.xml))
			assert.NoError(t, err)
			assert.Contains(t, result, tt.expectedText)
			assert.Equal(t, tt.expectedBlocks, blockCount)
		})
	}
}

func TestExtractStructuredInvalid(t *testing.T) {
	p := &Parser{}
	content := "<element>Item 1</element><element>Item 2</element>"

	// Test invalid range
	_, err := p.ExtractStructured(content, 3, 2)
	if err == nil {
		t.Fatal("ExtractStructured() expected error for invalid range, got nil")
	}

	// Test range exceeding element count
	_, err = p.ExtractStructured(content, 1, 10)
	if err == nil {
		t.Fatal("ExtractStructured() expected error for range exceeding element count, got nil")
	}
}
