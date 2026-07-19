package detect

import (
	"testing"

	"github.com/upendra7470/clip/internal/filetype"
)

func TestType(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantType      filetype.FileType
		wantErr       bool
		errorContains string
	}{
		// PDF files
		{"PDF lowercase", "document.pdf", filetype.FileTypePDF, false, ""},
		{"PDF uppercase", "DOCUMENT.PDF", filetype.FileTypePDF, false, ""},
		{"PDF mixed case", "Document.Pdf", filetype.FileTypePDF, false, ""},

		// DOCX files
		{"DOCX", "report.docx", filetype.FileTypeDOCX, false, ""},
		{"DOC", "legacy.doc", filetype.FileTypeDOC, false, ""},

		// Text files
		{"TXT", "notes.txt", filetype.FileTypeTXT, false, ""},
		{"Markdown MD", "readme.md", filetype.FileTypeMarkdown, false, ""},
		{"Markdown MARKDOWN", "guide.markdown", filetype.FileTypeMarkdown, false, ""},

		// Office files
		{"XLSX", "data.xlsx", filetype.FileTypeXLSX, false, ""},
		{"PPTX", "presentation.pptx", filetype.FileTypePPTX, false, ""},

		// Web files
		{"HTML", "index.html", filetype.FileTypeHTML, false, ""},
		{"HTM", "page.htm", filetype.FileTypeHTML, false, ""},
		{"JSON", "data.json", filetype.FileTypeJSON, false, ""},
		{"XML", "config.xml", filetype.FileTypeXML, false, ""},

		// Error cases
		{"No extension", "document", filetype.FileType(""), true, "no file extension found"},
		{"Unsupported extension", "file.xyz", filetype.FileType(""), true, "unsupported file extension: .xyz"},
		{"Empty string", "", filetype.FileType(""), true, "no file extension found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Type(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Type(%q) expected error, got nil", tt.path)
				} else if tt.errorContains != "" && !containsError(err.Error(), tt.errorContains) {
					t.Errorf("Type(%q) error = %q, want to contain %q", tt.path, err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("Type(%q) unexpected error: %v", tt.path, err)
				}
				if got != tt.wantType {
					t.Errorf("Type(%q) = %q, want %q", tt.path, got, tt.wantType)
				}
			}
		})
	}
}

func TestTypeCaseInsensitive(t *testing.T) {
	// Test that detection is case-insensitive
	lowercase := "file.pdf"
	uppercase := "FILE.PDF"
	mixed := "File.PdF"

	t1, err1 := Type(lowercase)
	t2, err2 := Type(uppercase)
	t3, err3 := Type(mixed)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("Unexpected errors: %v, %v, %v", err1, err2, err3)
	}

	if t1 != t2 || t2 != t3 {
		t.Errorf("Case-insensitive detection failed: %q, %q, %q", t1, t2, t3)
	}
}

func containsError(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
