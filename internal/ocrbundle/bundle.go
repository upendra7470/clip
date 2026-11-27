package ocrbundle

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io"
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
	clipDir := filepath.Join(cacheRoot, "clip")
	if err := ensurePrivateDirectory(clipDir); err != nil {
		return Paths{}, fmt.Errorf("create OCR cache: %w", err)
	}
	targetDir := filepath.Join(clipDir, bundleVersion)
	if err := ensurePrivateDirectory(targetDir); err != nil {
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
	matches, err := fileMatches(path, content, mode)
	if err != nil {
		return fmt.Errorf("verify OCR asset: %w", err)
	}
	if matches {
		return nil
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("write OCR asset: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() { _ = os.Remove(tempPath) }
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		cleanup()
		return fmt.Errorf("secure OCR asset: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		cleanup()
		return fmt.Errorf("write OCR asset: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		cleanup()
		return fmt.Errorf("sync OCR asset: %w", err)
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close OCR asset: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			cleanup()
			return fmt.Errorf("replace OCR asset: %w", err)
		}
		if retryErr := os.Rename(tempPath, path); retryErr != nil {
			cleanup()
			return fmt.Errorf("install OCR asset: %w", retryErr)
		}
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing unsafe cache directory %s", path)
	}
	// #nosec G302 -- this is an owner-only directory, not a data file.
	return os.Chmod(path, 0700)
}

func fileMatches(path string, content []byte, mode os.FileMode) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || info.Size() != int64(len(content)) {
		return false, nil
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != mode.Perm() {
		return false, nil
	}
	// #nosec G304 -- path is derived from the private OCR cache directory.
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return false, copyErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	expected := sha256.Sum256(content)
	return string(hash.Sum(nil)) == string(expected[:]), nil
}
