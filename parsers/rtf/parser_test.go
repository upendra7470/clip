package rtf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upendra7470/clip/internal/parser"
)

func TestFileType(t *testing.T) {
	p := &Parser{}
	assert.Equal(t, "RTF", string(p.FileType()))
}

func TestParse_MissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{File: "nonexistent.rtf"}

	_, err := p.Parse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")

	// Test error wrapping
	var rtfErr *RTFParserError
	assert.ErrorAs(t, err, &rtfErr)
}

func TestParse_EmptyFile(t *testing.T) {
	// Create empty file
	tempFile := createTempFile(t, "")
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	_, err := p.Parse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is empty")
}

func TestParse_InvalidRTF(t *testing.T) {
	// Create file with invalid RTF content
	invalidContent := "This is not valid RTF"
	tempFile := createTempFile(t, invalidContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	_, err := p.Parse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid RTF format")
}

func TestParse_UnbalancedGroups(t *testing.T) {
	// Create file with unbalanced RTF groups
	unbalancedContent := "{\\rtf1\\ansi This has unbalanced groups"
	tempFile := createTempFile(t, unbalancedContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	_, err := p.Parse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid RTF format")
}

func TestParse_PlainTextExtraction(t *testing.T) {
	// Simple RTF with plain text
	rtfContent := `{\rtf1\ansi\ansicpg1252\deff0\deflang1033{\fonttbl{\f0\fnil\fcharset0 Calibri;}}
{\*\generator Msftedit 5.41.21.2510;}\viewkind4\uc1\pard\sa200\sl276\slmult1\f0\fs22 This is simple text.\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "This is simple text")
}

func TestParse_BoldItalicRemoval(t *testing.T) {
	// RTF with bold and italic formatting
	rtfContent := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}
{\colortbl ;\red0\green0\blue0;}
\b This is bold\b0  and \i this is italic\i0  and normal text.\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "This is bold")
	assert.Contains(t, result.Text, "and")
	assert.Contains(t, result.Text, "this is italic")
	assert.Contains(t, result.Text, "and normal text")

	// Should not contain formatting codes
	assert.NotContains(t, result.Text, "\\b")
	assert.NotContains(t, result.Text, "\\i")
}

func TestParse_ParagraphPreservation(t *testing.T) {
	// RTF with multiple paragraphs
	rtfContent := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}
First paragraph\par
Second paragraph\par
Third paragraph\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "First paragraph")
	assert.Contains(t, result.Text, "Second paragraph")
	assert.Contains(t, result.Text, "Third paragraph")
}

func TestParse_UnicodeExtraction(t *testing.T) {
	// RTF with unicode characters
	rtfContent := `{\rtf1\ansi\ansicpg1252\deff0\deflang1033{\fonttbl{\f0\fnil\fcharset0 Calibri;}}
{\*\generator Msftedit 5.41.21.2510;}\viewkind4\uc1\pard\sa200\sl276\slmult1\f0\fs22 Hello \u9733 World\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Hello")
	assert.Contains(t, result.Text, "World")
}

func TestParse_EscapedCharacterDecoding(t *testing.T) {
	// RTF with escaped characters
	rtfContent := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}
Escaped chars: \\ \{ \} \'27 (apostrophe)\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Escaped chars")
	assert.Contains(t, result.Text, "\\")
	assert.Contains(t, result.Text, "{")
	assert.Contains(t, result.Text, "}")
	assert.Contains(t, result.Text, "'") // apostrophe
}

func TestParse_NestedGroups(t *testing.T) {
	// RTF with nested groups
	rtfContent := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}
{\b Outer {\i nested {\ul double nested} groups} text}\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Outer")
	assert.Contains(t, result.Text, "nested")
	assert.Contains(t, result.Text, "double nested")
	assert.Contains(t, result.Text, "groups")
	assert.Contains(t, result.Text, "text")
}

func TestParse_MultipleParagraphs(t *testing.T) {
	// RTF with multiple paragraphs and formatting
	rtfContent := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}
{\colortbl ;\red0\green0\blue0;}
\b Title\b0\par
\par
This is the first paragraph with some \i italic text\i0 and \b bold text\b0.\par
\par
This is the second paragraph.\par
\par
{\ul Underlined heading}\par
More content here.\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Title")
	assert.Contains(t, result.Text, "This is the first paragraph")
	assert.Contains(t, result.Text, "italic text")
	assert.Contains(t, result.Text, "bold text")
	assert.Contains(t, result.Text, "This is the second paragraph")
	assert.Contains(t, result.Text, "Underlined heading")
	assert.Contains(t, result.Text, "More content here")
}

func TestParse_HexCharacterDecoding(t *testing.T) {
	// RTF with hexadecimal character codes
	rtfContent := `{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}
Hex chars: \'e9 \'e0 \'e8 (accented letters)\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Hex chars")
	// Should decode the hex characters (é à è)
	assert.Contains(t, result.Text, "é")
	assert.Contains(t, result.Text, "à")
	assert.Contains(t, result.Text, "è")
}

func TestParse_ComplexRTF(t *testing.T) {
	// Complex RTF with various features
	rtfContent := `{\rtf1\ansi\ansicpg1252\deff0\deflang1033{\fonttbl{\f0\fnil\fcharset0 Calibri;}{\f1\fnil\fcharset2 Symbol;}}
{\colortbl ;\red0\green0\blue0;\red255\green0\blue0;}
{\*\generator Msftedit 5.41.21.2510;}\viewkind4\uc1\pard\sa200\sl276\slmult1\lang9\f0\fs22{\field{\*\fldinst{HYPERLINK "http://example.com"}}{\fldrslt{\ul\cf1 Example Link}}}\par
\par
This is a \b complex \b0 document with \i various \i0 formatting \ul options\ul0.\par
\par
\fs32 Large text \fs22 normal text \fs16 small text.\par
\par
\cf1 Red text \cf0 black text.\par
}`

	tempFile := createTempFile(t, rtfContent)
	defer os.Remove(tempFile)

	p := &Parser{}
	req := parser.ParseRequest{File: tempFile}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Example Link")
	assert.Contains(t, result.Text, "This is a")
	assert.Contains(t, result.Text, "complex")
	assert.Contains(t, result.Text, "document with")
	assert.Contains(t, result.Text, "various")
	assert.Contains(t, result.Text, "formatting")
	assert.Contains(t, result.Text, "options")
	assert.Contains(t, result.Text, "Large text")
	assert.Contains(t, result.Text, "normal text")
	assert.Contains(t, result.Text, "small text")
	assert.Contains(t, result.Text, "Red text")
	assert.Contains(t, result.Text, "black text")
}

// Helper function to create temporary test files
func createTempFile(t *testing.T, content string) string {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.rtf")

	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	return tempFile
}
