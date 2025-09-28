#!/usr/bin/env bash
set -euo pipefail

REPO="cubetiqlabs/tdb-cli"
NAME="tdb"

API_URL="https://api.github.com/repos/$REPO/releases/latest"
RELEASE_JSON=$(curl -fsSL "$API_URL")
TAG=$(echo "$RELEASE_JSON" | grep -m1 '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
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
    ;;
  msys*|cygwin*|mingw*|windows*)
    OS="windows"
    ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

ASSET=$(echo "$RELEASE_JSON" | \
  grep -oE "\"name\": *\"${NAME}_${OS}_${ARCH}\\.(tar\\.gz|zip)\"" | head -n1 | cut -d'"' -f4)
if [[ -z "$ASSET" ]]; then
  echo "No release asset found for ${OS}/${ARCH}" >&2
  exit 1
fi

EXT="${ASSET##*.}"
URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"
SUMS_URL="https://github.com/$REPO/releases/download/$TAG/SHA256SUMS"
TMP=$(mktemp -d)
ORIG_DIR=$(pwd)
cd "$TMP"
echo "Downloading $URL..."
curl -sSL -o "$ASSET" "$URL"

echo "Fetching checksums..."
curl -sSL -o SHA256SUMS "$SUMS_URL"
if [[ ! -s SHA256SUMS ]]; then
  echo "Failed to download SHA256SUMS from release" >&2
  exit 1
fi

if command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$ASSET" | awk '{print $1}')
else
  ACTUAL=$(sha256sum "$ASSET" | awk '{print $1}')
fi
EXPECTED=$(awk -v file="$ASSET" '$2 == file {print $1}' SHA256SUMS)
if [[ -z "$EXPECTED" ]]; then
  echo "Checksum entry for $ASSET not found" >&2
  exit 1
fi

if [[ "$ACTUAL" != "$EXPECTED" ]]; then
  echo "Checksum mismatch for $ASSET" >&2
  echo "Expected: $EXPECTED" >&2
  echo "Actual:   $ACTUAL" >&2
  exit 1
fi

if [[ "$EXT" == "tar.gz" ]]; then
  tar -xzf "$ASSET"
else
  unzip -q "$ASSET"
fi

BIN=$(find . -maxdepth 2 -type f \( -name "tdb" -o -name "tdb.exe" \) | head -n 1)
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
