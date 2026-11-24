package ocrbundle

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

const bundleVersion = "ocrs-0.14.0"

//go:embed assets/*
var bundledAssets embed.FS

type Paths struct {
	Engine           string
	DetectionModel   string
	RecognitionModel string
}

func Resolve() (Paths, error) {
	if engine := os.Getenv("CLIP_OCR_ENGINE"); engine != "" {
		return pathsFromDirectory(engine, os.Getenv("CLIP_OCR_MODELS"))
	}

	engineName := "clip-ocr"
	if runtime.GOOS == "windows" {
		engineName += ".exe"
	}
	if _, err := bundledAssets.Open("assets/" + engineName); err != nil {
		return Paths{}, fmt.Errorf("built-in OCR is unavailable in this development build; use an official Clip release")
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, fmt.Errorf("locate user cache: %w", err)
	}
	targetDir := filepath.Join(cacheRoot, "clip", bundleVersion)
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return Paths{}, fmt.Errorf("create OCR cache: %w", err)
	}

	entries, err := fs.ReadDir(bundledAssets, "assets")
	if err != nil {
		return Paths{}, fmt.Errorf("read built-in OCR assets: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "placeholder.txt" {
			continue
		}
		content, err := bundledAssets.ReadFile("assets/" + entry.Name())
		if err != nil {
			return Paths{}, fmt.Errorf("read built-in OCR asset %s: %w", entry.Name(), err)
		}
		mode := os.FileMode(0600)
		if entry.Name() == engineName {
			mode = 0700
		}
		if err := installFile(filepath.Join(targetDir, entry.Name()), content, mode); err != nil {
			return Paths{}, err
		}
	}

	return pathsFromDirectory(filepath.Join(targetDir, engineName), targetDir)
}

func pathsFromDirectory(engine, modelDir string) (Paths, error) {
	if modelDir == "" {
		modelDir = filepath.Join(filepath.Dir(engine), "models")
	}
	paths := Paths{
		Engine:           engine,
		DetectionModel:   filepath.Join(modelDir, "text-detection.onnx"),
		RecognitionModel: filepath.Join(modelDir, "text-recognition.onnx"),
	}
	for label, path := range map[string]string{
		"OCR engine":             paths.Engine,
		"text detection model":   paths.DetectionModel,
		"text recognition model": paths.RecognitionModel,
	} {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return Paths{}, fmt.Errorf("%s is unavailable at %s", label, path)
		}
	}
	return paths, nil
}

func installFile(path string, content []byte, mode os.FileMode) error {
	if info, err := os.Stat(path); err == nil && info.Size() == int64(len(content)) {
		return nil
	}
	tempPath := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tempPath, content, mode); err != nil {
		return fmt.Errorf("write OCR asset: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("install OCR asset: %w", err)
	}
	return nil
}
