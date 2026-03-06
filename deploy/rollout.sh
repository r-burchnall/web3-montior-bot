#!/usr/bin/env bash
set -euo pipefail

# Rollout fcsc-agent and promtail to all VMs listed in vms.conf.
# Uses lxc exec to push files and run installers.
#
# Usage:
#   ./deploy/rollout.sh              # Deploy to all VMs
#   ./deploy/rollout.sh vm1 vm2      # Deploy to specific VMs
#   ./deploy/rollout.sh --agent-only # Only deploy the agent
#   ./deploy/rollout.sh --promtail-only # Only deploy promtail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_ROOT/build"
VMS_FILE="$SCRIPT_DIR/vms.conf"

# Use sudo for lxc if not running as root and lxc requires it
LXC="lxc"
if [ "$(id -u)" -ne 0 ] && ! lxc list --format csv -c n 2>/dev/null | head -1 > /dev/null 2>&1; then
    LXC="sudo lxc"
fi

AGENT_ONLY=false
PROMTAIL_ONLY=false
SPECIFIC_VMS=()

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --agent-only) AGENT_ONLY=true ;;
        --promtail-only) PROMTAIL_ONLY=true ;;
        *) SPECIFIC_VMS+=("$arg") ;;
    esac
done

# Load VM list
if [ ${#SPECIFIC_VMS[@]} -gt 0 ]; then
    VMS=("${SPECIFIC_VMS[@]}")
elif [ -f "$VMS_FILE" ]; then
    mapfile -t VMS < <(grep -v '^\s*#' "$VMS_FILE" | grep -v '^\s*$')
else
    echo "ERROR: No VMs specified and $VMS_FILE not found."
    echo "Create vms.conf with one VM hostname per line, or pass VM names as arguments."
    exit 1
fi

# Verify build artifacts exist
if [ "$PROMTAIL_ONLY" = false ] && [ ! -f "$BUILD_DIR/fcsc-agent" ]; then
    echo "ERROR: Agent binary not found at $BUILD_DIR/fcsc-agent"
    echo "Run: ./scripts/build.sh"
    exit 1
fi

# Push a file into a container via lxc exec (lxc file push is forbidden by LXD policy).
# Removes any existing file first to avoid "Permission denied" from tee on
# files owned by another user.
push_file() {
    local src="$1"
    local vm="$2"
    local dest="$3"
    $LXC exec "$vm" -- rm -f "$dest"
    cat "$src" | $LXC exec "$vm" -- tee "$dest" > /dev/null
}

echo "Deploying to ${#VMS[@]} VM(s): ${VMS[*]}"
echo ""

FAILED=()

for vm in "${VMS[@]}"; do
    echo "--- $vm ---"

    # Check VM is accessible
    if ! $LXC info "$vm" &> /dev/null; then
        echo "  SKIP: VM '$vm' not found or not accessible."
        FAILED+=("$vm")
        continue
    fi

    # Deploy agent
    if [ "$PROMTAIL_ONLY" = false ]; then
        echo "  Stopping agent service..."
        $LXC exec "$vm" -- systemctl stop fcsc-agent 2>/dev/null || true

        echo "  Pushing agent binary..."
        push_file "$BUILD_DIR/fcsc-agent" "$vm" "/tmp/fcsc-agent"
        $LXC exec "$vm" -- chmod +x /tmp/fcsc-agent

        echo "  Pushing service file..."
        push_file "$SCRIPT_DIR/fcsc-agent.service" "$vm" "/tmp/fcsc-agent.service"

        echo "  Pushing installer..."
        push_file "$SCRIPT_DIR/install-agent.sh" "$vm" "/tmp/install-agent.sh"

        echo "  Running installer..."
        $LXC exec "$vm" -- chmod +x /tmp/install-agent.sh
        if $LXC exec "$vm" -- /tmp/install-agent.sh; then
            echo "  Agent: OK"
        else
            echo "  Agent: FAILED"
            FAILED+=("$vm:agent")
        fi
        $LXC exec "$vm" -- rm -f /tmp/install-agent.sh
    fi

    # Deploy promtail
    if [ "$AGENT_ONLY" = false ]; then
        echo "  Pushing promtail config..."
        push_file "$SCRIPT_DIR/promtail-config.yaml" "$vm" "/tmp/promtail-config.yaml"
        push_file "$SCRIPT_DIR/promtail.service" "$vm" "/tmp/promtail.service"

        echo "  Pushing promtail installer..."
        push_file "$SCRIPT_DIR/install-promtail.sh" "$vm" "/tmp/install-promtail.sh"

        echo "  Running promtail installer..."
        $LXC exec "$vm" -- chmod +x /tmp/install-promtail.sh
        if $LXC exec "$vm" -- /tmp/install-promtail.sh; then
            echo "  Promtail: OK"
        else
            echo "  Promtail: FAILED"
            FAILED+=("$vm:promtail")
        fi
        $LXC exec "$vm" -- rm -f /tmp/install-promtail.sh
    fi

    echo ""
done

# Summary
echo "========================================="
if [ ${#FAILED[@]} -eq 0 ]; then
    echo "All ${#VMS[@]} VM(s) deployed successfully."
else
    echo "FAILURES: ${FAILED[*]}"
    exit 1
fi
