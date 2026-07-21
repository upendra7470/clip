package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestCLIBuilds(t *testing.T) {
	// Test that the CLI can be built
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}

	// Clean up
	os.Remove("clip")
}

func TestCLIHelpFlag(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test --help flag
	cmd = exec.Command("./clip", "--help")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help flag failed: %v\n%s", err, output)
	}

	// Check that help output contains expected text
	outputStr := string(output)
	if !contains(outputStr, "Clip - Universal Document Extractor") {
		t.Errorf("Help output should contain 'Clip - Universal Document Extractor', got: %s", outputStr)
	}
	if !contains(outputStr, "Usage:") {
		t.Errorf("Help output should contain 'Usage:', got: %s", outputStr)
	}
	if !contains(outputStr, "clip <filename>") {
		t.Errorf("Help output should contain 'clip <filename>', got: %s", outputStr)
	}
}

func TestCLIVersionFlag(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test --version flag
	cmd = exec.Command("./clip", "--version")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version flag failed: %v\n%s", err, output)
	}

	// Check that version output contains expected text
	outputStr := string(output)
	if !contains(outputStr, "Clip v1.0.0") {
		t.Errorf("Version output should contain 'Clip v1.0.0', got: %s", outputStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCLIFileResolution(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Create a test file
	testContent := "Hello, this is a test file for Clip CLI."
	testFile := "test_clip.txt"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test file resolution and extraction
	cmd = exec.Command("./clip", testFile)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("File extraction failed: %v\n%s", err, output)
	}

	// Check that output contains success messages
	outputStr := string(output)
	if !contains(outputStr, "✓ Found:") {
		t.Errorf("Output should contain '✓ Found:', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Extracted text successfully") {
		t.Errorf("Output should contain '✓ Extracted text successfully', got: %s", outputStr)
	}
	if !contains(outputStr, "✓ Copied to clipboard") {
		t.Errorf("Output should contain '✓ Copied to clipboard', got: %s", outputStr)
	}
}

func TestCLIErrorHandling(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test with non-existent file
	cmd = exec.Command("./clip", "nonexistent_file.pdf")
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Expected error for non-existent file, but got none")
	}

	// Check that error message is user-friendly
	outputStr := string(output)
	if !contains(outputStr, "file \"nonexistent_file.pdf\" not found") {
		t.Errorf("Error output should contain file not found message, got: %s", outputStr)
	}
	if !contains(outputStr, "search locations checked:") {
		t.Errorf("Error output should contain search locations, got: %s", outputStr)
	}
}

func TestCLINoFileProvided(t *testing.T) {
	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", "clip", "../cmd/clip")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, output)
	}
	defer os.Remove("clip")

	// Test with no file provided - should show help and exit gracefully
	cmd = exec.Command("./clip")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Expected no error when no file provided (should show help), but got: %v\n%s", err, output)
	}

	// Check that help is shown
	outputStr := string(output)
	if !contains(outputStr, "Clip - Universal Document Extractor") {
		t.Errorf("Output should contain help message, got: %s", outputStr)
	}
	if !contains(outputStr, "Usage:") {
		t.Errorf("Output should contain usage information, got: %s", outputStr)
	}
}
