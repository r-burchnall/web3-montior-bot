#!/usr/bin/env bash
set -euo pipefail

# Runs both agent and promtail installers.
# Usage: ./install-all.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================="
echo " Installing FCSC monitoring components"
echo "========================================="
echo ""

"$SCRIPT_DIR/install-agent.sh"
echo ""
"$SCRIPT_DIR/install-promtail.sh"

echo ""
echo "========================================="
echo " All components installed successfully"
echo "========================================="
