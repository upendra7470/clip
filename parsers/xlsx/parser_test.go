package xlsx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

func TestXLSXParser(t *testing.T) {
	// Test data
	xlsxContent := `Row 1
Row 2
Row 3`

	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.xlsx")
	err := os.WriteFile(testFile, []byte(xlsxContent), 0644)
	assert.NoError(t, err)

	// Initialize parser
	xlsxParser := NewParser()

	// Test ParseFile
	t.Run("ParseFile", func(t *testing.T) {
		docUnit, err := xlsxParser.ParseFile(testFile)
		assert.NoError(t, err)
		assert.NotNil(t, docUnit)
		assert.Contains(t, docUnit.Text, "xlsx file content for")
		assert.Equal(t, filetype.FileTypeXLSX, docUnit.Meta["type"])
		assert.Equal(t, testFile, docUnit.Meta["path"])
	})

	// Test ParseWithContext
	t.Run("ParseWithContext", func(t *testing.T) {
		req := parser.ParseRequest{
			File: testFile,
		}
		result, err := xlsxParser.ParseWithContext(context.Background(), req)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		assert.Contains(t, result.Text, "Row 1")
		assert.Contains(t, result.Text, "Row 2")
		assert.Contains(t, result.Text, "Row 3")
	})

	// Test ParseRange
	t.Run("ParseRange", func(t *testing.T) {
		req := parser.ParseRequest{
			File: testFile,
		}
		result, err := xlsxParser.ParseRange(context.Background(), req, 2, 2)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		assert.Contains(t, result.Text, "Row 2")
	})

	// Test FileType
	t.Run("FileType", func(t *testing.T) {
		assert.Equal(t, filetype.FileTypeXLSX, xlsxParser.FileType())
	})

	// Test GetRangeUnit
	t.Run("GetRangeUnit", func(t *testing.T) {
		assert.Equal(t, "rows", xlsxParser.GetRangeUnit())
	})

	// Test ParseDirectory
	t.Run("ParseDirectory", func(t *testing.T) {
		_, err := xlsxParser.ParseDirectory(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}
