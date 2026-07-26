package docx

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upendra7470/clip/internal/parser"
)

func TestSectionDetection(t *testing.T) {
	// Test that headings correctly start logical sections
	// Test section detection algorithm

	// Create test paragraphs with headings
	testParagraphs := []StructuredParagraph{
		{Content: "Introduction text", IsTable: false},
		{Content: "# Heading 1", IsTable: false}, // This should start a new section
		{Content: "Content under heading 1", IsTable: false},
		{Content: "# Heading 2", IsTable: false}, // This should start another section
		{Content: "Content under heading 2", IsTable: false},
	}

	sections := detectSections(testParagraphs)

	// Should detect 3 sections (intro, heading1 content, heading2 content)
	assert.Equal(t, 3, len(sections), "Should detect 3 sections from headings")

	// Verify section content
	assert.Equal(t, 1, len(sections[0].Paragraphs))
	assert.Contains(t, sections[0].Paragraphs[0].Content, "Introduction text")

	assert.Equal(t, 2, len(sections[1].Paragraphs))
	assert.Contains(t, sections[1].Paragraphs[0].Content, "# Heading 1")
	assert.Contains(t, sections[1].Paragraphs[1].Content, "Content under heading 1")

	assert.Equal(t, 2, len(sections[2].Paragraphs))
	assert.Contains(t, sections[2].Paragraphs[0].Content, "# Heading 2")
	assert.Contains(t, sections[2].Paragraphs[1].Content, "Content under heading 2")
}

func TestSectionRangeExtraction(t *testing.T) {
	// Test that range extraction works with sections
	// Create a test DOCX file with sections
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>Introduction</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t># Section 1</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Content of section 1</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t># Section 2</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Content of section 2</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t># Section 3</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Content of section 3</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Test extracting sections 1-2 (should get intro and section 1)
	result, err := docParser.ParseRange(context.Background(), req, 1, 2)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Introduction")
	assert.Contains(t, result.Text, "# Section 1")
	assert.Contains(t, result.Text, "Content of section 1")
	assert.NotContains(t, result.Text, "# Section 2")
	assert.NotContains(t, result.Text, "Content of section 2")

	// Test extracting sections 2-3 (should get section 1 and section 2)
	result, err = docParser.ParseRange(context.Background(), req, 2, 3)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "# Section 1")
	assert.Contains(t, result.Text, "Content of section 1")
	assert.Contains(t, result.Text, "# Section 2")
	assert.Contains(t, result.Text, "Content of section 2")
	assert.NotContains(t, result.Text, "Introduction")
	assert.NotContains(t, result.Text, "# Section 3")
}

func TestSectionRangeWithTables(t *testing.T) {
	// Test that section-based range extraction preserves table structure
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>Section 1 Introduction</w:t>
	           </w:r>
	       </w:p>
	       <w:tbl>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Header</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Data</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	       </w:tbl>
	       <w:p>
	           <w:r>
	               <w:t># Section 2</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Section 2 content</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Test extracting section 1 (should include table)
	result, err := docParser.ParseRange(context.Background(), req, 1, 1)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Section 1 Introduction")
	assert.Contains(t, result.Text, "| Table Header |")
	assert.Contains(t, result.Text, "| Table Data |")
	assert.NotContains(t, result.Text, "# Section 2")
	assert.NotContains(t, result.Text, "Section 2 content")

	// Test extracting sections 1-2 (should include both sections with table)
	result, err = docParser.ParseRange(context.Background(), req, 1, 2)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Section 1 Introduction")
	assert.Contains(t, result.Text, "| Table Header |")
	assert.Contains(t, result.Text, "# Section 2")
	assert.Contains(t, result.Text, "Section 2 content")
}

// TestParseRangeWithTables tests that ParseRange preserves table structure

// Test that table structure is preserved in range extraction
// Test that headings correctly start logical sections
func TestParseRangeWithTables(t *testing.T) {
	// Create a test DOCX file with paragraphs and tables
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>First paragraph</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Second paragraph</w:t>
	           </w:r>
	       </w:p>
	       <w:tbl>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Cell 1</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Cell 2</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	       </w:tbl>
	       <w:p>
	           <w:r>
	               <w:t>Third paragraph</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()

	// Test 1: Parse full document should include table
	req := parser.ParseRequest{File: docxPath}
	fullResult, err := docParser.ParseWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, fullResult.Text, "First paragraph")
	assert.Contains(t, fullResult.Text, "Second paragraph")
	assert.Contains(t, fullResult.Text, "| Table Cell 1 | Table Cell 2 |")
	assert.Contains(t, fullResult.Text, "Third paragraph")

	// Test 2: ParseRange with range 1-1 should return the only section
	rangeResult, err := docParser.ParseRange(context.Background(), req, 1, 1)
	assert.NoError(t, err)
	assert.Contains(t, rangeResult.Text, "First paragraph")
	assert.Contains(t, rangeResult.Text, "Second paragraph")
	assert.Contains(t, rangeResult.Text, "| Table Cell 1 | Table Cell 2 |")
	assert.Contains(t, rangeResult.Text, "Third paragraph")
}

// TestParseRangeConsistency tests that Parse and ParseRange use the same parsing logic

