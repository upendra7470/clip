//go:build darwin

package clipboard

import (
	"testing"
)

func TestCopyDarwin(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{
			name:    "successful copy",
			text:    "test text",
			wantErr: false,
		},
		{
			name:    "empty text",
			text:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := copyDarwin(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyDarwin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
