# Clip Installation Guide

## Quick Installation

### Using Make (Recommended)

```bash
make install
```

This will:
1. Build the Clip binary
2. Install it to `/usr/local/bin/clip`
3. Make it executable
4. Clean up build artifacts

### Using go install

```bash
go install github.com/upendra7470/clip/cmd/clip@latest
```

This installs the binary to your `$GOPATH/bin` directory. Ensure this directory is in your PATH.

### Manual Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/upendra7470/clip.git
   cd clip
   ```

2. Build and install:
   ```bash
   go build -o clip ./cmd/clip
   sudo mv clip /usr/local/bin/clip
   ```

## PATH Setup

### Verify Installation

```bash
which clip
```

Should output: `/usr/local/bin/clip` or `~/go/bin/clip`

### Add to PATH (if needed)

If you installed to `~/go/bin` but it's not in your PATH:

```bash
# Add to your shell configuration (.bashrc, .zshrc, etc.)
export PATH=$PATH:~/go/bin

# Then reload
source ~/.zshrc  # or source ~/.bashrc
```

### Resolve PATH Conflicts

If `which clip` shows a Python package instead of the Go binary:

1. Check your PATH order:
   ```bash
   echo $PATH
   ```

2. Ensure `/usr/local/bin` or `~/go/bin` comes before Python directories.

3. Use the full path temporarily:
   ```bash
   /usr/local/bin/clip --version
   ```

## Uninstallation

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
make install    # Install globally
make uninstall  # Remove installation
make clean      # Clean build artifacts
make test       # Run tests
make fmt        # Format code
make help       # Show help
```

## Verification

After installation, verify everything works:

```bash
clip --version      # Should show "Clip v1.0.0"
clip --help         # Should show usage information
clip sample.txt     # Should extract and copy text
```
