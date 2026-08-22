package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/upendra7470/clip/internal/security"
)

// TesseractBinary is the external OCR engine binary name.
const TesseractBinary = "tesseract"

// DefaultLanguage is the OCR language passed to Tesseract.
// The "eng" traineddata ships with every Tesseract distribution.
const DefaultLanguage = "eng"

// DefaultEngine is the parser's default OCR engine: a locally installed
// Tesseract binary invoked over stdin/stdout. No network access, no API keys.
var DefaultEngine Engine = &TesseractEngine{Language: DefaultLanguage}

// TesseractEngine runs the tesseract command-line tool to perform local OCR.
type TesseractEngine struct {
	// Language selects the Tesseract language data (e.g. "eng", "deu+eng").
	Language string
}

// Name returns the engine name for diagnostics.
func (e *TesseractEngine) Name() string {
	return TesseractBinary
}

// IsAvailable reports whether the tesseract binary can be found on PATH.
func (e *TesseractEngine) IsAvailable() bool {
	_, err := exec.LookPath(TesseractBinary)
	return err == nil
}

// installHint returns platform-specific installation guidance.
func installHint() string {
	return "install it first:\n" +
		"  macOS:         brew install tesseract\n" +
		"  Debian/Ubuntu: sudo apt install tesseract-ocr\n" +
		"  Fedora:        sudo dnf install tesseract\n" +
		"  Arch:          sudo pacman -S tesseract\n" +
		"  Windows:       winget install UB-Mannheim.TesseractOCR"
}

// ExtractText runs OCR on the raw image bytes using the tesseract CLI.
// The image is piped via stdin and text is read from stdout, so no
// temporary files are created and nothing leaves the machine.
func (e *TesseractEngine) ExtractText(ctx context.Context, data []byte) (string, error) {
	if _, err := exec.LookPath(TesseractBinary); err != nil {
		return "", wrapError(fmt.Sprintf("OCR engine %q is not installed", TesseractBinary), &engineMissingError{})
	}

	language := e.Language
	if language == "" {
		language = DefaultLanguage
	}

	ctx, cancel := context.WithTimeout(ctx, security.MaxOCRDuration)
	defer cancel()

	cmd := exec.CommandContext(ctx, TesseractBinary, "stdin", "stdout", "-l", language)
	cmd.Stdin = bytes.NewReader(data)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if msg == "" {
				return "", wrapError(fmt.Sprintf("OCR engine %s failed with exit code %d", TesseractBinary, exitErr.ExitCode()), nil)
			}
			// Surface only the meaningful tail of Tesseract diagnostics.
			lines := strings.Split(msg, "\n")
			if len(lines) > 3 {
				msg = strings.Join(lines[len(lines)-3:], "; ")
			}
			return "", wrapError(fmt.Sprintf("OCR engine %s failed", TesseractBinary), fmt.Errorf("%s", msg))
		}
		return "", wrapError(fmt.Sprintf("failed to run OCR engine %s", TesseractBinary), err)
	}

	return string(output), nil
}

// engineMissingError marks the well-known "binary not installed" failure so
// callers can distinguish it from engine runtime failures.
type engineMissingError struct{}

func (e *engineMissingError) Error() string { return installHint() }
