// Package clipboardocr implements continuous, local OCR for images copied to
// the system clipboard.
package clipboardocr

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/clipboard"
	clipoutput "github.com/upendra7470/clip/internal/output"
	"github.com/upendra7470/clip/internal/security"
)

// Copier publishes recognized text back to the system clipboard.
type Copier interface {
	Copy(string) error
	CopyRich(string, string) error
}

// Recognizer performs OCR on one private temporary image.
type Recognizer func(context.Context, string) (string, error)

// Config controls how recognized clipboard images are rendered and delivered.
type Config struct {
	Format clipoutput.Format
	Rich   bool
	Stdout bool
}

// RecognizeOnce performs OCR on one clipboard image and returns plain text.
// It is used by universal input mode when rendering and delivery are handled
// by the main document pipeline.
func RecognizeOnce(ctx context.Context, event clipboard.ImageEvent, recognize Recognizer) (string, error) {
	if recognize == nil {
		return "", fmt.Errorf("OCR recognizer is required")
	}
	if event.Sensitive {
		return "", fmt.Errorf("refusing to OCR sensitive clipboard content")
	}
	if len(event.Bytes) == 0 {
		return "", fmt.Errorf("clipboard does not contain an image")
	}
	if len(event.Bytes) > security.MaxInputFileSize {
		return "", fmt.Errorf("clipboard image encoded size exceeds %d bytes", security.MaxInputFileSize)
	}
	return recognizeImage(ctx, event.Bytes, recognize)
}

// RunOnce recognizes one current clipboard image and exits.
func RunOnce(
	ctx context.Context,
	event clipboard.ImageEvent,
	config Config,
	recognize Recognizer,
	copier Copier,
	stdout, stderr io.Writer,
) error {
	if recognize == nil {
		return fmt.Errorf("OCR recognizer is required")
	}
	if copier == nil && !config.Stdout {
		return fmt.Errorf("clipboard destination is required")
	}
	if event.Sensitive {
		return fmt.Errorf("refusing to OCR sensitive clipboard content")
	}
	if len(event.Bytes) == 0 {
		return fmt.Errorf("clipboard does not contain an image")
	}
	if len(event.Bytes) > security.MaxInputFileSize {
		return fmt.Errorf("clipboard image encoded size exceeds %d bytes", security.MaxInputFileSize)
	}
	statusWriter := stdout
	if config.Stdout {
		statusWriter = stderr
	}
	return process(ctx, event.Bytes, config, recognize, copier, stdout, statusWriter)
}

// Run processes clipboard image events until the stream closes or ctx is
// canceled. A bad image is reported and skipped so watch mode stays available
// for the next clipboard change.
func Run(
	ctx context.Context,
	events <-chan clipboard.ImageEvent,
	config Config,
	recognize Recognizer,
	copier Copier,
	stdout, stderr io.Writer,
) error {
	if events == nil {
		return fmt.Errorf("clipboard image stream is required")
	}
	if recognize == nil {
		return fmt.Errorf("OCR recognizer is required")
	}
	if copier == nil && !config.Stdout {
		return fmt.Errorf("clipboard destination is required")
	}

	var previous [sha256.Size]byte
	havePrevious := false
	statusWriter := stdout
	if config.Stdout {
		statusWriter = stderr
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if event.Sensitive {
				havePrevious = false
				fmt.Fprintln(stderr, "Skipped a sensitive clipboard image")
				continue
			}
			if len(event.Bytes) == 0 {
				// Losing the image format is a real clipboard change. Reset the
				// digest so copying the same image again is intentional work.
				havePrevious = false
				continue
			}
			if len(event.Bytes) > security.MaxInputFileSize {
				havePrevious = false
				fmt.Fprintf(stderr, "Skipped clipboard image: encoded size exceeds %d bytes\n", security.MaxInputFileSize)
				continue
			}
			digest := sha256.Sum256(event.Bytes)
			if havePrevious && digest == previous {
				continue
			}
			previous, havePrevious = digest, true
			if err := process(ctx, event.Bytes, config, recognize, copier, stdout, statusWriter); err != nil {
				fmt.Fprintf(stderr, "Clipboard OCR warning: %v\n", err)
			}
		}
	}
}

func process(
	ctx context.Context,
	image []byte,
	config Config,
	recognize Recognizer,
	copier Copier,
	stdout, status io.Writer,
) error {
	text, err := recognizeImage(ctx, image, recognize)
	if err != nil {
		return err
	}
	documents := []clipoutput.Document{{Source: "clipboard.png", Type: "png", Text: text}}
	rendered, err := clipoutput.Render(documents, config.Format, 0)
	if err != nil {
		return err
	}
	destination := "clipboard"
	if config.Stdout {
		fmt.Fprintln(stdout, rendered)
		destination = "stdout"
	} else if config.Rich {
		html, err := clipoutput.RenderHTML(documents)
		if err != nil {
			return err
		}
		if err := copier.CopyRich(rendered, html); err != nil {
			return fmt.Errorf("copy rich text: %w", err)
		}
	} else if err := copier.Copy(rendered); err != nil {
		return fmt.Errorf("copy text: %w", err)
	}
	fmt.Fprintf(status, "✓ Clipboard image: recognized %d words → %s\n", len(strings.Fields(text)), destination)
	return nil
}

func recognizeImage(ctx context.Context, image []byte, recognize Recognizer) (string, error) {
	temporary, err := os.CreateTemp("", "clip-clipboard-*.png")
	if err != nil {
		return "", fmt.Errorf("create private image: %w", err)
	}
	path := temporary.Name()
	defer func() { _ = os.Remove(path) }()
	if err := os.Chmod(path, 0600); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("secure private image: %w", err)
	}
	if _, err := temporary.Write(image); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("write private image: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close private image: %w", err)
	}

	text, err := recognize(ctx, path)
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("OCR found no readable text")
	}
	return text, nil
}
