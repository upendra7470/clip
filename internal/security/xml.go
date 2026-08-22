package security

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// NewXMLDecoder creates a hardened XML decoder that prevents XXE and
// entity expansion attacks. It sets Strict mode and disables custom
// entity expansion.
func NewXMLDecoder(r io.Reader) *xml.Decoder {
	decoder := xml.NewDecoder(r)
	decoder.Strict = true
	decoder.Entity = map[string]string{}
	return decoder
}

// CheckXMLDepth tracks XML element nesting depth and returns an error
// if the depth exceeds MaxXMLDepth. This prevents stack exhaustion from
// deeply nested XML documents.
func CheckXMLDepth(depth int) error {
	if depth > MaxXMLDepth {
		return fmt.Errorf("XML nesting depth exceeds maximum allowed depth of %d", MaxXMLDepth)
	}
	return nil
}

// CheckTokenCount tracks the number of XML tokens processed and returns
// an error if the count exceeds MaxTokenCount. This prevents CPU
// exhaustion from pathological XML documents with excessive tokens.
func CheckTokenCount(count int) error {
	if count > MaxTokenCount {
		return fmt.Errorf("XML token count exceeds maximum allowed count of %d", MaxTokenCount)
	}
	return nil
}

// SafeXMLContent checks if XML content contains dangerous patterns
// before parsing. This is a defense-in-depth measure.
func SafeXMLContent(content string) bool {
	// Reject DOCTYPE declarations which can be used for XXE attacks
	if strings.Contains(content, "<!DOCTYPE") || strings.Contains(content, "<!doctype") {
		return false
	}
	return true
}
