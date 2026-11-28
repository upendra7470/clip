//go:build !windows

package clipboard

import nativeclipboard "golang.design/x/clipboard"

func richClipboardOptions(plainText, html string) []nativeclipboard.Option {
	return []nativeclipboard.Option{
		nativeclipboard.Item{Format: htmlFormat, Bytes: []byte(html)},
		nativeclipboard.Item{Format: nativeclipboard.FmtText, Bytes: []byte(plainText)},
	}
}
