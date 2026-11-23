package txt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/upendra7470/clip/internal/parser"
)

func TestParseRangeStopsAfterRequestedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.txt")
	content := append([]byte("first\nsecond\n"), 0xff)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	result, err := new(Parser).ParseRange(context.Background(), parser.ParseRequest{File: path}, 1, 2)
	if err != nil {
		t.Fatalf("ParseRange() error = %v", err)
	}
	if result.Text != "first\nsecond" {
		t.Fatalf("Text = %q", result.Text)
	}
}
