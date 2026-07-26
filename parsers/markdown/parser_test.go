package markdown

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
			content:  "key1: value1\n\nkey2: value2\n\nkey3: value3",
			start:    1,
			end:      3,
			expected: "key1: value1\nkey2: value2\nkey3: value3",
		},
		{
			name:     "Range extraction",
			content:  "key1: value1\n\nkey2: value2\n\nkey3: value3",
			start:    2,
			end:      2,
			expected: "key2: value2",
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
	want := "Markdown"

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
	filePath := filepath.Join(tempDir, "range_test.md")

	// Create Markdown with multiple sections
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
		t.Fatalf("Failed to create range test Markdown file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	// Test parsing specific section range
	result, err := p.ParseRange(context.Background(), req, 2, 3)
	if err != nil {
		t.Fatalf("ParseRange() unexpected error: %v", err)
	}

	// Check that the result contains expected content from sections 2-3
	expectedContent := "Getting started guide here"
	if !strings.Contains(result.Text, expectedContent) {
		t.Errorf("ParseRange() result missing expected content: got %q, want to contain %q", result.Text, expectedContent)
	}
}

func TestParseRangeInvalid(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "range_invalid.md")

	// Create Markdown file
	content := []byte(`# Section 1

Content 1

# Section 2

Content 2
`)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create range invalid Markdown file: %v", err)
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

	// Test range exceeding section count
	_, err = p.ParseRange(context.Background(), req, 1, 100)
	if err == nil {
		t.Fatal("ParseRange() expected error for range exceeding section count, got nil")
	}
}

func TestExtractBlocks(t *testing.T) {
	p := &Parser{}
	content := "Block 1\n\nBlock 2\n\nBlock 3\n\nBlock 4"

	// Test extracting blocks 2-3
	result, err := p.ExtractBlocks(content, 2, 3)
	if err != nil {
		t.Fatalf("ExtractBlocks() unexpected error: %v", err)
	}

	expected := "Block 2\n\nBlock 3"
	if result != expected {
		t.Errorf("ExtractBlocks() = %q, want %q", result, expected)
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

func TestExtractSections(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Full section extraction",
			content:  "# Section 1\nContent 1\n\n# Section 2\nContent 2\n\n# Section 3\nContent 3",
			start:    1,
			end:      3,
			expected: "Content 1\nContent 2\nContent 3",
		},
		{
			name:     "Single section extraction",
			content:  "# Section 1\nContent 1\n\n# Section 2\nContent 2\n\n# Section 3\nContent 3",
			start:    2,
			end:      2,
			expected: "Content 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test file with the content
			testFile := "test_sections.md"
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			assert.NoError(t, err)
			defer os.Remove(testFile)

			mdParser := NewParser()
			// Use ParseRange to test section extraction
			req := parser.ParseRequest{
				File: testFile,
			}
			result, err := mdParser.ParseRange(context.Background(), req, tt.start, tt.end)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.Text)
		})
	}
}
