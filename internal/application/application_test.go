package application

import (
	"context"
	"errors"
	"testing"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/registry"
)

// mockParser is a test parser that can simulate different behaviors.
type mockParser struct {
	fileType filetype.FileType
	content  string
	err      error
}

func (m *mockParser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	if m.err != nil {
		return parser.ParseResult{}, m.err
	}
	return parser.ParseResult{Text: m.content}, nil
}

func (m *mockParser) FileType() filetype.FileType {
	return m.fileType
}

func TestExtractSuccess(t *testing.T) {
	// Create registry with mock parser
	reg := registry.New()
	mock := &mockParser{
		fileType: filetype.FileTypeTXT,
		content:  "test content",
		err:      nil,
	}
	if err := reg.Register(filetype.FileTypeTXT, mock); err != nil {
		t.Fatalf("Failed to register mock parser: %v", err)
	}

	app := New(reg)

	// Test successful extraction
	err := app.Extract(context.Background(), "test.txt")
	if err != nil {
		t.Errorf("Extract() unexpected error: %v", err)
	}
}

func TestExtractUnsupportedFileType(t *testing.T) {
	reg := registry.New()
	app := New(reg)

	// Test unsupported file type
	err := app.Extract(context.Background(), "file.xyz")
	if err == nil {
		t.Fatal("Extract() expected error for unsupported file type, got nil")
	}

	if !containsError(err.Error(), "unsupported file type") {
		t.Errorf("Error = %q, want to contain 'unsupported file type'", err.Error())
	}
}

func TestExtractParserNotFound(t *testing.T) {
	// Create registry but don't register any parsers
	reg := registry.New()
	app := New(reg)

	// Test parser not found (even though file type is supported)
	err := app.Extract(context.Background(), "test.txt")
	if err == nil {
		t.Fatal("Extract() expected error for missing parser, got nil")
	}

	if !containsError(err.Error(), "parser not found") {
		t.Errorf("Error = %q, want to contain 'parser not found'", err.Error())
	}
}

func TestExtractParserError(t *testing.T) {
	// Create registry with failing mock parser
	reg := registry.New()
	expectedErr := errors.New("mock parse error")
	mock := &mockParser{
		fileType: filetype.FileTypeTXT,
		content:  "",
		err:      expectedErr,
	}
	if err := reg.Register(filetype.FileTypeTXT, mock); err != nil {
		t.Fatalf("Failed to register mock parser: %v", err)
	}

	app := New(reg)

	// Test parser error
	err := app.Extract(context.Background(), "test.txt")
	if err == nil {
		t.Fatal("Extract() expected error from parser, got nil")
	}

	if !containsError(err.Error(), "failed to extract text") {
		t.Errorf("Error = %q, want to contain 'failed to extract text'", err.Error())
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
