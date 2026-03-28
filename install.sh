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

echo "roamctl installed successfully!"

# Detect wireless interfaces
echo "Detecting wireless interfaces..."
# shellcheck disable=SC2011
IFACES=$(ls /sys/class/net/ | xargs -I{} sh -c 'test -d /sys/class/net/{}/wireless && echo {}' 2>/dev/null || true)
if [ -z "$IFACES" ]; then
  echo "No wireless interfaces found. You can set the interface later with: sudo roamctl -interface <name>"
else
  echo "Available wireless interfaces:"
  i=1
  for iface in $IFACES; do
    echo "  $i) $iface"
    i=$((i+1))
  done
  printf "Select interface [1]: "
  read -r SELECTION
  SELECTION=${SELECTION:-1}
  SELECTED=$(echo "$IFACES" | sed -n "${SELECTION}p")
  if [ -n "$SELECTED" ]; then
    echo "Setting interface to $SELECTED..."
    sudo roamctl -iface "$SELECTED"
  else
    echo "Invalid selection. You can set the interface later with: sudo roamctl -interface <name>"
  fi
fi

# Systemd service install
UNIT_URL="https://raw.githubusercontent.com/$REPO/master/systemd/roamctl.service"
printf "Install roamctl as a systemd service? [y/N]: "
read -r INSTALL_SERVICE
if [ "$INSTALL_SERVICE" = "y" ] || [ "$INSTALL_SERVICE" = "Y" ]; then
  echo "Installing systemd service..."
  curl -fsSL "$UNIT_URL" -o /tmp/roamctl.service
  sudo mv /tmp/roamctl.service /etc/systemd/system/roamctl.service
  sudo systemctl daemon-reload
  echo "Service installed."
  printf "Enable roamctl at boot? [y/N]: "
  read -r ENABLE_SERVICE
  if [ "$ENABLE_SERVICE" = "y" ] || [ "$ENABLE_SERVICE" = "Y" ]; then
    sudo systemctl enable roamctl
    echo "roamctl enabled at boot."
  fi
  echo "To start roamctl now, run: sudo systemctl start roamctl"
fi