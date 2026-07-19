package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/upendra7470/clip/internal/application"
	"github.com/upendra7470/clip/internal/clipboard"
	"github.com/upendra7470/clip/internal/registry"
	"github.com/upendra7470/clip/parsers/csv"
	"github.com/upendra7470/clip/parsers/docx"
	"github.com/upendra7470/clip/parsers/html"
	"github.com/upendra7470/clip/parsers/json"
	"github.com/upendra7470/clip/parsers/markdown"
	"github.com/upendra7470/clip/parsers/pdf"
	"github.com/upendra7470/clip/parsers/rtf"
	"github.com/upendra7470/clip/parsers/txt"
	"github.com/upendra7470/clip/parsers/xlsx"
	"github.com/upendra7470/clip/parsers/xml"
	"github.com/upendra7470/clip/parsers/yaml"
)

const version = "dev"

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
		fmt.Printf("Clip version %s\n", version)
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

	// Run the extraction pipeline
	if err := app.Extract(ctx, filePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Success
	fmt.Println("✓ Copied to clipboard.")
}

// realClipboard adapts the clipboard package to the application.Clipboard interface.
type realClipboard struct{}

func (r *realClipboard) Copy(text string) error {
	return clipboard.Copy(text)
}

func showHelp() {
	fmt.Println("Clip")
	fmt.Println()
	fmt.Println("A fast CLI for extracting text from documents.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  clip [file]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --help, -h    Show help message")
	fmt.Println("  --version     Show version information")
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
