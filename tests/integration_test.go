package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/parsers/html"
	"github.com/upendra7470/clip/parsers/json"
	"github.com/upendra7470/clip/parsers/markdown"
	"github.com/upendra7470/clip/parsers/xml"
	"github.com/upendra7470/clip/parsers/yaml"
)

// TestIntegrationJSON tests JSON parser integration with semantic range units
func TestIntegrationJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.json")

	content := []byte(`{
  "name": "John Doe",
  "age": 30,
  "city": "New York",
  "country": "USA",
  "skills": ["Go", "Python", "JavaScript"]
}`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	p := json.NewParser()
	req := parser.ParseRequest{File: filePath}

	// Test GetRangeUnit
	if p.GetRangeUnit() != string(parser.Entries) {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), parser.Entries)
	}

	// Test ParseRange with valid range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, `"age": 30`) {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}

	// Test ParseRange with invalid range
	_, err = p.ParseRange(context.Background(), req, 3, 2)
	if err == nil {
		t.Fatal("ParseRange() expected error for invalid range, got nil")
	}

	// Test ParseRange exceeding range
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding file lines, got nil")
	}

	// Test ExtractStructured
	structured, err := p.ExtractStructured(string(content), 1, 2)
	if err != nil {
		t.Fatalf("ExtractStructured() unexpected error: %v", err)
	}
	if !strings.Contains(structured, `"name": "John Doe"`) {
		t.Errorf("ExtractStructured() result missing expected content: got %q", structured)
	}
}

// TestIntegrationYAML tests YAML parser integration with semantic range units
func TestIntegrationYAML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.yaml")

	content := []byte(`name: John Doe
age: 30
city: New York
country: USA
skills:
  - Go
  - Python
  - JavaScript`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	p := yaml.NewParser()
	req := parser.ParseRequest{File: filePath}

	// Test GetRangeUnit
	if p.GetRangeUnit() != string(parser.Values) {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), parser.Values)
	}

	// Test ParseRange with valid range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "30") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}

	// Test ParseRange with invalid range
	_, err = p.ParseRange(context.Background(), req, 3, 2)
	if err == nil {
		t.Fatal("ParseRange() expected error for invalid range, got nil")
	}

	// Test ParseRange exceeding range
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding value count, got nil")
	}

	// Test ExtractStructured
	structured, err := p.ExtractStructured(string(content), 1, 2)
	if err != nil {
		t.Fatalf("ExtractStructured() unexpected error: %v", err)
	}
	if !strings.Contains(structured, "name: John Doe") {
		t.Errorf("ExtractStructured() result missing expected content: got %q", structured)
	}
}

// TestIntegrationMarkdown tests Markdown parser integration with semantic range units
func TestIntegrationMarkdown(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.md")

	content := []byte(`# Introduction

This is the introduction section.

## Getting Started

Getting started guide here.

# Features

List of features.

## Feature 1

Details about feature 1.

## Feature 2

Details about feature 2.
`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test Markdown file: %v", err)
	}

	p := markdown.NewParser()
	req := parser.ParseRequest{File: filePath}

	// Test GetRangeUnit
	if p.GetRangeUnit() != "sections" {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), "sections")
	}

	// Test ParseRange with valid range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "Getting Started") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}

	// Test ParseRange with invalid range
	_, err = p.ParseRange(context.Background(), req, 3, 2)
	if err == nil {
		t.Fatal("ParseRange() expected error for invalid range, got nil")
	}

	// Test ParseRange exceeding range
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding section count, got nil")
	}

	// Test ExtractBlocks
	blocks, err := p.ExtractBlocks(string(content), 2, 3)
	if err != nil {
		t.Fatalf("ExtractBlocks() unexpected error: %v", err)
	}
	if !strings.Contains(blocks, "Getting Started") {
		t.Errorf("ExtractBlocks() result missing expected content: got %q", blocks)
	}
}

