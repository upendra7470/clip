$ErrorActionPreference = "Stop"

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
if ($arch -ne "x64") {
    throw "Clip currently supports Windows x64. Detected: $arch"
}

$installDir = if ($env:CLIP_INSTALL_DIR) { $env:CLIP_INSTALL_DIR } else { Join-Path $HOME ".local\bin" }
$destination = Join-Path $installDir "clip.exe"
$asset = "clip_windows_amd64.exe"
$releaseUrl = "https://github.com/upendra7470/clip/releases/latest/download"

New-Item -ItemType Directory -Force $installDir | Out-Null
$nonce = [Guid]::NewGuid().ToString("N")
$tempBinary = Join-Path $installDir ".clip-install-$nonce.exe"
$tempChecksums = Join-Path $installDir ".clip-checksums-$nonce.txt"
try {
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseUrl/$asset" -OutFile $tempBinary
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseUrl/checksums.txt" -OutFile $tempChecksums

    $expected = $null
    foreach ($line in Get-Content $tempChecksums) {
        $parts = $line.Trim() -split '\s+', 2
        if ($parts.Count -eq 2 -and $parts[1].TrimStart('*') -eq $asset) {
            $expected = $parts[0]
            break
        }
    }
    if (-not $expected -or $expected -notmatch '^[0-9a-fA-F]{64}$') {
        throw "Release checksum for $asset is missing or invalid."
    }
    $actual = (Get-FileHash -Algorithm SHA256 -Path $tempBinary).Hash
    if ($actual -ne $expected) {
        throw "Checksum verification failed for $asset; installation aborted."
    }
    Move-Item -Force $tempBinary $destination
}
finally {
    Remove-Item -Force -ErrorAction SilentlyContinue $tempBinary, $tempChecksums
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$parts = @($userPath -split ";" | Where-Object { $_ })
if ($parts -notcontains $installDir) {
    $newPath = (@($parts) + $installDir) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$env:Path;$installDir"
}

Write-Host "Clip installed at $destination"
Write-Host "Run: clip --version"
