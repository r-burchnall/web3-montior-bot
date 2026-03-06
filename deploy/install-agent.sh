#!/usr/bin/env bash
set -euo pipefail

# Idempotent installer for fcsc-agent.
# Can be run directly on a VM or via: lxc exec <vm> -- /path/to/install-agent.sh
#
# Expects the binary and service file to already be present at known paths
# (placed there by lxc file push or other mechanism).

INSTALL_BIN="/usr/local/bin/fcsc-agent"
STAGING_BIN="/tmp/fcsc-agent"
INSTALL_SERVICE="/etc/systemd/system/fcsc-agent.service"
STAGING_SERVICE="/tmp/fcsc-agent.service"

echo "=== Installing fcsc-agent ==="

# Install binary
if [ -f "$STAGING_BIN" ]; then
    if [ -f "$INSTALL_BIN" ] && cmp -s "$STAGING_BIN" "$INSTALL_BIN"; then
        echo "Binary unchanged, skipping."
    else
        # Stop the service first to release the binary (Text file busy)
        systemctl stop fcsc-agent 2>/dev/null || true
        cp "$STAGING_BIN" "$INSTALL_BIN"
        chmod +x "$INSTALL_BIN"
        echo "Binary installed: $INSTALL_BIN"
    fi
    rm -f "$STAGING_BIN"
elif [ ! -f "$INSTALL_BIN" ]; then
    echo "ERROR: No binary found at $STAGING_BIN or $INSTALL_BIN"
    exit 1
else
    echo "Using existing binary: $INSTALL_BIN"
fi

# Install systemd unit
if [ -f "$STAGING_SERVICE" ]; then
    NEEDS_RELOAD=false
    if [ -f "$INSTALL_SERVICE" ] && cmp -s "$STAGING_SERVICE" "$INSTALL_SERVICE"; then
        echo "Service file unchanged, skipping."
    else
        cp "$STAGING_SERVICE" "$INSTALL_SERVICE"
        echo "Service file installed: $INSTALL_SERVICE"
        NEEDS_RELOAD=true
    fi
    rm -f "$STAGING_SERVICE"

    if [ "$NEEDS_RELOAD" = true ]; then
        systemctl daemon-reload
    fi
elif [ ! -f "$INSTALL_SERVICE" ]; then
    echo "WARNING: No service file found. Run configure-agent.sh first."
    exit 1
else
    echo "Using existing service file: $INSTALL_SERVICE"
fi

# Enable and start/restart
systemctl enable fcsc-agent
if systemctl is-active --quiet fcsc-agent; then
    systemctl restart fcsc-agent
    echo "Service restarted."
else
    systemctl start fcsc-agent
    echo "Service started."
fi

# Verify
sleep 1
if systemctl is-active --quiet fcsc-agent; then
    echo "fcsc-agent is running."
    # Quick health check
    LISTEN_ADDR=$(systemctl show fcsc-agent -p ExecStart --value | grep -oP '\-\-listen-addr\s+\K\S+' || echo ":9100")
    PORT=$(echo "$LISTEN_ADDR" | grep -oP '\d+$' || echo "9100")
    if curl -sf "http://localhost:${PORT}/health" > /dev/null 2>&1; then
        echo "Health check passed on port ${PORT}."
    else
        echo "WARNING: Health endpoint not yet responding on port ${PORT} (may still be starting)."
    fi
else
    echo "ERROR: fcsc-agent failed to start. Check: journalctl -u fcsc-agent"
    exit 1
fi

echo "=== Done ==="
