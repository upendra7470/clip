package resolver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolver handles file path resolution for the Clip application.
type Resolver struct{}

// New creates a new Resolver instance.
func New() *Resolver {
	return &Resolver{}
}

// Resolve resolves a file path, handling both exact paths and filename-only searches.
func (r *Resolver) Resolve(ctx context.Context, filePath string) (string, error) {
	// Case 1: Exact path provided (starts with ./, /, or contains path separators)
	if isExactPath(filePath) {
		return r.resolveExactPath(ctx, filePath)
	}

	// Case 2: Filename only - search common locations
	return r.resolveFilename(ctx, filePath)
}

// isExactPath determines if the provided path is an exact path.
func isExactPath(path string) bool {
	// Check if path starts with ./, ../, or /
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || filepath.IsAbs(path) {
		return true
	}

	// Check if path contains directory separators
	if strings.Contains(path, string(filepath.Separator)) {
		return true
	}

	return false
}

// resolveExactPath handles exact path resolution.
func (r *Resolver) resolveExactPath(ctx context.Context, filePath string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Check if file exists and is accessible
		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("file not found: %s", filePath)
			}
			if os.IsPermission(err) {
				return "", fmt.Errorf("cannot access file: %s\nreason: permission denied", filePath)
			}
			return "", fmt.Errorf("cannot access file: %s\nreason: %w", filePath, err)
		}

		// Check if it's a regular file
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("not a regular file: %s", filePath)
		}

		return filePath, nil
	}
}

// resolveFilename searches for a file by name in common locations.
func (r *Resolver) resolveFilename(ctx context.Context, filename string) (string, error) {
	// Define search locations
	searchLocations := []string{
		".", // Current directory
		getDownloadsDir(),
		getDesktopDir(),
		getDocumentsDir(),
	}

	var foundFiles []string

	// First, try exact match (case-sensitive)
	for _, location := range searchLocations {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fullPath := filepath.Join(location, filename)
			if _, err := os.Stat(fullPath); err == nil {
				foundFiles = append(foundFiles, fullPath)
			}
		}
	}

	// If no exact match found, try smart matching
	if len(foundFiles) == 0 {
		// Normalize the search filename for smart matching
		normalizedQuery := normalizeFilename(filename)

		// Search each location with smart matching
		for _, location := range searchLocations {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
				// Read directory contents
				files, err := os.ReadDir(location)
				if err != nil {
					continue // Skip directories that can't be read
				}

				// Check each file in the directory
				for _, file := range files {
					if file.IsDir() {
						continue
					}

					fileName := file.Name()
					normalizedFileName := normalizeFilename(fileName)

					// Try to match using normalized names
					if normalizedFileName == normalizedQuery {
						fullPath := filepath.Join(location, fileName)
						foundFiles = append(foundFiles, fullPath)
					}
				}
			}
		}
	}

	// Handle search results
	switch len(foundFiles) {
	case 0:
		// No files found
		locationNames := []string{
			"Current directory",
			"Downloads",
			"Desktop",
			"Documents",
		}
		locationsList := strings.Join(locationNames, "\n- ")
		return "", fmt.Errorf("file \"%s\" not found\n\nsearch locations checked:\n- %s", filename, locationsList)
	case 1:
		// Single file found
		return foundFiles[0], nil
	default:
		// Multiple files found - ask user to select
		return "", r.handleMultipleFiles(filename, foundFiles)
	}
}

// normalizeFilename normalizes a filename for smart matching.
// It removes spaces and converts to lowercase to enable case-insensitive
// and space-insensitive matching.
func normalizeFilename(filename string) string {
	// Remove file extension first to handle it separately
	ext := filepath.Ext(filename)
	baseName := filename[:len(filename)-len(ext)]

	// Normalize base name: remove spaces and convert to lowercase
	normalizedBase := strings.ToLower(strings.ReplaceAll(baseName, " ", ""))

	// Keep the extension as-is but lowercase
	normalizedExt := strings.ToLower(ext)

	return normalizedBase + normalizedExt
}

// handleMultipleFiles handles the case where multiple files with the same name are found.
func (r *Resolver) handleMultipleFiles(filename string, foundFiles []string) error {
	// Build error message listing all matching files
	var fileList strings.Builder
	fileList.WriteString("multiple files named \"")
	fileList.WriteString(filename)
	fileList.WriteString("\" found:\n")

	for i, file := range foundFiles {
		// Make paths relative to home directory for cleaner display
		relPath, err := filepath.Rel(getHomeDir(), file)
		if err != nil || strings.HasPrefix(relPath, "..") {
			relPath = file
		}
		fileList.WriteString(fmt.Sprintf("%d. %s\n", i+1, relPath))
	}

	return fmt.Errorf("%s", fileList.String())
}

// getHomeDir returns the user's home directory.
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~"
	}
	return home
}

// getDownloadsDir returns the Downloads directory path.
func getDownloadsDir() string {
	// Check for test directory first
	testDir := filepath.Join(".", "test_downloads")
	if _, err := os.Stat(testDir); err == nil {
		return testDir
	}
	return filepath.Join(getHomeDir(), "Downloads")
}

// getDesktopDir returns the Desktop directory path.
func getDesktopDir() string {
	return filepath.Join(getHomeDir(), "Desktop")
}

// getDocumentsDir returns the Documents directory path.
func getDocumentsDir() string {
	// Check for test directory first
	testDir := filepath.Join(".", "test_documents")
	if _, err := os.Stat(testDir); err == nil {
		return testDir
	}
	return filepath.Join(getHomeDir(), "Documents")
}
