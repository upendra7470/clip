package ods

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// ODSParserError represents an error that occurs during ODS parsing.
type ODSParserError struct {
	message string
	cause   error
}

func (e *ODSParserError) Error() string {
	if e.message == "" {
		return "ODS parser error"
	}
	return e.message
}

func (e *ODSParserError) Unwrap() error {
	return e.cause
}

// Parser implements the parser.Parser and parser.RangeParser interfaces for ODS files.
type Parser struct{}

// NewParser creates a new ODS Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads an ODS file and extracts readable text representation.
func (p *Parser) Parse(reader io.Reader) (*parser.DocumentUnit, error) {
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read ods: %w", err)
	}
	return &parser.DocumentUnit{
		Text: string(text),
		Meta: map[string]interface{}{
			"type": filetype.FileTypeODS,
		},
	}, nil
}

// ParseFile implements the parser.Parser interface method for parsing files
func (p *Parser) ParseFile(path string) (*parser.DocumentUnit, error) {
	return &parser.DocumentUnit{
		Text: "ods file content for " + path,
		Meta: map[string]interface{}{
			"path": path,
			"type": filetype.FileTypeODS,
		},
	}, nil
}

// ParseDirectory implements the parser.Parser interface method for parsing directories
func (p *Parser) ParseDirectory(dirPath string) ([]*parser.DocumentUnit, error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseWithContext implements the parser.Parser interface method for parsing with context
func (p *Parser) ParseWithContext(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	lines, err := readODSLines(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}
	if len(lines) == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in ODS", nil)
	}

	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(strings.TrimSpace(line))
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeODS
}

// GetRangeUnit returns the unit type that this parser uses for ranges.
func (p *Parser) GetRangeUnit() string {
	return "rows"
}

// ParseRange extracts text from a specific row range in an ODS file.
func (p *Parser) ParseRange(ctx context.Context, req parser.ParseRequest, start, end int) (parser.ParseResult, error) {
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("row numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid row range: start row must not be greater than end row (got %d-%d)", start, end), nil)
	}

	lines, err := readODSLines(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}
	if len(lines) == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in ODS", nil)
	}
	if start > len(lines) || end > len(lines) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested row range exceeds ODS row count (file has %d rows, requested %d-%d)", len(lines), start, end), nil)
	}

	var result strings.Builder
	for i := start - 1; i < end && i < len(lines); i++ {
		if i > start-1 {
			result.WriteString("\n")
		}
		result.WriteString(strings.TrimSpace(lines[i]))
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("no text content found in rows %d-%d", start, end), nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// readODSLines opens an ODS file and returns row strings.
// It parses content.xml from the ZIP archive when possible,
// otherwise falls back to splitting raw file text by newline.
func readODSLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, wrapError("Could not open ODS file:\n"+path+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return nil, wrapError("Could not open ODS file:\n"+path+"\n\nReason:\npermission denied", err)
		}
		return nil, wrapError("Could not open ODS file:\n"+path+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, wrapError("failed to get ODS file info", err)
	}

	zipReader, zipErr := zip.NewReader(file, fileInfo.Size())
	if zipErr == nil {
		var contentXML string
		for _, zf := range zipReader.File {
			if zf.Name == "content.xml" {
				rc, err := zf.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}
				contentXML = string(data)
				break
			}
		}

		if contentXML != "" {
			return extractODSRows(contentXML), nil
		}
	}

	// Fallback: raw text split by newline.
	file, err = os.Open(path)
	if err != nil {
		return nil, wrapError("Could not open ODS file for fallback:\n"+path+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, wrapError("failed to read ODS file", err)
	}
	if len(data) == 0 {
		return []string{}, nil
	}
	return strings.Split(string(data), "\n"), nil
}

// extractODSRows parses ODS content.xml text into rows by string scanning.
func extractODSRows(xmlContent string) []string {
	var rows []string
	content := xmlContent

	for {
		rs := strings.Index(content, "<table:table-row")
		if rs == -1 {
			break
		}
		re := strings.Index(content[rs:], "</table:table-row>")
		if re == -1 {
			break
		}
		re += rs + len("</table:table-row>")
		rowXML := content[rs:re]
		content = content[re:]

		var cells []string
		cellContent := rowXML
		for {
			cs := strings.Index(cellContent, "<table:table-cell")
			if cs == -1 {
				break
			}
			ce := strings.Index(cellContent[cs:], "</table:table-cell>")
			if ce == -1 {
				break
			}
			ce += cs + len("</table:table-cell>")
			cellXML := cellContent[cs:ce]
			cellContent = cellContent[ce:]

			ps := strings.Index(cellXML, "<text:p>")
			pe := strings.Index(cellXML, "</text:p>")
			if ps != -1 && pe != -1 {
				cells = append(cells, cellXML[ps+len("<text:p>"):pe])
			} else {
				cells = append(cells, "")
			}
		}

		if len(cells) == 0 {
			cells = []string{""}
		}
		rows = append(rows, strings.Join(cells, ", "))
	}

	if len(rows) == 0 {
		return []string{""}
	}
	return rows
}

// wrapError wraps an error with additional context.
func wrapError(message string, err error) error {
	if err == nil {
		return &ODSParserError{
			message: message,
			cause:   nil,
		}
	}
	return &ODSParserError{
		message: message,
		cause:   err,
	}
}
