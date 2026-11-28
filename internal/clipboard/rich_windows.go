//go:build windows

package clipboard

import nativeclipboard "golang.design/x/clipboard"

var windowsHTMLFormat = nativeclipboard.Register("HTML Format")

func richClipboardOptions(plainText, html string) []nativeclipboard.Option {
	return []nativeclipboard.Option{
		// Microsoft Office and most Windows applications consume the CF_HTML
		// registered format, which includes byte-offset metadata.
		nativeclipboard.Item{Format: windowsHTMLFormat, Bytes: cfHTML(html)},
		nativeclipboard.Item{Format: htmlFormat, Bytes: []byte(html)},
		nativeclipboard.Item{Format: nativeclipboard.FmtText, Bytes: []byte(plainText)},
	}
}
