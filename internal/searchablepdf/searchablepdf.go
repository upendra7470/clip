// Package searchablepdf adds an invisible OCR text layer to scanned PDF
// pages while preserving the original page appearance.
package searchablepdf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	ledpdf "github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/upendra7470/clip/internal/parser"
)

const maxPageTextRunes = 50_000

// Extractor is the extraction surface required from application.Application.
type Extractor interface {
	ExtractText(context.Context, string, *parser.Range, parser.Selection) (parser.ParseResult, error)
}

// Create preserves inputPath's pages and adds invisible searchable text to
// pages that do not already contain a native text layer. It never overwrites
// an existing output file.
func Create(ctx context.Context, extractor Extractor, inputPath, outputPath string, selection parser.Selection) (int, error) {
	if extractor == nil {
		return 0, fmt.Errorf("searchable PDF extractor is required")
	}
	inputAbs, err := filepath.Abs(inputPath)
	if err != nil {
		return 0, fmt.Errorf("resolve input path: %w", err)
	}
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return 0, fmt.Errorf("resolve output path: %w", err)
	}
	if filepath.Clean(inputAbs) == filepath.Clean(outputAbs) {
		return 0, fmt.Errorf("output must be different from the input PDF")
	}
	if _, err := os.Lstat(outputAbs); err == nil {
		return 0, fmt.Errorf("output already exists: %s", outputPath)
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("check output: %w", err)
	}

	// #nosec G304 -- Clip intentionally opens the local PDF selected by the user.
	input, err := os.Open(inputAbs)
	if err != nil {
		return 0, fmt.Errorf("open PDF: %w", err)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return 0, fmt.Errorf("inspect PDF: %w", err)
	}
	reader, err := ledpdf.NewReader(input, info.Size())
	if err != nil {
		return 0, fmt.Errorf("read PDF: %w", err)
	}

	watermarks := make(map[int]*model.Watermark)
	// Stateless configuration keeps the one-command binary self-contained and
	// never attempts to create a pdfcpu config directory in the user's profile.
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	for pageNumber := 1; pageNumber <= reader.NumPage(); pageNumber++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		nativeText, err := reader.Page(pageNumber).GetPlainText(nil)
		if err != nil {
			return 0, fmt.Errorf("read page %d text: %w", pageNumber, err)
		}
		if strings.TrimSpace(nativeText) != "" {
			continue
		}
		pageRange := parser.Range{Start: pageNumber, End: pageNumber}
		result, err := extractor.ExtractText(ctx, inputAbs, &pageRange, selection)
		if err != nil {
			return 0, fmt.Errorf("OCR page %d: %w", pageNumber, err)
		}
		text := watermarkText(result.Text)
		if text == "" {
			continue
		}
		watermark, err := api.TextWatermark(ctx, text,
			"font:Helvetica, points:1, scale:1 abs, pos:bl, off:0 0, rot:0, op:0",
			true, false, types.POINTS, conf)
		if err != nil {
			return 0, fmt.Errorf("prepare page %d text layer: %w", pageNumber, err)
		}
		watermarks[pageNumber] = watermark
	}

	if err := os.MkdirAll(filepath.Dir(outputAbs), 0700); err != nil {
		return 0, fmt.Errorf("create output directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(outputAbs), ".clip-searchable-*.pdf")
	if err != nil {
		return 0, fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return 0, fmt.Errorf("close temporary output: %w", err)
	}
	defer os.Remove(temporaryPath)

	if len(watermarks) == 0 {
		if err := copyFile(inputAbs, temporaryPath); err != nil {
			return 0, err
		}
	} else if err := api.AddWatermarksMapFile(ctx, inputAbs, temporaryPath, watermarks, conf); err != nil {
		return 0, fmt.Errorf("write searchable PDF: %w", err)
	}
	if err := os.Chmod(temporaryPath, 0600); err != nil {
		return 0, fmt.Errorf("secure output permissions: %w", err)
	}
	// A hard-link publish is atomic and fails if another process creates the
	// destination after our preflight check. Unlike Rename on Unix, it cannot
	// silently replace an existing user file.
	if err := os.Link(temporaryPath, outputAbs); err != nil {
		return 0, fmt.Errorf("publish searchable PDF: %w", err)
	}
	return len(watermarks), nil
}

func watermarkText(value string) string {
	value = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case unicode.IsControl(r):
			return -1
		default:
			return r
		}
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > maxPageTextRunes {
		runes = runes[:maxPageTextRunes]
	}
	// pdfcpu interprets percent sequences as page-number placeholders.
	return strings.ReplaceAll(string(runes), "%", "%%")
}

func copyFile(source, destination string) error {
	// #nosec G304 -- source is the resolved user-selected PDF.
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source PDF: %w", err)
	}
	defer in.Close()
	// #nosec G304 -- destination was created by os.CreateTemp in Create.
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open temporary output: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		return errors.Join(fmt.Errorf("copy PDF: %w", err), out.Close())
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close copied PDF: %w", err)
	}
	return nil
}
