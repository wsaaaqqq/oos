#!/bin/bash
set -e

REPO="wsaaaqqq/oos"
BIN="oos"
INSTALL_DIR="${HOME}/.local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest release URL
LATEST_URL=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" \
  | grep "browser_download_url.*oos_${OS}_${ARCH}" \
  | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
  echo "Error: could not find release for ${OS}/${ARCH}"
  exit 1
fi

mkdir -p "$INSTALL_DIR"

echo "Downloading oos ${OS}/${ARCH}..."
curl -fsSL "$LATEST_URL" -o "${INSTALL_DIR}/${BIN}"
chmod +x "${INSTALL_DIR}/${BIN}"

# Check if install dir is in PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  echo "Add to your shell profile to use 'oos' globally:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""
echo "oos installed to ${INSTALL_DIR}/${BIN}"
echo "Try: oos"
