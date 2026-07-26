package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCLIHelpFlag tests the --help flag.
func TestCLIHelpFlag(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	cmd := exec.Command(clipPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip --help failed: %v\nOutput: %s", err, output)
	}

	helpText := string(output)
	if !strings.Contains(helpText, "Usage:") {
		t.Errorf("Help text missing 'Usage:' section: %s", helpText)
	}
	if !strings.Contains(helpText, "--help") {
		t.Errorf("Help text missing --help flag: %s", helpText)
	}
	if !strings.Contains(helpText, "--version") {
		t.Errorf("Help text missing --version flag: %s", helpText)
	}
}

// TestCLIVersionFlag tests the --version flag.
func TestCLIVersionFlag(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	cmd := exec.Command(clipPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip --version failed: %v\nOutput: %s", err, output)
	}

	versionText := strings.TrimSpace(string(output))
	if versionText == "" {
		t.Error("Version output is empty")
	}
	if !strings.Contains(versionText, "clip") {
		t.Errorf("Version output doesn't contain 'clip': %s", versionText)
	}
}

// TestCLIFileResolution tests file resolution with exact path.
func TestCLIFileResolution(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(clipPath, testFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip failed: %v\nOutput: %s", err, output)
	}

	// Verify clipboard content by running clip again with --help to see if it worked
	// Actually, we can't easily verify clipboard in subprocess tests without a test clipboard
	// For now, just verify the command succeeds
	if strings.Contains(string(output), "error") {
		t.Errorf("Unexpected error in output: %s", output)
	}
}

// TestCLIErrorHandling tests error handling for non-existent file.
func TestCLIErrorHandling(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	cmd := exec.Command(clipPath, "nonexistent.txt")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for nonexistent file, got success")
	}

	errorText := string(output)
	if !strings.Contains(strings.ToLower(errorText), "error") &&
		!strings.Contains(strings.ToLower(errorText), "not found") &&
		!strings.Contains(strings.ToLower(errorText), "unsupported") {
		t.Errorf("Error message not informative: %s", errorText)
	}
}

// TestCLINoFileProvided tests behavior when no file is provided.
func TestCLINoFileProvided(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	cmd := exec.Command(clipPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error when no file provided, got success")
	}

	errorText := string(output)
	if !strings.Contains(strings.ToLower(errorText), "usage") &&
		!strings.Contains(strings.ToLower(errorText), "error") &&
		!strings.Contains(strings.ToLower(errorText), "required") {
		t.Errorf("Error message not informative: %s", errorText)
	}
}

// TestCLIFilenameWithSpaces tests filename with spaces.
func TestCLIFilenameWithSpaces(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file with spaces
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test file.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(clipPath, testFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip failed with spaced filename: %v\nOutput: %s", err, output)
	}
}

// TestCLIRangeExtraction tests range extraction.
func TestCLIRangeExtraction(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file with multiple lines
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line 1\nline 2\nline 3\nline 4\nline 5"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(clipPath, testFile, "2-4")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip range extraction failed: %v\nOutput: %s", err, output)
	}
}

// TestCLISmartFilenameResolution tests smart filename resolution.
func TestCLISmartFilenameResolution(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "TestFile.TXT")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	cmd := exec.Command(clipPath, "testfile.txt")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip smart resolution failed: %v\nOutput: %s", err, output)
	}
}

// TestCLIFilenameWithSpacesNoQuotes tests filename with spaces without quotes.
func TestCLIFilenameWithSpacesNoQuotes(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file with spaces
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test file.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change dir: %v", err)
	}

	cmd := exec.Command(clipPath, "test file.txt")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip failed with spaced filename (no quotes): %v\nOutput: %s", err, output)
	}
}

// TestCLIRangeExtractionCSV tests range extraction on CSV.
func TestCLIRangeExtractionCSV(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test CSV file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.csv")
	testContent := "col1,col2,col3\n1,2,3\n4,5,6\n7,8,9\n10,11,12"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(clipPath, testFile, "2-3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip CSV range extraction failed: %v\nOutput: %s", err, output)
	}
}

// TestCLIRangeExtractionMarkdown tests range extraction on Markdown.
func TestCLIRangeExtractionMarkdown(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test Markdown file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	testContent := "# Header 1\n\nParagraph 1\n\n## Header 2\n\nParagraph 2\n\n### Header 3\n\nParagraph 3"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(clipPath, testFile, "2-3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip Markdown range extraction failed: %v\nOutput: %s", err, output)
	}
}

