package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

type Format string

const (
	Text     Format = "text"
	Markdown Format = "markdown"
	JSON     Format = "json"
	JSONL    Format = "jsonl"
)

type Document struct {
	Source string `json:"source"`
	Type   string `json:"type,omitempty"`
	Text   string `json:"text"`
}

type Chunk struct {
	Source string `json:"source"`
	Index  int    `json:"index"`
	Text   string `json:"text"`
}

type payload struct {
	Version   string     `json:"version"`
	Documents []Document `json:"documents,omitempty"`
	Chunks    []Chunk    `json:"chunks,omitempty"`
}

func ParseFormat(value string) (Format, error) {
	switch Format(strings.ToLower(value)) {
	case Text, Markdown, JSON, JSONL:
		return Format(strings.ToLower(value)), nil
	default:
		return "", fmt.Errorf("unsupported output format %q (use text, markdown, json, or jsonl)", value)
	}
}

func Render(documents []Document, format Format, chunkSize int) (string, error) {
	if len(documents) == 0 {
		return "", fmt.Errorf("no documents to render")
	}
	chunks := makeChunks(documents, chunkSize)

	switch format {
	case Text:
		return renderText(documents, chunks), nil
	case Markdown:
		return renderMarkdown(documents, chunks), nil
	case JSON:
		value := payload{Version: "2.0", Documents: documents}
		if chunkSize > 0 {
			value.Documents = nil
			value.Chunks = chunks
		}
		encoded, err := json.MarshalIndent(value, "", "  ")
		return string(encoded), err
	case JSONL:
		return renderJSONL(documents, chunks, chunkSize)
	default:
		return "", fmt.Errorf("unsupported output format %q", format)
	}
}

// RenderHTML converts the structure-preserving Markdown representation to a
// safe HTML fragment suitable for a rich clipboard. Goldmark's safe defaults
// omit raw HTML from extracted documents.
func RenderHTML(documents []Document) (string, error) {
	if len(documents) == 0 {
		return "", fmt.Errorf("no documents to render")
	}
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM))
	var result strings.Builder
	result.WriteString(`<div class="clip-content"><style>.clip-content{font-family:system-ui,-apple-system,sans-serif;line-height:1.45;color:#1f2328}.clip-content table{border-collapse:collapse}.clip-content th,.clip-content td{border:1px solid #d0d7de;padding:6px 10px;text-align:left}.clip-content h1,.clip-content h2,.clip-content h3{line-height:1.25}</style>`)
	for index, document := range documents {
		if index > 0 {
			result.WriteString("<hr>")
		}
		if len(documents) > 1 {
			fmt.Fprintf(&result, "<h1>%s</h1>", html.EscapeString(filepath.Base(document.Source)))
		}
		var body bytes.Buffer
		if err := markdown.Convert([]byte(markdownDocument(document)), &body); err != nil {
			return "", fmt.Errorf("render HTML: %w", err)
		}
		result.Write(body.Bytes())
	}
	result.WriteString("</div>")
	return result.String(), nil
}

func renderText(documents []Document, chunks []Chunk) string {
	if len(chunks) > 0 {
		parts := make([]string, 0, len(chunks))
		for _, chunk := range chunks {
			parts = append(parts, chunk.Text)
		}
		return strings.Join(parts, "\n\n")
	}
	if len(documents) == 1 {
		return strings.TrimSpace(documents[0].Text)
	}
	var result strings.Builder
	for index, document := range documents {
		if index > 0 {
			result.WriteString("\n\n")
		}
		fmt.Fprintf(&result, "--- %s ---\n\n%s", document.Source, strings.TrimSpace(document.Text))
	}
	return result.String()
}

func renderMarkdown(documents []Document, chunks []Chunk) string {
	if len(chunks) > 0 {
		var result strings.Builder
		for index, chunk := range chunks {
			if index > 0 {
				result.WriteString("\n\n---\n\n")
			}
			fmt.Fprintf(&result, "## %s · chunk %d\n\n%s", filepath.Base(chunk.Source), chunk.Index, markdownTables(chunk.Text))
		}
		return result.String()
	}
	var result strings.Builder
	for index, document := range documents {
		if index > 0 {
			result.WriteString("\n\n---\n\n")
		}
		if len(documents) > 1 {
			fmt.Fprintf(&result, "# %s\n\n", filepath.Base(document.Source))
		}
		result.WriteString(markdownDocument(document))
	}
	return result.String()
}

