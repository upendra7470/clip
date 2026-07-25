package html

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upendra7470/clip/internal/parser"
)

func TestExtractBlocks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Full extraction",
			content:  "Block 1\n\nBlock 2\n\nBlock 3",
			start:    1,
			end:      3,
			expected: "Block 1\n\nBlock 2\n\nBlock 3",
		},
		{
			name:     "Block range extraction",
			content:  "Block 1\n\nBlock 2\n\nBlock 3",
			start:    2,
			end:      2,
			expected: "Block 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			result, err := parser.ExtractBlocks(tt.content, tt.start, tt.end)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRangeWithSections(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Full section extraction",
			content:  "<html><body><section>Section 1</section><section>Section 2</section></body></html>",
			start:    1,
			end:      2,
			expected: "Section 1\n\nSection 2",
		},
		{
			name:     "Single section extraction",
			content:  "<html><body><section>Section 1</section><section>Section 2</section></body></html>",
			start:    1,
			end:      1,
			expected: "Section 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			req := parser.ParseRequest{
				File: tt.content,
			}
			result, err := parser.ParseRange(context.Background(), req, tt.start, tt.end)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.Text)
		})
	}
}

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "HTML"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestGetRangeUnit(t *testing.T) {
	p := &Parser{}
	want := "sections"

	if got := p.GetRangeUnit(); got != want {
		t.Errorf("GetRangeUnit() = %q, want %q", got, want)
	}
}

func TestParseRange(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_test.html")

	// Create HTML with multiple text blocks
	content := []byte(`<html>
<body>
<h1>Title</h1>
<p>First paragraph with some content.</p>
<p>Second paragraph with more content.</p>
<div>
  <p>Third paragraph inside div.</p>
</div>
<h2>Subtitle</h2>
<p>Fourth paragraph after subtitle.</p>
</body>
</html>
`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range test HTML file: %v", err)
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
	expectedContent := "Second paragraph"
	if !strings.Contains(result.Text, expectedContent) {
		t.Errorf("ParseRange() result missing expected content: got %q, want to contain %q", result.Text, expectedContent)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.html")

	// Create HTML file
	content := []byte(`<html><body><p>Block 1</p><p>Block 2</p></body></html>`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid HTML file: %v", err)
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

func TestExtractTextFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Simple HTML",
			html:     "<html><body><p>Hello World</p></body></html>",
			expected: "Hello World",
		},
		{
			name:     "Multiple paragraphs",
			html:     "<html><body><p>First</p><p>Second</p></body></html>",
			expected: "First\n\nSecond",
		},
		{
			name:     "Headings",
			html:     "<html><body><h1>Title</h1><h2>Subtitle</h2></body></html>",
			expected: "Title\n\nSubtitle",
		},
		{
			name:     "Script and style tags removed",
			html:     "<html><body><script>alert('test')</script><p>Content</p><style>body{color:red}</style></body></html>",
			expected: "Content",
		},
		{
			name:     "Comments removed",
			html:     "<html><body><!-- comment --><p>Content</p></body></html>",
			expected: "Content",
		},
		{
			name:     "Nested elements",
			html:     "<html><body><div><p>Nested content</p></div></body></html>",
			expected: "Nested content",
		},
		{
			name:     "List items",
			html:     "<html><body><ul><li>Item 1</li><li>Item 2</li></ul></body></html>",
			expected: "Item 1\n\nItem 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTextFromHTML(tt.html)
			assert.NoError(t, err)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestExtractBlocksInvalid(t *testing.T) {
	p := &Parser{}
	content := "Block 1\n\nBlock 2"

	// Test invalid range
	_, err := p.ExtractBlocks(content, 3, 2)
	if err == nil {
		t.Fatal("ExtractBlocks() expected error for invalid range, got nil")
	}

	// Test range exceeding block count
	_, err = p.ExtractBlocks(content, 1, 10)
	if err == nil {
		t.Fatal("ExtractBlocks() expected error for range exceeding block count, got nil")
	}
}
