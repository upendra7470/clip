package ods

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

	var tests = []struct {
		name     string
		content  string
		start    int
		end      int
		expected string
	}{
		{
			name:     "Full extraction",
			content:  "Row 1\nRow 2\nRow 3",
			start:    1,
			end:      3,
			expected: "Row 1\nRow 2\nRow 3",
		},
		{
			name:     "Row range extraction",
			content:  "Row 1\nRow 2\nRow 3",
			start:    2,
			end:      2,
			expected: "Row 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			result, err := parser.ExtractRows(tt.content, tt.start, tt.end)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
