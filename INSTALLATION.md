# Installing Clip

Official Clip releases are self-contained. You do not need to install an OCR engine, PDF renderer, language runtime, package manager, or clipboard utility.

## One-command install

### macOS and Linux

```sh
curl -fsSL https://raw.githubusercontent.com/upendra7470/clip/main/install.sh | sh
```

The installer detects the operating system and CPU, downloads the matching binary, and installs it as `~/.local/bin/clip`. Override the destination when needed:

```sh
curl -fsSL https://raw.githubusercontent.com/upendra7470/clip/main/install.sh | CLIP_INSTALL_DIR=/usr/local/bin sh
```

If `~/.local/bin` is not already on `PATH`, add this line to `~/.zshrc` or `~/.bashrc`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/upendra7470/clip/main/install.ps1 | iex
```

The installer places `clip.exe` in `%USERPROFILE%\.local\bin` and adds that directory to the user `PATH`. Open a new terminal after installation if the current terminal does not find `clip`.

## Verify

```sh
clip --version
clip --ocr-status
```

The OCR status should report that image and scanned-PDF OCR are ready.

## Direct download

Download one file from [GitHub Releases](https://github.com/upendra7470/clip/releases/latest):

| System | File |
|---|---|
| Windows x64 | `clip_windows_amd64.exe` |
| macOS Apple Silicon | `clip_darwin_arm64` |
| macOS Intel | `clip_darwin_amd64` |
| Linux x64 | `clip_linux_amd64` |
| Linux ARM64 | `clip_linux_arm64` |

Rename it to `clip` (`clip.exe` on Windows), make it executable on macOS/Linux with `chmod +x clip`, and place it in a directory on `PATH`. Compare its SHA-256 hash with the release's `checksums.txt` before running it in security-sensitive environments.

## Build from source

Source builds require Go 1.26 or newer:

```sh
git clone https://github.com/upendra7470/clip.git
cd clip
go test ./...
go build -o clip ./cmd/clip
```

The GitHub release workflow additionally compiles the native OCR engine for each target and embeds the pinned models. For that reason, use an official release when you need the zero-dependency OCR package.

## Uninstall

macOS or Linux:

```sh
rm "$HOME/.local/bin/clip"
```

Windows PowerShell:

```powershell
Remove-Item "$HOME\.local\bin\clip.exe"
```
