package parser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/upendra7470/clip/internal/filetype"
)

// Parser interface defines the contract for all parsers
//go:generate mockgen -destination=./mocks/mock_parser.go -package=mocks github.com/yourorg/clip/internal/parser Parser
//go:generate mockgen -destination=./mocks/mock_parser.go -package=mocks github.com/yourorg/clip/internal/parser Parser

// Parser represents a generic file parser
//go:generate mockgen -destination=./mocks/mock_parser.go -package=mocks github.com/yourorg/clip/internal/parser Parser

// Parser represents a generic file parser
//go:generate mockgen -destination=./mocks/mock_parser.go -package=mocks github.com/yourorg/clip/internal/parser Parser

// ParseRequest represents a request to parse a file with optional selection criteria
type ParseRequest struct {
	File      string
	Selection Selection
}

// Selection represents the selection criteria for parsing
type Selection struct {
	Pages string
	Range string
	Query string
}

// ParseResult represents the result of a parse operation
type ParseResult struct {
	Text string
}

// RangeParser interface defines the contract for parsers that support range-based parsing
type RangeParser interface {
	ParseRange(ctx context.Context, req ParseRequest, start, end int) (ParseResult, error)
	GetRangeUnit() string
}

type Parser interface {
	ParseWithContext(ctx context.Context, req ParseRequest) (ParseResult, error)
	ParseFile(path string) (*DocumentUnit, error)
	ParseDirectory(dirPath string) ([]*DocumentUnit, error)
	FileType() filetype.FileType
}

// DocumentUnit represents a parsed document unit
//go:generate mockgen -destination=./mocks/mock_document_unit.go -package=mocks github.com/yourorg/clip/internal/parser DocumentUnit

// NewDocumentUnit creates a new DocumentUnit with the given content and metadata
func NewDocumentUnit(text string, meta map[string]interface{}) *DocumentUnit {
	return &DocumentUnit{
		Text: text,
		Meta: meta,
	}
}

// ParseFile parses the given file and returns a DocumentUnit
func ParseFile(path string) (*DocumentUnit, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	text, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	meta := map[string]interface{}{
		"path": path,
		"type": "text",
	}

	return NewDocumentUnit(string(text), meta), nil
}

// ParseDirectory parses all files in the given directory and returns a slice of DocumentUnits
func ParseDirectory(dirPath string) ([]*DocumentUnit, error) {
	var documentUnits []*DocumentUnit

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		docUnit, err := ParseFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse file %s: %w", path, err)
		}

		documentUnits = append(documentUnits, docUnit)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %w", err)
	}

	return documentUnits, nil
}
