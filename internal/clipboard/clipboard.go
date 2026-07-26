package clipboard

import (
	"errors"
	"fmt"
	"sync"
)

// Copy copies the given text to the system clipboard.
// It returns an error if the clipboard is unavailable or the operation fails.
func Copy(text string) error {
	// Platform-specific implementation will be provided by the build tag files
	return copyImpl(text)
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %w", message, err)
}

// TestClipboard is a test implementation of the clipboard for testing purposes.
// It implements the application.Clipboard interface.
type TestClipboard struct {
	mu         sync.Mutex
	copiedText string
}

// NewTestClipboard creates a new test clipboard.
func NewTestClipboard() *TestClipboard {
	return &TestClipboard{}
}

// Copy copies the given text to the test clipboard.
func (t *TestClipboard) Copy(text string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.copiedText = text
	return nil
}

// GetCopiedText returns the text that was copied to the test clipboard.
func (t *TestClipboard) GetCopiedText() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.copiedText
}

// Reset clears the test clipboard.
func (t *TestClipboard) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.copiedText = ""
}

// testClipboardImpl is a package-level variable that can be set for testing.
// When set, Copy() will use this implementation instead of the platform-specific one.
var testClipboardImpl *TestClipboard
var testClipboardMu sync.Mutex

// SetTestClipboard sets a test clipboard implementation for testing.
// Pass nil to restore the real clipboard implementation.
func SetTestClipboard(tc *TestClipboard) {
	testClipboardMu.Lock()
	defer testClipboardMu.Unlock()
	testClipboardImpl = tc
}

// copyImpl is the platform-specific implementation for copying to clipboard.
// It can be overridden for testing by calling SetTestClipboard.
func copyImpl(text string) error {
	testClipboardMu.Lock()
	tc := testClipboardImpl
	testClipboardMu.Unlock()

	if tc != nil {
		return tc.Copy(text)
	}

	// Platform-specific implementation will be provided by build tag files
	return copyPlatform(text)
}
