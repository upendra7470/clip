package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/upendra7470/clip/internal/application"
	"github.com/upendra7470/clip/internal/clipboard"
	"github.com/upendra7470/clip/internal/clipboardocr"
	"github.com/upendra7470/clip/internal/detect"
	"github.com/upendra7470/clip/internal/filetype"
	clipLayout "github.com/upendra7470/clip/internal/layout"
	clipoutput "github.com/upendra7470/clip/internal/output"
	"github.com/upendra7470/clip/internal/parser"
	"github.com/upendra7470/clip/internal/registry"
	"github.com/upendra7470/clip/internal/resolver"
	"github.com/upendra7470/clip/internal/screencapture"
	"github.com/upendra7470/clip/internal/searchablepdf"
	"github.com/upendra7470/clip/internal/security"
	"github.com/upendra7470/clip/parsers/csv"
	"github.com/upendra7470/clip/parsers/docx"
	"github.com/upendra7470/clip/parsers/html"
	"github.com/upendra7470/clip/parsers/json"
	"github.com/upendra7470/clip/parsers/markdown"
	"github.com/upendra7470/clip/parsers/ocr"
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

// version is set at build time via -ldflags "-X main.version=<version>".
// Builds without injection (plain `go build`) report "dev".
var version = "dev"

var captureScreen = screencapture.Capture

