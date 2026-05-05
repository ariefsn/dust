#!/usr/bin/env bash
# install.sh — download and install dust.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ariefsn/dust/main/install.sh | bash
#
# Environment overrides:
#   DUST_VERSION   pin a specific tag (default: latest release)
#   BIN_DIR        install location  (default: /usr/local/bin, or ~/.local/bin if not writable)
#   DUST_REPO      override the repo (default: ariefsn/dust)

set -euo pipefail

REPO="${DUST_REPO:-ariefsn/dust}"
VERSION="${DUST_VERSION:-latest}"

err() { printf '\033[31merror:\033[0m %s\n' "$*" >&2; exit 1; }
log() { printf '\033[36m==>\033[0m %s\n' "$*"; }

# --- detect OS + arch ---------------------------------------------------------

uname_s="$(uname -s)"
uname_m="$(uname -m)"

case "$uname_s" in
  Darwin) os="darwin" ;;
  Linux)  os="linux" ;;
  *) err "unsupported OS: $uname_s (only macOS and Linux are supported)" ;;
esac

case "$uname_m" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) err "unsupported architecture: $uname_m" ;;
esac

# --- pick install dir ---------------------------------------------------------

if [ -z "${BIN_DIR:-}" ]; then
  if [ -w "/usr/local/bin" ]; then
    BIN_DIR="/usr/local/bin"
  else
    BIN_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$BIN_DIR"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) PATH_WARNING="Note: $BIN_DIR is not on your \$PATH. Add it to your shell profile." ;;
esac

# --- resolve version ----------------------------------------------------------

if [ "$VERSION" = "latest" ]; then
  log "resolving latest release..."
  # Use the redirect from /releases/latest to find the tag without the GH API
  # (works without auth).
  VERSION="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" \
              | sed 's|.*/tag/||')"
  if [ -z "$VERSION" ]; then
    err "could not resolve latest version (check network or pin DUST_VERSION=v0.1.0)"
  fi
fi

# Strip leading 'v' for archive filenames (goreleaser .Version is no-v).
plain_version="${VERSION#v}"

archive="dust_${plain_version}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${archive}"
sums_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

log "downloading $archive"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "$url"      -o "$tmp/$archive" || err "download failed: $url"
curl -fsSL "$sums_url" -o "$tmp/checksums.txt" || err "checksums download failed"

# --- verify checksum ----------------------------------------------------------

log "verifying checksum"
expected="$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')"
if [ -z "$expected" ]; then
  err "no checksum entry for $archive in checksums.txt"
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
else
  err "no sha256sum or shasum on \$PATH"
fi

if [ "$actual" != "$expected" ]; then
  err "checksum mismatch: expected $expected, got $actual"
fi

# --- extract + install --------------------------------------------------------

log "extracting"
tar -xzf "$tmp/$archive" -C "$tmp"
[ -f "$tmp/dust" ] || err "expected 'dust' binary inside archive, not found"

log "installing to $BIN_DIR/dust"
install -m 0755 "$tmp/dust" "$BIN_DIR/dust"

log "$($BIN_DIR/dust version --short 2>/dev/null || echo "$VERSION") installed at $BIN_DIR/dust"

if [ -n "${PATH_WARNING:-}" ]; then
  printf '\033[33m%s\033[0m\n' "$PATH_WARNING"
fi
