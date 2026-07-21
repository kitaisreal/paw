#!/bin/sh
# paw installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/kitaisreal/paw/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/kitaisreal/paw/main/install.sh | sh -s -- v0.1.0
#
# Environment overrides:
#   PAW_REPO    GitHub "owner/repo"      (default: kitaisreal/paw)
#   PAW_VERSION release tag to install   (default: latest, or first arg)
#   BINDIR      install directory        (default: /usr/local/bin, else ~/.local/bin)
set -eu

PAW_REPO="${PAW_REPO:-kitaisreal/paw}"
PAW_VERSION="${PAW_VERSION:-${1:-latest}}"
BINARY="paw"

log()  { printf '%s\n' "$*" >&2; }
fail() { log "error: $*"; exit 1; }

# --- pick a downloader -------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  http_get()  { curl -fsSL "$1"; }
  download()  { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  http_get()  { wget -qO- "$1"; }
  download()  { wget -qO "$2" "$1"; }
else
  fail "need curl or wget installed"
fi

# --- detect os/arch (must match .goreleaser.yaml targets) --------------------
os="$(uname -s)"
case "$os" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) fail "unsupported OS: $os (linux and darwin only)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64)  arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *) fail "unsupported architecture: $arch (amd64 and arm64 only)" ;;
esac

# --- resolve the release tag -------------------------------------------------
if [ "$PAW_VERSION" = "latest" ]; then
  log "Resolving latest release of $PAW_REPO..."
  tag="$(http_get "https://api.github.com/repos/${PAW_REPO}/releases/latest" \
    | grep '"tag_name":' | head -n1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  [ -n "$tag" ] || fail "could not determine latest release tag"
else
  tag="$PAW_VERSION"
fi

# GoReleaser archives use the version without a leading "v".
version="${tag#v}"
archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/${PAW_REPO}/releases/download/${tag}"

# --- download + verify + extract ---------------------------------------------
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

log "Downloading ${archive} (${tag})..."
download "${base}/${archive}"      "${tmp}/${archive}" || fail "download failed: ${base}/${archive}"

if download "${base}/checksums.txt" "${tmp}/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then sha="sha256sum";
  elif command -v shasum   >/dev/null 2>&1; then sha="shasum -a 256";
  else sha=""; fi
  if [ -n "$sha" ]; then
    want="$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')"
    got="$( (cd "$tmp" && $sha "$archive") | awk '{print $1}')"
    [ -n "$want" ] && [ "$want" = "$got" ] || fail "checksum verification failed for ${archive}"
    log "Checksum OK."
  fi
else
  log "warning: checksums.txt not found, skipping verification"
fi

tar -xzf "${tmp}/${archive}" -C "$tmp" "$BINARY" || fail "failed to extract ${BINARY}"

# --- install -----------------------------------------------------------------
if [ -z "${BINDIR:-}" ]; then
  if [ -w /usr/local/bin ] || [ "$(id -u)" = "0" ]; then
    BINDIR="/usr/local/bin"
  else
    BINDIR="${HOME}/.local/bin"
  fi
fi
mkdir -p "$BINDIR"

if [ -w "$BINDIR" ]; then
  install -m 0755 "${tmp}/${BINARY}" "${BINDIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1; then
  log "Installing to ${BINDIR} (requires sudo)..."
  sudo install -m 0755 "${tmp}/${BINARY}" "${BINDIR}/${BINARY}"
else
  fail "cannot write to ${BINDIR}; set BINDIR to a writable path"
fi

log "Installed ${BINARY} ${tag} to ${BINDIR}/${BINARY}"
case ":${PATH}:" in
  *":${BINDIR}:"*) ;;
  *) log "note: ${BINDIR} is not on your PATH; add it with: export PATH=\"${BINDIR}:\$PATH\"" ;;
esac
