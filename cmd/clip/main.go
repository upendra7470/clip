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
	"github.com/upendra7470/clip/internal/detect"
	"github.com/upendra7470/clip/internal/parser"
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

	// Get the file path and optional range argument
	filePath, rangeArg := getFilePathAndRange()
	if filePath == "" {
		fmt.Fprintf(os.Stderr, "No file specified\n")
		showHelp()
		os.Exit(1)
	}

	// Parse optional range argument
	var rangeObj *parser.Range
	if rangeArg != "" {
		parsedRange, err := parser.ParseRange(rangeArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			showHelp()
			os.Exit(1)
		}
		rangeObj = &parsedRange
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

	// Detect file type to determine range unit for success message
	fileType, err := detect.Type(resolvedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Lookup parser to get range unit
	parserObj, err := reg.Lookup(fileType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Run the extraction pipeline
	var extractErr error
	if rangeObj != nil {
		extractErr = app.ExtractWithRange(ctx, resolvedPath, rangeObj)
	} else {
		extractErr = app.Extract(ctx, resolvedPath)
	}
	if extractErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", extractErr)
		os.Exit(1)
	}

	// Success
	fmt.Printf("✓ Found: %s\n", resolvedPath)
	if rangeObj != nil {
		// Determine the correct range unit based on file type
		rangeUnit := "pages" // default
		if rangeParser, ok := parserObj.(parser.RangeParser); ok {
			rangeUnit = rangeParser.GetRangeUnit()
		}

		fmt.Printf("✓ Extracted %s %d-%d successfully\n", rangeUnit, rangeObj.Start, rangeObj.End)
	} else {
		fmt.Println("✓ Extracted text successfully")
	}
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
	fmt.Println("    clip <filename>")
	fmt.Println("    clip <filename> <range>")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("    clip report.pdf")
	fmt.Println("    clip report.pdf 5")
	fmt.Println("    clip report.pdf 5-10")
	fmt.Println("    clip \"The Brain.docx\"")
	fmt.Println("    clip \"The Brain.docx\" 5-10")
	fmt.Println("    clip presentation.pptx 5-10")
	fmt.Println("    clip file.txt 5-10")
	fmt.Println()
	fmt.Println("Range units by file type:")
	fmt.Println("  PDF    -> pages")
	fmt.Println("  DOCX   -> paragraphs")
	fmt.Println("  PPTX   -> slides")
	fmt.Println("  TXT    -> lines")
	fmt.Println("  Markdown -> lines")
	fmt.Println()
	fmt.Println("Note: Remember to quote filenames containing spaces.")
	fmt.Println("      Example: clip \"The Brain.docx\" 5-10")
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

// getFilePathAndRange extracts the file path and optional range argument from command line arguments.
// This function handles quoted filenames with spaces and intelligently distinguishes between
// filename words and range arguments.
func getFilePathAndRange() (string, string) {
	nonFlagArgs := []string{}
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" || arg == "--version" {
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			continue // Skip other flags
		}
		nonFlagArgs = append(nonFlagArgs, arg)
	}

	// If no arguments, return empty
	if len(nonFlagArgs) == 0 {
		return "", ""
	}

	// If only one argument, it's the filename
	if len(nonFlagArgs) == 1 {
		return nonFlagArgs[0], ""
	}

	// If multiple arguments, we need to determine if the last argument is a range
	// A range argument should contain digits and optionally a dash
	lastArg := nonFlagArgs[len(nonFlagArgs)-1]
	if isRangeArgument(lastArg) {
		// Last argument is a range, join the rest as filename
		filename := strings.Join(nonFlagArgs[:len(nonFlagArgs)-1], " ")
		return filename, lastArg
	}

	// If last argument is not a range, treat all arguments as filename
	filename := strings.Join(nonFlagArgs, " ")
	return filename, ""
}

// isRangeArgument checks if an argument looks like a range specification.
// A valid range contains digits and optionally a dash (e.g., "5", "5-10").
func isRangeArgument(arg string) bool {
	// Remove any quotes from the argument
	arg = strings.Trim(arg, `"'`)

	// Check if it contains only digits and optionally a dash
	hasDigits := false
	hasOtherChars := false

	for _, c := range arg {
		if c >= '0' && c <= '9' {
			hasDigits = true
		} else if c != '-' {
			hasOtherChars = true
		}
	}

	// Valid range: has digits, may have dashes, no other characters
	return hasDigits && !hasOtherChars
}
