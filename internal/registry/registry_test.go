package registry

import (
	"context"
	"testing"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// mockParser is a minimal parser implementation for testing.
type mockParser struct {
	fileType filetype.FileType
}

func (m *mockParser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	return parser.ParseResult{Text: "mock result"}, nil
}

func (m *mockParser) FileType() filetype.FileType {
	return m.fileType
}

func TestRegistry(t *testing.T) {
	t.Run("New registry is empty", func(t *testing.T) {
		r := New()
		if r.Count() != 0 {
			t.Errorf("New registry count = %d, want 0", r.Count())
		}
	})

	t.Run("Register and Lookup", func(t *testing.T) {
		r := New()
		p := &mockParser{fileType: filetype.FileTypePDF}

		// Register should succeed
		err := r.Register(filetype.FileTypePDF, p)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		// Lookup should return the same parser
		found, err := r.Lookup(filetype.FileTypePDF)
		if err != nil {
			t.Fatalf("Lookup failed: %v", err)
		}
		if found != p {
			t.Errorf("Lookup returned different parser")
		}
	})

	t.Run("Duplicate registration fails", func(t *testing.T) {
		r := New()
		p1 := &mockParser{fileType: filetype.FileTypePDF}
		p2 := &mockParser{fileType: filetype.FileTypePDF}

		// First registration should succeed
		err := r.Register(filetype.FileTypePDF, p1)
		if err != nil {
			t.Fatalf("First Register failed: %v", err)
		}

		// Second registration should fail
		err = r.Register(filetype.FileTypePDF, p2)
		if err == nil {
			t.Fatal("Expected duplicate registration error, got nil")
		}

		_, ok := err.(*DuplicateRegistrationError)
		if !ok {
			t.Errorf("Expected DuplicateRegistrationError, got %T: %v", err, err)
		}
	})

	t.Run("Lookup non-existent parser fails", func(t *testing.T) {
		r := New()

		_, err := r.Lookup(filetype.FileTypePDF)
		if err == nil {
			t.Fatal("Expected parser not found error, got nil")
		}

		_, ok := err.(*ParserNotFoundError)
		if !ok {
			t.Errorf("Expected ParserNotFoundError, got %T: %v", err, err)
		}
	})

	t.Run("Registry is concurrency-safe", func(t *testing.T) {
		r := New()

		// Concurrent registrations
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(n int) {
				ft := filetype.FileType(string([]byte{'A' + byte(n)}))
				err := r.Register(ft, &mockParser{fileType: ft})
				if err != nil {
					t.Errorf("Concurrent registration failed: %v", err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		if r.Count() != 10 {
			t.Errorf("Concurrent registry count = %d, want 10", r.Count())
		}
	})

	t.Run("FileTypes returns all registered types", func(t *testing.T) {
		r := New()

		expectedTypes := []filetype.FileType{
			filetype.FileTypePDF,
			filetype.FileTypeDOCX,
			filetype.FileTypeTXT,
		}

		for _, ft := range expectedTypes {
			r.Register(ft, &mockParser{fileType: ft})
		}

		actualTypes := r.FileTypes()
		if len(actualTypes) != len(expectedTypes) {
			t.Fatalf("FileTypes returned %d types, want %d", len(actualTypes), len(expectedTypes))
		}

		// Check that all expected types are present
		typeSet := make(map[filetype.FileType]bool)
		for _, ft := range actualTypes {
			typeSet[ft] = true
		}

		for _, ft := range expectedTypes {
			if !typeSet[ft] {
				t.Errorf("FileTypes missing expected type: %s", ft)
			}
		}
	})
}

// asError converts an error to a specific error type if possible.
func asError(err error, target interface{}) bool {
	t, ok := target.(interface{ As(interface{}) bool })
	if !ok {
		return false
	}
	return t.As(err)
}
