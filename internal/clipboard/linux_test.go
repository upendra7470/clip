//go:build linux

package clipboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxFallsBackWhenWaylandBackendFails(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "wl-copy"), "#!/bin/sh\nexit 1\n")
	marker := filepath.Join(dir, "xclip-used")
	writeExecutable(t, filepath.Join(dir, "xclip"), "#!/bin/sh\ncat >/dev/null\ntouch \""+marker+"\"\n")
	t.Setenv("PATH", dir+":/usr/bin:/bin")

	if err := copyLinux("hello"); err != nil {
		t.Fatalf("copyLinux() error = %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("xclip fallback was not used: %v", err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
}
