package ocrbundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvironmentOverride(t *testing.T) {
	dir := t.TempDir()
	engine := filepath.Join(dir, "clip-ocr")
	for _, path := range []string{
		engine,
		filepath.Join(dir, "text-detection.onnx"),
		filepath.Join(dir, "text-recognition.onnx"),
	} {
		if err := os.WriteFile(path, []byte("test"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("CLIP_OCR_ENGINE", engine)
	t.Setenv("CLIP_OCR_MODELS", dir)

	paths, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if paths.Engine != engine {
		t.Fatalf("Engine = %q", paths.Engine)
	}
}

func TestInstallFileReplacesSameSizeTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clip-ocr")
	if err := os.WriteFile(path, []byte("evil"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := installFile(path, []byte("good"), 0700); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "good" {
		t.Fatalf("cached asset was not repaired: %q", content)
	}
}

func TestEnsurePrivateDirectoryRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := ensurePrivateDirectory(link); err == nil {
		t.Fatal("expected cache symlink to be rejected")
	}
}
