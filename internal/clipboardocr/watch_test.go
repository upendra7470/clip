package clipboardocr

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/upendra7470/clip/internal/clipboard"
	clipoutput "github.com/upendra7470/clip/internal/output"
)

type testCopier struct {
	plain string
	html  string
}

func (c *testCopier) Copy(value string) error {
	c.plain = value
	return nil
}

func (c *testCopier) CopyRich(plain, html string) error {
	c.plain, c.html = plain, html
	return nil
}

func TestRunRecognizesEachUniqueImageAndDeletesTemporaryFiles(t *testing.T) {
	events := make(chan clipboard.ImageEvent, 5)
	events <- clipboard.ImageEvent{Bytes: []byte("first image")}
	events <- clipboard.ImageEvent{Bytes: []byte("first image")}
	events <- clipboard.ImageEvent{Bytes: []byte("second image")}
	events <- clipboard.ImageEvent{}
	events <- clipboard.ImageEvent{Bytes: []byte("second image")}
	close(events)

	var paths []string
	recognitions := 0
	recognize := func(_ context.Context, path string) (string, error) {
		paths = append(paths, path)
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		// Windows reports synthesized Unix permission bits; privacy there is
		// enforced by the current user's temporary-directory ACL.
		if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
			t.Fatalf("temporary image permissions = %o", info.Mode().Perm())
		}
		recognitions++
		return "hello clipboard", nil
	}
	copier := &testCopier{}
	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), events, Config{Format: clipoutput.Markdown, Rich: true}, recognize, copier, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if recognitions != 3 {
		t.Fatalf("recognitions = %d, want 3", recognitions)
	}
	if copier.plain != "hello clipboard" || !strings.Contains(copier.html, "<p>hello clipboard</p>") {
		t.Fatalf("copied plain=%q html=%q", copier.plain, copier.html)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("temporary image still exists: %s", path)
		}
	}
}

func TestRunSkipsSensitiveImages(t *testing.T) {
	events := make(chan clipboard.ImageEvent, 1)
	events <- clipboard.ImageEvent{Bytes: []byte("secret"), Sensitive: true}
	close(events)
	called := false
	var stderr bytes.Buffer
	err := Run(context.Background(), events, Config{Stdout: true, Format: clipoutput.Text}, func(context.Context, string) (string, error) {
		called = true
		return "", nil
	}, nil, &bytes.Buffer{}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("sensitive clipboard image was OCRed")
	}
	if !strings.Contains(stderr.String(), "Skipped a sensitive clipboard image") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
