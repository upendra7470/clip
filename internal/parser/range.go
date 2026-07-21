package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// PageRange represents a range of pages to extract from a document.
type PageRange struct {
	Start int
	End   int
}

// ParsePageRange parses a page range string and returns a PageRange.
// Supported formats:
// - "5" (single page)
// - "5-10" (range of pages)
// Returns error if the format is invalid.
func ParsePageRange(input string) (PageRange, error) {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check for single page format
	if !strings.Contains(input, "-") {
		pageNum, err := strconv.Atoi(input)
		if err != nil {
			return PageRange{}, fmt.Errorf("invalid page range: expected format like 5 or 5-10, got %q", input)
		}
		if pageNum < 1 {
			return PageRange{}, fmt.Errorf("page numbers must start from 1, got %d", pageNum)
		}
		return PageRange{Start: pageNum, End: pageNum}, nil
	}

	// Check for range format
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return PageRange{}, fmt.Errorf("invalid page range: expected format like 5-10, got %q", input)
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	// Parse start page
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return PageRange{}, fmt.Errorf("invalid page range: start page must be a number, got %q", startStr)
	}
	if start < 1 {
		return PageRange{}, fmt.Errorf("page numbers must start from 1, got %d", start)
	}

	// Parse end page
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return PageRange{}, fmt.Errorf("invalid page range: end page must be a number, got %q", endStr)
	}
	if end < 1 {
		return PageRange{}, fmt.Errorf("page numbers must start from 1, got %d", end)
	}

	// Validate range
	if end < start {
		return PageRange{}, fmt.Errorf("invalid page range: start page must not be greater than end page (got %d-%d)", start, end)
	}

	return PageRange{Start: start, End: end}, nil
}

// ValidatePageRangeAgainstTotal validates that a page range is within the bounds of a document.
func ValidatePageRangeAgainstTotal(rangeObj PageRange, totalPages int) error {
	if rangeObj.Start < 1 || rangeObj.End < 1 {
		return fmt.Errorf("page numbers must start from 1")
	}
	if rangeObj.Start > totalPages || rangeObj.End > totalPages {
		return fmt.Errorf("requested page range exceeds document page count (document has %d pages, requested %d-%d)", totalPages, rangeObj.Start, rangeObj.End)
	}
	if rangeObj.End < rangeObj.Start {
		return fmt.Errorf("invalid page range: start page must not be greater than end page")
	}
	return nil
}
