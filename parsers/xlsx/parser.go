package xlsx

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

// Parser implements the parser.Parser interface for XLSX files.
type Parser struct{}

// Parse reads an XLSX file and extracts text content.
// XLSX files are ZIP archives containing XML files.
// This parser extracts data from xl/sharedStrings.xml and xl/worksheets/sheet*.xml.
func (p *Parser) Parse(ctx context.Context, req parser.ParseRequest) (parser.ParseResult, error) {
	// Open the XLSX file (which is a ZIP archive)
	file, err := os.Open(req.File)
	if err != nil {
		if os.IsNotExist(err) {
			return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\nfile does not exist", err)
		}
		if os.IsPermission(err) {
			return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\npermission denied", err)
		}
		return parser.ParseResult{}, wrapError("Could not open XLSX file:\n"+req.File+"\n\nReason:\n"+err.Error(), err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to get file info", err)
	}

	// Read the ZIP archive
	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to read XLSX as ZIP archive", err)
	}

	// Parse shared strings
	sharedStrings, err := parseSharedStrings(zipReader)
	if err != nil {
		return parser.ParseResult{}, wrapError("failed to parse shared strings", err)
	}

	// Parse all worksheets
	var result strings.Builder
	sheetFound := false

	for _, zipFile := range zipReader.File {
		if strings.HasPrefix(zipFile.Name, "xl/worksheets/sheet") && strings.HasSuffix(zipFile.Name, ".xml") {
			sheetFound = true
			sheetData, err := parseWorksheet(zipFile, sharedStrings)
			if err != nil {
				return parser.ParseResult{}, wrapError("failed to parse worksheet", err)
			}
			if result.Len() > 0 {
				result.WriteString("\n\n")
			}
			result.WriteString(sheetData)
		}
	}

	if !sheetFound {
		return parser.ParseResult{}, wrapError("no worksheets found in XLSX", nil)
	}

	if result.Len() == 0 {
		return parser.ParseResult{}, wrapError("no content found in XLSX", nil)
	}

	return parser.ParseResult{
		Text: result.String(),
	}, nil
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() filetype.FileType {
	return filetype.FileTypeXLSX
}

// parseSharedStrings parses the shared strings table from xl/sharedStrings.xml.
func parseSharedStrings(zipReader *zip.Reader) ([]string, error) {
	var sharedStrings []string

	for _, zipFile := range zipReader.File {
		if zipFile.Name == "xl/sharedStrings.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				return nil, wrapError("failed to open sharedStrings.xml", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, wrapError("failed to read sharedStrings.xml", err)
			}

			// Parse the XML to extract shared strings
			decoder := xml.NewDecoder(strings.NewReader(string(content)))
			var inSi, inT bool
			var currentString strings.Builder

			for {
				token, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, wrapError("failed to parse sharedStrings.xml", err)
				}

				switch t := token.(type) {
				case xml.StartElement:
					if t.Name.Local == "si" {
						inSi = true
						currentString.Reset()
					} else if t.Name.Local == "t" && inSi {
						inT = true
					}
				case xml.CharData:
					if inT {
						currentString.Write(t)
					}
				case xml.EndElement:
					if t.Name.Local == "t" && inT {
						inT = false
					} else if t.Name.Local == "si" && inSi {
						inSi = false
						sharedStrings = append(sharedStrings, currentString.String())
					}
				}
			}
			break
		}
	}

	return sharedStrings, nil
}

// parseWorksheet parses a worksheet XML file and extracts cell data.
func parseWorksheet(zipFile *zip.File, sharedStrings []string) (string, error) {
	rc, err := zipFile.Open()
	if err != nil {
		return "", wrapError("failed to open worksheet", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", wrapError("failed to read worksheet", err)
	}

	var result strings.Builder
	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	var inRow, inC, inIS, inT bool
	var currentRow strings.Builder
	var currentCell strings.Builder
	var cellType string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", wrapError("failed to parse worksheet", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "row" {
				inRow = true
				currentRow.Reset()
			} else if t.Name.Local == "c" && inRow {
				inC = true
				currentCell.Reset()
				// Get cell type
				cellType = ""
				for _, attr := range t.Attr {
					if attr.Name.Local == "t" {
						cellType = attr.Value
					}
				}
			} else if t.Name.Local == "is" && inC {
				inIS = true
			} else if t.Name.Local == "t" && (inIS || inC) {
				inT = true
			} else if t.Name.Local == "v" && inC {
				// Cell value - only relevant for non-inline-string cells
				// For inline strings, we get the value from <t> elements
			}
		case xml.CharData:
			if inT {
				currentCell.Write(t)
			}
		case xml.EndElement:
			if t.Name.Local == "t" && inT {
				inT = false
			} else if t.Name.Local == "is" && inIS {
				inIS = false
			} else if t.Name.Local == "c" && inC {
				// Process cell value based on type
				cellValue := strings.TrimSpace(currentCell.String())
				if cellType == "s" {
					// Shared string reference
					if idx, err := strconv.Atoi(cellValue); err == nil && idx < len(sharedStrings) {
						cellValue = sharedStrings[idx]
					}
				} else if cellType == "inlineStr" {
					// Inline string - value is already in currentCell from <t> element
				} else {
					// Numeric or other types - value would be in <v> element
					// But for inline strings, we already have the value
				}

				if cellValue != "" { // Only add non-empty cells
					if currentRow.Len() > 0 {
						currentRow.WriteString(", ")
					}
					currentRow.WriteString(cellValue)
				}

				inC = false
				inIS = false
			} else if t.Name.Local == "row" && inRow {
				inRow = false
				if currentRow.Len() > 0 { // Only add non-empty rows
					if result.Len() > 0 {
						result.WriteString("\n")
					}
					result.WriteString(currentRow.String())
				}
			}
		}
	}

	return result.String(), nil
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
