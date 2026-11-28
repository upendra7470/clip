// Package layout reconstructs readable text from positioned PDF and OCR tokens.
package layout

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// Mode controls how aggressively spatial layout is retained.
type Mode string

const (
	Auto  Mode = "auto"
	Plain Mode = "plain"
	Exact Mode = "exact"
)

// ParseMode validates a user-facing layout mode. An empty value means auto so
// older callers automatically benefit from layout reconstruction.
func ParseMode(value string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(value))) {
	case "", Auto:
		return Auto, nil
	case Plain:
		return Plain, nil
	case Exact:
		return Exact, nil
	default:
		return "", fmt.Errorf("unsupported layout %q (use auto, plain, or exact)", value)
	}
}

// Token is one positioned fragment of recognized text. X/Y identify the
// upper-left corner and Width/Height are expressed in the source coordinate
// system. Y must increase from top to bottom.
type Token struct {
	Text          string
	X, Y          float64
	Width, Height float64
}

type line struct {
	tokens []Token
	center float64
	height float64
}

// Render converts positioned tokens to clipboard-friendly text. Auto keeps
// paragraphs and uses tabs for column/table-sized gaps. Exact uses spaces to
// approximate the source geometry. Plain retains reading-order lines only.
func Render(tokens []Token, mode Mode) string {
	tokens = validTokens(tokens)
	if len(tokens) == 0 {
		return ""
	}
	lines := groupLines(tokens)
	charWidth := medianCharacterWidth(tokens)
	lineGap := medianLineGap(lines)
	minX := tokens[0].X
	for _, token := range tokens[1:] {
		minX = math.Min(minX, token.X)
	}

	var result strings.Builder
	for index, current := range lines {
		if index > 0 {
			gap := current.center - lines[index-1].center
			if mode != Plain && lineGap > 0 && gap > lineGap*1.25 {
				result.WriteString("\n\n")
			} else {
				result.WriteByte('\n')
			}
		}
		result.WriteString(renderLine(current, mode, charWidth, minX))
	}
	rendered := strings.TrimRight(result.String(), " \t\r\n")
	return strings.TrimLeft(rendered, "\r\n")
}

func validTokens(tokens []Token) []Token {
	valid := make([]Token, 0, len(tokens))
	for _, token := range tokens {
		token.Text = strings.TrimSpace(token.Text)
		if token.Text == "" || !finite(token.X) || !finite(token.Y) || !finite(token.Width) || !finite(token.Height) {
			continue
		}
		if token.Width <= 0 {
			token.Width = float64(max(1, utf8.RuneCountInString(token.Text)))
		}
		if token.Height <= 0 {
			token.Height = 1
		}
		valid = append(valid, token)
	}
	return valid
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func groupLines(tokens []Token) []line {
	sort.SliceStable(tokens, func(i, j int) bool {
		iCenter := tokens[i].Y + tokens[i].Height/2
		jCenter := tokens[j].Y + tokens[j].Height/2
		if math.Abs(iCenter-jCenter) > 0.5 {
			return iCenter < jCenter
		}
		return tokens[i].X < tokens[j].X
	})

	lines := make([]line, 0)
	for _, token := range tokens {
		center := token.Y + token.Height/2
		best := -1
		bestDistance := math.MaxFloat64
		for index := len(lines) - 1; index >= 0; index-- {
			distance := math.Abs(lines[index].center - center)
			tolerance := math.Max(1.5, math.Min(lines[index].height, token.Height)*0.45)
			if distance <= tolerance && distance < bestDistance {
				best, bestDistance = index, distance
			}
			if lines[index].center < center-tolerance*2 {
				break
			}
		}
		if best == -1 {
			lines = append(lines, line{tokens: []Token{token}, center: center, height: token.Height})
			continue
		}
		current := &lines[best]
		count := float64(len(current.tokens))
		current.center = (current.center*count + center) / (count + 1)
		current.height = math.Max(current.height, token.Height)
		current.tokens = append(current.tokens, token)
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].center < lines[j].center })
	for index := range lines {
		sort.SliceStable(lines[index].tokens, func(i, j int) bool {
			return lines[index].tokens[i].X < lines[index].tokens[j].X
		})
	}
	return lines
}

func renderLine(current line, mode Mode, charWidth, minX float64) string {
	if charWidth <= 0 {
		charWidth = 1
	}
	var result strings.Builder
	if mode == Exact {
		indent := boundedSpaces((current.tokens[0].X - minX) / charWidth)
		result.WriteString(strings.Repeat(" ", indent))
	}
	previousRight := current.tokens[0].X
	for index, token := range current.tokens {
		if index > 0 {
			gap := token.X - previousRight
			switch mode {
			case Exact:
				spaces := boundedSpaces(gap / charWidth)
				if spaces < 1 {
					spaces = 1
				}
				result.WriteString(strings.Repeat(" ", spaces))
			case Auto:
				if gap > charWidth*3.5 {
					result.WriteByte('\t')
				} else {
					result.WriteByte(' ')
				}
			default:
				result.WriteByte(' ')
			}
		}
		result.WriteString(token.Text)
		previousRight = math.Max(previousRight, token.X+token.Width)
	}
	return strings.TrimRight(result.String(), " \t")
}

func boundedSpaces(value float64) int {
	spaces := int(math.Round(value))
	if spaces < 0 {
		return 0
	}
	if spaces > 120 {
		return 120
	}
	return spaces
}

func medianCharacterWidth(tokens []Token) float64 {
	widths := make([]float64, 0, len(tokens))
	for _, token := range tokens {
		count := utf8.RuneCountInString(token.Text)
		if count > 0 && token.Width > 0 {
			widths = append(widths, token.Width/float64(count))
		}
	}
	return median(widths)
}

func medianLineGap(lines []line) float64 {
	if len(lines) < 2 {
		return 0
	}
	gaps := make([]float64, 0, len(lines)-1)
	for index := 1; index < len(lines); index++ {
		if gap := lines[index].center - lines[index-1].center; gap > 0 {
			gaps = append(gaps, gap)
		}
	}
	return median(gaps)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 0 {
		return (values[middle-1] + values[middle]) / 2
	}
	return values[middle]
}
