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
asset="clip_${os}_${arch}"
release_url="https://github.com/$repo/releases/latest/download"
tmp=$(mktemp "$install_dir/.clip-install.XXXXXX")
checksums=$(mktemp "$install_dir/.clip-checksums.XXXXXX")
trap 'rm -f "$tmp" "$checksums"' EXIT HUP INT TERM

curl --fail --location --retry 3 --progress-bar "$release_url/$asset" --output "$tmp"
curl --fail --location --retry 3 --silent --show-error "$release_url/checksums.txt" --output "$checksums"
expected=$(awk -v asset="$asset" '$2 == asset { print $1 }' "$checksums")
case "$expected" in
  [0-9a-fA-F][0-9a-fA-F]*) ;;
  *) echo "Release checksum for $asset is missing or invalid." >&2; exit 1 ;;
esac
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmp" | awk '{ print $1 }')
else
  echo "A built-in SHA-256 tool (sha256sum or shasum) is required." >&2
  exit 1
fi
if [ "$(printf '%s' "$actual" | tr 'A-F' 'a-f')" != "$(printf '%s' "$expected" | tr 'A-F' 'a-f')" ]; then
  echo "Checksum verification failed for $asset; installation aborted." >&2
  exit 1
fi
chmod 755 "$tmp"
mv "$tmp" "$install_dir/clip"

echo "Clip installed at $install_dir/clip"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to PATH, then run: clip --version" ;;
esac
