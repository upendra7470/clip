package output

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSONChunks(t *testing.T) {
	rendered, err := Render([]Document{{Source: "notes.txt", Text: "alpha beta gamma delta"}}, JSON, 10)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(rendered), &value); err != nil {
		t.Fatal(err)
	}
	chunks, ok := value["chunks"].([]any)
	if !ok || len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %s", rendered)
	}
}

func TestMarkdownTable(t *testing.T) {
	rendered, err := Render([]Document{{Source: "data.csv", Text: "Name\tScore\nAda\t10"}}, Markdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "| Name | Score |\n| --- | --- |") {
		t.Fatalf("table was not preserved:\n%s", rendered)
	}
}

func TestCSVMarkdownTable(t *testing.T) {
	rendered, err := Render([]Document{{Source: "data.csv", Type: "csv", Text: "Name, Score\nAda, 10"}}, Markdown, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "| Name | Score |\n| --- | --- |") {
		t.Fatalf("CSV table was not preserved:\n%s", rendered)
	}
}

func TestChunkTextDoesNotSplitUnicode(t *testing.T) {
	chunks := chunkText("नमस्ते दुनिया", 5)
	if len(chunks) < 2 {
		t.Fatalf("expected chunks, got %#v", chunks)
	}
	for _, chunk := range chunks {
		if !utf8Valid(chunk) {
			t.Fatalf("invalid UTF-8 chunk: %q", chunk)
		}
	}
}

func utf8Valid(value string) bool {
	return strings.ToValidUTF8(value, "") == value
}
