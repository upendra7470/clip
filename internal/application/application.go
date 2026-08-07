package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/upendra7470/clip/internal/detect"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/registry"
)

// Clipboard defines the interface for clipboard operations.
type Clipboard interface {
	// Copy copies the given text to the system clipboard.
	Copy(text string) error
}

// Application handles the document extraction workflow.
type Application struct {
	reg       *registry.Registry
	clipboard Clipboard
}

// New creates a new Application with the given registry and clipboard.
func New(reg *registry.Registry, clipboard Clipboard) *Application {
	return &Application{
		reg:       reg,
		clipboard: clipboard,
	}
}

// Extract processes a document file through the complete pipeline:
// detect → lookup parser → parse → copy to clipboard.
func (app *Application) Extract(ctx context.Context, filePath string) error {
	return app.ExtractWithRange(ctx, filePath, nil)
}

// ExtractWithRange processes a document file with optional range through the complete pipeline:
// detect → lookup parser → parse → copy to clipboard.
func (app *Application) ExtractWithRange(ctx context.Context, filePath string, rangeObj *parser.Range) error {
	// Step 0: Guard file size (prevent huge files)
	if info, statErr := os.Stat(filePath); statErr == nil {
		const maxSize int64 = 500 * 1024 * 1024 // 500 MiB
		if info.Size() > maxSize {
			return fmt.Errorf("file size %d bytes exceeds maximum allowed %d bytes", info.Size(), maxSize)
		}
	} else if !os.IsNotExist(statErr) && !os.IsPermission(statErr) {
		// If we can't stat for other reasons, return the error
		return fmt.Errorf("cannot stat file %s: %w", filePath, statErr)
	}

	// Step 1: Detect file type
	fileType, err := detect.Type(filePath)
	if err != nil {
		// Extract file extension for better error message
		ext := filepath.Ext(filePath)
		if ext == "" {
			ext = "unknown"
		}
		return fmt.Errorf("unsupported file type: %s\n\nsupported formats:\nPDF, DOCX, TXT, Markdown, PPTX, CSV, XLSX, JSON, XML, HTML, YAML, RTF, ODT, ODS, PPT", ext)
	}

	// Step 2: Lookup parser
	parserObj, err := app.reg.Lookup(fileType)
	if err != nil {
		return fmt.Errorf("parser not found for file type: %s", fileType)
	}

	// Step 3: Parse document
	req := parser.ParseRequest{
		File: filePath,
		// Selection is intentionally empty for now
		Selection: parser.Selection{},
	}

	var result parser.ParseResult

	// Check if parser supports range extraction and if a range was requested
	if rangeObj != nil {
		if rangeParser, ok := parserObj.(parser.RangeParser); ok {
			start := rangeObj.Start
			end := rangeObj.End
			if start == -1 {
				start = 1 // start from the beginning
			}
			// end == -1 is interpreted by the parser as "to end"
			result, err = rangeParser.ParseRange(ctx, req, start, end)
			if err != nil {
				// keep err as is
			}
		} else {
			return fmt.Errorf("range extraction is not currently supported for %s files", fileType)
		}
	} else {
		result, err = parserObj.ParseWithContext(ctx, req)
	}

	if err != nil {
		// Check for permission errors
		if os.IsPermission(err) {
			return fmt.Errorf("cannot access file: %s\nreason: permission denied", filePath)
		}
		// Check for file not found errors
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("failed to extract text from file: %w", err)
	}

	// Step 4: Copy to clipboard
	if err := app.clipboard.Copy(result.Text); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	return nil
}
