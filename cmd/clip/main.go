package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/upendra7470/clip/internal/application"
	"github.com/upendra7470/clip/internal/registry"
	"github.com/upendra7470/clip/parsers/txt"
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
	txtParser := &txt.Parser{}
	if err := reg.Register(txtParser.FileType(), txtParser); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register TXT parser: %v\n", err)
		os.Exit(1)
	}

	// Create application
	app := application.New(reg)

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
