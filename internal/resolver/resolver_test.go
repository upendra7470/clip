package resolver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExactPathResolution(t *testing.T) {
	t.Run("exact path with ./ prefix", func(t *testing.T) {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		resolver := New()
		ctx := context.Background()

		// Test with ./ prefix
		relativePath := "./" + filepath.Base(tmpFile.Name())
		_, err = resolver.Resolve(ctx, relativePath)
		if err == nil {
			t.Errorf("Expected error for non-existent relative path, got nil")
		}
	})

	t.Run("absolute path", func(t *testing.T) {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		resolver := New()
		ctx := context.Background()

		// Test with absolute path
		resolvedPath, err := resolver.Resolve(ctx, tmpFile.Name())
		if err != nil {
			t.Errorf("Failed to resolve absolute path: %v", err)
		}

		if resolvedPath != tmpFile.Name() {
			t.Errorf("Expected %s, got %s", tmpFile.Name(), resolvedPath)
		}
	})
}

func TestFilenameSearch(t *testing.T) {
	t.Run("file in current directory", func(t *testing.T) {
		// Create a temporary file in current directory
		tmpFile, err := os.CreateTemp(".", "test*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		filename := filepath.Base(tmpFile.Name())
		resolver := New()
		ctx := context.Background()

		resolvedPath, err := resolver.Resolve(ctx, filename)
		if err != nil {
			t.Errorf("Failed to resolve filename: %v", err)
		}

		// Compare file names only, not full paths
		expectedFilename := filepath.Base(tmpFile.Name())
		actualFilename := filepath.Base(resolvedPath)
		if actualFilename != expectedFilename {
			t.Errorf("Expected filename %s, got %s", expectedFilename, actualFilename)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		resolver := New()
		ctx := context.Background()

		_, err := resolver.Resolve(ctx, "nonexistent_file.txt")
		if err == nil {
			t.Errorf("Expected error for non-existent file, got nil")
		}

		expectedErrorPrefix := "file \"nonexistent_file.txt\" not found"
		if !strings.HasPrefix(err.Error(), expectedErrorPrefix) {
			t.Errorf("Expected error message to start with %q, got %q", expectedErrorPrefix, err.Error())
		}
	})
}

func TestContextCancellation(t *testing.T) {
	t.Run("context cancellation during resolution", func(t *testing.T) {
		resolver := New()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// This should timeout
		_, err := resolver.Resolve(ctx, "test.txt")
		if err == nil {
			t.Errorf("Expected context deadline exceeded error, got nil")
		}

		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
	})
}

func TestPermissionErrors(t *testing.T) {
	t.Run("permission denied", func(t *testing.T) {
		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "testdir")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a file with no read permissions
		tmpFile := filepath.Join(tmpDir, "test.txt")
		f, err := os.Create(tmpFile)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		f.Close()

		// Change permissions to make it unreadable
		err = os.Chmod(tmpFile, 0000)
		if err != nil {
			t.Fatalf("Failed to change file permissions: %v", err)
		}
		defer func() {
			// Restore permissions even if test fails
			os.Chmod(tmpFile, 0644)
		}()

		resolver := New()
		ctx := context.Background()

		_, err = resolver.Resolve(ctx, tmpFile)
		// On some systems, this might still work due to root privileges or other factors
		// So we'll check if we get a permission error or any other error
		if err == nil {
			t.Skip("Permission test skipped - file was accessible despite 0000 permissions")
		}

		// Check if it's a permission-related error
		if strings.Contains(err.Error(), "permission denied") {
			expectedErrorMsg := "cannot access file: " + tmpFile + "\nreason: permission denied"
			if err.Error() != expectedErrorMsg {
				t.Errorf("Expected error message %q, got %q", expectedErrorMsg, err.Error())
			}
		} else {
			t.Logf("Got different error (expected permission denied): %v", err)
		}
	})
}

func TestMultipleFilesFound(t *testing.T) {
	t.Run("multiple files with same name", func(t *testing.T) {
		// Create temporary files in different locations
		tmpFile1, err := os.CreateTemp(".", "test*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file 1: %v", err)
		}
		defer os.Remove(tmpFile1.Name())

		// Create a Downloads directory for testing
		downloadsDir := filepath.Join(".", "test_downloads")
		err = os.MkdirAll(downloadsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test downloads dir: %v", err)
		}
		defer os.RemoveAll(downloadsDir)

		tmpFile2, err := os.CreateTemp(downloadsDir, "test*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file 2: %v", err)
		}
		defer os.Remove(tmpFile2.Name())

		// Rename both files to have the same name
		filename := "test_file.txt"
		err = os.Rename(tmpFile1.Name(), filepath.Join(".", filename))
		if err != nil {
			t.Fatalf("Failed to rename temp file 1: %v", err)
		}

		err = os.Rename(tmpFile2.Name(), filepath.Join(downloadsDir, filename))
		if err != nil {
			t.Fatalf("Failed to rename temp file 2: %v", err)
		}

		resolver := New()
		ctx := context.Background()

		_, err = resolver.Resolve(ctx, filename)
		if err == nil {
			t.Errorf("Expected error for multiple files, got nil")
		}

		// The error should indicate multiple files were found
		expectedErrorPrefix := "multiple files named"
		if !strings.Contains(err.Error(), expectedErrorPrefix) {
			t.Errorf("Expected error message to contain %q, got %q", expectedErrorPrefix, err.Error())
		}
	})
}

