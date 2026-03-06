#!/usr/bin/env bash
set -euo pipefail

# Rollout fcsc-agent and promtail to all VMs listed in vms.conf.
# Uses lxc file push to stage files and lxc exec to run installers.
#
# Usage:
#   ./rollout.sh              # Deploy to all VMs
#   ./rollout.sh vm1 vm2      # Deploy to specific VMs
#   ./rollout.sh --agent-only # Only deploy the agent
#   ./rollout.sh --promtail-only # Only deploy promtail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_ROOT/build"
VMS_FILE="$SCRIPT_DIR/vms.conf"

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

echo "Deploying to ${#VMS[@]} VM(s): ${VMS[*]}"
echo ""

FAILED=()

for vm in "${VMS[@]}"; do
    echo "--- $vm ---"

    # Check VM is accessible
    if ! lxc info "$vm" &> /dev/null; then
        echo "  SKIP: VM '$vm' not found or not accessible."
        FAILED+=("$vm")
        continue
    fi

    # Deploy agent
    if [ "$PROMTAIL_ONLY" = false ]; then
        echo "  Pushing agent binary..."
        lxc file push "$BUILD_DIR/fcsc-agent" "$vm/tmp/fcsc-agent"

        # Push service file if it exists for this VM
        SERVICE_FILE="$BUILD_DIR/fcsc-agent.service"
        if [ -f "$SERVICE_FILE" ]; then
            lxc file push "$SERVICE_FILE" "$vm/tmp/fcsc-agent.service"
        fi

        echo "  Pushing installer..."
        lxc file push "$SCRIPT_DIR/install-agent.sh" "$vm/tmp/install-agent.sh"

        echo "  Running installer..."
        lxc exec "$vm" -- chmod +x /tmp/install-agent.sh
        if lxc exec "$vm" -- /tmp/install-agent.sh; then
            echo "  Agent: OK"
        else
            echo "  Agent: FAILED"
            FAILED+=("$vm:agent")
        fi
        lxc exec "$vm" -- rm -f /tmp/install-agent.sh
    fi

    # Deploy promtail
    if [ "$AGENT_ONLY" = false ]; then
        PROMTAIL_CONFIG="$BUILD_DIR/promtail-config.yaml"
        PROMTAIL_SERVICE="$BUILD_DIR/promtail.service"

        if [ -f "$PROMTAIL_CONFIG" ]; then
            lxc file push "$PROMTAIL_CONFIG" "$vm/tmp/promtail-config.yaml"
        fi
        if [ -f "$PROMTAIL_SERVICE" ]; then
            lxc file push "$PROMTAIL_SERVICE" "$vm/tmp/promtail.service"
        fi

        echo "  Pushing promtail installer..."
        lxc file push "$SCRIPT_DIR/install-promtail.sh" "$vm/tmp/install-promtail.sh"

        echo "  Running promtail installer..."
        lxc exec "$vm" -- chmod +x /tmp/install-promtail.sh
        if lxc exec "$vm" -- /tmp/install-promtail.sh; then
            echo "  Promtail: OK"
        else
            echo "  Promtail: FAILED"
            FAILED+=("$vm:promtail")
        fi
        lxc exec "$vm" -- rm -f /tmp/install-promtail.sh
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
