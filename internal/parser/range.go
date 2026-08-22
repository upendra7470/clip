package parser

type Range struct {
	Start int
	End   int
}

// NormalizeRange normalizes a range against a total count of units.
// If end == -1, it sets end to totalUnits.
func NormalizeRange(r Range, totalUnits int) Range {
	if r.End == -1 {
		r.End = totalUnits
	}
	return r
}
