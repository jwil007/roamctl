#!/bin/bash
set -e

REPO="jwil007/roamctl"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="roamctl"

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)         GOARCH="amd64" ;;
  aarch64|arm64)  GOARCH="arm64" ;;
  armv7*|armhf)   GOARCH="arm" ;;
  armv6*)         GOARCH="arm" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release tag from GitHub API
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

echo "Latest version: $LATEST"

# Check if already installed and up to date
if command -v $BINARY_NAME &>/dev/null; then
  INSTALLED=$($BINARY_NAME --version 2>/dev/null || echo "unknown")
  echo "Installed version: $INSTALLED"
  if [ "$INSTALLED" = "$LATEST" ]; then
    echo "Already up to date. Exiting."
    exit 0
  fi
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/roamctl-$GOARCH"

echo "Downloading roamctl-$GOARCH from $LATEST..."
curl -fsSL "$DOWNLOAD_URL" -o /tmp/roamctl

chmod +x /tmp/roamctl

echo "Installing to $INSTALL_DIR/$BINARY_NAME..."
sudo mv /tmp/roamctl "$INSTALL_DIR/$BINARY_NAME"

echo "Done! Run: roamctl"