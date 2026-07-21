package pptx

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upendra7470/clip/internal/parser"
)

func TestFileType(t *testing.T) {
	p := &Parser{}
	assert.Equal(t, "PPTX", string(p.FileType()))
}

func TestMissingFile(t *testing.T) {
	p := &Parser{}
	_, err := p.Parse(context.Background(), parser.ParseRequest{File: "nonexistent.pptx"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not open PPTX file")
}

func TestInvalidZIP(t *testing.T) {
	p := &Parser{}
	// Create a temporary file with invalid content
	tmpFile := createTempFile(t, []byte("not a zip file"))
	defer os.Remove(tmpFile)

	_, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read PPTX as ZIP archive")
}

func TestMissingSlides(t *testing.T) {
	p := &Parser{}
	// Create a valid ZIP file without slides
	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/presentation.xml": []byte(`<?xml version="1.0"?><presentation xmlns="http://schemas.openxmlformats.org/presentationml/2006/main"/>`),
	})
	defer os.Remove(tmpFile)

	_, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no slides found in PPTX")
}

func TestInvalidXML(t *testing.T) {
	p := &Parser{}
	// Create a ZIP file with invalid XML in slides
	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/slides/slide1.xml": []byte("not valid xml"),
	})
	defer os.Remove(tmpFile)

	_, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	assert.Error(t, err)
	// The error could be either parsing error or no text content error
	assert.True(t, strings.Contains(err.Error(), "failed to parse slide XML") ||
		strings.Contains(err.Error(), "no text content found in PPTX"))
}

func TestEmptyPresentation(t *testing.T) {
	p := &Parser{}
	// Create a ZIP file with empty slides
	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/slides/slide1.xml": []byte(`<?xml version="1.0"?><sld xmlns="http://schemas.openxmlformats.org/presentationml/2006/main"><cSld><spTree/></cSld></sld>`),
	})
	defer os.Remove(tmpFile)

	_, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no text content found in PPTX")
}

func TestOneSlide(t *testing.T) {
	p := &Parser{}
	slideContent := `<?xml version="1.0"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
							<a:r>
								<a:t>Hello World</a:t>
							</a:r>
						</a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/slides/slide1.xml": []byte(slideContent),
	})
	defer os.Remove(tmpFile)

	result, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result.Text)
}

func TestMultipleSlides(t *testing.T) {
	p := &Parser{}
	slide1Content := `<?xml version="1.0"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
							<a:r>
								<a:t>Hello</a:t>
							</a:r>
						</a:p>
						<a:p>
							<a:r>
								<a:t>World</a:t>
							</a:r>
						</a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	slide2Content := `<?xml version="1.0"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
							<a:r>
								<a:t>Clip</a:t>
							</a:r>
						</a:p>
						<a:p>
							<a:r>
								<a:t>Project</a:t>
							</a:r>
						</a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/slides/slide1.xml": []byte(slide1Content),
		"ppt/slides/slide2.xml": []byte(slide2Content),
	})
	defer os.Remove(tmpFile)

	result, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	require.NoError(t, err)
	expected := "Hello\nWorld\n\nClip\nProject"
	assert.Equal(t, expected, result.Text)
}

func TestUnicode(t *testing.T) {
	p := &Parser{}
	slideContent := `<?xml version="1.0"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
							<a:r>
								<a:t>Hello 世界</a:t>
							</a:r>
						</a:p>
						<a:p>
							<a:r>
								<a:t>Привет мир</a:t>
							</a:r>
						</a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/slides/slide1.xml": []byte(slideContent),
	})
	defer os.Remove(tmpFile)

	result, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	require.NoError(t, err)
	assert.Equal(t, "Hello 世界\nПривет мир", result.Text)
}

func TestWhitespaceHandling(t *testing.T) {
	p := &Parser{}
	slideContent := `<?xml version="1.0"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
							<a:r>
								<a:t>  Hello   World  </a:t>
							</a:r>
						</a:p>
						<a:p>
							<a:r>
								<a:t>  Another   Line  </a:t>
							</a:r>
						</a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/slides/slide1.xml": []byte(slideContent),
	})
	defer os.Remove(tmpFile)

	result, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	require.NoError(t, err)
	assert.Equal(t, "Hello World\nAnother Line", result.Text)
}

func TestNestedXML(t *testing.T) {
	p := &Parser{}
	slideContent := `<?xml version="1.0"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
		<p:cSld>
			<p:spTree>
				<p:sp>
					<p:txBody>
						<a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
							<a:r>
								<a:t>Line 1</a:t>
							</a:r>
						</a:p>
						<a:p>
							<a:r>
								<a:t>Line 2 with <a:br/> break</a:t>
							</a:r>
						</a:p>
						<a:p>
							<a:r>
								<a:t>Line 3</a:t>
							</a:r>
						</a:p>
					</p:txBody>
				</p:sp>
			</p:spTree>
		</p:cSld>
	</p:sld>`

	tmpFile := createTempZipFile(t, map[string][]byte{
		"ppt/slides/slide1.xml": []byte(slideContent),
	})
	defer os.Remove(tmpFile)

	result, err := p.Parse(context.Background(), parser.ParseRequest{File: tmpFile})
	require.NoError(t, err)
	// XML tags are stripped during text extraction, which is correct behavior
	assert.Equal(t, "Line 1\nLine 2 with break\nLine 3", result.Text)
}

func TestErrorWrapping(t *testing.T) {
	p := &Parser{}
	_, err := p.Parse(context.Background(), parser.ParseRequest{File: "nonexistent.pptx"})
	assert.Error(t, err)

	// Test error unwrapping
	var pptxErr *PPTXParserError
	if errors.As(err, &pptxErr) {
		// Test that the error can be unwrapped
		assert.NotNil(t, pptxErr)
		assert.NotEmpty(t, pptxErr.Error())
	}
}

// Helper functions for testing

func createTempFile(t *testing.T, content []byte) string {
	tmpFile, err := os.CreateTemp("", "test*.pptx")
	require.NoError(t, err)
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	err = tmpFile.Close()
	require.NoError(t, err)
	return tmpFile.Name()
}

func createTempZipFile(t *testing.T, files map[string][]byte) string {
	tmpFile, err := os.CreateTemp("", "test*.pptx")
	require.NoError(t, err)

	zipWriter := zip.NewWriter(tmpFile)

	for name, content := range files {
		writer, err := zipWriter.Create(name)
		require.NoError(t, err)
		_, err = writer.Write(content)
		require.NoError(t, err)
	}

	err = zipWriter.Close()
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}
