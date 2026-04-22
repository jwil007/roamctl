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

# Download and install binary
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/roamctl_${GOARCH}.tar.gz"
echo "Downloading roamctl-$GOARCH from $LATEST..."
curl -fsSL "$DOWNLOAD_URL" -o /tmp/roamctl.tar.gz
tar -xzf /tmp/roamctl.tar.gz -C /tmp roamctl
chmod +x /tmp/roamctl
echo "Installing to $INSTALL_DIR/$BINARY_NAME..."
sudo mv /tmp/roamctl "$INSTALL_DIR/$BINARY_NAME"
echo "roamctl installed successfully!"

# Download and install roamctl-tui
TUI_BINARY_NAME="roamctl-tui"
TUI_DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST/roamctl-tui_${GOARCH}.tar.gz"
echo "Downloading roamctl-tui-$GOARCH from $LATEST..."
curl -fsSL "$TUI_DOWNLOAD_URL" -o /tmp/roamctl-tui.tar.gz
tar -xzf /tmp/roamctl-tui.tar.gz -C /tmp roamctl-tui
chmod +x /tmp/roamctl-tui
echo "Installing to $INSTALL_DIR/$TUI_BINARY_NAME..."
sudo mv /tmp/roamctl-tui "$INSTALL_DIR/$TUI_BINARY_NAME"
echo "roamctl-tui installed successfully!"

# Detect and configure wireless interface
echo ""
echo "Detecting wireless interfaces..."
# shellcheck disable=SC2011
IFACES=$(ls /sys/class/net/ | xargs -I{} sh -c 'test -d /sys/class/net/{}/wireless && echo {}' 2>/dev/null || true)
SELECTED=""
if [ -z "$IFACES" ]; then
  echo "No wireless interfaces found. You can configure one later with: sudo roamctl -iface <name>"
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
    echo "Initializing config for $SELECTED..."
    sudo roamctl -iface "$SELECTED"
  else
    echo "Invalid selection. You can configure one later with: sudo roamctl -iface <name>"
  fi
fi

# Systemd service install
UNIT_URL="https://raw.githubusercontent.com/$REPO/master/systemd/roamctl@.service"
SERVICE_INSTALLED=false
printf "\nInstall roamctl as a systemd service? [y/N]: "
read -r INSTALL_SERVICE
if [ "$INSTALL_SERVICE" = "y" ] || [ "$INSTALL_SERVICE" = "Y" ]; then
  echo "Installing systemd service..."
  curl -fsSL "$UNIT_URL" -o /tmp/roamctl@.service
  sudo mv /tmp/roamctl@.service /etc/systemd/system/roamctl@.service
  sudo systemctl daemon-reload
  SERVICE_INSTALLED=true
  if [ -n "$SELECTED" ]; then
    printf "Enable roamctl@%s at boot? [y/N]: " "$SELECTED"
    read -r ENABLE_SERVICE
    if [ "$ENABLE_SERVICE" = "y" ] || [ "$ENABLE_SERVICE" = "Y" ]; then
      sudo systemctl enable "roamctl@$SELECTED"
      echo "roamctl@$SELECTED enabled at boot."
    fi
  else
    echo "No interface selected — enable manually with: sudo systemctl enable roamctl@<iface>"
  fi
fi

# Summary
echo ""
echo "================================================"
echo " roamctl $LATEST installed"
echo "================================================"
echo ""
if [ -n "$SELECTED" ]; then
  echo " Config file:      /etc/roamctl/$SELECTED.toml"
  echo " Edit config:      sudo roamctl -iface $SELECTED -edit"
else
  echo " Config file:      /etc/roamctl/<iface>.toml"
  echo " Edit config:      sudo roamctl -iface <iface> -edit"
fi
echo " Launch TUI:       sudo roamctl-tui"
echo ""
if [ "$SERVICE_INSTALLED" = true ] && [ -n "$SELECTED" ]; then
  echo " Run as daemon:    sudo systemctl start roamctl@$SELECTED"
  echo " View logs:        journalctl -u roamctl@$SELECTED -f"
elif [ "$SERVICE_INSTALLED" = true ]; then
  echo " Run as daemon:    sudo systemctl start roamctl@<iface>"
  echo " View logs:        journalctl -u roamctl@<iface> -f"
else
  echo " Run in foreground: sudo roamctl -iface $SELECTED"
fi
echo ""