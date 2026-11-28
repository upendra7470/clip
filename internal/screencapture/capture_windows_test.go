//go:build windows

package screencapture

import (
	"os"
	"os/exec"
	"testing"
)

func TestWindowsCaptureScriptParses(t *testing.T) {
	command := `$tokens=$null; $errors=$null; [System.Management.Automation.Language.Parser]::ParseInput([Text.Encoding]::Unicode.GetString([Convert]::FromBase64String($env:CLIP_TEST_SCRIPT)), [ref]$tokens, [ref]$errors) | Out-Null; if ($errors.Count -gt 0) { $errors | ForEach-Object { Write-Error $_ }; exit 1 }`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command)
	cmd.Env = append(os.Environ(), "CLIP_TEST_SCRIPT="+encodePowerShell(windowsCaptureScript))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("PowerShell parser rejected capture script: %v\n%s", err, output)
	}
}
