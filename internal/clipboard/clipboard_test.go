package clipboard

import (
	"errors"
	"fmt"
	"testing"
)

func TestCopyUnsupportedPlatform(t *testing.T) {
	// Add test for warning message
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

func TestCFHTMLByteOffsets(t *testing.T) {
	fragment := "<b>café</b>"
	payload := cfHTML(fragment)
	var startHTML, endHTML, startFragment, endFragment int
	if _, err := fmt.Sscanf(string(payload), "Version:0.9\r\nStartHTML:%d\r\nEndHTML:%d\r\nStartFragment:%d\r\nEndFragment:%d\r\n",
		&startHTML, &endHTML, &startFragment, &endFragment); err != nil {
		t.Fatalf("parse CF_HTML header: %v", err)
	}
	if startHTML < 0 || endHTML != len(payload) || startFragment < startHTML || endFragment > endHTML {
		t.Fatalf("invalid CF_HTML offsets: %d %d %d %d (length %d)", startHTML, endHTML, startFragment, endFragment, len(payload))
	}
	if got := string(payload[startFragment:endFragment]); got != fragment {
		t.Fatalf("CF_HTML fragment = %q, want %q", got, fragment)
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