// TestIntegrationHTML tests HTML parser integration with semantic range units
func TestIntegrationHTML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.html")

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
		t.Fatalf("Failed to create test HTML file: %v", err)
	}

	p := html.NewParser()
	req := parser.ParseRequest{File: filePath}

	// Test GetRangeUnit
	if p.GetRangeUnit() != string(parser.Sections) {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), parser.Sections)
	}

	// Test ParseRange with valid range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "Second paragraph") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}

	// Test ParseRange with invalid range
	_, err = p.ParseRange(context.Background(), req, 3, 2)
	if err == nil {
		t.Fatal("ParseRange() expected error for invalid range, got nil")
	}

	// Test ParseRange exceeding range
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding block count, got nil")
	}

	// Test ExtractBlocks
	blocks, err := p.ExtractBlocks(string(content), 2, 3)
	if err != nil {
		t.Fatalf("ExtractBlocks() unexpected error: %v", err)
	}
	if !strings.Contains(blocks, "Second paragraph") {
		t.Errorf("ExtractBlocks() result missing expected content: got %q", blocks)
	}
}

// TestIntegrationXML tests XML parser integration with semantic range units
func TestIntegrationXML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.xml")

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
		t.Fatalf("Failed to create test XML file: %v", err)
	}

	p := xml.NewParser()
	req := parser.ParseRequest{File: filePath}

	// Test GetRangeUnit
	if p.GetRangeUnit() != "elements" {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), "elements")
	}

	// Test ParseRange with valid range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "Second item content") {
		t.Errorf("ParseRange() result missing expected content: got %q", result.Text)
	}

	// Test ParseRange with invalid range
	_, err = p.ParseRange(context.Background(), req, 3, 2)
	if err == nil {
		t.Fatal("ParseRange() expected error for invalid range, got nil")
	}

	// Test ParseRange exceeding range
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding block count, got nil")
	}

	// Test ExtractStructured
	structured, err := p.ExtractStructured(string(content), 1, 2)
	if err != nil {
		t.Fatalf("ExtractStructured() unexpected error: %v", err)
	}
	if !strings.Contains(structured, "First item content") {
		t.Errorf("ExtractStructured() result missing expected content: got %q", structured)
	}
}

// TestIntegrationAllFormats tests all formats together
func TestIntegrationAllFormats(t *testing.T) {
	tests := []struct {
		name      string
		ext       string
		content   string
		parser    parser.RangeParser
		rangeUnit string
	}{
		{
			name:      "JSON",
			ext:       ".json",
			content:   `{"name": "John", "age": 30, "city": "NYC"}`,
			parser:    json.NewParser(),
			rangeUnit: string(parser.Entries),
		},
		{
			name:      "YAML",
			ext:       ".yaml",
			content:   "name: John\nage: 30\ncity: NYC",
			parser:    yaml.NewParser(),
			rangeUnit: string(parser.Values),
		},
		{
			name:      "Markdown",
			ext:       ".md",
			content:   "# Section 1\nContent 1\n\n# Section 2\nContent 2",
			parser:    markdown.NewParser(),
			rangeUnit: "sections",
		},
		{
			name:      "HTML",
			ext:       ".html",
			content:   "<html><body><p>Block 1</p><p>Block 2</p></body></html>",
			parser:    html.NewParser(),
			rangeUnit: string(parser.Sections),
		},
		{
			name:      "XML",
			ext:       ".xml",
			content:   `<?xml version="1.0"?><root><item>Item 1</item><item>Item 2</item></root>`,
			parser:    xml.NewParser(),
			rangeUnit: "elements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test"+tt.ext)

			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			req := parser.ParseRequest{File: filePath}

			// Test GetRangeUnit
			if tt.parser.GetRangeUnit() != tt.rangeUnit {
				t.Errorf("GetRangeUnit() = %q, want %q", tt.parser.GetRangeUnit(), tt.rangeUnit)
			}

			// Test ParseRange with valid range
			result, err := tt.parser.ParseRange(context.Background(), req, 1, 2)
			if err != nil {
				t.Fatalf("ParseRange() unexpected error: %v", err)
			}
			if result.Text == "" {
				t.Error("ParseRange() returned empty result")
			}

			// Test ParseRange with invalid range
			_, err = tt.parser.ParseRange(context.Background(), req, 3, 2)
			if err == nil {
				t.Fatal("ParseRange() expected error for invalid range, got nil")
			}
		})
	}
}

