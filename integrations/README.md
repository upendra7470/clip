# Clip file-manager integrations

These optional integrations add **Copy text with Clip** to the desktop workflow. Clip itself remains a standalone command and does not modify the operating system during installation.

## Windows Explorer

Run `windows/install-context-menu.ps1` in PowerShell. It creates per-user context-menu entries for files and folders, so administrator access is not required. Run `windows/uninstall-context-menu.ps1` to remove them.

## macOS Finder

Create a Finder Quick Action in Automator, choose **Run Shell Script**, pass input as arguments, and use `macos/clip-quick-action.sh` as the script. Each selected file is processed as smart Markdown; the final selected file remains on the clipboard.

## Linux file managers

The `linux/clip-smart.desktop` action can be installed in file managers that support desktop actions. Desktop environments differ, so the file is intentionally provided as a template rather than installed automatically.
