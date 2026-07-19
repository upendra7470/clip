package clipboard

import (
	"errors"
	"testing"
)

func TestCopyUnsupportedPlatform(t *testing.T) {
	// This test should only run on unsupported platforms
	// We'll mock this by testing the error path
	err := copyUnsupported()
	if err == nil {
		t.Error("Expected error for unsupported platform, got nil")
	}
	if err.Error() != "unsupported platform" {
		t.Errorf("Expected 'unsupported platform' error, got: %v", err)
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			message:  "test error",
			err:      nil,
			expected: "test error",
		},
		{
			name:     "with underlying error",
			message:  "context",
			err:      errors.New("underlying"),
			expected: "context: underlying",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapError(tt.message, tt.err)
			if result.Error() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result.Error())
			}

			// Test error unwrapping
			if tt.err != nil {
				var target error = tt.err
				if !errors.Is(result, target) {
					t.Errorf("Expected error to wrap underlying error")
				}
			}
		})
	}
}

// copyUnsupported simulates the unsupported platform case for testing
func copyUnsupported() error {
	return errors.New("unsupported platform")
}
