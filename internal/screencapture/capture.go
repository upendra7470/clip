// Package screencapture provides interactive, native screen-region capture.
package screencapture

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// ErrCancelled is returned when the user cancels region selection.
var ErrCancelled = errors.New("screen capture cancelled")

// Result describes a temporary screenshot. The caller owns Path and must
// remove it after OCR completes.
type Result struct {
	Path          string
	Width, Height int
}

// Capture opens the platform-native region selector and returns a private
// temporary image containing the selected area.
func Capture(ctx context.Context) (Result, error) {
	path, err := capturePlatform(ctx)
	if err != nil {
		return Result{}, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = os.Remove(path)
		return Result{}, fmt.Errorf("secure captured image: %w", err)
	}
	// #nosec G304 -- every platform returns a private Clip temporary path.
	file, err := os.Open(path)
	if err != nil {
		_ = os.Remove(path)
		return Result{}, fmt.Errorf("open captured image: %w", err)
	}
	config, _, decodeErr := image.DecodeConfig(file)
	closeErr := file.Close()
	if decodeErr != nil {
		_ = os.Remove(path)
		return Result{}, fmt.Errorf("decode captured image: %w", decodeErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return Result{}, fmt.Errorf("close captured image: %w", closeErr)
	}
	if config.Width <= 0 || config.Height <= 0 {
		_ = os.Remove(path)
		return Result{}, fmt.Errorf("captured region is empty")
	}
	return Result{Path: path, Width: config.Width, Height: config.Height}, nil
}

func temporaryPNG() (string, error) {
	file, err := os.CreateTemp("", "clip-screen-*.png")
	if err != nil {
		return "", fmt.Errorf("create screenshot file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close screenshot file: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("secure screenshot file: %w", err)
	}
	return path, nil
}
