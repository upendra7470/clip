$ErrorActionPreference = "Stop"
Remove-Item "HKCU:\Software\Classes\*\shell\ClipSmart" -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item "HKCU:\Software\Classes\Directory\shell\ClipSmart" -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "Clip context-menu actions removed."
