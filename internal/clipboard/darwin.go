//go:build darwin

package clipboard

import (
	"os/exec"
)

// copyImpl is the platform-specific implementation for darwin.
func copyImpl(text string) error {
	return copyDarwin(text)
}

// copyDarwin copies text to the clipboard on macOS using pbcopy.
func copyDarwin(text string) error {
	cmd := exec.Command("pbcopy")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return wrapError("macOS clipboard unavailable", err)
	}

	if err := cmd.Start(); err != nil {
		return wrapError("failed to start pbcopy", err)
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		stdin.Close()
		return wrapError("failed to write to pbcopy", err)
	}

	if err := stdin.Close(); err != nil {
		return wrapError("failed to close pbcopy stdin", err)
	}

	if err := cmd.Wait(); err != nil {
		return wrapError("pbcopy failed", err)
	}

	return nil
}
