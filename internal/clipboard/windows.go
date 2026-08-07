//go:build windows

package clipboard

import (
	"context"
	"os/exec"
	"time"
)

// copyPlatform is the platform-specific implementation for windows.
func copyPlatform(text string) error {
	return copyWindows(text)
}

// copyWindows copies text to the clipboard on Windows using clip.exe.
func copyWindows(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cmd", "/c", "clip")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return wrapError("Windows clipboard unavailable", err)
	}

	if err := cmd.Start(); err != nil {
		return wrapError("failed to start clip", err)
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		stdin.Close()
		return wrapError("failed to write to clip", err)
	}

	if err := stdin.Close(); err != nil {
		return wrapError("failed to close clip stdin", err)
	}

	if err := cmd.Wait(); err != nil {
		return wrapError("clip failed", err)
	}

	return nil
}
