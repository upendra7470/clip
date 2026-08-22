//go:build linux

package clipboard

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// copyPlatform is the platform-specific implementation for linux.
func copyPlatform(text string) error {
	return copyLinux(text)
}

// copyLinux copies text to the clipboard on Linux using xclip (X11)
// or wl-copy (Wayland).
func copyLinux(text string) error {
	// Try xclip first (X11)
	if path, err := exec.LookPath("xclip"); err == nil {
		return runCopyCommand(path, []string{"-selection", "clipboard"}, text)
	}

	// Fall back to wl-copy (Wayland)
	if path, err := exec.LookPath("wl-copy"); err == nil {
		return runCopyCommand(path, nil, text)
	}

	return fmt.Errorf("Linux clipboard unavailable: install xclip (X11) or wl-copy (Wayland)")
}

// runCopyCommand pipes text into a clipboard command's stdin.
func runCopyCommand(name string, args []string, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return wrapError(fmt.Sprintf("failed to open stdin pipe for %s", name), err)
	}

	if err := cmd.Start(); err != nil {
		return wrapError(fmt.Sprintf("failed to start %s", name), err)
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		stdin.Close()
		return wrapError(fmt.Sprintf("failed to write to %s", name), err)
	}

	if err := stdin.Close(); err != nil {
		return wrapError(fmt.Sprintf("failed to close %s stdin", name), err)
	}

	if err := cmd.Wait(); err != nil {
		return wrapError(fmt.Sprintf("%s failed", name), err)
	}

	return nil
}
