//go:build windows

package clipboard

// copyPlatform is the platform-specific implementation for windows.
func copyPlatform(text string) error {
	return copyWindows(text)
}

// copyWindows copies text through the native Windows clipboard API.
func copyWindows(text string) error {
	return copyNative(text)
}
