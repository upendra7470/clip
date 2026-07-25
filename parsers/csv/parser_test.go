package csv

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/upendra7470/clip/internal/filetype"
)

func TestCSVParser(t *testing.T) {
	// Test data
	csvContent := `Name,Age,Occupation
John,30,Engineer
Jane,25,Designer
Bob,40,Manager`

	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.csv")
	err := os.WriteFile(testFile, []byte(csvContent), 0644)
	assert.NoError(t, err)

	// Initialize parser
	parser := NewParser()

	// Test ParseFile
	t.Run("ParseFile", func(t *testing.T) {
		docUnit, err := parser.ParseFile(testFile)
		assert.NoError(t, err)
		assert.NotNil(t, docUnit)
		assert.Contains(t, docUnit.Text, "csv file content for")
		assert.Equal(t, filetype.FileTypeCSV, docUnit.Meta["type"])
		assert.Equal(t, testFile, docUnit.Meta["path"])
	})

	// Test ParseWithContext
	t.Run("ParseWithContext", func(t *testing.T) {
		req := parser.ParseRequest{
			File: testFile,
		}
		result, err := parser.ParseWithContext(context.Background(), req)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		assert.Contains(t, result.Text, "John, 30, Engineer")
		assert.Contains(t, result.Text, "Jane, 25, Designer")
		assert.Contains(t, result.Text, "Bob, 40, Manager")
	})

	// Test ParseRange
	t.Run("ParseRange", func(t *testing.T) {
		req := parser.ParseRequest{
			File: testFile,
		}
		result, err := parser.ParseRange(context.Background(), req, 2, 3)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.Text)
		assert.Contains(t, result.Text, "Jane, 25, Designer")
		assert.Contains(t, result.Text, "Bob, 40, Manager")
	})

	// Test FileType
	t.Run("FileType", func(t *testing.T) {
		assert.Equal(t, filetype.FileTypeCSV, parser.FileType())
	})

	// Test GetRangeUnit
	t.Run("GetRangeUnit", func(t *testing.T) {
		assert.Equal(t, "rows", parser.GetRangeUnit())
	})

	// Test ParseDirectory
	t.Run("ParseDirectory", func(t *testing.T) {
		_, err := parser.ParseDirectory(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})

	// Test error cases
	t.Run("ErrorCases", func(t *testing.T) {
		// Non-existent file
		req := parser.ParseRequest{
			File: "nonexistent.csv",
		}
		_, err := parser.ParseWithContext(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not exist")

		// Invalid row range
		_, err = parser.ParseRange(context.Background(), req, 0, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "row numbers must start from 1")

		// Invalid row range (start > end)
		_, err = parser.ParseRange(context.Background(), req, 2, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "start row must not be greater than end row")

		// Row range exceeds file content
		_, err = parser.ParseRange(context.Background(), req, 1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requested row range exceeds CSV row count")
	})
}
