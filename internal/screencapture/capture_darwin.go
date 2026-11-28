//go:build darwin

package screencapture

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func capturePlatform(ctx context.Context) (string, error) {
	path, err := temporaryPNG()
	if err != nil {
		return "", err
	}
	// screencapture ships with macOS. -i enables the native interactive region
	// selector and -x suppresses the camera sound.
	cmd := exec.CommandContext(ctx, "/usr/sbin/screencapture", "-i", "-x", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(path)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if len(output) > 0 {
			return "", fmt.Errorf("capture screen region: %s", output)
		}
		return "", ErrCancelled
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		_ = os.Remove(path)
		return "", ErrCancelled
	}
	return path, nil
}
