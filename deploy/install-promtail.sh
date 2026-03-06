#!/usr/bin/env bash
set -euo pipefail

# Idempotent installer for Promtail.
# Can be run directly on a VM or via: lxc exec <vm> -- /path/to/install-promtail.sh

PROMTAIL_VERSION="3.2.1"
INSTALL_BIN="/usr/local/bin/promtail"
CONFIG_DIR="/etc/promtail"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
POSITIONS_DIR="/var/lib/promtail"
STAGING_CONFIG="/tmp/promtail-config.yaml"
INSTALL_SERVICE="/etc/systemd/system/promtail.service"
STAGING_SERVICE="/tmp/promtail.service"

echo "=== Installing Promtail ==="

# Install binary if not present or version mismatch
NEEDS_BINARY=false
if [ ! -f "$INSTALL_BIN" ]; then
    NEEDS_BINARY=true
elif ! "$INSTALL_BIN" --version 2>&1 | grep -q "$PROMTAIL_VERSION"; then
    NEEDS_BINARY=true
    echo "Upgrading promtail to v${PROMTAIL_VERSION}..."
fi

if [ "$NEEDS_BINARY" = true ]; then
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
    esac

    DOWNLOAD_URL="https://github.com/grafana/loki/releases/download/v${PROMTAIL_VERSION}/promtail-linux-${ARCH}.zip"
    echo "Downloading promtail v${PROMTAIL_VERSION} for ${ARCH}..."

    TMPDIR=$(mktemp -d)
    cd "$TMPDIR"

    if command -v curl &> /dev/null; then
        curl -sLO "$DOWNLOAD_URL"
    elif command -v wget &> /dev/null; then
        wget -q "$DOWNLOAD_URL"
    else
        echo "ERROR: neither curl nor wget available"
        exit 1
    fi

    # Install unzip if needed
    if ! command -v unzip &> /dev/null; then
        apt-get update -qq && apt-get install -y -qq unzip > /dev/null
    fi

    unzip -q "promtail-linux-${ARCH}.zip"
    mv "promtail-linux-${ARCH}" "$INSTALL_BIN"
    chmod +x "$INSTALL_BIN"
    cd /
    rm -rf "$TMPDIR"
    echo "Promtail installed: $INSTALL_BIN"
else
    echo "Promtail v${PROMTAIL_VERSION} already installed."
fi

# Create directories
mkdir -p "$CONFIG_DIR" "$POSITIONS_DIR"

# Install config
if [ -f "$STAGING_CONFIG" ]; then
    # Substitute hostname into the config
    ACTUAL_HOSTNAME="$(hostname)"
    sed "s/\${HOSTNAME}/$ACTUAL_HOSTNAME/g" "$STAGING_CONFIG" > "${STAGING_CONFIG}.resolved"
    if [ -f "$CONFIG_FILE" ] && cmp -s "${STAGING_CONFIG}.resolved" "$CONFIG_FILE"; then
        echo "Config unchanged, skipping."
    else
        cp "${STAGING_CONFIG}.resolved" "$CONFIG_FILE"
        echo "Config installed: $CONFIG_FILE (vm=$ACTUAL_HOSTNAME)"
    fi
    rm -f "$STAGING_CONFIG" "${STAGING_CONFIG}.resolved"
elif [ ! -f "$CONFIG_FILE" ]; then
    echo "WARNING: No config found. Run configure-promtail.sh first."
    exit 1
else
    echo "Using existing config: $CONFIG_FILE"
fi

# Install systemd unit
NEEDS_RELOAD=false
if [ -f "$STAGING_SERVICE" ]; then
    if [ -f "$INSTALL_SERVICE" ] && cmp -s "$STAGING_SERVICE" "$INSTALL_SERVICE"; then
        echo "Service file unchanged, skipping."
    else
        cp "$STAGING_SERVICE" "$INSTALL_SERVICE"
        echo "Service file installed: $INSTALL_SERVICE"
        NEEDS_RELOAD=true
    fi
    rm -f "$STAGING_SERVICE"
elif [ ! -f "$INSTALL_SERVICE" ]; then
    # Generate a default service file
    cat > "$INSTALL_SERVICE" <<EOF
[Unit]
Description=Promtail Log Shipper
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/promtail -config.file=/etc/promtail/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=promtail

[Install]
WantedBy=multi-user.target
EOF
    NEEDS_RELOAD=true
    echo "Default service file created: $INSTALL_SERVICE"
fi

if [ "$NEEDS_RELOAD" = true ]; then
    systemctl daemon-reload
fi

# Enable and start/restart
systemctl enable promtail
if systemctl is-active --quiet promtail; then
    systemctl restart promtail
    echo "Promtail restarted."
else
    systemctl start promtail
    echo "Promtail started."
fi

sleep 1
if systemctl is-active --quiet promtail; then
    echo "Promtail is running."
else
    echo "ERROR: Promtail failed to start. Check: journalctl -u promtail"
    exit 1
fi

echo "=== Done ==="
