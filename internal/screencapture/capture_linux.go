//go:build linux

package screencapture

import (
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/bits"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func capturePlatform(ctx context.Context) (string, error) {
	wayland := os.Getenv("WAYLAND_DISPLAY") != "" || strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland")
	if wayland {
		path, err := captureWayland(ctx)
		if err == nil || os.Getenv("DISPLAY") == "" {
			return path, err
		}
	}
	if os.Getenv("DISPLAY") == "" {
		return "", fmt.Errorf("screen capture requires a Wayland desktop portal or an X11 DISPLAY")
	}
	return captureX11(ctx)
}

func captureWayland(ctx context.Context) (string, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return "", fmt.Errorf("connect to Wayland screenshot portal: %w", err)
	}
	defer conn.Close()

	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return "", fmt.Errorf("listen for screenshot portal: %w", err)
	}
	defer conn.RemoveMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
	)

	token := fmt.Sprintf("clip%d", time.Now().UnixNano())
	options := map[string]dbus.Variant{
		"interactive":  dbus.MakeVariant(true),
		"modal":        dbus.MakeVariant(false),
		"handle_token": dbus.MakeVariant(token),
	}
	portal := conn.Object("org.freedesktop.portal.Desktop", dbus.ObjectPath("/org/freedesktop/portal/desktop"))
	var requestPath dbus.ObjectPath
	if err := portal.CallWithContext(ctx, "org.freedesktop.portal.Screenshot.Screenshot", 0, "", options).Store(&requestPath); err != nil {
		return "", fmt.Errorf("open Wayland screenshot portal: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case signal := <-signals:
			if signal == nil || signal.Path != requestPath || len(signal.Body) < 2 {
				continue
			}
			response, ok := signal.Body[0].(uint32)
			if !ok {
				return "", fmt.Errorf("invalid response from screenshot portal")
			}
			if response != 0 {
				return "", ErrCancelled
			}
			results, ok := signal.Body[1].(map[string]dbus.Variant)
			if !ok {
				return "", fmt.Errorf("screenshot portal returned no result")
			}
			uriValue, ok := results["uri"]
			if !ok {
				return "", fmt.Errorf("screenshot portal returned no image")
			}
			uri, ok := uriValue.Value().(string)
			if !ok {
				return "", fmt.Errorf("screenshot portal returned an invalid image URI")
			}
			return copyPortalImage(uri)
		}
	}
}

func copyPortalImage(uri string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return "", fmt.Errorf("invalid screenshot portal URI")
	}
	// #nosec G304 -- the path is issued by the trusted desktop portal.
	file, err := os.Open(parsed.Path)
	if err != nil {
		return "", fmt.Errorf("open portal screenshot: %w", err)
	}
	img, _, decodeErr := image.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil {
		return "", fmt.Errorf("decode portal screenshot: %w", decodeErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close portal screenshot: %w", closeErr)
	}
	return writePNG(img)
}

func captureX11(ctx context.Context) (string, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return "", fmt.Errorf("connect to X11 display: %w", err)
	}
	defer conn.Close()
	screen := xproto.Setup(conn).DefaultScreen(conn)

	gc, err := xproto.NewGcontextId(conn)
	if err != nil {
		return "", fmt.Errorf("create X11 selector: %w", err)
	}
	mask := uint32(xproto.GcFunction | xproto.GcForeground | xproto.GcLineWidth | xproto.GcSubwindowMode)
	values := []uint32{xproto.GxXor, screen.WhitePixel, 2, xproto.SubwindowModeIncludeInferiors}
	if err := xproto.CreateGCChecked(conn, gc, xproto.Drawable(screen.Root), mask, values).Check(); err != nil {
		return "", fmt.Errorf("create X11 selection overlay: %w", err)
	}
	defer xproto.FreeGC(conn, gc)

	grab, err := xproto.GrabPointer(conn, false, screen.Root,
		xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease|xproto.EventMaskPointerMotion,
		xproto.GrabModeAsync, xproto.GrabModeAsync, xproto.WindowNone, xproto.CursorNone, xproto.TimeCurrentTime).Reply()
	if err != nil || grab.Status != xproto.GrabStatusSuccess {
		return "", fmt.Errorf("another application is using the X11 pointer")
	}
	defer xproto.UngrabPointer(conn, xproto.TimeCurrentTime)

	var startX, startY, currentX, currentY int
	var dragging, drawn bool
	draw := func() {
		if !dragging {
			return
		}
		rect := selectionRectangle(startX, startY, currentX, currentY)
		if rect.Width == 0 || rect.Height == 0 {
			return
		}
		xproto.PolyRectangle(conn, xproto.Drawable(screen.Root), gc, []xproto.Rectangle{{
			// #nosec G115 -- coordinates originate as signed X11 int16 values and dimensions are their bounded difference.
			X: int16(rect.X), Y: int16(rect.Y), Width: uint16(rect.Width), Height: uint16(rect.Height),
		}})
		conn.Sync()
	}

	for {
		select {
		case <-ctx.Done():
			if drawn {
				draw()
			}
			return "", ctx.Err()
		default:
		}
		event, eventErr := conn.PollForEvent()
		if eventErr != nil {
			return "", fmt.Errorf("read X11 selection: %w", eventErr)
		}
		if event == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		switch value := event.(type) {
		case xproto.ButtonPressEvent:
			if value.Detail != 1 {
				continue
			}
			startX, startY = int(value.RootX), int(value.RootY)
			currentX, currentY = startX, startY
			dragging = true
		case xproto.MotionNotifyEvent:
			if !dragging {
				continue
			}
			if drawn {
				draw()
			}
			currentX, currentY = int(value.RootX), int(value.RootY)
			draw()
			drawn = true
		case xproto.ButtonReleaseEvent:
			if !dragging || value.Detail != 1 {
				continue
			}
			if drawn {
				draw()
			}
			currentX, currentY = int(value.RootX), int(value.RootY)
			xproto.UngrabPointer(conn, xproto.TimeCurrentTime)
			conn.Sync()
			rect := selectionRectangle(startX, startY, currentX, currentY)
			if rect.Width < 2 || rect.Height < 2 {
				return "", ErrCancelled
			}
			return captureX11Rectangle(conn, screen, rect)
		}
	}
}

