//go:build linux

package screencapture

import "testing"

func TestSelectionRectangleNormalizesDragDirection(t *testing.T) {
	rect := selectionRectangle(100, 80, 20, 10)
	if rect.X != 20 || rect.Y != 10 || rect.Width != 80 || rect.Height != 70 {
		t.Fatalf("rectangle = %#v", rect)
	}
}

func TestX11PixelDecoding(t *testing.T) {
	if got := x11Pixel([]byte{0x33, 0x22, 0x11, 0x00}, 0); got != 0x00112233 {
		t.Fatalf("little-endian pixel = %#x", got)
	}
	if got := x11Pixel([]byte{0x00, 0x11, 0x22, 0x33}, 1); got != 0x00112233 {
		t.Fatalf("big-endian pixel = %#x", got)
	}
	if got := x11Channel(0x00112233, 0x00ff0000); got != 0x11 {
		t.Fatalf("red channel = %#x", got)
	}
}
