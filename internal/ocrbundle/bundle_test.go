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
