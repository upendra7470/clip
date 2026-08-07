//go:build linux

package clipboard

import (
	"context"
	"os/exec"
	"time"
)

// copyPlatform is the platform-specific implementation for linux.
func copyPlatform(text string) error {
	return copyLinux(text)
}

// copyLinux copies text to the clipboard on Linux using xclip or wl-copy.
func copyLinux(text string) error {
	// Try xclip first (X11)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
	stdin, err := cmd.StdinPipe()
	if err == nil {
		// xclip is available
		if err := cmd.Start(); err != nil {
			return wrapError("failed to start xclip", err)
		}

		if _, err := stdin.Write([]byte(text)); err != nil {
			stdin.Close()
			return wrapError("failed to write to xclip", err)
		}

		if err := stdin.Close(); err != nil {
			return wrapError("failed to close xclip stdin", err)
		}

		if err := cmd.Wait(); err != nil {
			return wrapError("xclip failed", err)
		}

		return nil
	}

	// Fall back to wl-copy (Wayland)
	cmd = exec.CommandContext(ctx, "wl-copy")
	stdin, err = cmd.StdinPipe()
	if err != nil {
		return wrapError("Linux clipboard unavailable (neither xclip nor wl-copy found)", err)
	}

	if err := cmd.Start(); err != nil {
		return wrapError("failed to start wl-copy", err)
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		stdin.Close()
		return wrapError("failed to write to wl-copy", err)
	}

	if err := stdin.Close(); err != nil {
		return wrapError("failed to close wl-copy stdin", err)
	}

	if err := cmd.Wait(); err != nil {
		return wrapError("wl-copy failed", err)
	}

	return nil
}
