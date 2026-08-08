package parser

// Security constants for file processing
const (
	// MaxFileSize defines the maximum allowed file size for parsing
	// This prevents memory exhaustion attacks from large files
	MaxFileSize = 50 * 1024 * 1024 // 50 MiB
)
