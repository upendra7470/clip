//go:build darwin

package clipboard

// copyPlatform is the platform-specific implementation for darwin.
func copyPlatform(text string) error {
	return copyDarwin(text)
}

// copyDarwin copies text through the native macOS pasteboard API.
func copyDarwin(text string) error {
	return copyNative(text)
}
