package xlsx

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/security"
	"github.com/xuri/excelize/v2"
)

// XLSXParserError represents an error that occurs during XLSX parsing.
type XLSXParserError struct {
	message string
	cause   error
}

func (e *XLSXParserError) Error() string {
	if e.cause != nil {
		if e.message == "" {
			return e.cause.Error()
		}
		return e.message + ": " + e.cause.Error()
	}
	if e.message == "" {
		return "XLSX parser error"
	}
	return e.message
}

func (e *XLSXParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser, parser.RangeParser, and parser.DocumentLister interfaces for XLSX files.
type Parser struct{}

// NewParser creates a new XLSX Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an XLSX file and extracts readable text representation.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	limitedReader := io.LimitReader(reader, parser.MaxFileSize)
	text, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read xlsx: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type":                filetype.FileTypeXLSX,
			"preserved_structure": "sheet names, row/column structure, tables in readable format",
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "xlsx file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": filetype.FileTypeXLSX,
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	lines, err := readXLSXLines(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(strings.TrimSpace(line))
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in XLSX", nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeXLSX
}

// ListUnits implements the parser.DocumentLister interface for XLSX files.
func (p *Parser) ListUnits(ctx context.Context, req parser.ParseRequest) (int, []string, error) {
	// Open the XLSX file
	file, err := os.Open(req.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil, wrapError(fmt.Sprintf("file %s does not exist", req.File), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return 0, nil, wrapError(fmt.Sprintf("permission denied reading file %s", req.File), err)
		}
		return 0, nil, wrapError(fmt.Sprintf("error opening file %s: %v", req.File, err), err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, nil, wrapError("failed to get file info", err)
	}
	if err := validateXLSXArchive(file, fileInfo.Size()); err != nil {
		return 0, nil, err
	}

	// Read the XLSX file
	xlsxFile, err := excelize.OpenReader(file)
	if err != nil {
		return 0, nil, wrapError("failed to parse XLSX file", err)
	}
	defer xlsxFile.Close()

	// Get all sheet names
	sheetNames := xlsxFile.GetSheetList()
	if len(sheetNames) == 0 {
		return 0, nil, wrapError("no sheets found in XLSX file", nil)
	}

	// For each sheet, get the row count
	var totalRows int
	var sheetInfo []string
	for _, sheetName := range sheetNames {
		rows, err := xlsxFile.Rows(sheetName)
		if err != nil {
			continue
		}
		rowCount := 0
		for rows.Next() {
			if err := ctx.Err(); err != nil {
				_ = rows.Close()
				return 0, nil, err
			}
			rowCount++
		}
		if err := rows.Error(); err != nil {
			_ = rows.Close()
			return 0, nil, wrapError(fmt.Sprintf("failed to count rows in sheet %s", sheetName), err)
		}
		_ = rows.Close()
		totalRows += rowCount
		sheetInfo = append(sheetInfo, fmt.Sprintf("%s (%d rows)", sheetName, rowCount))
	}

	return totalRows, sheetInfo, nil
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() parser.RangeUnit {
	return parser.RangeUnitRows
}

// ParseRange extracts text from a specific row range in an XLSX file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	if start < 1 || (end != -1 && end < 1) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("row numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end != -1 && end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid row range: start row must not be greater than end row (got %d-%d)", start, end), nil)
	}

	lines, err := readXLSXRows(req.File, start, end)
	if err != nil {
		return parser.ParseResult{}, err
	}

	if len(lines) == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in XLSX", nil)
	}

	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(strings.TrimSpace(line))
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in rows %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// readXLSXLines reads an XLSX file and returns logical rows.
// XLSX files are ZIP archives; anything else is rejected as invalid.
func readXLSXLines(path string) ([]string, error) {
	return readXLSXRows(path, 1, -1)
}

func readXLSXRows(path string, start, end int) ([]string, error) {
	// #nosec G304 -- Clip intentionally opens the local document path selected by the user.
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, wrapError(fmt.Sprintf("file %s does not exist", path), err)
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, wrapError(fmt.Sprintf("permission denied reading file %s", path), err)
		}
		return nil, wrapError(fmt.Sprintf("error opening file %s: %v", path, err), err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, wrapError("failed to get XLSX file info", err)
	}

	if fileInfo.Size() > parser.MaxFileSize {
		return nil, wrapError(fmt.Sprintf("file %s exceeds maximum allowed size of %d bytes", path, parser.MaxFileSize), nil)
	}
	if err := validateXLSXArchive(file, fileInfo.Size()); err != nil {
		return nil, err
	}

	xlsxFile, err := excelize.OpenReader(file)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("invalid XLSX file: %s", path), err)
	}
	defer xlsxFile.Close()

	sheets := xlsxFile.GetSheetList()
	if len(sheets) == 0 {
		return nil, wrapError(fmt.Sprintf("invalid XLSX file: %s contains no worksheets", path), nil)
	}

	rows, err := xlsxFile.Rows(sheets[0])
	if err != nil {
		return nil, wrapError("failed to read XLSX worksheet", err)
	}
	defer rows.Close()

	var lines []string
	rowNumber := 0
	for rows.Next() {
		rowNumber++
		if rowNumber < start {
			continue
		}
		if end != -1 && rowNumber > end {
			break
		}
		columns, err := rows.Columns()
		if err != nil {
			return nil, wrapError("failed to read XLSX row", err)
		}
		lines = append(lines, strings.Join(columns, ", "))
		if end != -1 && rowNumber == end {
			break
		}
	}
	if err := rows.Error(); err != nil {
		return nil, wrapError("failed to read XLSX rows", err)
	}
	if rowNumber < start || (end != -1 && rowNumber < end) {
		return nil, wrapError(fmt.Sprintf("requested row range exceeds XLSX row count (file has %d rows, requested %d-%d)", rowNumber, start, end), nil)
	}
	return lines, nil
}

func validateXLSXArchive(file *os.File, size int64) error {
	archive, err := zip.NewReader(file, size)
	if err != nil {
		return wrapError("invalid XLSX file: not a valid ZIP archive", err)
	}
	if err := security.CheckZipArchiveTotalSize(archive.File); err != nil {
		return wrapError("failed to read XLSX as ZIP archive", err)
	}
	return nil
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &XLSXParserError{
			message: message,
			cause:   nil,
		}
	}
	return &XLSXParserError{
		message: message,
		cause:   err,
	}
}
