package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/upendra7470/clip/internal/application"
	"github.com/upendra7470/clip/internal/clipboard"
	"github.com/upendra7470/clip/internal/registry"
	"github.com/upendra7470/clip/internal/resolver"
	"github.com/upendra7470/clip/parsers/csv"
	"github.com/upendra7470/clip/parsers/docx"
	"github.com/upendra7470/clip/parsers/html"
	"github.com/upendra7470/clip/parsers/json"
	"github.com/upendra7470/clip/parsers/markdown"
	"github.com/upendra7470/clip/parsers/ods"
	"github.com/upendra7470/clip/parsers/odt"
	"github.com/upendra7470/clip/parsers/pdf"
	"github.com/upendra7470/clip/parsers/ppt"
	"github.com/upendra7470/clip/parsers/pptx"
	"github.com/upendra7470/clip/parsers/rtf"
	"github.com/upendra7470/clip/parsers/txt"
	"github.com/upendra7470/clip/parsers/xlsx"
	"github.com/upendra7470/clip/parsers/xml"
	"github.com/upendra7470/clip/parsers/yaml"
)

const version = "1.0.0"

func main() {
	// Parse command line flags
	helpFlag := flag.Bool("help", false, "Show help message")
	hFlag := flag.Bool("h", false, "Show help message")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show help if requested
	if *helpFlag || *hFlag {
		showHelp()
		return
	}

	// Show version if requested
	if *versionFlag {
		fmt.Printf("Clip v%s\n", version)
		return
	}

	// Set up the parser registry
	reg := registry.New()

	// Register TXT parser
	txtParser := &txt.Parser{}
	if err := reg.Register(txtParser.FileType(), txtParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register TXT parser: %v\n", err)
		os.Exit(1)
	}

	// Register Markdown parser
	markdownParser := &markdown.Parser{}
	if err := reg.Register(markdownParser.FileType(), markdownParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register Markdown parser: %v\n", err)
		os.Exit(1)
	}

	// Register PDF parser
	pdfParser := &pdf.Parser{}
	if err := reg.Register(pdfParser.FileType(), pdfParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register PDF parser: %v\n", err)
		os.Exit(1)
	}

	// Register DOCX parser
	docxParser := &docx.Parser{}
	if err := reg.Register(docxParser.FileType(), docxParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register DOCX parser: %v\n", err)
		os.Exit(1)
	}

	// Register PPT parser
	pptParser := &ppt.Parser{}
	if err := reg.Register(pptParser.FileType(), pptParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register PPT parser: %v\n", err)
		os.Exit(1)
	}

	// Register PPTX parser
	pptxParser := &pptx.Parser{}
	if err := reg.Register(pptxParser.FileType(), pptxParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register PPTX parser: %v\n", err)
		os.Exit(1)
	}

	// Register CSV parser
	csvParser := &csv.Parser{}
	if err := reg.Register(csvParser.FileType(), csvParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register CSV parser: %v\n", err)
		os.Exit(1)
	}

	// Register XLSX parser
	xlsxParser := &xlsx.Parser{}
	if err := reg.Register(xlsxParser.FileType(), xlsxParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register XLSX parser: %v\n", err)
		os.Exit(1)
	}

	// Register JSON parser
	jsonParser := &json.Parser{}
	if err := reg.Register(jsonParser.FileType(), jsonParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register JSON parser: %v\n", err)
		os.Exit(1)
	}

	// Register XML parser
	xmlParser := &xml.Parser{}
	if err := reg.Register(xmlParser.FileType(), xmlParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register XML parser: %v\n", err)
		os.Exit(1)
	}

	// Register HTML parser
	htmlParser := &html.Parser{}
	if err := reg.Register(htmlParser.FileType(), htmlParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register HTML parser: %v\n", err)
		os.Exit(1)
	}

	// Register YAML parser
	yamlParser := &yaml.Parser{}
	if err := reg.Register(yamlParser.FileType(), yamlParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register YAML parser: %v\n", err)
		os.Exit(1)
	}

	// Register RTF parser
	rtfParser := &rtf.Parser{}
	if err := reg.Register(rtfParser.FileType(), rtfParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register RTF parser: %v\n", err)
		os.Exit(1)
	}

	// Register ODT parser
	odtParser := &odt.Parser{}
	if err := reg.Register(odtParser.FileType(), odtParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register ODT parser: %v\n", err)
		os.Exit(1)
	}

	// Register ODS parser
	odsParser := &ods.Parser{}
	if err := reg.Register(odsParser.FileType(), odsParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register ODS parser: %v\n", err)
		os.Exit(1)
	}

	// Create resolver
	fileResolver := resolver.New()

	// Create clipboard adapter
	clipboardAdapter := &realClipboard{}

	// Create application with registry and clipboard
	app := application.New(reg, clipboardAdapter)

	// Handle file argument
	if len(os.Args) == 1 {
		showHelp()
		return
	}

	// Get the file path (first non-flag argument)
	filePath := getFilePath()
	if filePath == "" {
		fmt.Fprintf(os.Stderr, "No file specified\n")
		showHelp()
		os.Exit(1)
	}

	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Resolve file path using the resolver
	resolvedPath, err := fileResolver.Resolve(ctx, filePath)
	if err != nil {
		// Handle special case for multiple files selection
		if strings.HasPrefix(err.Error(), "selected:") {
			// Extract the selected file path
			selectedPath := strings.TrimPrefix(err.Error(), "selected:")
			resolvedPath = selectedPath
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Run the extraction pipeline
	if err := app.Extract(ctx, resolvedPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Success
	fmt.Printf("✓ Found: %s\n", resolvedPath)
	fmt.Println("✓ Extracted text successfully")
	fmt.Println("✓ Copied to clipboard")
}

// realClipboard adapts the clipboard package to the application.Clipboard interface.
type realClipboard struct{}

func (r *realClipboard) Copy(text string) error {
	return clipboard.Copy(text)
}

func showHelp() {
	fmt.Println("Clip - Universal Document Extractor")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("    clip <filename>")
	fmt.Println()
	fmt.Println("Supported formats:")
	fmt.Println()
	fmt.Println("TXT")
	fmt.Println("MD")
	fmt.Println("PDF")
	fmt.Println("DOCX")
	fmt.Println("CSV")
	fmt.Println("XLSX")
	fmt.Println("JSON")
	fmt.Println("XML")
	fmt.Println("HTML")
	fmt.Println("YAML")
	fmt.Println("RTF")
	fmt.Println("ODT")
	fmt.Println("ODS")
	fmt.Println("PPTX")
	fmt.Println("PPT")
}

// getFilePath extracts the file path from command line arguments.
// Returns the first non-flag argument.
func getFilePath() string {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" || arg == "--version" {
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			continue // Skip other flags
		}
		return arg
	}
	return ""
}
