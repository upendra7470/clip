package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestListReturnsFailureWhenInspectionFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--list", path}, &stdout, &stderr); code == 0 {
		t.Fatalf("Run() returned success, stderr = %q", stderr.String())
	}
}
