package parser

import (
	"testing"
)

func TestParsePageRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    PageRange
		wantErr bool
		errMsg  string
	}{
		{
			name:    "single page",
			input:   "5",
			want:    PageRange{Start: 5, End: 5},
			wantErr: false,
		},
		{
			name:    "page range",
			input:   "5-10",
			want:    PageRange{Start: 5, End: 10},
			wantErr: false,
		},
		{
			name:    "single page with whitespace",
			input:   "  3  ",
			want:    PageRange{Start: 3, End: 3},
			wantErr: false,
		},
		{
			name:    "page range with whitespace",
			input:   "  2 - 8  ",
			want:    PageRange{Start: 2, End: 8},
			wantErr: false,
		},
		{
			name:    "invalid format - no digits",
			input:   "abc",
			wantErr: true,
			errMsg:  "invalid page range: expected format like 5 or 5-10, got \"abc\"",
		},
		{
			name:    "invalid format - multiple dashes",
			input:   "1-2-3",
			wantErr: true,
			errMsg:  "invalid page range: expected format like 5-10, got \"1-2-3\"",
		},
		{
			name:    "page number too low",
			input:   "0",
			wantErr: true,
			errMsg:  "page numbers must start from 1, got 0",
		},
		{
			name:    "range with start too low",
			input:   "0-5",
			wantErr: true,
			errMsg:  "page numbers must start from 1, got 0",
		},
		{
			name:    "range with end too low",
			input:   "5-0",
			wantErr: true,
			errMsg:  "page numbers must start from 1, got 0",
		},
		{
			name:    "reversed range",
			input:   "10-5",
			wantErr: true,
			errMsg:  "invalid page range: start page must not be greater than end page (got 10-5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePageRange(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePageRange(%q) expected error, got none", tt.input)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ParsePageRange(%q) error = %v, want %v", tt.input, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ParsePageRange(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("ParsePageRange(%q) = %+v, want %+v", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestValidatePageRangeAgainstTotal(t *testing.T) {
	tests := []struct {
		name       string
		rangeObj   PageRange
		totalPages int
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid range",
			rangeObj:   PageRange{Start: 2, End: 4},
			totalPages: 5,
			wantErr:    false,
		},
		{
			name:       "single page valid",
			rangeObj:   PageRange{Start: 3, End: 3},
			totalPages: 5,
			wantErr:    false,
		},
		{
			name:       "range exceeds total pages",
			rangeObj:   PageRange{Start: 4, End: 6},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "requested page range exceeds document page count (document has 5 pages, requested 4-6)",
		},
		{
			name:       "start page exceeds total",
			rangeObj:   PageRange{Start: 6, End: 7},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "requested page range exceeds document page count (document has 5 pages, requested 6-7)",
		},
		{
			name:       "page number too low",
			rangeObj:   PageRange{Start: 0, End: 2},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "page numbers must start from 1",
		},
		{
			name:       "reversed range",
			rangeObj:   PageRange{Start: 4, End: 2},
			totalPages: 5,
			wantErr:    true,
			errMsg:     "invalid page range: start page must not be greater than end page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePageRangeAgainstTotal(tt.rangeObj, tt.totalPages)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePageRangeAgainstTotal(%+v, %d) expected error, got none", tt.rangeObj, tt.totalPages)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ValidatePageRangeAgainstTotal(%+v, %d) error = %v, want %v", tt.rangeObj, tt.totalPages, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePageRangeAgainstTotal(%+v, %d) unexpected error: %v", tt.rangeObj, tt.totalPages, err)
				}
			}
		})
	}
}
