// Package autoinput classifies text copied to the clipboard for Clip's
// universal input mode.
package autoinput

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Kind string

const (
	Text Kind = "text"
	Path Kind = "path"
	URL  Kind = "url"
)

type Input struct {
	Kind  Kind
	Value string
}

// Classify recognizes an existing local path, an HTTPS URL, or plain text.
// Local paths win over URL parsing so filenames containing ':' remain usable.
func Classify(value string) (Input, error) {
	if strings.IndexByte(value, 0) >= 0 {
		return Input{}, fmt.Errorf("clipboard text contains a NUL byte")
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Input{}, fmt.Errorf("clipboard text is empty")
	}

	candidate := unquotePath(trimmed)
	if info, err := os.Stat(candidate); err == nil {
		if info.Mode().IsRegular() || info.IsDir() {
			absolute, err := filepath.Abs(candidate)
			if err != nil {
				return Input{}, fmt.Errorf("resolve clipboard path: %w", err)
			}
			return Input{Kind: Path, Value: absolute}, nil
		}
	}

	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		if parsed.Scheme != "https" {
			return Input{}, fmt.Errorf("remote input requires an https URL")
		}
		if parsed.User != nil {
			return Input{}, fmt.Errorf("remote input URL must not contain credentials")
		}
		return Input{Kind: URL, Value: parsed.String()}, nil
	}

	return Input{Kind: Text, Value: value}, nil
}

func unquotePath(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
