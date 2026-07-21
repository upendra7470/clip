package parser

import (
	"testing"
)

func TestParseRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Range
		wantErr bool
		errMsg  string
	}{
		{
			name:    "single page",
			input:   "5",
			want:    Range{Start: 5, End: 5},
			wantErr: false,
		},
		{
			name:    "page range",
			input:   "5-10",
			want:    Range{Start: 5, End: 10},
			wantErr: false,
		},
		{
			name:    "single page with whitespace",
			input:   "  3  ",
			want:    Range{Start: 3, End: 3},
			wantErr: false,
		},
		{
			name:    "page range with whitespace",
			input:   "  2 - 8  ",
			want:    Range{Start: 2, End: 8},
			wantErr: false,
		},
		{
			name:    "invalid format - no digits",
			input:   "abc",
			wantErr: true,
			errMsg:  "invalid range: expected format like 5 or 5-10, got \"abc\"",
		},
		{
			name:    "invalid format - multiple dashes",
			input:   "1-2-3",
			wantErr: true,
			errMsg:  "invalid range: expected format like 5-10, got \"1-2-3\"",
		},
		{
			name:    "page number too low",
			input:   "0",
			wantErr: true,
			errMsg:  "range values must start from 1, got 0",
		},
		{
			name:    "range with start too low",
			input:   "0-5",
			wantErr: true,
			errMsg:  "range values must start from 1, got 0",
		},
		{
			name:    "range with end too low",
			input:   "5-0",
			wantErr: true,
			errMsg:  "range values must start from 1, got 0",
		},
		{
			name:    "reversed range",
			input:   "10-5",
			wantErr: true,
			errMsg:  "invalid range: start value must not be greater than end value (got 10-5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRange(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRange(%q) expected error, got none", tt.input)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ParseRange(%q) error = %v, want %v", tt.input, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ParseRange(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("ParseRange(%q) = %+v, want %+v", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestValidateRangeAgainstTotal(t *testing.T) {
	tests := []struct {
		name       string
		rangeObj   Range
		totalPages int
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid range",
			rangeObj:   Range{Start: 2, End: 4},
			totalPages: 5,
			wantErr:    false,
		},
		{
			name:       "single page valid",
			rangeObj:   Range{Start: 3, End: 3},
			totalPages: 5,
			wantErr:    false,
		},
		{
			name:       "range exceeds total pages",
			rangeObj:   Range{Start: 4, End: 6},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "requested range exceeds document unit count (document has 5 units, requested 4-6)",
		},
		{
			name:       "start page exceeds total",
			rangeObj:   Range{Start: 6, End: 7},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "requested range exceeds document unit count (document has 5 units, requested 6-7)",
		},
		{
			name:       "page number too low",
			rangeObj:   Range{Start: 0, End: 2},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "range values must start from 1",
		},
		{
			name:       "reversed range",
			rangeObj:   Range{Start: 4, End: 2},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "invalid range: start value must not be greater than end value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRangeAgainstTotal(tt.rangeObj, tt.totalPages)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateRangeAgainstTotal(%+v, %d) expected error, got none", tt.rangeObj, tt.totalPages)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ValidateRangeAgainstTotal(%+v, %d) error = %v, want %v", tt.rangeObj, tt.totalPages, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRangeAgainstTotal(%+v, %d) unexpected error: %v", tt.rangeObj, tt.totalPages, err)
				}
			}
		})
	}
}
