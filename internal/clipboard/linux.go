//go:build linux

package clipboard

// copyPlatform is the platform-specific implementation for linux.
func copyPlatform(text string) error {
	return copyLinux(text)
}

// copyLinux copies text directly through X11 or Wayland without external tools.
func copyLinux(text string) error {
	return copyNative(text)
}
