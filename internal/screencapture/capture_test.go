package screencapture

import (
	"os"
	"runtime"
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
	// Windows reports synthesized POSIX mode bits; privacy there comes from
	// the current user's temporary-directory ACL rather than chmod metadata.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
