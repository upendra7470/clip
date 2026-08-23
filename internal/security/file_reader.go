package security

import (
	"fmt"
	"io"
	"os"
)

// MaxInputFileSize defines the maximum allowed size for input files.
// This is a safety limit to prevent memory exhaustion from pathological files.
const MaxInputFileSize = 50 * 1024 * 1024 // 50 MiB

// ErrFileTooLarge is returned when a file exceeds the maximum allowed size.
var ErrFileTooLarge = fmt.Errorf("file exceeds maximum allowed size of %d bytes", MaxInputFileSize)

// ReadFileBounded reads a file with a size limit.
// It checks the file size before reading to prevent excessive memory usage.
func ReadFileBounded(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if fileInfo.Size() > MaxInputFileSize {
		return nil, fmt.Errorf("%w: %d bytes", ErrFileTooLarge, fileInfo.Size())
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// ReadAllBounded reads from a reader with a hard size limit.
// Unlike io.LimitReader, this returns an error if the reader
// exceeds the limit rather than silently truncating.
func ReadAllBounded(r io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(r, limit+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("content exceeds maximum allowed size of %d bytes", limit)
	}
	return content, nil
}
