package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRange parses a range string into a Range struct
func ParseRange(rangeStr string) (Range, error) {
	// Trim whitespace from input
	trimmed := strings.TrimSpace(rangeStr)
	if trimmed == "" {
		return Range{}, fmt.Errorf("invalid range: empty string")
	}

	var start, end int
	var err error

	// Handle different range formats
	// Check for dash first
	if strings.Contains(trimmed, "-") {
		// Format: "start-end"
		parts := strings.Split(trimmed, "-")
		if len(parts) != 2 {
			return Range{}, fmt.Errorf("invalid range: expected format like 5-10, got \"%s\"", trimmed)
		}

		// Handle open-ended ranges: "1-" or "-10"
		startStr := strings.TrimSpace(parts[0])
		endStr := strings.TrimSpace(parts[1])

		if startStr == "" {
			// "-10" format: start from beginning
			start = -1
		} else {
			start, err = strconv.Atoi(startStr)
			if err != nil {
				return Range{}, fmt.Errorf("invalid range: expected format like 5-10, got \"%s\"", trimmed)
			}
		}

		if endStr == "" {
			// "1-" format: go to end
			end = -1
		} else {
			end, err = strconv.Atoi(endStr)
			if err != nil {
				return Range{}, fmt.Errorf("invalid range: expected format like 5-10, got \"%s\"", trimmed)
			}
		}
	} else if strings.Contains(trimmed, ":") {
		// Format: "start:end"
		parts := strings.Split(trimmed, ":")
		if len(parts) != 2 {
			return Range{}, fmt.Errorf("invalid range: expected format like 5 or 5-10, got \"%s\"", trimmed)
		}

		start, err = strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return Range{}, fmt.Errorf("invalid range: expected format like 5 or 5-10, got \"%s\"", trimmed)
		}

		end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return Range{}, fmt.Errorf("invalid range: expected format like 5 or 5-10, got \"%s\"", trimmed)
		}
	} else {
		// Single number format
		start, err = strconv.Atoi(trimmed)
		if err != nil {
			return Range{}, fmt.Errorf("invalid range: expected format like 5 or 5-10, got \"%s\"", trimmed)
		}
		end = start
	}

	// Handle sentinel values BEFORE validation
	// -1 means "from start" for start, or "to end" for end
	// These will be normalized by the parser implementations

	// Validate range values (only for non-sentinel values)
	if start != -1 && start < 1 {
		return Range{}, fmt.Errorf("range values must start from 1, got %d", start)
	}
	if end != -1 && end < 1 {
		return Range{}, fmt.Errorf("range values must start from 1, got %d", end)
	}
	// Only validate start > end if both are non-sentinel
	if start != -1 && end != -1 && start > end {
		return Range{}, fmt.Errorf("invalid range: start value must not be greater than end value (got %d-%d)", start, end)
	}

	return Range{Start: start, End: end}, nil
}

// ValidateRange validates a range against a document's unit
func ValidateRange(r Range, docUnit DocumentUnit) error {
	if r.Start < 1 {
		return fmt.Errorf("start value must be positive: %d", r.Start)
	}
	if r.End < 1 {
		return fmt.Errorf("end value must be positive: %d", r.End)
	}
	if r.Start > r.End {
		return fmt.Errorf("start value must be less than or equal to end value: %d > %d", r.Start, r.End)
	}

	// Additional validation based on document unit
	switch docUnit.Type {
	case "sections":
	// No additional validation for blocks
	case "pages":
	// No additional validation for pages
	case "characters":
	// No additional validation for characters
	default:
		return fmt.Errorf("unknown document unit: %s", docUnit.Type)
	}

	return nil
}

// ValidateRangeAgainstTotal validates a range against a total count of units
func ValidateRangeAgainstTotal(r Range, totalUnits int) error {
	if r.Start < 1 {
		return fmt.Errorf("range values must start from 1")
	}
	if r.End < 1 {
		return fmt.Errorf("range values must start from 1")
	}
	if r.Start > r.End {
		return fmt.Errorf("invalid range: start value must not be greater than end value")
	}
	if r.End > totalUnits {
		return fmt.Errorf("requested range exceeds document unit count (document has %d units, requested %d-%d)", totalUnits, r.Start, r.End)
	}
	return nil
}
