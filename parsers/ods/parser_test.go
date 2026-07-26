package ods

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upendra7470/clip/internal/filetype"
	"github.com/upendra7470/clip/internal/parser"
)

func TestODSParser(t *testing.T) {
	// Test data
	odsContent := `Row 1
Row 2
Row 3`

	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.ods")
	err := os.WriteFile(testFile, []byte(odsContent), 0644)
	assert.NoError(t, err)

	// Initialize parser
	odsParser := NewParser()

	// Test ParseFile
	t.Run("ParseFile", func(t *testing.T) {
		docUnit, err := odsParser.ParseFile(testFile)
		assert.NoError(t, err)
		assert.NotNil(t, docUnit)
		assert.Contains(t, docUnit.Text, "ods file content for")
		assert.Equal(t, filetype.FileTypeODS, docUnit.Meta["type"])
		assert.Equal(t, testFile, docUnit.Meta["path"])
	})

	// Test ParseWithContext
	t.Run("ParseWithContext", func(t *testing.T) {
		req := parser.ParseRequest{
			File: testFile,
		}
		result, err := odsParser.ParseWithContext(context.Background(), req)
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
		result, err := odsParser.ParseRange(context.Background(), req, 2, 2)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		assert.Contains(t, result.Text, "Row 2")
	})

	// Test FileType
	t.Run("FileType", func(t *testing.T) {
		assert.Equal(t, filetype.FileTypeODS, odsParser.FileType())
	})

	// Test GetRangeUnit
	t.Run("GetRangeUnit", func(t *testing.T) {
		assert.Equal(t, parser.RangeUnit("rows"), odsParser.GetRangeUnit())
	})

	// Test ParseDirectory
	t.Run("ParseDirectory", func(t *testing.T) {
		_, err := odsParser.ParseDirectory(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}
