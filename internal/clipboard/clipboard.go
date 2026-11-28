package clipboard

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	nativeclipboard "golang.design/x/clipboard"
)

var nativeInit sync.Once
var nativeInitErr error
var htmlFormat = nativeclipboard.Register("text/html")

// Copy copies the given text to the system clipboard.
// It returns an error if the clipboard is unavailable or the operation fails.
func Copy(text string) error {
	// Platform-specific implementation will be provided by the build tag files
	return copyImpl(text)
}

// CopyRich publishes HTML and plain-text representations in one clipboard
// operation. Rich-text applications receive HTML while every other
// application can still paste the plain-text fallback.
func CopyRich(plainText, html string) error {
	testClipboardMu.Lock()
	tc := testClipboardImpl
	testClipboardMu.Unlock()
	if tc != nil {
		return tc.CopyRich(plainText, html)
	}
	return copyRichNative(plainText, html)
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
	copiedHTML string
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
	t.copiedHTML = ""
	return nil
}

// CopyRich records both clipboard representations for tests.
func (t *TestClipboard) CopyRich(plainText, html string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.copiedText = plainText
	t.copiedHTML = html
	return nil
}

// GetCopiedText returns the text that was copied to the test clipboard.
func (t *TestClipboard) GetCopiedText() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.copiedText
}

// GetCopiedHTML returns the rich clipboard representation.
func (t *TestClipboard) GetCopiedHTML() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.copiedHTML
}

// Reset clears the test clipboard.
func (t *TestClipboard) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.copiedText = ""
	t.copiedHTML = ""
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

// copyNative uses native, pure-Go clipboard integrations on Windows, macOS,
// X11, and Wayland. It does not shell out to an operating-system utility.
func copyNative(text string) error {
	nativeInit.Do(func() { nativeInitErr = nativeclipboard.Init() })
	if nativeInitErr != nil {
		return wrapError("system clipboard unavailable", nativeInitErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := nativeclipboard.Write(ctx, nativeclipboard.FmtText, []byte(text)); err != nil {
		return wrapError("failed to copy text", err)
	}
	return nil
}

func copyRichNative(plainText, html string) error {
	nativeInit.Do(func() { nativeInitErr = nativeclipboard.Init() })
	if nativeInitErr != nil {
		return wrapError("system clipboard unavailable", nativeInitErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := nativeclipboard.WriteAll(ctx, richClipboardOptions(plainText, html)...); err != nil {
		return wrapError("failed to copy rich text", err)
	}
	return nil
}

func cfHTML(fragment string) []byte {
	prefix := "<html><body><!--StartFragment-->"
	suffix := "<!--EndFragment--></body></html>"
	const headerTemplate = "Version:0.9\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\n"
	// All four decimal fields have a fixed width, so formatting with zeroes
	// first yields the exact byte offset at which the HTML begins.
	headerLength := len(fmt.Sprintf(headerTemplate, 0, 0, 0, 0))
	startHTML := headerLength
	startFragment := startHTML + len(prefix)
	endFragment := startFragment + len(fragment)
	endHTML := endFragment + len(suffix)
	header := fmt.Sprintf(headerTemplate, startHTML, endHTML, startFragment, endFragment)
	return []byte(header + prefix + fragment + suffix)
}
