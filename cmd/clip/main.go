package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	"github.com/upendra7470/clip/parsers/pdf"
	"github.com/upendra7470/clip/parsers/ppt"
	"github.com/upendra7470/clip/parsers/pptx"
	"github.com/upendra7470/clip/parsers/txt"
	"github.com/upendra7470/clip/parsers/xlsx"
	"github.com/upendra7470/clip/parsers/xml"
)

const version = "1.0.0"

func Run(args []string, stdout, stderr io.Writer) int {
	// Create a new FlagSet to avoid flag redefinition
	flagSet := flag.NewFlagSet("clip", flag.ContinueOnError)

	// Parse command line flags
	helpFlag := flagSet.Bool("help", false, "Show help message")
	hFlag := flagSet.Bool("h", false, "Show help message")
	versionFlag := flagSet.Bool("version", false, "Show version information")
	listFlag := flagSet.Bool("list", false, "List document units (pages, sections, slides, etc.)")
	inspectFlag := flagSet.Bool("inspect", false, "Inspect document units (pages, sections, slides, etc.)")
	err := flagSet.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	// Show help if requested
	if *helpFlag || *hFlag {
		showHelp()
		return 0
	}

	// Show version if requested
	if *versionFlag {
		fmt.Fprintf(stdout, "clip v%s\n", version)
		return 0
	}

	// Set up the parser registry
	reg := registry.New()

	// Register TXT parser
	txtParser := &txt.Parser{}
	if err := reg.Register(txtParser.FileType(), txtParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register TXT parser: %v\n", err)
		return 1
	}

	// Register Markdown parser
	markdownParser := &markdown.Parser{}
	if err := reg.Register(markdownParser.FileType(), markdownParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register Markdown parser: %v\n", err)
		return 1
	}

	// Register PDF parser
	pdfParser := &pdf.Parser{}
	if err := reg.Register(pdfParser.FileType(), pdfParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register PDF parser: %v\n", err)
		return 1
	}

	// Register DOCX parser
	docxParser := &docx.Parser{}
	if err := reg.Register(docxParser.FileType(), docxParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register DOCX parser: %v\n", err)
		return 1
	}

	// Register PPT parser
	pptParser := &ppt.Parser{}
	if err := reg.Register(pptParser.FileType(), pptParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register PPT parser: %v\n", err)
		return 1
	}

	// Register PPTX parser
	pptxParser := &pptx.Parser{}
	if err := reg.Register(pptxParser.FileType(), pptxParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register PPTX parser: %v\n", err)
		return 1
	}

	// Register CSV parser
	csvParser := &csv.Parser{}
	if err := reg.Register(csvParser.FileType(), csvParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register CSV parser: %v\n", err)
		return 1
	}

	// Register XLSX parser
	xlsxParser := &xlsx.Parser{}
	if err := reg.Register(xlsxParser.FileType(), xlsxParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register XLSX parser: %v\n", err)
		return 1
	}

	// Register JSON parser
	jsonParser := &json.Parser{}

	// Register XML parser
	xmlParser := &xml.Parser{}
	if err := reg.Register(xmlParser.FileType(), xmlParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register XML parser: %v\n", err)
		return 1
	}

	// Register HTML parser
	htmlParser := &html.Parser{}
	if err := reg.Register(htmlParser.FileType(), htmlParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register HTML parser: %v\n", err)
		return 1
	}

	if err := reg.Register(jsonParser.FileType(), jsonParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register JSON parser: %v\n", err)
		return 1
	}

	// Create resolver
	fileResolver := resolver.New()

	// Create clipboard adapter
	clipboardAdapter := &realClipboard{}

	// Create application with registry and clipboard
	app := application.New(reg, clipboardAdapter)

	// Handle file argument
	if len(flagSet.Args()) == 0 {
		showHelp()
		return 1
	}

	// Get the file path and optional range argument
	filePath, rangeArg := getFilePathAndRangeFromArgs(flagSet.Args())
	if filePath == "" {
		fmt.Fprintf(stderr, "No file specified\n")
		showHelp()
		return 1
	}

	// Parse optional range argument
	var rangeObj *parser.Range
	if rangeArg != "" {
		parsedRange, err := parser.ParseRange(rangeArg)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			showHelp()
			return 1
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
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
	}

	// Detect file type to determine range unit for success message
	fileType, err := detect.Type(resolvedPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	// Lookup parser to get range unit
	parserObj, err := reg.Lookup(fileType)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	// Handle --list or --inspect flag: inspect document units
	if *listFlag || *inspectFlag {
		listDocumentUnits(resolvedPath, parserObj, ctx)
		return 0
	}

	// Run the extraction pipeline
	var extractErr error
	if rangeObj != nil {
		extractErr = app.ExtractWithRange(ctx, resolvedPath, rangeObj)
	} else {
		extractErr = app.Extract(ctx, resolvedPath)
	}
	if extractErr != nil {
		fmt.Fprintf(stderr, "Error: %v\n", extractErr)
		return 1
	}

	// Success
	fmt.Fprintf(stdout, "✓ Found: %s\n", resolvedPath)
	if rangeObj != nil {
		// Determine the correct range unit based on file type
		rangeUnit := parser.RangeUnit("blocks") // default
		if rangeParser, ok := parserObj.(parser.RangeParser); ok {
			rangeUnit = rangeParser.GetRangeUnit()
		}

		// Handle special range formats
		if rangeObj.Start == -1 {
			fmt.Fprintf(stdout, "✓ Extracted from start to %s %d successfully\n", rangeUnit, rangeObj.End)
		} else if rangeObj.End == -1 {
			fmt.Fprintf(stdout, "✓ Extracted from %s %d to end successfully\n", rangeUnit, rangeObj.Start)
		} else {
			fmt.Fprintf(stdout, "✓ Extracted %s %d-%d successfully\n", rangeUnit, rangeObj.Start, rangeObj.End)
		}
	} else {
		fmt.Fprintln(stdout, "✓ Extracted text successfully")
	}
	fmt.Fprintln(stdout, "✓ Copied to clipboard")

	return 0
}

func main() {
	// Run the application with os.Args and standard output/error streams
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

// listDocumentUnits prints the available units in a document.
func listDocumentUnits(resolvedPath string, parserObj parser.Parser, ctx context.Context) {
	fmt.Printf("Document: %s\n", resolvedPath)

	// Detect format from file extension
	ext := strings.ToLower(filepath.Ext(resolvedPath))
	ext = strings.TrimPrefix(ext, ".")
	fmt.Printf("Format: %s\n", strings.ToUpper(ext))

	// Get range unit
	rangeUnit := parser.RangeUnit("blocks")
	if rangeParser, ok := parserObj.(parser.RangeParser); ok {
		rangeUnit = rangeParser.GetRangeUnit()
	}
	fmt.Printf("Range unit: %s\n", rangeUnit)

	// Try to list units
	if lister, ok := parserObj.(parser.DocumentLister); ok {
		total, unitNames, err := lister.ListUnits(ctx, parser.ParseRequest{File: resolvedPath})
		if err != nil {
			fmt.Printf("Error listing units: %v\n", err)
			return
		}

		// Print unit names
		if len(unitNames) > 0 {
			fmt.Printf("%s:\n", capitalizeUnit(rangeUnit))
			for i, name := range unitNames {
				if name != "" {
					fmt.Printf("%d. %s\n", i+1, name)
				} else {
					fmt.Printf("%d.\n", i+1)
				}
			}
		} else {
			fmt.Printf("%s: %d\n", capitalizeUnit(rangeUnit), total)
		}
	} else {
		fmt.Printf("%s: %d\n", capitalizeUnit(rangeUnit), 0)
	}
}

// capitalizeUnit returns the plural form of the unit name for display.
func capitalizeUnit(unit parser.RangeUnit) string {
	switch unit {
	case parser.RangeUnitPages:
		return "Pages"
	case parser.RangeUnitSections:
		return "Sections"
	case parser.RangeUnitSlides:
		return "Slides"
	case parser.RangeUnitRows:
		return "Rows"
	case parser.RangeUnitLines:
		return "Lines"
	case parser.RangeUnitParagraphs:
		return "Paragraphs"
	case parser.RangeUnitEntries:
		return "Entries"
	case parser.RangeUnitValues:
		return "Values"
	default:
		return string(unit)
	}
}

// realClipboard adapts the clipboard package to the application.Clipboard interface.
type realClipboard struct{}

func (r *realClipboard) Copy(text string) error {
	return clipboard.Copy(text)
}

func showHelp() {
	fmt.Println("Clip - Copy text from documents to your clipboard")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("    clip <filename>")
	fmt.Println("    clip <filename> <range>")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("    clip report.pdf")
	fmt.Println("    clip report.pdf 5-10")
	fmt.Println("    clip \"The Brain.docx\" 1-3")
	fmt.Println("    clip presentation.pptx 2-5")
	fmt.Println("    clip spreadsheet.xlsx 2-10")
	fmt.Println()
	fmt.Println("Clip searches common locations when only a filename is provided.")
	fmt.Println()
	fmt.Println("Supported formats and their range units:")
	fmt.Println("  DOCX      -> blocks")
	fmt.Println("  PPT/PPTX  -> slides")
	fmt.Println("  ODT       -> paragraphs")
	fmt.Println("  RTF       -> paragraphs")
	fmt.Println("  TXT       -> lines")
	fmt.Println("  Markdown  -> blocks")
	fmt.Println("  CSV       -> rows")
	fmt.Println("  XLSX      -> rows")
	fmt.Println("  ODS       -> rows")
	fmt.Println("  JSON      -> entries")
	fmt.Println("  XML       -> entries")
	fmt.Println("  HTML      -> blocks")
	fmt.Println("  YAML      -> extracted values")
	fmt.Println()
	fmt.Println("Additional commands:")
	fmt.Println("  clip --version")
	fmt.Println("  clip --help")
}

// getFilePathAndRangeFromArgs extracts the file path and optional range argument from parsed flag arguments.
// This function handles quoted filenames with spaces and intelligently distinguishes between
// filename words and range arguments.
func getFilePathAndRangeFromArgs(args []string) (string, string) {
	// If no arguments, return empty
	if len(args) == 0 {
		return "", ""
	}

	// If only one argument, it's the filename
	if len(args) == 1 {
		return args[0], ""
	}

	// If multiple arguments, we need to determine if the last argument is a range
	// A range argument should contain digits and optionally a dash
	lastArg := args[len(args)-1]
	if isRangeArgument(lastArg) {
		// Last argument is a range, join the rest as filename
		filename := strings.Join(args[:len(args)-1], " ")
		return filename, lastArg
	}

	// If last argument is not a range, treat all arguments as filename
	filename := strings.Join(args, " ")
	return filename, ""
}

// getFilePathAndRange is now deprecated in favor of getFilePathAndRangeFromArgs
func getFilePathAndRange() (string, string) {
	return getFilePathAndRangeFromArgs(os.Args[1:])
}

// isRangeArgument checks if an argument looks like a range specification.
// A valid range contains digits and optionally a dash (e.g., "5", "5-10").
func isRangeArgument(arg string) bool {
	// Remove any quotes from the argument
	arg = strings.Trim(arg, `"`)

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
