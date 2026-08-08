package parser

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrFileTooLarge is returned when a file exceeds the maximum allowed size
var ErrFileTooLarge = errors.New("file size exceeds maximum allowed size")

// CheckFileSize checks if a file exceeds the maximum allowed size
// Returns nil if file size is acceptable, ErrFileTooLarge if too large
func CheckFileSize(file *os.File) error {
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.Size() > MaxFileSize {
		return fmt.Errorf("%w: %d bytes (max: %d bytes)", ErrFileTooLarge, fileInfo.Size(), MaxFileSize)
	}

	return nil
}

// LimitReader wraps io.LimitReader with our MaxFileSize constant
func LimitReader(r io.Reader) io.Reader {
	return io.LimitReader(r, MaxFileSize)
}
