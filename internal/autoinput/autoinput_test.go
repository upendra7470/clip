package autoinput

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassify(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "report.txt")
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		value string
		kind  Kind
	}{
		{name: "path", value: `"` + path + `"`, kind: Path},
		{name: "url", value: "https://example.com/report.pdf", kind: URL},
		{name: "text", value: "Notes for tomorrow\nsecond line", kind: Text},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := Classify(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if input.Kind != test.kind {
				t.Fatalf("kind = %q, want %q", input.Kind, test.kind)
			}
		})
	}
}

func TestClassifyRejectsUnsafeRemoteInputs(t *testing.T) {
	for _, value := range []string{
		"http://example.com/report.pdf",
		"https://user:secret@example.com/report.pdf",
		"\x00bad",
	} {
		if _, err := Classify(value); err == nil {
			t.Fatalf("Classify(%q) succeeded", value)
		}
	}
}