// Test that table structure is preserved in range extraction
// Test that headings correctly start logical sections
func TestParseRangeConsistency(t *testing.T) {
	// Create a test DOCX file with complex content
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>Paragraph 1</w:t>
	           </w:r>
	       </w:p>
	       <w:tbl>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Header</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Data</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	       </w:tbl>
	       <w:p>
	           <w:r>
	               <w:t>Paragraph 2</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Parse full document
	fullResult, err := docParser.ParseWithContext(context.Background(), req)
	assert.NoError(t, err)

	// Parse full range - get total paragraphs first by parsing
	// We'll use a large number for end to get all content
	rangeResult, err := docParser.ParseRange(context.Background(), req, 1, 100)
	assert.NoError(t, err)

	// Both should contain the same content (tables and paragraphs)
	// Check that both contain the essential elements
	assert.Contains(t, fullResult.Text, "Paragraph 1")
	assert.Contains(t, fullResult.Text, "| Header |")
	assert.Contains(t, fullResult.Text, "| Data |")
	assert.Contains(t, fullResult.Text, "Paragraph 2")

	assert.Contains(t, rangeResult.Text, "Paragraph 1")
	assert.Contains(t, rangeResult.Text, "| Header |")
	assert.Contains(t, rangeResult.Text, "| Data |")
	assert.Contains(t, rangeResult.Text, "Paragraph 2")
}

// TestParseRangePreservesUnicode tests that ParseRange preserves Unicode characters

// Test that table structure is preserved in range extraction
// Test that headings correctly start logical sections
func TestParseRangePreservesUnicode(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>English text</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>中文文字</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Русский текст</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Parse range for the only section (1-1)
	result, err := docParser.ParseRange(context.Background(), req, 1, 1)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "English text")
	assert.Contains(t, result.Text, "中文文字")
	assert.Contains(t, result.Text, "Русский текст")
}

func TestExtractTables(t *testing.T) {
	// Test that headings correctly start logical sections
	tests := []struct {
		name     string
		content  string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Table in selected range",
			content:  "Table 1\nTable 2\nTable 3",
			start:    2,
			end:      2,
			expected: "Table 2",
		},
		{
			name:     "Nested table paragraphs",
			content:  "Paragraph before table\nCell with nested paragraphs\nFirst nested paragraph\nSecond nested paragraph\nSimple cell\nParagraph after table",
			start:    2,
			end:      3,
			expected: "Cell with nested paragraphs\nFirst nested paragraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			result, err := parser.ExtractTables(tt.content, tt.start, tt.end)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFullDOCXExtraction(t *testing.T) {
	// Test that headings correctly start logical sections
	// Test that full DOCX extraction returns all content including tables and paragraphs
	// Create a test DOCX file with complex content including tables and paragraphs
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>First paragraph</w:t>
	           </w:r>
	       </w:p>
	       <w:tbl>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Cell 1</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Cell 2</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	       </w:tbl>
	       <w:p>
	           <w:r>
	               <w:t>Second paragraph</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Parse full document should include all content including tables and paragraphs
	result, err := docParser.ParseWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "First paragraph")
	assert.Contains(t, result.Text, "| Table Cell 1 | Table Cell 2 |")
	assert.Contains(t, result.Text, "Second paragraph")
}

func TestDOCXExtractionWithUnicode(t *testing.T) {
	// Test that headings correctly start logical sections
	// Test that DOCX extraction preserves Unicode characters
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>English text</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>中文文字</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Русский текст</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Parse full document should include all Unicode text
	result, err := docParser.ParseWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "English text")
	assert.Contains(t, result.Text, "中文文字")
	assert.Contains(t, result.Text, "Русский текст")
}

func TestDOCXExtractionWithTables(t *testing.T) {
	// Test that headings correctly start logical sections
	// Test that DOCX extraction preserves table structure
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:tbl>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Header 1</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Header 2</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Data 1</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Data 2</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	       </w:tbl>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Parse full document should include table structure
	result, err := docParser.ParseWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "| Header 1 | Header 2 |")
	assert.Contains(t, result.Text, "| Data 1 | Data 2 |")
}

func TestDOCXExtractionWithNestedParagraphs(t *testing.T) {
	// Test that headings correctly start logical sections
	// Test that DOCX extraction preserves nested paragraph structure
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>Paragraph 1</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Nested Paragraph 1</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Nested Paragraph 2</w:t>
	           </w:r>
	       </w:p>
	       <w:p>
	           <w:r>
	               <w:t>Paragraph 2</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Parse full document should include nested paragraph structure
	result, err := docParser.ParseWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Paragraph 1")
	assert.Contains(t, result.Text, "Nested Paragraph 1")
	assert.Contains(t, result.Text, "Nested Paragraph 2")
	assert.Contains(t, result.Text, "Paragraph 2")
}

func TestDOCXExtractionWithMixedContent(t *testing.T) {
	// Test that headings correctly start logical sections
	// Test that DOCX extraction preserves mixed content including paragraphs and tables
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	   <w:body>
	       <w:p>
	           <w:r>
	               <w:t>Paragraph 1</w:t>
	           </w:r>
	       </w:p>
	       <w:tbl>
	           <w:tr>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Cell 1</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	               <w:tc>
	                   <w:p>
	                       <w:r>
	                           <w:t>Table Cell 2</w:t>
	                       </w:r>
	                   </w:p>
	               </w:tc>
	           </w:tr>
	       </w:tbl>
	       <w:p>
	           <w:r>
	               <w:t>Paragraph 2</w:t>
	           </w:r>
	       </w:p>
	   </w:body>
</w:document>`

	// Create temporary DOCX file
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "test.docx")
	createTestDOCXFromXML(t, docxPath, xmlContent)

	docParser := NewParser()
	req := parser.ParseRequest{File: docxPath}

	// Parse full document should include mixed content including paragraphs and tables
	result, err := docParser.ParseWithContext(context.Background(), req)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Paragraph 1")
	assert.Contains(t, result.Text, "| Table Cell 1 | Table Cell 2 |")
	assert.Contains(t, result.Text, "Paragraph 2")
}
