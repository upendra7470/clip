$ErrorActionPreference = "Stop"

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
if ($arch -ne "x64") {
    throw "Clip currently supports Windows x64. Detected: $arch"
}

$installDir = if ($env:CLIP_INSTALL_DIR) { $env:CLIP_INSTALL_DIR } else { Join-Path $HOME ".local\bin" }
$destination = Join-Path $installDir "clip.exe"
$url = "https://github.com/upendra7470/clip/releases/latest/download/clip_windows_amd64.exe"

New-Item -ItemType Directory -Force $installDir | Out-Null
Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $destination

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$parts = @($userPath -split ";" | Where-Object { $_ })
if ($parts -notcontains $installDir) {
    $newPath = (@($parts) + $installDir) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$env:Path;$installDir"
}

Write-Host "Clip installed at $destination"
Write-Host "Run: clip --version"
