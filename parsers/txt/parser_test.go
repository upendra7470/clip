package txt

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
			name:          "single line",
			content:       "Hello, World!",
			wantText:      "Hello, World!",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "multiple lines",
			content:       "Line 1\nLine 2\nLine 3",
			wantText:      "Line 1\nLine 2\nLine 3",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "unicode text",
			content:       "Hello, 世界! 🌍\nПривет, мир!",
			wantText:      "Hello, 世界! 🌍\nПривет, мир!",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "trailing newlines",
			content:       "Text with trailing newlines\n\n\n",
			wantText:      "Text with trailing newlines\n\n\n",
			wantErr:       false,
			errorContains: "",
		},
		{
			name:          "whitespace preservation",
			content:       "  leading spaces\n\tabs and tabs  \n  trailing spaces  ",
			wantText:      "  leading spaces\n\tabs and tabs  \n  trailing spaces  ",
			wantErr:       false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tempDir, "test.txt")
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
		File: "nonexistent.txt",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for missing file, got nil")
	}

	if !containsError(err.Error(), "Could not open TXT file") {
		t.Errorf("Parse() error = %q, want to contain 'Could not open TXT file'", err.Error())
	}
}

func TestParseInvalidUTF8(t *testing.T) {
	// Create a file with invalid UTF-8 content
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.txt")

	// Write invalid UTF-8 (truncated UTF-8 sequence)
	invalidUTF8 := []byte{0xFF, 0xFE} // Invalid UTF-8
	err := os.WriteFile(filePath, invalidUTF8, 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid UTF-8 test file: %v", err)
	}

	p := &Parser{}
	req := parser.ParseRequest{
		File: filePath,
	}

	_, err = p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Parse() expected error for invalid UTF-8, got nil")
	}

	if !containsError(err.Error(), "invalid UTF-8") {
		t.Errorf("Parse() error = %q, want to contain 'invalid UTF-8'", err.Error())
	}
}

func TestFileType(t *testing.T) {
	p := &Parser{}
	want := "TXT"

	if got := p.FileType(); string(got) != want {
		t.Errorf("FileType() = %q, want %q", got, want)
	}
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.txt",
	}

	_, err := p.Parse(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}

	// Check that error contains expected message
	if !containsError(err.Error(), "Could not open TXT file") {
		t.Errorf("Error message = %q, want to contain 'Could not open TXT file'", err.Error())
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

// asError converts an error to a specific error type if possible.
func asError(err error, target interface{}) bool {
	t, ok := target.(interface{ As(interface{}) bool })
	if !ok {
		return false
	}
	return t.As(err)
}
