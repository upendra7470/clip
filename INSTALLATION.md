# Clip Installation Guide

## Quick Installation

### Download a prebuilt binary (all platforms)

Download the archive for your platform from the [Releases](https://github.com/upendra7470/clip/releases) page and extract it:

- **macOS Apple Silicon:** `clip_*_darwin_arm64.tar.gz`
- **macOS Intel:** `clip_*_darwin_amd64.tar.gz`
- **Linux:** `clip_*_linux_amd64.tar.gz` (or `linux_arm64`)
- **Windows:** `clip_*_windows_amd64.zip`

Then follow the steps in the [README](README.md#option-1--download-a-prebuilt-binary-all-platforms).

### Using Make (macOS / Linux)

```bash
make install
```

This will:
1. Build the Clip binary
2. Install it to `~/.local/bin/clip`
3. Make sure the directory exists
4. Clean up build artifacts

### Using go install (all platforms)

```bash
go install github.com/upendra7470/clip/cmd/clip@latest
```

This installs the binary to your `$GOPATH/bin` directory. Ensure this directory is in your PATH.

### Build from Source (all platforms)

1. Clone the repository:
   ```bash
   git clone https://github.com/upendra7470/clip.git
   cd clip
   ```

2. Build and install:
   ```bash
   # macOS / Linux
   go build -o clip ./cmd/clip
   mv clip ~/.local/bin/clip

   # Windows (PowerShell)
   go build -o clip.exe ./cmd/clip
   Move-Item clip.exe "$HOME\.local\bin\"
   ```

## Linux Clipboard Backend

Clip requires `xclip` (X11) or `wl-copy` (Wayland) on Linux:

```bash
sudo apt install xclip      # Debian / Ubuntu
sudo dnf install xclip      # Fedora
sudo pacman -S xclip        # Arch
```

No backend needed on macOS (`pbcopy`) or Windows (`clip.exe`) — both are built in.

## OCR Engine (optional)

Extracting text from images requires [Tesseract](https://github.com/tesseract-ocr/tesseract):

```bash
brew install tesseract          # macOS
sudo apt install tesseract-ocr  # Debian / Ubuntu
sudo dnf install tesseract      # Fedora
winget install UB-Mannheim.TesseractOCR  # Windows
```

All other file formats work without it; image files show a clear install hint when Tesseract is missing.

## PATH Setup

### Verify Installation

```bash
which clip
```

Should output: `/usr/local/bin/clip`, `~/.local/bin/clip`, or `~/go/bin/clip`

### Add to PATH (if needed)

If you installed to `~/.local/bin` or `~/go/bin` and it's not in your PATH:

```bash
# Add to your shell configuration (.bashrc, .zshrc, etc.)
export PATH=$PATH:$HOME/.local/bin:$HOME/go/bin

# Then reload
source ~/.zshrc  # or source ~/.bashrc
```

### Resolve PATH Conflicts

If `which clip` shows a Python package instead of the Go binary:

1. Check your PATH order:
   ```bash
   echo $PATH
   ```

2. Ensure `/usr/local/bin`, `~/.local/bin`, or `~/go/bin` comes before Python directories.

3. Use the full path temporarily:
   ```bash
   /usr/local/bin/clip --version
   ```

## Uninstallation

### Remove from ~/.local/bin

```bash
rm ~/.local/bin/clip
```

### Remove from /usr/local/bin

```bash
sudo rm /usr/local/bin/clip
```

### Remove from Go bin directory

```bash
rm ~/go/bin/clip
```

### Using Make

```bash
make uninstall
```

## Usage Examples

### Basic usage
```bash
clip document.pdf
```

### With exact path
```bash
clip ./files/report.pdf
```

### With absolute path
```bash
clip /full/path/to/document.pdf
```

### Help
```bash
clip --help
```

### Version
```bash
clip --version
```

## Smart File Resolution

Clip automatically searches for files in these locations:
1. Current directory
2. Absolute path if provided
3. ~/Downloads
4. ~/Desktop
5. ~/Documents

If multiple files with the same name are found, Clip will prompt you to select one.

## Troubleshooting

### "command not found" after installation

1. Verify the binary exists:
   ```bash
   ls /usr/local/bin/clip
   ```

2. Check your PATH:
   ```bash
   echo $PATH
   ```

3. Try running with full path:
   ```bash
   /usr/local/bin/clip --version
   ```

### Permission denied

```bash
sudo chmod +x /usr/local/bin/clip
```

### Python clip conflict persists

1. Rename the Go binary:
   ```bash
   sudo mv /usr/local/bin/clip /usr/local/bin/clip-doc
   ```

2. Use the renamed command:
   ```bash
   clip-doc document.pdf
   ```

## Makefile Commands

```bash
make build      # Build the binary
make install    # Install to ~/.local/bin
make uninstall  # Remove installation
make clean      # Clean build artifacts
make test       # Run tests
make fmt        # Format code
make vet        # Run go vet
make help       # Show help
```

## Verification

After installation, verify everything works:

```bash
clip --version      # Should show "clip v1.0.0"
clip --help         # Should show usage information
clip sample.txt     # Should extract and copy text
```
