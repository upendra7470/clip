package layout

import "testing"

func TestAutoPreservesLinesParagraphsAndColumns(t *testing.T) {
	tokens := []Token{
		{Text: "Invoice", X: 10, Y: 10, Width: 35, Height: 10},
		{Text: "Total", X: 10, Y: 30, Width: 25, Height: 10},
		{Text: "$42", X: 100, Y: 30, Width: 18, Height: 10},
		{Text: "Thank", X: 10, Y: 70, Width: 25, Height: 10},
		{Text: "you", X: 40, Y: 70, Width: 15, Height: 10},
	}
	want := "Invoice\nTotal\t$42\n\nThank you"
	if got := Render(tokens, Auto); got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestExactUsesSpatialIndentation(t *testing.T) {
	tokens := []Token{
		{Text: "A", X: 0, Y: 0, Width: 10, Height: 10},
		{Text: "B", X: 30, Y: 0, Width: 10, Height: 10},
	}
	if got := Render(tokens, Exact); got != "A  B" {
		t.Fatalf("Render() = %q", got)
	}
}

func TestExactDoesNotIndentLeftMarginOnLaterLines(t *testing.T) {
	tokens := []Token{
		{Text: "first", X: 0, Y: 0, Width: 25, Height: 10},
		{Text: "second", X: 0, Y: 15, Width: 30, Height: 10},
	}
	if got := Render(tokens, Exact); got != "first\nsecond" {
		t.Fatalf("Render() = %q", got)
	}
}

func TestExactPreservesFirstLineIndentation(t *testing.T) {
	tokens := []Token{
		{Text: "indented", X: 20, Y: 0, Width: 40, Height: 10},
		{Text: "margin", X: 0, Y: 15, Width: 30, Height: 10},
	}
	if got := Render(tokens, Exact); got != "    indented\nmargin" {
		t.Fatalf("Render() = %q", got)
	}
}

func TestParseMode(t *testing.T) {
	if mode, err := ParseMode(""); err != nil || mode != Auto {
		t.Fatalf("ParseMode(empty) = %q, %v", mode, err)
	}
	if _, err := ParseMode("pixel-perfect"); err == nil {
		t.Fatal("expected invalid layout error")
	}
}