type rectangle struct{ X, Y, Width, Height int }

func selectionRectangle(x1, y1, x2, y2 int) rectangle {
	x, y := min(x1, x2), min(y1, y2)
	return rectangle{X: x, Y: y, Width: max(x1, x2) - x, Height: max(y1, y2) - y}
}

func captureX11Rectangle(conn *xgb.Conn, screen *xproto.ScreenInfo, rect rectangle) (string, error) {
	img, err := readX11Image(conn, screen, rect)
	if err != nil {
		return "", fmt.Errorf("capture X11 region: %w", err)
	}
	return writePNG(img)
}

func readX11Image(conn *xgb.Conn, screen *xproto.ScreenInfo, rect rectangle) (*image.RGBA, error) {
	if rect.X < 0 || rect.Y < 0 || rect.X > 32767 || rect.Y > 32767 || rect.Width < 1 || rect.Height < 1 || rect.Width > 65535 || rect.Height > 65535 {
		return nil, fmt.Errorf("selected rectangle is outside X11 capture limits")
	}
	reply, err := xproto.GetImage(conn, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root),
		// #nosec G115 -- the bounds check above proves coordinates and dimensions fit the X11 wire types.
		int16(rect.X), int16(rect.Y), uint16(rect.Width), uint16(rect.Height), 0xffffffff).Reply()
	if err != nil {
		return nil, err
	}
	setup := xproto.Setup(conn)
	var format *xproto.Format
	for index := range setup.PixmapFormats {
		if setup.PixmapFormats[index].Depth == reply.Depth {
			format = &setup.PixmapFormats[index]
			break
		}
	}
	if format == nil || (format.BitsPerPixel != 16 && format.BitsPerPixel != 24 && format.BitsPerPixel != 32) {
		return nil, fmt.Errorf("unsupported X11 pixel depth %d", reply.Depth)
	}
	var visual *xproto.VisualInfo
	for _, depth := range screen.AllowedDepths {
		for index := range depth.Visuals {
			if depth.Visuals[index].VisualId == screen.RootVisual {
				visual = &depth.Visuals[index]
				break
			}
		}
	}
	if visual == nil {
		return nil, fmt.Errorf("X11 root visual was not found")
	}
	bpp := int(format.BitsPerPixel)
	pad := int(format.ScanlinePad)
	if pad == 0 {
		pad = 32
	}
	stride := ((rect.Width*bpp + pad - 1) / pad) * (pad / 8)
	if stride <= 0 || len(reply.Data) < stride*rect.Height {
		return nil, fmt.Errorf("X11 returned incomplete image data")
	}
	bytesPerPixel := bpp / 8
	result := image.NewRGBA(image.Rect(0, 0, rect.Width, rect.Height))
	for y := 0; y < rect.Height; y++ {
		row := reply.Data[y*stride:]
		for x := 0; x < rect.Width; x++ {
			offset := x * bytesPerPixel
			pixel := x11Pixel(row[offset:offset+bytesPerPixel], setup.ImageByteOrder)
			result.SetRGBA(x, y, color.RGBA{
				R: x11Channel(pixel, visual.RedMask),
				G: x11Channel(pixel, visual.GreenMask),
				B: x11Channel(pixel, visual.BlueMask),
				A: 255,
			})
		}
	}
	return result, nil
}

func x11Pixel(data []byte, order byte) uint32 {
	var buffer [4]byte
	if order == xproto.ImageOrderMSBFirst {
		copy(buffer[4-len(data):], data)
		return binary.BigEndian.Uint32(buffer[:])
	}
	copy(buffer[:], data)
	return binary.LittleEndian.Uint32(buffer[:])
}

func x11Channel(pixel, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	maximum := mask >> shift
	value := (pixel & mask) >> shift
	// #nosec G115 -- value is bounded by maximum, so the scaled result is in [0, 255].
	return uint8((uint64(value)*255 + uint64(maximum)/2) / uint64(maximum))
}

func writePNG(img image.Image) (string, error) {
	path, err := temporaryPNG()
	if err != nil {
		return "", err
	}
	// #nosec G304 -- path was created by temporaryPNG in the private system temp directory.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("open screenshot file: %w", err)
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("encode screenshot: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close screenshot file: %w", err)
	}
	return path, nil
}
