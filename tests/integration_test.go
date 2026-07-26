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
	if p.GetRangeUnit() != parser.RangeUnitEntries {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), parser.RangeUnitEntries)
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
	if p.GetRangeUnit() != parser.RangeUnitValues {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), parser.RangeUnitValues)
	}

	// Test ParseRange with valid range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "age: 30") {
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
	if !strings.Contains(result.Text, "Getting started guide here.") {
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
	if p.GetRangeUnit() != parser.RangeUnitSections {
		t.Errorf("GetRangeUnit() = %q, want %q", p.GetRangeUnit(), parser.RangeUnitSections)
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
			content:   "{\n  \"name\": \"John\",\n  \"age\": 30,\n  \"city\": \"NYC\"\n}",
			parser:    json.NewParser(),
			rangeUnit: string(parser.RangeUnitEntries),
		},
		{
			name:      "YAML",
			ext:       ".yaml",
			content:   "name: John\nage: 30\ncity: NYC",
			parser:    yaml.NewParser(),
			rangeUnit: string(parser.RangeUnitValues),
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
			content:   "<html><body><section><p>Block 1</p></section><section><p>Block 2</p></section></body></html>",
			parser:    html.NewParser(),
			rangeUnit: string(parser.RangeUnitSections),
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
			if string(tt.parser.GetRangeUnit()) != tt.rangeUnit {
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
