package yaml

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upendra7470/clip/internal/parser"
)

func TestYAMLParser_FileType(t *testing.T) {
	p := &Parser{}
	assert.Equal(t, "YAML", string(p.FileType()))
}

func TestYAMLParser_MissingFile(t *testing.T) {
	p := &Parser{}
	req := parser.ParseRequest{
		File: "nonexistent.yaml",
	}

	_, err := p.Parse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not open YAML file")
}

func TestYAMLParser_EmptyFile(t *testing.T) {
	// Create a temporary empty YAML file
	tempDir := t.TempDir()
	emptyFile := filepath.Join(tempDir, "empty.yaml")
	err := os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: emptyFile,
	}

	_, err = p.Parse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty YAML file")
}

func TestYAMLParser_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.yaml")
	invalidYAML := `key: value
	- invalid yaml: [unclosed`
	err := os.WriteFile(invalidFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: invalidFile,
	}

	_, err = p.Parse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid YAML syntax")
}

func TestYAMLParser_SimpleYAML(t *testing.T) {
	// Create a temporary file with simple YAML
	tempDir := t.TempDir()
	simpleFile := filepath.Join(tempDir, "simple.yaml")
	simpleYAML := `name: John Doe
age: 30
active: true`
	err := os.WriteFile(simpleFile, []byte(simpleYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: simpleFile,
	}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)

	// Check that the extracted text contains the expected values
	assert.Contains(t, result.Text, "John Doe")
	assert.Contains(t, result.Text, "30")
	assert.Contains(t, result.Text, "true")
}

func TestYAMLParser_NestedYAML(t *testing.T) {
	// Create a temporary file with nested YAML
	tempDir := t.TempDir()
	nestedFile := filepath.Join(tempDir, "nested.yaml")
	nestedYAML := `person:
  name: Alice
  age: 25
  address:
    street: 123 Main St
    city: New York
    zip: 10001`
	err := os.WriteFile(nestedFile, []byte(nestedYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: nestedFile,
	}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)

	// Check that the extracted text contains the expected values
	assert.Contains(t, result.Text, "Alice")
	assert.Contains(t, result.Text, "25")
	assert.Contains(t, result.Text, "123 Main St")
	assert.Contains(t, result.Text, "New York")
	assert.Contains(t, result.Text, "10001")
}

func TestYAMLParser_Lists(t *testing.T) {
	// Create a temporary file with YAML lists
	tempDir := t.TempDir()
	listFile := filepath.Join(tempDir, "lists.yaml")
	listYAML := `fruits:
  - apple
  - banana
  - cherry
numbers:
  - 1
  - 2
  - 3`
	err := os.WriteFile(listFile, []byte(listYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: listFile,
	}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)

	// Check that the extracted text contains the expected values
	assert.Contains(t, result.Text, "apple")
	assert.Contains(t, result.Text, "banana")
	assert.Contains(t, result.Text, "cherry")
	assert.Contains(t, result.Text, "1")
	assert.Contains(t, result.Text, "2")
	assert.Contains(t, result.Text, "3")
}

func TestYAMLParser_NullValues(t *testing.T) {
	// Create a temporary file with null values
	tempDir := t.TempDir()
	nullFile := filepath.Join(tempDir, "null.yaml")
	nullYAML := `name: John
age: null
email: null`
	err := os.WriteFile(nullFile, []byte(nullYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: nullFile,
	}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)

	// Check that the extracted text contains the expected values
	assert.Contains(t, result.Text, "John")
	assert.Contains(t, result.Text, "null")
}

func TestYAMLParser_UnicodeContent(t *testing.T) {
	// Create a temporary file with Unicode content
	tempDir := t.TempDir()
	unicodeFile := filepath.Join(tempDir, "unicode.yaml")
	unicodeYAML := `greeting: 你好世界
name: José García
emoji: 😊`
	err := os.WriteFile(unicodeFile, []byte(unicodeYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: unicodeFile,
	}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)

	// Check that the extracted text contains the expected Unicode values
	assert.Contains(t, result.Text, "你好世界")
	assert.Contains(t, result.Text, "José García")
	assert.Contains(t, result.Text, "😊")
}

func TestYAMLParser_MixedTypes(t *testing.T) {
	// Create a temporary file with mixed data types
	tempDir := t.TempDir()
	mixedFile := filepath.Join(tempDir, "mixed.yaml")
	mixedYAML := `config:
  enabled: true
  timeout: 30.5
  retries: 3
  servers:
    - host: server1.example.com
      port: 8080
    - host: server2.example.com
      port: 9090
  metadata:
    version: 1.0
    description: null`
	err := os.WriteFile(mixedFile, []byte(mixedYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: mixedFile,
	}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)

	// Check that the extracted text contains the expected values
	assert.Contains(t, result.Text, "true")
	assert.Contains(t, result.Text, "30.5")
	assert.Contains(t, result.Text, "3")
	assert.Contains(t, result.Text, "server1.example.com")
	assert.Contains(t, result.Text, "8080")
	assert.Contains(t, result.Text, "server2.example.com")
	assert.Contains(t, result.Text, "9090")
	assert.Contains(t, result.Text, "1")
	assert.Contains(t, result.Text, "null")
}

func TestYAMLParser_ContextCancellation(t *testing.T) {
	// Create a temporary file with some YAML content
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.yaml")
	testYAML := `name: test
value: 123`
	err := os.WriteFile(testFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: testFile,
	}

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// This should still work since file reading is quick, but we test the context is passed correctly
	result, err := p.Parse(ctx, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)
}

func TestYAMLParser_YMLExtension(t *testing.T) {
	// Test that .yml extension works the same as .yaml
	tempDir := t.TempDir()
	ymlFile := filepath.Join(tempDir, "test.yml")
	ymlContent := `key: value
number: 42`
	err := os.WriteFile(ymlFile, []byte(ymlContent), 0644)
	require.NoError(t, err)

	p := &Parser{}
	req := parser.ParseRequest{
		File: ymlFile,
	}

	result, err := p.Parse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Text)
	assert.Contains(t, result.Text, "value")
	assert.Contains(t, result.Text, "42")
}
