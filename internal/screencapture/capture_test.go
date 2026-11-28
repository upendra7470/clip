package screencapture

import (
	"os"
	"testing"
)

func TestTemporaryPNGIsPrivate(t *testing.T) {
	path, err := temporaryPNG()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
