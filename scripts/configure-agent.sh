#!/usr/bin/env bash
set -euo pipefail

# Interactive wizard to generate a systemd unit file and optional YAML config
# for fcsc-agent. Outputs files to the specified directory.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${1:-$PROJECT_ROOT/build}"

mkdir -p "$OUTPUT_DIR"

echo "=== fcsc-agent Configuration Wizard ==="
echo ""

# Hostname
DEFAULT_HOSTNAME="$(hostname)"
read -rp "VM hostname [$DEFAULT_HOSTNAME]: " HOSTNAME
HOSTNAME="${HOSTNAME:-$DEFAULT_HOSTNAME}"

# Listen address
read -rp "Listen address [:9100]: " LISTEN_ADDR
LISTEN_ADDR="${LISTEN_ADDR:-:9100}"

# RPC URL
read -rp "Solana RPC URL [http://64.130.40.37:8899]: " RPC_URL
RPC_URL="${RPC_URL:-http://64.130.40.37:8899}"

# ClickHouse URL
read -rp "ClickHouse URL [http://monitoring.lxd:8123]: " CH_URL
CH_URL="${CH_URL:-http://monitoring.lxd:8123}"

# Log scan paths
read -rp "Log scan paths (comma-separated) [/home]: " LOG_PATHS
LOG_PATHS="${LOG_PATHS:-/home}"

# Log scan interval
read -rp "Log scan interval [60s]: " LOG_INTERVAL
LOG_INTERVAL="${LOG_INTERVAL:-60s}"

# Check interval
read -rp "Health check interval [5s]: " CHECK_INTERVAL
CHECK_INTERVAL="${CHECK_INTERVAL:-5s}"

# Bot process name
read -rp "Bot process name (empty = auto-detect): " BOT_PROCESS
BOT_PROCESS="${BOT_PROCESS:-}"

# Traders garage dir
read -rp "Traders-garage directory (empty = auto-detect from logs): " TG_DIR
TG_DIR="${TG_DIR:-}"

# Enable checks
echo ""
echo "Available checks: geyser, clickhouse, rpc, process"
read -rp "Enable checks [all]: " CHECKS
CHECKS="${CHECKS:-all}"

# Build the ExecStart command line
EXEC_ARGS="--listen-addr ${LISTEN_ADDR} --rpc-url ${RPC_URL} --clickhouse-url ${CH_URL}"
EXEC_ARGS+=" --log-scan-paths ${LOG_PATHS} --log-scan-interval ${LOG_INTERVAL}"
EXEC_ARGS+=" --check-interval ${CHECK_INTERVAL} --enable-checks ${CHECKS}"
if [ -n "$BOT_PROCESS" ]; then
    EXEC_ARGS+=" --bot-process-name ${BOT_PROCESS}"
fi
if [ -n "$TG_DIR" ]; then
    EXEC_ARGS+=" --traders-garage-dir ${TG_DIR}"
fi

# Generate systemd unit file
UNIT_FILE="$OUTPUT_DIR/fcsc-agent.service"
cat > "$UNIT_FILE" <<EOF
[Unit]
Description=FCSC Health Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fcsc-agent ${EXEC_ARGS}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=fcsc-agent

[Install]
WantedBy=multi-user.target
EOF

echo ""
echo "Generated: $UNIT_FILE"

# Generate YAML config (as reference/backup)
YAML_FILE="$OUTPUT_DIR/fcsc-agent.yaml"
cat > "$YAML_FILE" <<EOF
listen_addr: "${LISTEN_ADDR}"
rpc_url: "${RPC_URL}"
clickhouse_url: "${CH_URL}"
log_scan_paths:
$(echo "$LOG_PATHS" | tr ',' '\n' | sed 's/^ *//' | sed 's/^/  - "/' | sed 's/$/"/')
log_scan_interval: "${LOG_INTERVAL}"
check_interval: "${CHECK_INTERVAL}"
bot_process_name: "${BOT_PROCESS}"
traders_garage_dir: "${TG_DIR}"
enable_checks:
$(echo "$CHECKS" | tr ',' '\n' | sed 's/^ *//' | sed 's/^/  - "/' | sed 's/$/"/')
EOF

echo "Generated: $YAML_FILE"
echo ""
echo "To install, run:"
echo "  ./deploy/install-agent.sh $OUTPUT_DIR"
