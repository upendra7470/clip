package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// Range represents a range of units to extract from a document.
// This generic range can represent pages, slides, paragraphs, lines, rows, etc.
type Range struct {
	Start int
	End   int
}

// ParseRange parses a range string and returns a Range.
// Supported formats:
// - "5" (single unit)
// - "5-10" (range of units)
// Returns error if the format is invalid.
func ParseRange(input string) (Range, error) {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check for single page format
	if !strings.Contains(input, "-") {
		pageNum, err := strconv.Atoi(input)
		if err != nil {
			return Range{}, fmt.Errorf("invalid range: expected format like 5 or 5-10, got %q", input)
		}
		if pageNum < 1 {
			return Range{}, fmt.Errorf("range values must start from 1, got %d", pageNum)
		}
		return Range{Start: pageNum, End: pageNum}, nil
	}

	// Check for range format
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return Range{}, fmt.Errorf("invalid range: expected format like 5-10, got %q", input)
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	// Parse start page
	start, err := strconv.Atoi(startStr)
	if err != nil {
		return Range{}, fmt.Errorf("invalid range: start value must be a number, got %q", startStr)
	}
	if start < 1 {
		return Range{}, fmt.Errorf("range values must start from 1, got %d", start)
	}

	// Parse end page
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return Range{}, fmt.Errorf("invalid range: end value must be a number, got %q", endStr)
	}
	if end < 1 {
		return Range{}, fmt.Errorf("range values must start from 1, got %d", end)
	}

	// Validate range
	if end < start {
		return Range{}, fmt.Errorf("invalid range: start value must not be greater than end value (got %d-%d)", start, end)
	}

	return Range{Start: start, End: end}, nil
}

// ValidateRangeAgainstTotal validates that a range is within the bounds of a document.
func ValidateRangeAgainstTotal(rangeObj Range, totalUnits int) error {
	if rangeObj.Start < 1 || rangeObj.End < 1 {
		return fmt.Errorf("range values must start from 1")
	}
	if rangeObj.Start > totalUnits || rangeObj.End > totalUnits {
		return fmt.Errorf("requested range exceeds document unit count (document has %d units, requested %d-%d)", totalUnits, rangeObj.Start, rangeObj.End)
	}
	if rangeObj.End < rangeObj.Start {
		return fmt.Errorf("invalid range: start value must not be greater than end value")
	}
	return nil
}
