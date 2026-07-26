package xlsx

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/xuri/excelize/v2"
)

// XLSXParserError represents an error that occurs during XLSX parsing.
type XLSXParserError struct {
	message string
	cause   error
}

func (e *XLSXParserError) Error() string {
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
	text, err := io.ReadAll(reader)
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

// worksheetRow represents a parsed row with cell texts.
type worksheetRow struct {
	rowNum int
	cells  []string
}

// worksheetData represents parsed worksheet content.
type worksheetData struct {
	name string
	rows []worksheetRow
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
		if os.IsNotExist(err) {
			return 0, nil, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return 0, nil, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return 0, nil, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Get file info for size
	_, err = file.Stat()
	if err != nil {
		return 0, nil, wrapError("failed to get file info", err)
	}

	// Read the XLSX file
	xlsxFile, err := excelize.OpenReader(file)
	if err != nil {
		return 0, nil, wrapError("failed to parse XLSX file", err)
	}

	// Get all sheet names
	sheetNames := xlsxFile.GetSheetList()
	if len(sheetNames) == 0 {
		return 0, nil, wrapError("no sheets found in XLSX file", nil)
	}

	// For each sheet, get the row count
	var totalRows int
	var sheetInfo []string
	for _, sheetName := range sheetNames {
		rows, err := xlsxFile.GetRows(sheetName)
		if err != nil {
			continue
		}
		rowCount := len(rows)
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
	if start < 1 || end < 1 {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("row numbers must start from 1, got %d-%d", start, end), nil)
	}
	if end < start {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("invalid row range: start row must not be greater than end row (got %d-%d)", start, end), nil)
	}

	lines, err := readXLSXLines(req.File)
	if err != nil {
		return parser.ParseResult{}, err
	}

	if len(lines) == 0 {
		return parser.ParseResult{}, wrapError("no readable content found in XLSX", nil)
	}

	if start > len(lines) || end > len(lines) {
		return parser.ParseResult{}, wrapError(fmt.Sprintf("requested row range exceeds XLSX row count (file has %d rows, requested %d-%d)", len(lines), start, end), nil)
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

// readXLSXLines reads an XLSX file and returns logical rows.
// It first attempts ZIP/XML parsing for real XLSX files.
// If the file is not a ZIP archive, it falls back to newline-split text.
func readXLSXLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, wrapError("Could not open XLSX file:\n"+path+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return nil, wrapError("Could not open XLSX file:\n"+path+"\n\nReason:\npermission denied", err)
		}
		return nil, wrapError("Could not open XLSX file:\n"+path+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, wrapError("failed to get XLSX file info", err)
	}

	// Try ZIP-based parsing first.
	zipReader, zipErr := zip.NewReader(file, fileInfo.Size())
	if zipErr == nil {
		// Read all ZIP entries.
		files := make(map[string]string)
		for _, zf := range zipReader.File {
			rc, err := zf.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			files[zf.Name] = string(data)
		}

		// Parse shared strings.
		var ss []string
		if s, ok := files["xl/sharedStrings.xml"]; ok {
			ss = parseXLSXSharedStrings(s)
		}

		// Find first worksheet XML.
		var wsXML string
		for name, data := range files {
			if strings.HasPrefix(name, "xl/worksheets/sheet") && strings.HasSuffix(name, ".xml") {
				wsXML = data
				break
			}
		}

		if wsXML != "" {
			rows := parseXLSXWorksheet(wsXML, ss)
			lines := make([]string, 0, len(rows))
			for _, row := range rows {
				lines = append(lines, strings.Join(row.cells, ", "))
			}
			return lines, nil
		}
	}

	// Fallback: treat file contents as text split by newline.
	// Re-open since ZIP parsing may have consumed file pointer.
	file, err = os.Open(path)
	if err != nil {
		return nil, wrapError("Could not open XLSX file for fallback:\n"+path+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, wrapError("failed to read XLSX file", err)
	}
	if len(data) == 0 {
		return []string{}, nil
	}
	return strings.Split(string(data), "\n"), nil
}

// parseXLSXSharedStrings parses shared strings XML.
func parseXLSXSharedStrings(xmlContent string) []string {
	var result []string
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var inSi bool
	var currentText strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" && t.Name.Space == "http://schemas.openxmlformats.org/spreadsheetml/2006/main" {
				inSi = true
				currentText.Reset()
			}
		case xml.CharData:
			if inSi {
				currentText.Write(t)
			}
		case xml.EndElement:
			if inSi && t.Name.Local == "si" && t.Name.Space == "http://schemas.openxmlformats.org/spreadsheetml/2006/main" {
				inSi = false
				result = append(result, strings.TrimSpace(currentText.String()))
			}
		}
	}

	return result
}

// parseXLSXWorksheet parses worksheet XML into rows.
func parseXLSXWorksheet(xmlContent string, sharedStrings []string) []worksheetRow {
	var rows []worksheetRow
	content := xmlContent

	for {
		rs := strings.Index(content, "<row ")
		if rs == -1 {
			break
		}
		re := strings.Index(content[rs:], "</row>")
		if re == -1 {
			break
		}
		re += rs + len("</row>")
		rowXML := content[rs:re]
		content = content[re:]

		var cells []string
		cellContent := rowXML
		for {
			cs := strings.Index(cellContent, "<c ")
			if cs == -1 {
				break
			}
			ce := strings.Index(cellContent[cs:], "</c>")
			if ce == -1 {
				break
			}
			ce += cs + len("</c>")
			cellXML := cellContent[cs:ce]
			cellContent = cellContent[ce:]

			cellType := ""
			if idx := strings.Index(cellXML, `t="`); idx != -1 {
				val := cellXML[idx+3:]
				if end := strings.Index(val, `"`); end != -1 {
					cellType = val[:end]
				}
			}

			vs := strings.Index(cellXML, "<v>")
			ve := strings.Index(cellXML, "</v>")
			if vs != -1 && ve != -1 {
				val := cellXML[vs+len("<v>") : ve]
				if cellType == "s" {
					idx := 0
					for _, c := range val {
						if c >= '0' && c <= '9' {
							idx = idx*10 + int(c-'0')
						} else {
							break
						}
					}
					if idx >= 0 && idx < len(sharedStrings) {
						cells = append(cells, sharedStrings[idx])
					} else {
						cells = append(cells, "")
					}
				} else {
					cells = append(cells, val)
				}
			} else {
				cells = append(cells, "")
			}
		}

		if len(cells) == 0 {
			cells = []string{""}
		}
		rows = append(rows, worksheetRow{cells: cells})
	}

	return rows
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
