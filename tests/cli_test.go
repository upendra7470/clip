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
	if !contains(outputStr, "Clip") {
		t.Errorf("Help output should contain 'Clip', got: %s", outputStr)
	}
	if !contains(outputStr, "A fast CLI for extracting text from documents") {
		t.Errorf("Help output should contain project description, got: %s", outputStr)
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
	if !contains(outputStr, "Clip version") {
		t.Errorf("Version output should contain 'Clip version', got: %s", outputStr)
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