// TestIntegrationExtractStructured tests ExtractStructured across all formats
func TestIntegrationExtractStructured(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		parser   parser.RangeParser
		start    int
		end      int
		expected string
	}{
		{
			name:     "JSON",
			content:  `{"key1": "value1", "key2": "value2", "key3": "value3"}`,
			parser:   json.NewParser(),
			start:    1,
			end:      2,
			expected: `"key1": "value1"`,
		},
		{
			name:     "YAML",
			content:  "key1: value1\nkey2: value2\nkey3: value3",
			parser:   yaml.NewParser(),
			start:    1,
			end:      2,
			expected: "key1: value1",
		},
		{
			name:     "XML",
			content:  "<element>Item 1</element><element>Item 2</element><element>Item 3</element>",
			parser:   xml.NewParser(),
			start:    1,
			end:      2,
			expected: "<element>Item 1</element>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.parser.ExtractStructured(tt.content, tt.start, tt.end)
			if err != nil {
				t.Fatalf("ExtractStructured() unexpected error: %v", err)
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("ExtractStructured() result missing expected content: got %q, want to contain %q", result, tt.expected)
			}
		})
	}
}

// TestIntegrationExtractBlocks tests ExtractBlocks across formats that support it
func TestIntegrationExtractBlocks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		parser   parser.RangeParser
		start    int
		end      int
		expected string
	}{
		{
			name:     "Markdown",
			content:  "Block 1\n\nBlock 2\n\nBlock 3\n\nBlock 4",
			parser:   markdown.NewParser(),
			start:    2,
			end:      3,
			expected: "Block 2\n\nBlock 3",
		},
		{
			name:     "HTML",
			content:  "Block 1\n\nBlock 2\n\nBlock 3",
			parser:   html.NewParser(),
			start:    2,
			end:      2,
			expected: "Block 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.parser.ExtractBlocks(tt.content, tt.start, tt.end)
			if err != nil {
				t.Fatalf("ExtractBlocks() unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ExtractBlocks() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestIntegrationErrorHandling tests error handling across all formats
func TestIntegrationErrorHandling(t *testing.T) {
	tests := []struct {
		name   string
		parser parser.RangeParser
	}{
		{"JSON", json.NewParser()},
		{"YAML", yaml.NewParser()},
		{"Markdown", markdown.NewParser()},
		{"HTML", html.NewParser()},
		{"XML", xml.NewParser()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "nonexistent.txt")

			req := parser.ParseRequest{File: filePath}

			// Test file not found error
			_, err := tt.parser.ParseRange(context.Background(), req, 1, 2)
			if err == nil {
				t.Fatal("ParseRange() expected error for nonexistent file, got nil")
			}

			// Test invalid range (start > end)
			// Create a valid file first
			validFile := filepath.Join(tempDir, "valid.txt")
			err = os.WriteFile(validFile, []byte("content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create valid file: %v", err)
			}

			req.File = validFile
			_, err = tt.parser.ParseRange(context.Background(), req, 3, 2)
			if err == nil {
				t.Fatal("ParseRange() expected error for invalid range (start > end), got nil")
			}
		})
	}
}

// TestIntegrationEmptyFiles tests handling of empty files across all formats
func TestIntegrationEmptyFiles(t *testing.T) {
	tests := []struct {
		name   string
		ext    string
		parser parser.RangeParser
	}{
		{"JSON", ".json", json.NewParser()},
		{"YAML", ".yaml", yaml.NewParser()},
		{"Markdown", ".md", markdown.NewParser()},
		{"HTML", ".html", html.NewParser()},
		{"XML", ".xml", xml.NewParser()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "empty"+tt.ext)

			err := os.WriteFile(filePath, []byte(""), 0644)
			if err != nil {
				t.Fatalf("Failed to create empty file: %v", err)
			}

			req := parser.ParseRequest{File: filePath}
			_, err = tt.parser.ParseRange(context.Background(), req, 1, 1)
			if err == nil {
				t.Fatal("ParseRange() expected error for empty file, got nil")
			}
		})
	}
}
