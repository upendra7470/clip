//go:build windows

package screencapture

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
)

const windowsCaptureScript = `
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition 'using System.Runtime.InteropServices; public static class ClipDpi { [DllImport("user32.dll")] public static extern bool SetProcessDPIAware(); }'
[ClipDpi]::SetProcessDPIAware() | Out-Null
[System.Windows.Forms.Application]::EnableVisualStyles()

$bounds = [System.Windows.Forms.SystemInformation]::VirtualScreen
$form = New-Object System.Windows.Forms.Form
$form.FormBorderStyle = [System.Windows.Forms.FormBorderStyle]::None
$form.StartPosition = [System.Windows.Forms.FormStartPosition]::Manual
$form.Bounds = $bounds
$form.TopMost = $true
$form.ShowInTaskbar = $false
$form.BackColor = [System.Drawing.Color]::Black
$form.Opacity = 0.25
$form.Cursor = [System.Windows.Forms.Cursors]::Cross
$form.KeyPreview = $true

$script:start = [System.Drawing.Point]::Empty
$script:current = [System.Drawing.Point]::Empty
$script:selected = [System.Drawing.Rectangle]::Empty
$script:dragging = $false

$form.Add_KeyDown({
    if ($_.KeyCode -eq [System.Windows.Forms.Keys]::Escape) {
        $form.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
        $form.Close()
    }
})
$form.Add_MouseDown({
    if ($_.Button -eq [System.Windows.Forms.MouseButtons]::Left) {
        $script:start = $_.Location
        $script:current = $_.Location
        $script:dragging = $true
    }
})
$form.Add_MouseMove({
    if ($script:dragging) {
        $script:current = $_.Location
        $form.Invalidate()
    }
})
$form.Add_Paint({
    if ($script:dragging) {
        $x = [Math]::Min($script:start.X, $script:current.X)
        $y = [Math]::Min($script:start.Y, $script:current.Y)
        $w = [Math]::Abs($script:start.X - $script:current.X)
        $h = [Math]::Abs($script:start.Y - $script:current.Y)
        $pen = [System.Drawing.Pen]::new([System.Drawing.Color]::White, 2)
        $_.Graphics.DrawRectangle($pen, $x, $y, $w, $h)
        $pen.Dispose()
    }
})
$form.Add_MouseUp({
    if ($script:dragging -and $_.Button -eq [System.Windows.Forms.MouseButtons]::Left) {
        $x = [Math]::Min($script:start.X, $_.X)
        $y = [Math]::Min($script:start.Y, $_.Y)
        $w = [Math]::Abs($script:start.X - $_.X)
        $h = [Math]::Abs($script:start.Y - $_.Y)
        $script:selected = [System.Drawing.Rectangle]::new($x + $bounds.X, $y + $bounds.Y, $w, $h)
        $script:dragging = $false
        if ($w -ge 2 -and $h -ge 2) {
            $form.DialogResult = [System.Windows.Forms.DialogResult]::OK
        } else {
            $form.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
        }
        $form.Close()
    }
})

$result = $form.ShowDialog()
$form.Dispose()
if ($result -ne [System.Windows.Forms.DialogResult]::OK) { exit 2 }

[System.Threading.Thread]::Sleep(120)
$bitmap = [System.Drawing.Bitmap]::new($selected.Width, $selected.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen($selected.Location, [System.Drawing.Point]::Empty, $selected.Size)
$graphics.Dispose()
$bitmap.Save($env:CLIP_SCREEN_OUTPUT, [System.Drawing.Imaging.ImageFormat]::Png)
$bitmap.Dispose()
`

func capturePlatform(ctx context.Context) (string, error) {
	path, err := temporaryPNG()
	if err != nil {
		return "", err
	}
	encoded := encodePowerShell(windowsCaptureScript)
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-STA", "-EncodedCommand", encoded)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = append(os.Environ(), "CLIP_SCREEN_OUTPUT="+path)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		_ = os.Remove(path)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return "", ErrCancelled
		}
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = runErr.Error()
		}
		return "", fmt.Errorf("capture screen region: %s", detail)
	}
	return path, nil
}

func encodePowerShell(script string) string {
	encoded := utf16.Encode([]rune(script))
	bytes := make([]byte, len(encoded)*2)
	for index, value := range encoded {
		bytes[index*2] = byte(value)
		bytes[index*2+1] = byte(value >> 8)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}