func Run(args []string, stdout, stderr io.Writer) int {
	// Create a new FlagSet to avoid flag redefinition
	flagSet := flag.NewFlagSet("clip", flag.ContinueOnError)

	// Parse command line flags
	helpFlag := flagSet.Bool("help", false, "Show help message")
	hFlag := flagSet.Bool("h", false, "Show help message")
	versionFlag := flagSet.Bool("version", false, "Show version information")
	listFlag := flagSet.Bool("list", false, "List document units (pages, sections, slides, etc.)")
	inspectFlag := flagSet.Bool("inspect", false, "Inspect document units (pages, sections, slides, etc.)")
	ocrStatus := flagSet.Bool("ocr-status", false, "Show OCR engine availability")
	formatFlag := flagSet.String("format", "text", "Output format: text, markdown, json, or jsonl")
	chunkSize := flagSet.Int("chunks", 0, "Split output into AI-ready chunks of approximately N characters")
	outputPath := flagSet.String("output", "", "Write extracted content to a file instead of the clipboard")
	stdoutFlag := flagSet.Bool("stdout", false, "Write extracted content to standard output")
	smartFlag := flagSet.Bool("smart", false, "Enable OCR cleanup and structure-preserving Markdown output")
	recursiveFlag := flagSet.Bool("recursive", false, "Process supported files in subdirectories")
	languageFlag := flagSet.String("lang", "auto", "OCR language hint: auto, latin, en, fr, de, es, it, pt, or nl")
	completionFlag := flagSet.String("completion", "", "Print shell completion: bash, zsh, fish, or powershell")
	imageFlag := flagSet.Bool("image", false, "Extract an image with layout-aware OCR")
	pdfFlag := flagSet.Bool("pdf", false, "Extract a PDF while preserving its layout")
	layoutFlag := flagSet.String("layout", "auto", "Layout reconstruction: auto, plain, or exact")
	screenFlag := flagSet.Bool("screen", false, "Select a screen region and copy its text")
	richFlag := flagSet.Bool("rich", false, "Copy formatted HTML with a plain-text fallback")
	searchableFlag := flagSet.Bool("make-searchable", false, "Add an invisible OCR text layer to a scanned PDF")
	watchFlag := flagSet.Bool("watch", false, "Continuously OCR images copied to the clipboard")
	clipboardFlag := flagSet.Bool("clipboard", false, "OCR the image currently on the clipboard")
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

	if *completionFlag != "" {
		script, err := completionScript(*completionFlag)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Fprint(stdout, script)
		return 0
	}
	if *chunkSize < 0 {
		fmt.Fprintln(stderr, "Error: --chunks must be zero or greater")
		return 1
	}
	if *imageFlag && *pdfFlag {
		fmt.Fprintln(stderr, "Error: --image and --pdf cannot be used together")
		return 1
	}
	if *screenFlag && (*imageFlag || *pdfFlag || *listFlag || *inspectFlag || *recursiveFlag || *searchableFlag) {
		fmt.Fprintln(stderr, "Error: --screen cannot be combined with file input modes, inspection, or recursion")
		return 1
	}
	if *searchableFlag && (*imageFlag || *pdfFlag || *richFlag || *listFlag || *inspectFlag || *recursiveFlag || *stdoutFlag || *chunkSize > 0 || *formatFlag != "text") {
		fmt.Fprintln(stderr, "Error: --make-searchable only accepts OCR, layout, language, and --output options")
		return 1
	}
	if *watchFlag && (*clipboardFlag || *screenFlag || *imageFlag || *pdfFlag || *searchableFlag || *listFlag || *inspectFlag || *recursiveFlag || *outputPath != "" || *chunkSize > 0) {
		fmt.Fprintln(stderr, "Error: --watch cannot be combined with file modes, inspection, recursion, chunks, or --output")
		return 1
	}
	if *clipboardFlag && (*screenFlag || *imageFlag || *pdfFlag || *searchableFlag || *listFlag || *inspectFlag || *recursiveFlag || *outputPath != "" || *chunkSize > 0) {
		fmt.Fprintln(stderr, "Error: --clipboard cannot be combined with file modes, inspection, recursion, chunks, or --output")
		return 1
	}
	if _, err := clipLayout.ParseMode(*layoutFlag); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if err := validateOCRLanguage(*languageFlag); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	outputFormat, err := clipoutput.ParseFormat(*formatFlag)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if *smartFlag && outputFormat == clipoutput.Text {
		outputFormat = clipoutput.Markdown
	}
	if *richFlag && (*stdoutFlag || *outputPath != "") {
		fmt.Fprintln(stderr, "Error: --rich writes to the clipboard and cannot be combined with --stdout or --output")
		return 1
	}
	if *richFlag && outputFormat != clipoutput.Text && outputFormat != clipoutput.Markdown {
		fmt.Fprintln(stderr, "Error: --rich supports text or markdown format, not JSON")
		return 1
	}

	if *ocrStatus {
		if path, ok := ocr.Available(); ok {
			fmt.Fprintf(stdout, "Built-in OCR ready: %s\n", path)
			fmt.Fprintln(stdout, "Scanned PDF OCR ready: built in")
			return 0
		}
		fmt.Fprintln(stderr, "OCR unavailable: this development build does not contain the bundled OCR assets")
		return 1
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
	pdfParser := pdf.NewParser()
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
	if err := reg.Register(jsonParser.FileType(), jsonParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register JSON parser: %v\n", err)
		return 1
	}

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

	// Register YAML parser
	yamlParser := &yaml.Parser{}
	if err := reg.Register(yamlParser.FileType(), yamlParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register YAML parser: %v\n", err)
		return 1
	}

	// Register ODT parser
	odtParser := &odt.Parser{}
	if err := reg.Register(odtParser.FileType(), odtParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register ODT parser: %v\n", err)
		return 1
	}

	// Register ODS parser
	odsParser := &ods.Parser{}
	if err := reg.Register(odsParser.FileType(), odsParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register ODS parser: %v\n", err)
		return 1
	}

	// Register RTF parser
	rtfParser := &rtf.Parser{}
	if err := reg.Register(rtfParser.FileType(), rtfParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register RTF parser: %v\n", err)
		return 1
	}

	ocrParser := ocr.NewParser()
	if err := reg.Register(ocrParser.FileType(), ocrParser); err != nil {
		fmt.Fprintf(stderr, "Failed to register OCR parser: %v\n", err)
		return 1
	}

	// Create resolver
	fileResolver := resolver.New()

	// Create clipboard adapter
	clipboardAdapter := &realClipboard{}

	// Create application with registry and clipboard
	app := application.New(reg, clipboardAdapter)
	if *clipboardFlag {
		if len(flagSet.Args()) != 0 {
			fmt.Fprintln(stderr, "Error: --clipboard does not accept a filename")
			return 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		event, err := clipboard.ReadImage(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		err = clipboardocr.RunOnce(ctx, event, clipboardocr.Config{
			Format: outputFormat,
			Rich:   *richFlag,
			Stdout: *stdoutFlag,
		}, func(ctx context.Context, path string) (string, error) {
			return ocr.RecognizeSmartLayout(ctx, path, *smartFlag, *layoutFlag)
		}, clipboardAdapter, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}
	if *watchFlag {
		if len(flagSet.Args()) != 0 {
			fmt.Fprintln(stderr, "Error: --watch does not accept a filename")
			return 1
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		events, err := clipboard.WatchImages(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		statusWriter := stdout
		if *stdoutFlag {
			statusWriter = stderr
		}
		fmt.Fprintln(statusWriter, "Watching the clipboard for images… Press Ctrl+C to stop.")
		err = clipboardocr.Run(ctx, events, clipboardocr.Config{
			Format: outputFormat,
			Rich:   *richFlag,
			Stdout: *stdoutFlag,
		}, func(ctx context.Context, path string) (string, error) {
			return ocr.RecognizeSmartLayout(ctx, path, *smartFlag, *layoutFlag)
		}, clipboardAdapter, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Fprintln(statusWriter, "✓ Clipboard OCR stopped")
		return 0
	}
	if *screenFlag {
		if len(flagSet.Args()) != 0 {
			fmt.Fprintln(stderr, "Error: --screen does not accept a filename")
			return 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		return runScreenCapture(ctx, app, clipboardAdapter, outputFormat, *chunkSize, *outputPath, *stdoutFlag, *richFlag,
			parser.Selection{Smart: *smartFlag, Language: *languageFlag, Layout: *layoutFlag}, stdout, stderr)
	}

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
	if *searchableFlag && rangeObj != nil {
		fmt.Fprintln(stderr, "Error: --make-searchable processes the complete PDF and does not accept a range")
		return 1
	}

	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Resolve file path using the resolver
	resolvedPath, err := fileResolver.Resolve(ctx, filePath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if !info.IsDir() && info.Size() > security.MaxInputFileSize {
		fmt.Fprintf(stderr, "Error: file size %d bytes exceeds maximum allowed %d bytes\n", info.Size(), security.MaxInputFileSize)
		return 1
	}
	if *searchableFlag {
		if info.IsDir() {
			fmt.Fprintln(stderr, "Error: --make-searchable requires a PDF file")
			return 1
		}
		detectedType, err := detect.Type(resolvedPath)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if detectedType != filetype.FileTypePDF {
			fmt.Fprintf(stderr, "Error: --make-searchable requires a PDF file, got %s\n", detectedType)
			return 1
		}
		destination := *outputPath
		if destination == "" {
			destination = searchableOutputPath(resolvedPath)
		}
		pages, err := searchablepdf.Create(ctx, app, resolvedPath, destination, parser.Selection{
			Smart: true, Language: *languageFlag, Layout: *layoutFlag,
		})
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if pages == 0 {
			fmt.Fprintf(stdout, "✓ PDF already had searchable text → %s\n", destination)
		} else {
			fmt.Fprintf(stdout, "✓ Added searchable text to %d page(s) → %s\n", pages, destination)
		}
		return 0
	}
	if *imageFlag || *pdfFlag {
		if info.IsDir() {
			fmt.Fprintln(stderr, "Error: --image and --pdf require a file")
			return 1
		}
		detectedType, err := detect.Type(resolvedPath)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if *imageFlag && detectedType != filetype.FileTypeImage {
			fmt.Fprintf(stderr, "Error: --image requires a supported image, got %s\n", detectedType)
			return 1
		}
		if *pdfFlag && detectedType != filetype.FileTypePDF {
			fmt.Fprintf(stderr, "Error: --pdf requires a PDF file, got %s\n", detectedType)
			return 1
		}
	}
	v2Mode := *smartFlag || *imageFlag || *pdfFlag || *richFlag || *layoutFlag != "auto" || *formatFlag != "text" || *chunkSize > 0 || *outputPath != "" || *stdoutFlag || *recursiveFlag || *languageFlag != "auto" || info.IsDir()
	if v2Mode {
		paths := []string{resolvedPath}
		if info.IsDir() {
			paths, err = batchFiles(resolvedPath, *recursiveFlag)
			if err != nil {
				fmt.Fprintf(stderr, "Error: %v\n", err)
				return 1
			}
		}
		selection := parser.Selection{Smart: *smartFlag, Language: *languageFlag, Layout: *layoutFlag}
		documents, err := extractDocuments(ctx, app, paths, rangeObj, selection)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		rendered, err := clipoutput.Render(documents, outputFormat, *chunkSize)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		destination := "clipboard"
		switch {
		case *outputPath != "":
			if err := os.WriteFile(*outputPath, []byte(rendered), 0600); err != nil {
				fmt.Fprintf(stderr, "Error: write output: %v\n", err)
				return 1
			}
			destination = *outputPath
		case *stdoutFlag:
			fmt.Fprintln(stdout, rendered)
			destination = "stdout"
		default:
			if *richFlag {
				richHTML, err := clipoutput.RenderHTML(documents)
				if err != nil {
					fmt.Fprintf(stderr, "Error: %v\n", err)
					return 1
				}
				if err := clipboardAdapter.CopyRich(rendered, richHTML); err != nil {
					fmt.Fprintf(stderr, "Error: failed to copy rich text: %v\n", err)
					return 1
				}
			} else if err := clipboardAdapter.Copy(rendered); err != nil {
				fmt.Fprintf(stderr, "Error: failed to copy to clipboard: %v\n", err)
				return 1
			}
		}
		statusWriter := stdout
		if *stdoutFlag {
			statusWriter = stderr
		}
		fmt.Fprintf(statusWriter, "✓ Processed %d file(s) as %s → %s\n", len(documents), outputFormat, destination)
		return 0
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
		if err := listDocumentUnits(resolvedPath, parserObj, ctx); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
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
func listDocumentUnits(resolvedPath string, parserObj parser.Parser, ctx context.Context) error {
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
			return fmt.Errorf("listing document units: %w", err)
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
	return nil
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

func (r *realClipboard) CopyRich(plainText, html string) error {
	return clipboard.CopyRich(plainText, html)
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
	fmt.Println("    clip --smart scanned.pdf")
	fmt.Println("    clip --image receipt.png")
	fmt.Println("    clip --pdf --layout exact report.pdf")
	fmt.Println("    clip --screen")
	fmt.Println("    clip --screen --format markdown --stdout")
	fmt.Println("    clip --rich --pdf report.pdf")
	fmt.Println("    clip --make-searchable scanned.pdf")
	fmt.Println("    clip --make-searchable --output archive.pdf scanned.pdf")
	fmt.Println("    clip --watch")
	fmt.Println("    clip --watch --smart --rich")
	fmt.Println("    clip --clipboard")
	fmt.Println("    clip --clipboard --smart --stdout")
	fmt.Println("    clip --format jsonl --chunks 1200 --stdout handbook.pdf")
	fmt.Println("    clip --recursive --output knowledge.md ./documents")
	fmt.Println()
	fmt.Println("Clip searches common locations when only a filename is provided.")
	fmt.Println()
	fmt.Println("Supported formats and their range units:")
	fmt.Println("  DOCX      -> sections")
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
	fmt.Println("  YAML      -> values")
	fmt.Println("  Images    -> OCR (PNG, JPEG, GIF, BMP, TIFF, WebP)")
	fmt.Println()
	fmt.Println("Additional commands:")
	fmt.Println("  clip --version")
	fmt.Println("  clip --help")
	fmt.Println("  clip --ocr-status")
	fmt.Println("  clip --completion bash|zsh|fish|powershell")
	fmt.Println()
	fmt.Println("Advanced OCR and workflow options:")
	fmt.Println("  --smart             Clean OCR and preserve structure as Markdown")
	fmt.Println("  --image             Run layout-aware OCR on an image")
	fmt.Println("  --pdf               Preserve the spatial layout of a PDF")
	fmt.Println("  --layout <mode>     auto, plain, or exact")
	fmt.Println("  --screen            Select a screen region, OCR it, and copy the text")
	fmt.Println("  --rich              Copy formatted text with a plain-text fallback")
	fmt.Println("  --make-searchable   Preserve a PDF and add an invisible OCR text layer")
	fmt.Println("  --watch             OCR each image copied to the clipboard until stopped")
	fmt.Println("  --clipboard         OCR the current clipboard image once and exit")
	fmt.Println("  --format <format>   text, markdown, json, or jsonl")
	fmt.Println("  --chunks <size>     Create AI-ready chunks")
	fmt.Println("  --output <file>     Write to a file instead of clipboard")
	fmt.Println("  --stdout            Print extracted content")
	fmt.Println("  --recursive         Process directories recursively")
	fmt.Println("  --lang <hint>       Auto-detect supported Latin-script languages")
	fmt.Println()
	fmt.Println("OCR is built in and works offline; no external tools are required.")
}

func extractDocuments(ctx context.Context, app *application.Application, paths []string, rangeObj *parser.Range, selection parser.Selection) ([]clipoutput.Document, error) {
	documents := make([]clipoutput.Document, 0, len(paths))
	for _, path := range paths {
		result, err := app.ExtractText(ctx, path, rangeObj, selection)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		documents = append(documents, clipoutput.Document{
			Source: path,
			Type:   strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
			Text:   result.Text,
		})
	}
	return documents, nil
}

func batchFiles(root string, recursive bool) ([]string, error) {
	const maxBatchFiles = 1000
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && !recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSupportedBatchExtension(filepath.Ext(path)) {
			return nil
		}
		if _, err := detect.Type(path); err != nil {
			return nil
		}
		paths = append(paths, path)
		if len(paths) > maxBatchFiles {
			return fmt.Errorf("batch exceeds the %d-file safety limit", maxBatchFiles)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan directory: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no supported documents found in %s", root)
	}
	sort.Strings(paths)
	return paths, nil
}

func isSupportedBatchExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".pdf", ".docx", ".ppt", ".pptx", ".odt", ".rtf", ".txt", ".md", ".csv", ".xlsx", ".ods", ".json", ".xml", ".html", ".htm", ".yaml", ".yml", ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tif", ".tiff", ".webp":
		return true
	default:
		return false
	}
}

func validateOCRLanguage(language string) error {
	switch strings.ToLower(language) {
	case "auto", "latin", "en", "eng", "fr", "de", "es", "it", "pt", "nl":
		return nil
	default:
		return fmt.Errorf("unsupported OCR language hint %q; v2 supports automatic Latin-script recognition", language)
	}
}

func completionScript(shell string) (string, error) {
	flags := "--help --version --list --inspect --ocr-status --smart --image --pdf --screen --rich --make-searchable --watch --clipboard --layout --format --chunks --output --stdout --recursive --lang --completion"
	switch strings.ToLower(shell) {
	case "bash":
		return "_clip() { COMPREPLY=( $(compgen -W '" + flags + "' -- \"${COMP_WORDS[COMP_CWORD]}\") ); }\ncomplete -F _clip clip\n", nil
	case "zsh":
		return "#compdef clip\n_arguments '*:file:_files' '" + strings.ReplaceAll(flags, " ", "' '") + "'\n", nil
	case "fish":
		var result strings.Builder
		for _, value := range strings.Fields(flags) {
			fmt.Fprintf(&result, "complete -c clip -l %s\n", strings.TrimPrefix(value, "--"))
		}
		return result.String(), nil
	case "powershell":
		return "Register-ArgumentCompleter -Native -CommandName clip -ScriptBlock { param($wordToComplete) @('" + strings.ReplaceAll(flags, " ", "','") + "') | Where-Object { $_ -like \"$wordToComplete*\" } }\n", nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func runScreenCapture(
	ctx context.Context,
	app *application.Application,
	clipboardAdapter *realClipboard,
	outputFormat clipoutput.Format,
	chunkSize int,
	outputPath string,
	stdoutFlag bool,
	rich bool,
	selection parser.Selection,
	stdout, stderr io.Writer,
) int {
	statusWriter := stdout
	if stdoutFlag {
		statusWriter = stderr
	}
	fmt.Fprintln(statusWriter, "Select an area of the screen…")
	capture, err := captureScreen(ctx)
	if err != nil {
		if errors.Is(err, screencapture.ErrCancelled) {
			fmt.Fprintln(stderr, "Screen capture cancelled")
			return 1
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	defer os.Remove(capture.Path)

	result, err := app.ExtractText(ctx, capture.Path, nil, selection)
	if err != nil {
		fmt.Fprintf(stderr, "Error: screen OCR failed: %v\n", err)
		return 1
	}
	rendered, err := clipoutput.Render([]clipoutput.Document{{
		Source: "screen.png",
		Type:   "png",
		Text:   result.Text,
	}}, outputFormat, chunkSize)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	destination := "clipboard"
	switch {
	case outputPath != "":
		if err := os.WriteFile(outputPath, []byte(rendered), 0600); err != nil {
			fmt.Fprintf(stderr, "Error: write output: %v\n", err)
			return 1
		}
		destination = outputPath
	case stdoutFlag:
		fmt.Fprintln(stdout, rendered)
		destination = "stdout"
	default:
		if rich {
			richHTML, err := clipoutput.RenderHTML([]clipoutput.Document{{Source: "screen.png", Type: "png", Text: result.Text}})
			if err != nil {
				fmt.Fprintf(stderr, "Error: %v\n", err)
				return 1
			}
			if err := clipboardAdapter.CopyRich(rendered, richHTML); err != nil {
				fmt.Fprintf(stderr, "Error: failed to copy rich text: %v\n", err)
				return 1
			}
		} else if err := clipboardAdapter.Copy(rendered); err != nil {
			fmt.Fprintf(stderr, "Error: failed to copy to clipboard: %v\n", err)
			return 1
		}
	}

	fmt.Fprintf(statusWriter, "✓ Captured %d × %d\n", capture.Width, capture.Height)
	fmt.Fprintf(statusWriter, "✓ Recognized %d words\n", len(strings.Fields(result.Text)))
	fmt.Fprintf(statusWriter, "✓ Sent to %s\n", destination)
	return 0
}

func searchableOutputPath(inputPath string) string {
	extension := filepath.Ext(inputPath)
	return strings.TrimSuffix(inputPath, extension) + ".searchable.pdf"
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

// isRangeArgument checks if an argument looks like a range specification.
// A valid range contains digits and optionally a dash (e.g., "5", "5-10", "1-", "-10").
func isRangeArgument(arg string) bool {
	// Remove any quotes from the argument
	arg = strings.Trim(arg, `"`)

	// Check if empty
	if arg == "" {
		return false
	}

	// Check if it contains only digits, dashes, and optional whitespace
	hasDigits := false
	hasOtherChars := false

	for _, c := range arg {
		if c >= '0' && c <= '9' {
			hasDigits = true
		} else if c == '-' {
			// dash is handled below
		} else if !isSpace(c) {
			hasOtherChars = true
		}
	}

	// Must have at least one digit
	if !hasDigits {
		return false
	}

	// Count dashes - valid ranges have at most one dash
	dashCount := strings.Count(arg, "-")
	if dashCount > 1 {
		return false
	}

	// Check for open-ended ranges: "N-" or "-N"
	if dashCount == 1 {
		if arg[0] == '-' {
			// "-N" format - must have digits after the dash
			for _, c := range arg[1:] {
				if c >= '0' && c <= '9' {
					return true
				}
			}
			return false
		}
		if arg[len(arg)-1] == '-' {
			// "N-" format - must have digits before the dash
			for _, c := range arg[:len(arg)-1] {
				if c >= '0' && c <= '9' {
					return true
				}
			}
			return false
		}
	}

	// Check for valid dash-separated format: "N" or "N-M"
	// Must not have other characters
	if !hasOtherChars {
		return true
	}

	return false
}

// isSpace checks if a character is whitespace
func isSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
