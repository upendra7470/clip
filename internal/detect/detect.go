package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
)

// Type determines the file type from the given path. It first checks the file
// extension, then validates against magic bytes when the file is readable.
// If the magic bytes contradict the extension, the magic-byte type wins.
func Type(path string) (filetype.FileType, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "", fmt.Errorf("no file extension found")
	}
	ext = strings.TrimPrefix(ext, ".")

	extType := typeFromExtension(ext)

	// Attempt magic-byte validation for ambiguous or unsupported extensions.
	f, err := os.Open(path)
	if err != nil {
		// Can't read the file — fall back to extension-only.
		if extType != "" {
			return extType, nil
		}
		return "", fmt.Errorf("unsupported file extension: .%s", ext)
	}
	defer f.Close()

	header := make([]byte, 16)
	n, _ := f.Read(header)
	header = header[:n]

	magicType := typeFromMagic(header)

	if magicType != "" && extType != magicType {
		// Magic bytes disagree with extension — trust the bytes.
		return magicType, nil
	}
	if extType != "" {
		return extType, nil
	}
	if magicType != "" {
		return magicType, nil
	}

	return "", fmt.Errorf("unsupported file type: .%s", ext)
}

// typeFromExtension maps a lowercase extension (without dot) to a FileType.
func typeFromExtension(ext string) filetype.FileType {
	switch ext {
	case "pdf":
		return filetype.FileTypePDF
	case "doc":
		return filetype.FileTypeDOC
	case "docx":
		return filetype.FileTypeDOCX
	case "txt":
		return filetype.FileTypeTXT
	case "md", "markdown":
		return filetype.FileTypeMarkdown
	case "rtf":
		return filetype.FileTypeRTF
	case "odt":
		return filetype.FileTypeODT
	case "csv":
		return filetype.FileTypeCSV
	case "xls":
		return filetype.FileTypeXLS
	case "xlsx":
		return filetype.FileTypeXLSX
	case "ods":
		return filetype.FileTypeODS
	case "ppt":
		return filetype.FileTypePPT
	case "pptx":
		return filetype.FileTypePPTX
	case "odp":
		return filetype.FileTypeODP
	case "json":
		return filetype.FileTypeJSON
	case "xml":
		return filetype.FileTypeXML
	case "html", "htm":
		return filetype.FileTypeHTML
	case "yaml", "yml":
		return filetype.FileTypeYAML
	default:
		return ""
	}
}

// typeFromMagic inspects the first bytes of a file and returns a FileType
// when the magic signature is unambiguous. Returns "" for ambigous cases
// (e.g. ZIP, which could be DOCX/XLSX/PPTX/ODS/ODT).
func typeFromMagic(header []byte) filetype.FileType {
	if len(header) < 4 {
		return ""
	}

	// PDF: %PDF
	if header[0] == '%' && header[1] == 'P' && header[2] == 'D' && header[3] == 'F' {
		return filetype.FileTypePDF
	}

	// RTF: {\rtf
	if header[0] == '{' && header[1] == '\\' && header[2] == 'r' && header[3] == 't' && header[4] == 'f' {
		return filetype.FileTypeRTF
	}

	// PNG: 0x89 P N G
	if header[0] == 0x89 && header[1] == 'P' && header[2] == 'N' && header[3] == 'G' {
		return "" // image type not registered without OCR
	}

	// JPEG: 0xFF 0xD8 0xFF
	if header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return "" // image type not registered without OCR
	}

	// GIF: GIF8
	if header[0] == 'G' && header[1] == 'I' && header[2] == 'F' && header[3] == '8' {
		return "" // image type not registered without OCR
	}

	// TIFF: II*\0 or MM\0*
	if (header[0] == 'I' && header[1] == 'I' && header[2] == 0x2A && header[3] == 0x00) ||
		(header[0] == 'M' && header[1] == 'M' && header[2] == 0x00 && header[3] == 0x2A) {
		return "" // image type not registered without OCR
	}

	// WebP: RIFF....WEBP
	if len(header) >= 12 &&
		header[0] == 'R' && header[1] == 'I' && header[2] == 'F' && header[3] == 'F' &&
		header[8] == 'W' && header[9] == 'E' && header[10] == 'B' && header[11] == 'P' {
		return "" // image type not registered without OCR
	}

	// ZIP: PK\x03\x04 — could be DOCX, XLSX, PPTX, ODS, ODT, ODP.
	// Return empty; extension wins for these.
	if header[0] == 'P' && header[1] == 'K' && header[2] == 0x03 && header[3] == 0x04 {
		return ""
	}

	// JSON: starts with { or [
	if header[0] == '{' || header[0] == '[' {
		return filetype.FileTypeJSON
	}

	// XML: starts with < (with optional BOM/whitespace)
	if header[0] == '<' {
		return filetype.FileTypeXML
	}

	return ""
}
