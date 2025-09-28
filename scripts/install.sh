#!/usr/bin/env bash
set -euo pipefail

REPO="cubetiqlabs/tdb-cli"
NAME="tdb"

API_URL="https://api.github.com/repos/$REPO/releases/latest"
TAG=$(curl -fsSL "$API_URL" | grep -m1 '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [[ -z "$TAG" ]]; then
  echo "Failed to determine latest release tag from $API_URL" >&2
  exit 1
fi
VERSION="${TAG#v}"

OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then ARCH="amd64"; fi
if [[ "$ARCH" == "aarch64" ]]; then ARCH="arm64"; fi
case "$OS" in
  linux|darwin)
    EXT="tar.gz"
    ;;
  msys*|cygwin*|mingw*|windows*)
    OS="windows"
    EXT="zip"
    ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac
ASSET="${NAME}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"
TMP=$(mktemp -d)
ORIG_DIR=$(pwd)
cd "$TMP"
echo "Downloading $URL..."
curl -sSL -o "$ASSET" "$URL"
if [[ "$EXT" == "tar.gz" ]]; then
  tar -xzf "$ASSET"
else
  unzip -q "$ASSET"
fi

BIN=$(find . -maxdepth 1 -type f -name "tdb*" | head -n 1)
if [[ -z "$BIN" ]]; then
  echo "Failed to locate extracted binary" >&2
  exit 1
fi
chmod +x "$BIN"
DEST="${TDB_INSTALL_DIR:-/usr/local/bin}/$NAME"
sudo mkdir -p "$(dirname "$DEST")"
sudo mv "$BIN" "$DEST"
echo "Installed $NAME $VERSION to $DEST"

cd "$ORIG_DIR"
rm -rf "$TMP"
