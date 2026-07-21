package markdown

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/upendra7470/clip/internal/parser"
)

func TestParse(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		content       string
		wantText      string
		wantErr       bool
		errorContains string
	}{
		{
			name:          "empty file",
			content:       "",
			wantText:      "",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "plain text",
			content:       "Just plain text",
			wantText:      "Just plain text",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "headings",
			content:       "# Heading 1\n## Heading 2\n### Heading 3",
			wantText:      "Heading 1\nHeading 2\nHeading 3",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "bold and italic",
			content:       "This is **bold** and this is *italic*",
			wantText:      "This is bold and this is italic",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "links",
			content:       "Visit [Google](https://google.com) and [GitHub](https://github.com)",
			wantText:      "Visit Google and GitHub",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "unordered lists",
			content:       "- Item one\n- Item two\n- Item three",
			wantText:      "Item one\nItem two\nItem three",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "ordered lists",
			content:       "1. First item\n2. Second item\n3. Third item",
			wantText:      "First item\nSecond item\nThird item",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "code blocks",
			content:       "Some text\n```go\nfmt.Println(\"hello\")\n```\nMore text",
			wantText:      "Some text\n\nMore text",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "mixed markdown",
			content:       "# Title\n\nThis is **important** text.\n\n- Feature one\n- Feature two\n\n[Learn more](https://example.com)",
			wantText:      "Title\n\nThis is important text.\n\nFeature one\nFeature two\n\nLearn more",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "unicode content",
			content:       "# Hello 世界 🌍\n\nThis is **bold 文字** and *italic 文字*.",
			wantText:      "Hello 世界 🌍\n\nThis is bold 文字 and italic 文字.",
			wantErr:       false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tempDir, "test.md")
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test the parser
			p := &Parser{}
			req := parser.ParseRequest{
				File: filePath,
				Selection: parser.Selection{
					Pages: "1-2",       // Should be ignored
					Range: "1:10-2:20", // Should be ignored
					Query: "test",      // Should be ignored
				},
			}

			result, err := p.Parse(context.Background(), req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				} else if tt.errorContains != "" && !containsError(err.Error(), tt.errorContains) {
					t.Errorf("Parse() error = %q, want to contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("Parse() unexpected error: %v", err)
				}
				if result.Text != tt.wantText {
					t.Errorf("Parse() result = %q, want %q", result.Text, tt.wantText)
				}
			}
		})
	}
}

func TestParseMissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.md",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "Could not open Markdown file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open Markdown file'", err.Error())
	}
}

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "Markdown"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.md",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "Could not open Markdown file") {
		t.Errorf("Error message = %q, want to contain 'Could not open Markdown file'", err.Error())
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
