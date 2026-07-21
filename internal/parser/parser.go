package parser

import (
	"context"

	"github.com/upendra7470/clip/internal/filetype"
)

// Parser defines the interface that all document parsers must implement.
type Parser interface {
	// Parse extracts text from a document file.
	// The context allows for cancellation and timeouts.
	// The ParseRequest contains the file path and any selection criteria.
	// Returns the extracted text in ParseResult or an error.
	Parse(ctx context.Context, req ParseRequest) (ParseResult, error)
}

// RangeParser is an optional interface that parsers can implement
// to support range extraction for their specific document type.
type RangeParser interface {
	Parser

	// ParseRange extracts text from a specific range in a document.
	// The context allows for cancellation and timeouts.
	// The ParseRequest contains the file path and any selection criteria.
	// start and end are 1-based unit numbers (pages, slides, paragraphs, lines, rows, etc.).
	// Returns the extracted text in ParseResult or an error.
	ParseRange(ctx context.Context, req ParseRequest, start, end int) (ParseResult, error)

	// GetRangeUnit returns the unit type that this parser uses for ranges.
	// Returns a human-readable string like "pages", "slides", "paragraphs", "lines", "rows".
	GetRangeUnit() string
}

// ParseRequest contains the input parameters for parsing a document.
type ParseRequest struct {
	// File is the path to the document file to parse.
	File string

	// Selection specifies which parts of the document to extract.
	// If empty, the entire document should be extracted.
	Selection Selection
}

// ParseResult contains the output of document parsing.
type ParseResult struct {
	// Text is the extracted text from the document.
	Text string
}

// Selection represents criteria for selecting specific content from a document.
// This is a minimal data model that can be extended in future phases.
type Selection struct {
	// Pages specifies which pages to extract (e.g., "1-3,5").
	// Empty means all pages.
	Pages string

	// Range specifies a text range to extract (e.g., "1:10-20:30").
	// Format and interpretation are parser-specific.
	Range string

	// Query specifies a search query for targeted extraction.
	// Empty means no query filtering.
	Query string
}

// FileType returns the file type that this parser handles.
// This should be implemented by each parser implementation.
type FileTypeAware interface {
	FileType() filetype.FileType
}
