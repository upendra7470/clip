package security

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// XMLDecoder enforces the shared XML nesting and token limits for every
// document parser that consumes XML through NewXMLDecoder.
type XMLDecoder struct {
	*xml.Decoder
	depth  int
	tokens int
}

// NewXMLDecoder creates a hardened XML decoder that prevents XXE and
// entity expansion attacks. It sets Strict mode and disables custom
// entity expansion.
func NewXMLDecoder(r io.Reader) *XMLDecoder {
	decoder := xml.NewDecoder(r)
	decoder.Strict = true
	decoder.Entity = map[string]string{}
	return &XMLDecoder{Decoder: decoder}
}

// Token returns the next token while enforcing depth and token-count limits.
func (d *XMLDecoder) Token() (xml.Token, error) {
	token, err := d.Decoder.Token()
	if err != nil {
		return nil, err
	}
	d.tokens++
	if err := CheckTokenCount(d.tokens); err != nil {
		return nil, err
	}
	switch token.(type) {
	case xml.StartElement:
		d.depth++
		if err := CheckXMLDepth(d.depth); err != nil {
			return nil, err
		}
	case xml.EndElement:
		if d.depth > 0 {
			d.depth--
		}
	}
	return token, nil
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
