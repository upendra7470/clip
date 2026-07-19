package detect

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/upendra7470/clip/internal/filetype"
)

// Type determines the file type from the given path based on file extension.
// It is case-insensitive and does not access the filesystem.
func Type(path string) (filetype.FileType, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "", fmt.Errorf("no file extension found")
	}

	// Remove leading dot
	ext = strings.TrimPrefix(ext, ".")

	// Map extensions to file types
	switch ext {
	case "pdf":
		return filetype.FileTypePDF, nil
	case "doc":
		return filetype.FileTypeDOC, nil
	case "docx":
		return filetype.FileTypeDOCX, nil
	case "txt":
		return filetype.FileTypeTXT, nil
	case "md", "markdown":
		return filetype.FileTypeMarkdown, nil
	case "rtf":
		return filetype.FileTypeRTF, nil
	case "odt":
		return filetype.FileTypeODT, nil
	case "csv":
		return filetype.FileTypeCSV, nil
	case "xls":
		return filetype.FileTypeXLS, nil
	case "xlsx":
		return filetype.FileTypeXLSX, nil
	case "ods":
		return filetype.FileTypeODS, nil
	case "ppt":
		return filetype.FileTypePPT, nil
	case "pptx":
		return filetype.FileTypePPTX, nil
	case "odp":
		return filetype.FileTypeODP, nil
	case "json":
		return filetype.FileTypeJSON, nil
	case "xml":
		return filetype.FileTypeXML, nil
	case "html", "htm":
		return filetype.FileTypeHTML, nil
	default:
		return "", fmt.Errorf("unsupported file extension: .%s", ext)
	}
}