func markdownDocument(document Document) string {
	text := strings.TrimSpace(document.Text)
	switch strings.ToLower(document.Type) {
	case "csv", "xlsx", "ods":
		return delimitedTable(text, ", ")
	default:
		return markdownTables(text)
	}
}

func delimitedTable(text, delimiter string) string {
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return text
	}
	rows := make([][]string, 0, len(lines))
	width := -1
	for _, line := range lines {
		cells := strings.Split(line, delimiter)
		if len(cells) < 2 || (width != -1 && len(cells) != width) {
			return text
		}
		width = len(cells)
		rows = append(rows, cells)
	}
	var result []string
	result = append(result, markdownRow(rows[0]))
	separator := make([]string, width)
	for index := range separator {
		separator[index] = "---"
	}
	result = append(result, markdownRow(separator))
	for _, row := range rows[1:] {
		result = append(result, markdownRow(row))
	}
	return strings.Join(result, "\n")
}

func renderJSONL(documents []Document, chunks []Chunk, chunkSize int) (string, error) {
	var lines bytes.Buffer
	encoder := json.NewEncoder(&lines)
	encoder.SetEscapeHTML(false)
	if chunkSize > 0 {
		for _, chunk := range chunks {
			if err := encoder.Encode(chunk); err != nil {
				return "", err
			}
		}
	} else {
		for _, document := range documents {
			if err := encoder.Encode(document); err != nil {
				return "", err
			}
		}
	}
	return strings.TrimSuffix(lines.String(), "\n"), nil
}

func makeChunks(documents []Document, size int) []Chunk {
	if size <= 0 {
		return nil
	}
	var chunks []Chunk
	for _, document := range documents {
		parts := chunkText(strings.TrimSpace(document.Text), size)
		for index, part := range parts {
			chunks = append(chunks, Chunk{Source: document.Source, Index: index + 1, Text: part})
		}
	}
	return chunks
}

func chunkText(text string, size int) []string {
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= size {
		return []string{text}
	}
	runes := []rune(text)
	var chunks []string
	for len(runes) > 0 {
		end := size
		if end > len(runes) {
			end = len(runes)
		}
		if end < len(runes) {
			floor := end / 2
			for cursor := end; cursor > floor; cursor-- {
				if runes[cursor-1] == '\n' || runes[cursor-1] == ' ' {
					end = cursor
					break
				}
			}
		}
		part := strings.TrimSpace(string(runes[:end]))
		if part != "" {
			chunks = append(chunks, part)
		}
		runes = runes[end:]
	}
	return chunks
}

// markdownTables turns consecutive tab-separated rows into Markdown tables.
// Spreadsheet parsers already emit tabular rows, so this preserves useful
// structure without guessing at ordinary prose.
func markdownTables(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for index := 0; index < len(lines); {
		if !strings.Contains(lines[index], "\t") {
			result = append(result, lines[index])
			index++
			continue
		}
		end := index
		for end < len(lines) && strings.Contains(lines[end], "\t") {
			end++
		}
		if end-index < 2 {
			result = append(result, lines[index])
			index++
			continue
		}
		head := strings.Split(lines[index], "\t")
		result = append(result, markdownRow(head))
		separator := make([]string, len(head))
		for column := range separator {
			separator[column] = "---"
		}
		result = append(result, markdownRow(separator))
		for row := index + 1; row < end; row++ {
			result = append(result, markdownRow(strings.Split(lines[row], "\t")))
		}
		index = end
	}
	return strings.Join(result, "\n")
}

func markdownRow(cells []string) string {
	for index := range cells {
		cells[index] = strings.ReplaceAll(strings.TrimSpace(cells[index]), "|", "\\|")
	}
	return "| " + strings.Join(cells, " | ") + " |"
}