// TestCLIRangeExtractionJSON tests range extraction on JSON.
func TestCLIRangeExtractionJSON(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test JSON file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	testContent := `{
	 "items": [{"id": 1}, {"id": 2}, {"id": 3}, {"id": 4}]
}`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(clipPath, testFile, "2-3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip JSON range extraction failed: %v\nOutput: %s", err, output)
	}
}

// TestCLISingleUnitRange tests single unit range (e.g., "3-3").
func TestCLISingleUnitRange(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line 1\nline 2\nline 3\nline 4"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command(clipPath, testFile, "3-3")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip single unit range failed: %v\nOutput: %s", err, output)
	}
}

// TestCLIInvalidRange tests invalid range handling.
func TestCLIInvalidRange(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line 1\nline 2"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test invalid range format
	cmd := exec.Command(clipPath, testFile, "invalid")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for invalid range, got success")
	}

	// Test out of bounds range
	cmd = exec.Command(clipPath, testFile, "5-10")
	_, err = cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for out of bounds range, got success")
	}
}

// BuildCLIBinary builds the CLI binary in a temporary directory and returns its absolute path.
// This function is designed for use in tests and ensures the binary is built for the current
// platform/architecture with executable permissions.
func BuildCLIBinary(t *testing.T) string {
	t.Helper()

	// Create a temporary directory for the binary
	tmpDir := t.TempDir()
	clipPath := filepath.Join(tmpDir, "clip")

	// Get the module root directory (where go.mod is)
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	moduleRoot := filepath.Dir(testDir)
	cmdDir := filepath.Join(moduleRoot, "cmd", "clip")

	// Build the binary for the current platform/architecture
	var buildEnv []string
	buildEnv = append(buildEnv, os.Environ()...)
	buildEnv = append(buildEnv, "CGO_ENABLED=0")

	// Set GOOS and GOARCH to match the current platform
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	buildEnv = append(buildEnv, "GOOS="+goos, "GOARCH="+goarch)

	cmd := exec.Command("go", "build", "-o", clipPath, ".")
	cmd.Dir = cmdDir
	cmd.Env = buildEnv
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build clip binary: %v\nOutput: %s", err, output)
	}

	// Verify the binary exists and is executable
	info, err := os.Stat(clipPath)
	if err != nil {
		t.Fatalf("Built binary not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		// Make executable if not already
		if err := os.Chmod(clipPath, 0755); err != nil {
			t.Fatalf("Failed to make binary executable: %v", err)
		}
	}

	// Resolve and return the absolute path to the built binary
	absPath, err := filepath.Abs(clipPath)
	if err != nil {
		t.Fatalf("Failed to resolve absolute path: %v", err)
	}

	return absPath
}

// TestCLIBinaryBuiltInTempDir tests that the CLI binary is built in a temporary directory.
func TestCLIBinaryBuiltInTempDir(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Verify the binary is in a temporary directory
	if !strings.Contains(clipPath, "Temp") && !strings.Contains(clipPath, "temp") {
		t.Errorf("Binary not built in temporary directory: %s", clipPath)
	}

	// Verify the binary exists
	if _, err := os.Stat(clipPath); err != nil {
		t.Fatalf("Built binary not found: %v", err)
	}
}

// TestCLIWorksWithoutClipDirectory tests that the CLI works when no ./clip directory exists.
func TestCLIWorksWithoutClipDirectory(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a temporary directory without a ./clip subdirectory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to the temporary directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Verify no ./clip directory exists
	clipDir := filepath.Join(tmpDir, "clip")
	if _, err := os.Stat(clipDir); !os.IsNotExist(err) {
		t.Fatal("clip directory should not exist")
	}

	// Run the CLI command
	cmd := exec.Command(clipPath, testFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clip failed: %v\nOutput: %s", err, output)
	}
}

// TestCLIMultipleInvocationsNoFlagRedefinition tests that multiple calls to the CLI entry point do not cause flag redefinition.
func TestCLIMultipleInvocationsNoFlagRedefinition(t *testing.T) {
	clipPath := BuildCLIBinary(t)
	defer os.Remove(clipPath)

	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run the CLI command multiple times
	for i := 0; i < 3; i++ {
		cmd := exec.Command(clipPath, testFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("clip failed on invocation %d: %v\nOutput: %s", i+1, err, output)
		}
	}
}
