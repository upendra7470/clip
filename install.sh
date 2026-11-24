#!/bin/sh
set -eu

repo="upendra7470/clip"
install_dir="${CLIP_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "Clip supports Windows, macOS, and Linux." >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "Unsupported processor: $(uname -m)" >&2; exit 1 ;;
esac

mkdir -p "$install_dir"
url="https://github.com/$repo/releases/latest/download/clip_${os}_${arch}"
tmp="${TMPDIR:-/tmp}/clip-install-$$"
trap 'rm -f "$tmp"' EXIT HUP INT TERM

curl --fail --location --retry 3 --progress-bar "$url" --output "$tmp"
chmod 755 "$tmp"
mv "$tmp" "$install_dir/clip"

echo "Clip installed at $install_dir/clip"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to PATH, then run: clip --version" ;;
esac
