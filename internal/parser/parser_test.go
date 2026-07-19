package parser

import (
	"context"
	"testing"
)

type mockParser struct{}

func (m *mockParser) Parse(ctx context.Context, req ParseRequest) (ParseResult, error) {
	return ParseResult{Text: "mock parsed text"}, nil
}

func TestParseRequest(t *testing.T) {
	req := ParseRequest{
		File: "test.docx",
		Selection: Selection{
			Pages: "1-3",
			Range: "1:10-20:30",
			Query: "important",
		},
	}

	if req.File != "test.docx" {
		t.Errorf("ParseRequest.File = %q, want %q", req.File, "test.docx")
	}

	if req.Selection.Pages != "1-3" {
		t.Errorf("ParseRequest.Selection.Pages = %q, want %q", req.Selection.Pages, "1-3")
	}

	if req.Selection.Range != "1:10-20:30" {
		t.Errorf("ParseRequest.Selection.Range = %q, want %q", req.Selection.Range, "1:10-20:30")
	}

	if req.Selection.Query != "important" {
		t.Errorf("ParseRequest.Selection.Query = %q, want %q", req.Selection.Query, "important")
	}
}

func TestParseResult(t *testing.T) {
	result := ParseResult{
		Text: "extracted text content",
	}

	if result.Text != "extracted text content" {
		t.Errorf("ParseResult.Text = %q, want %q", result.Text, "extracted text content")
	}
}

func TestSelection(t *testing.T) {
	sel := Selection{
		Pages: "1,3-5",
		Range: "start-end",
		Query: "search term",
	}

	if sel.Pages != "1,3-5" {
		t.Errorf("Selection.Pages = %q, want %q", sel.Pages, "1,3-5")
	}

	if sel.Range != "start-end" {
		t.Errorf("Selection.Range = %q, want %q", sel.Range, "start-end")
	}

	if sel.Query != "search term" {
		t.Errorf("Selection.Query = %q, want %q", sel.Query, "search term")
	}
}

func TestEmptySelection(t *testing.T) {
	sel := Selection{}

	if sel.Pages != "" {
		t.Errorf("Empty Selection.Pages = %q, want empty string", sel.Pages)
	}

	if sel.Range != "" {
		t.Errorf("Empty Selection.Range = %q, want empty string", sel.Range)
	}

	if sel.Query != "" {
		t.Errorf("Empty Selection.Query = %q, want empty string", sel.Query)
	}
}

func TestParserInterface(t *testing.T) {
	var p Parser = &mockParser{}
	_, ok := p.(*mockParser)
	if !ok {
		t.Errorf("mockParser does not implement Parser interface")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	p := &mockParser{}
	req := ParseRequest{File: "test.docx"}

	// This should still work since our mock doesn't check context
	// but real implementations should handle context cancellation
	result, err := p.Parse(ctx, req)
	if err != nil {
		t.Errorf("Parse with cancelled context failed: %v", err)
	}

	if result.Text != "mock parsed text" {
		t.Errorf("Parse result = %q, want %q", result.Text, "mock parsed text")
	}
}
