package parser

// RangeUnit represents the unit of a range (e.g., pages, lines, characters)
type RangeUnit string

const (
	Pages    RangeUnit = "pages"
	Slides   RangeUnit = "slides"
	Blocks   RangeUnit = "blocks"
	Lines    RangeUnit = "lines"
	Rows     RangeUnit = "rows"
	Sections RangeUnit = "sections"
	Entries  RangeUnit = "entries"
	Values   RangeUnit = "values"
)
