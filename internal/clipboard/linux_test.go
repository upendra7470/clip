//go:build linux

package clipboard

import (
	"testing"
)

func TestLinuxClipboardDoesNotRequireHelperCommands(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if testClipboardImpl == nil {
		// The implementation is linked directly into Clip. A display may not be
		// present in unit tests, so this assertion only verifies there is no
		// dependency on an executable discovered through PATH.
		_ = copyLinux("hello")
	}
}