func TestIsExactPath(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{"./file.txt", true},
		{"../file.txt", true},
		{"/absolute/path/file.txt", true},
		{"subdir/file.txt", true},
		{"file.txt", false},
		{"document.pdf", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := isExactPath(tc.path)
			if result != tc.expected {
				t.Errorf("isExactPath(%q) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

func TestNormalizeFilename(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"The Brain.docx", "thebrain.docx"},
		{"the brain.docx", "thebrain.docx"},
		{"THEBRAIN.DOCX", "thebrain.docx"},
		{"jntuh.pdf", "jntuh.pdf"},
		{"file name with spaces.txt", "filenamewithspaces.txt"},
		{"Another File.PDF", "anotherfile.pdf"},
		{"test", "test"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeFilename(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeFilename(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestSmartFilenameResolution(t *testing.T) {
	t.Run("case-insensitive matching", func(t *testing.T) {
		// Create a test file with mixed case
		tmpFile, err := os.CreateTemp(".", "TestFile*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		// Rename to have specific case
		actualName := "TheBrain.docx"
		err = os.Rename(tmpFile.Name(), filepath.Join(".", actualName))
		if err != nil {
			t.Fatalf("Failed to rename temp file: %v", err)
		}
		defer os.Remove(actualName)

		resolver := New()
		ctx := context.Background()

		// Test different case variations
		testCases := []string{
			"thebrain.docx",
			"THEBRAIN.DOCX",
			"the brain.docx",
			"The Brain.docx",
		}

		for _, query := range testCases {
			t.Run(query, func(t *testing.T) {
				resolvedPath, err := resolver.Resolve(ctx, query)
				if err != nil {
					t.Errorf("Failed to resolve %q: %v", query, err)
					return
				}

				// Check that the resolved file exists and is the one we created
				// Since exact match takes precedence, it should find "TheBrain.docx"
				// But if smart matching is used, it should still resolve to our file
				resolvedFilename := filepath.Base(resolvedPath)
				normalizedResolved := normalizeFilename(resolvedFilename)
				normalizedExpected := normalizeFilename(actualName)

				if normalizedResolved != normalizedExpected {
					t.Errorf("Expected normalized filename %q, got %q (original: %q)", normalizedExpected, normalizedResolved, resolvedFilename)
				}
			})
		}
	})

	t.Run("space-insensitive matching", func(t *testing.T) {
		// Create a test file with spaces
		tmpFile, err := os.CreateTemp(".", "Test File*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		// Rename to have specific name with spaces
		actualName := "The Brain.docx"
		err = os.Rename(tmpFile.Name(), filepath.Join(".", actualName))
		if err != nil {
			t.Fatalf("Failed to rename temp file: %v", err)
		}
		defer os.Remove(actualName)

		resolver := New()
		ctx := context.Background()

		// Test query without spaces
		query := "thebrain.docx"
		resolvedPath, err := resolver.Resolve(ctx, query)
		if err != nil {
			t.Errorf("Failed to resolve %q: %v", query, err)
			return
		}

		// Check that the resolved file exists and is the one we created
		// Since exact match takes precedence, it should find "The Brain.docx"
		// But if smart matching is used, it should still resolve to our file
		resolvedFilename := filepath.Base(resolvedPath)
		normalizedResolved := normalizeFilename(resolvedFilename)
		normalizedExpected := normalizeFilename(actualName)

		if normalizedResolved != normalizedExpected {
			t.Errorf("Expected normalized filename %q, got %q (original: %q)", normalizedExpected, normalizedResolved, resolvedFilename)
		}
	})

	t.Run("exact match takes precedence", func(t *testing.T) {
		// Create two files: one exact match, one smart match
		exactFile, err := os.CreateTemp(".", "exact*.txt")
		if err != nil {
			t.Fatalf("Failed to create exact file: %v", err)
		}
		defer os.Remove(exactFile.Name())

		smartFile, err := os.CreateTemp(".", "smart*.txt")
		if err != nil {
			t.Fatalf("Failed to create smart file: %v", err)
		}
		defer os.Remove(smartFile.Name())

		// Rename files
		exactName := "thebrain.docx"
		smartName := "The Brain.docx"
		err = os.Rename(exactFile.Name(), filepath.Join(".", exactName))
		if err != nil {
			t.Fatalf("Failed to rename exact file: %v", err)
		}
		defer os.Remove(exactName)

		err = os.Rename(smartFile.Name(), filepath.Join(".", smartName))
		if err != nil {
			t.Fatalf("Failed to rename smart file: %v", err)
		}
		defer os.Remove(smartName)

		resolver := New()
		ctx := context.Background()

		// When querying with exact name, should get exact match
		resolvedPath, err := resolver.Resolve(ctx, exactName)
		if err != nil {
			t.Errorf("Failed to resolve exact match: %v", err)
			return
		}

		expectedFilename := filepath.Base(resolvedPath)
		if expectedFilename != exactName {
			t.Errorf("Expected exact match filename %q, got %q", exactName, expectedFilename)
		}
	})
}

func TestMultipleFilesSmartMatching(t *testing.T) {
	t.Run("multiple files with smart matching", func(t *testing.T) {
		// Create temporary files in different locations that would match the same smart query
		tmpFile1, err := os.CreateTemp(".", "The Brain*.docx")
		if err != nil {
			t.Fatalf("Failed to create temp file 1: %v", err)
		}
		defer os.Remove(tmpFile1.Name())

		// Create a Downloads directory for testing
		downloadsDir := filepath.Join(".", "test_downloads")
		err = os.MkdirAll(downloadsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test downloads dir: %v", err)
		}
		defer os.RemoveAll(downloadsDir)

		tmpFile2, err := os.CreateTemp(downloadsDir, "TheBrain*.docx")
		if err != nil {
			t.Fatalf("Failed to create temp file 2: %v", err)
		}
		defer os.Remove(tmpFile2.Name())

		// Rename both files to have names that would smart-match the same query
		filename1 := "The Brain.docx"
		filename2 := "The Brain.docx" // Same name to trigger multiple exact matches
		err = os.Rename(tmpFile1.Name(), filepath.Join(".", filename1))
		if err != nil {
			t.Fatalf("Failed to rename temp file 1: %v", err)
		}

		err = os.Rename(tmpFile2.Name(), filepath.Join(downloadsDir, filename2))
		if err != nil {
			t.Fatalf("Failed to rename temp file 2: %v", err)
		}

		resolver := New()
		ctx := context.Background()

		// Query with exact match that would find both files
		query := "The Brain.docx"
		_, err = resolver.Resolve(ctx, query)
		if err == nil {
			t.Errorf("Expected error for multiple files, got nil")
		}

		// The error should indicate multiple files were found
		expectedErrorPrefix := "multiple files named"
		if !strings.Contains(err.Error(), expectedErrorPrefix) {
			t.Errorf("Expected error message to contain %q, got %q", expectedErrorPrefix, err.Error())
		}
	})
}
