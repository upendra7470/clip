package filetype

// FileType represents a supported document file type.
type FileType string

// Supported file types.
const (
	FileTypePDF      FileType = "PDF"
	FileTypeDOC      FileType = "DOC"
	FileTypeDOCX     FileType = "DOCX"
	FileTypeTXT      FileType = "TXT"
	FileTypeMarkdown FileType = "Markdown"
	FileTypeRTF      FileType = "RTF"
	FileTypeODT      FileType = "ODT"
	FileTypeCSV      FileType = "CSV"
	FileTypeXLS      FileType = "XLS"
	FileTypeXLSX     FileType = "XLSX"
	FileTypeODS      FileType = "ODS"
	FileTypePPT      FileType = "PPT"
	FileTypePPTX     FileType = "PPTX"
	FileTypeODP      FileType = "ODP"
	FileTypeJSON     FileType = "JSON"
	FileTypeXML      FileType = "XML"
	FileTypeHTML     FileType = "HTML"
)
