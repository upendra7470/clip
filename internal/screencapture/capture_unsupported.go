//go:build !darwin && !linux && !windows

package screencapture

import (
	"context"
	"fmt"
	"runtime"
)

func capturePlatform(context.Context) (string, error) {
	return "", fmt.Errorf("screen capture is not supported on %s", runtime.GOOS)
}
