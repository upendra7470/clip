package pdf

import (
	"testing"

	ledongpdf "github.com/ledongthuc/pdf"
	clipLayout "github.com/upendra7470/clip/internal/layout"
)

func TestRenderPagePreservesWordsLinesAndColumns(t *testing.T) {
	fragments := []ledongpdf.Text{
		{S: "Title", X: 10, Y: 100, W: 25, FontSize: 12},
		{S: "Left", X: 10, Y: 80, W: 20, FontSize: 10},
		{S: "Right", X: 100, Y: 80, W: 25, FontSize: 10},
		{S: "Next", X: 10, Y: 60, W: 20, FontSize: 10},
		{S: "line", X: 35, Y: 60, W: 18, FontSize: 10},
	}
	want := "Title\nLeft\tRight\nNext line"
	if got := renderPage(fragments, clipLayout.Auto); got != want {
		t.Fatalf("renderPage() = %q, want %q", got, want)
	}
}

func TestRenderPageSortsPDFCoordinatesTopToBottom(t *testing.T) {
	fragments := []ledongpdf.Text{
		{S: "bottom", X: 0, Y: 10, W: 30, FontSize: 10},
		{S: "top", X: 0, Y: 100, W: 15, FontSize: 10},
	}
	if got := renderPage(fragments, clipLayout.Plain); got != "top\nbottom" {
		t.Fatalf("renderPage() = %q", got)
	}
}
