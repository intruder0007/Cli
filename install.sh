#!/bin/sh
# Installs the bootstrap CLI (github.com/intruder0007/Cli) for the
# current platform: resolves OS/arch, downloads the matching release
# archive + SHA256SUMS.txt, verifies the checksum, and installs just the
# bootstrap binary onto PATH. See docs/architecture/adr/0012-universal-install-architecture.md —
# the binary is self-sufficient (embeds its own plugin set as a
# fallback), so this script never needs to preserve sibling directories
# the way a package-manager formula extracting the whole archive does.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/intruder0007/Cli/main/install.sh | sh
#   VERSION=v0.2.0 INSTALL_DIR=/usr/local/bin sh install.sh   # pin a version / install dir

set -eu

REPO="intruder0007/Cli"
VERSION="${VERSION:-latest}"

say() { printf '%s\n' "$*"; }
die() { printf 'install.sh: error: %s\n' "$*" >&2; exit 1; }

need_cmd() { command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"; }
need_cmd curl
need_cmd tar
need_cmd mktemp

detect_os() {
  case "$(uname -s)" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    *) die "unsupported OS: $(uname -s) (only linux/darwin are released; on Windows use install.ps1)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) echo amd64 ;;
    arm64 | aarch64) echo arm64 ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

if [ "$VERSION" = "latest" ]; then
  need_cmd grep
  need_cmd sed
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  [ -n "$VERSION" ] || die "could not resolve the latest release version"
fi

ARCHIVE="cli_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

say "Downloading ${ARCHIVE} (${VERSION})..."
curl -fsSL -o "${TMP_DIR}/${ARCHIVE}" "${BASE_URL}/${ARCHIVE}"
curl -fsSL -o "${TMP_DIR}/SHA256SUMS.txt" "${BASE_URL}/SHA256SUMS.txt"

say "Verifying checksum..."
(
  cd "$TMP_DIR"
  if command -v sha256sum >/dev/null 2>&1; then
    grep " ${ARCHIVE}\$" SHA256SUMS.txt | sha256sum -c -
  elif command -v shasum >/dev/null 2>&1; then
    grep " ${ARCHIVE}\$" SHA256SUMS.txt | shasum -a 256 -c -
  else
    die "neither sha256sum nor shasum found; cannot verify checksum"
  fi
)

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"
EXTRACTED_DIR="${TMP_DIR}/cli_${VERSION}_${OS}_${ARCH}"

if [ -n "${INSTALL_DIR:-}" ]; then
  DEST="$INSTALL_DIR"
else
  DEST="${HOME}/.local/bin"
fi
mkdir -p "$DEST"
cp "${EXTRACTED_DIR}/bootstrap" "${DEST}/bootstrap"
chmod +x "${DEST}/bootstrap"

say "Installed bootstrap ${VERSION} to ${DEST}/bootstrap"

case ":$PATH:" in
  *":${DEST}:"*) say "Run 'bootstrap new' to get started." ;;
  *) say "${DEST} is not on your PATH. Add it, e.g.:"; say "  export PATH=\"${DEST}:\$PATH\"" ;;
esac
