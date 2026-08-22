package security

import "strings"

// IsSafeZipEntry checks if a ZIP entry name is safe for processing.
// It rejects paths containing:
// - ../ (path traversal)
// - absolute paths (starting with /)
// - path traversal after cleaning
func IsSafeZipEntry(name string) bool {
	// Reject absolute paths
	if strings.HasPrefix(name, "/") {
		return false
	}

	// Reject paths containing traversal
	if strings.Contains(name, "..") {
		return false
	}

	// Clean the path and check for traversal
	cleaned := strings.TrimPrefix(name, "./")
	cleaned = strings.TrimPrefix(cleaned, ".\\")

	// Check if the cleaned path still has traversal components
	parts := strings.Split(cleaned, "/")
	for _, part := range parts {
		if part == ".." {
			return false
		}
	}

	return true
}

// IsSafeZipEntryWithPath checks if a ZIP entry path is safe when combined with a base path.
// This prevents directory traversal when extracting to a target directory.
func IsSafeZipEntryWithPath(entryName, basePath string) bool {
	// First check the entry name itself
	if !IsSafeZipEntry(entryName) {
		return false
	}

	// Check for path traversal when joined with base path
	fullPath := basePath + "/" + entryName

	// Normalize the path
	normalized := strings.ReplaceAll(fullPath, "\\", "/")

	// Check if the normalized path starts with the base path
	// This prevents traversal like "../../etc/passwd"
	if strings.HasPrefix(normalized, basePath+"/") || normalized == basePath {
		return true
	}

	// Also check if it's just the base path itself
	if normalized == basePath {
		return true
	}

	return false
}
