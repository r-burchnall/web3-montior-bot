#!/usr/bin/env bash
set -euo pipefail

# Interactive wizard to generate promtail config for a VM.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${1:-$PROJECT_ROOT/build}"

mkdir -p "$OUTPUT_DIR"

echo "=== Promtail Configuration Wizard ==="
echo ""

# Hostname
DEFAULT_HOSTNAME="$(hostname)"
read -rp "VM hostname [$DEFAULT_HOSTNAME]: " HOSTNAME
HOSTNAME="${HOSTNAME:-$DEFAULT_HOSTNAME}"

# Loki URL
read -rp "Loki push URL [http://monitoring.lxd:3100/loki/api/v1/push]: " LOKI_URL
LOKI_URL="${LOKI_URL:-http://monitoring.lxd:3100/loki/api/v1/push}"

# Log scan paths
read -rp "Log base directories (comma-separated) [/home]: " LOG_BASES
LOG_BASES="${LOG_BASES:-/home}"

# Promtail listen port
read -rp "Promtail HTTP listen port [9080]: " PROMTAIL_PORT
PROMTAIL_PORT="${PROMTAIL_PORT:-9080}"

# Generate the __path__ entries
PATH_ENTRIES=""
IFS=',' read -ra BASES <<< "$LOG_BASES"
for base in "${BASES[@]}"; do
    base=$(echo "$base" | xargs) # trim whitespace
    PATH_ENTRIES+="          - ${base}/**/traders-garage/logs/metrics.log*"$'\n'
done

CONFIG_FILE="$OUTPUT_DIR/promtail-config.yaml"
cat > "$CONFIG_FILE" <<EOF
server:
  http_listen_port: ${PROMTAIL_PORT}
  grpc_listen_port: 0

positions:
  filename: /var/lib/promtail/positions.yaml

clients:
  - url: ${LOKI_URL}

scrape_configs:
  - job_name: arb-bot-logs
    static_configs:
      - targets:
          - localhost
        labels:
          vm: "${HOSTNAME}"
          job: arb-bot
          __path__: placeholder
    file_sd_configs: []
    pipeline_stages:
      - regex:
          expression: '^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+(?P<level>\w+)\s+(?P<thread>ThreadId\(\d+\))\s+(?P<message>.*)'
      - labels:
          level:
      - timestamp:
          source: timestamp
          format: "2006-01-02T15:04:05.999999999Z"

  - job_name: arb-bot-logs-discovery
    static_configs:
      - targets:
          - localhost
        labels:
          vm: "${HOSTNAME}"
          job: arb-bot
          __path__: ""
    file_sd_configs: []

# NOTE: Promtail doesn't support recursive glob natively in all versions.
# Use the following approach: one static_config per known log directory,
# or use a discovery mechanism. The install script will configure this
# based on discovered log directories at install time.
EOF

# Generate the actual working config with discovered paths
WORKING_CONFIG="$OUTPUT_DIR/promtail-config.yaml"
cat > "$WORKING_CONFIG" <<EOF
server:
  http_listen_port: ${PROMTAIL_PORT}
  grpc_listen_port: 0

positions:
  filename: /var/lib/promtail/positions.yaml

clients:
  - url: ${LOKI_URL}

scrape_configs:
  - job_name: arb-bot-logs
    static_configs:
      - targets:
          - localhost
        labels:
          vm: "${HOSTNAME}"
          job: arb-bot
          __path__: /home/*/traders-garage/logs/metrics.log*
    pipeline_stages:
      - regex:
          expression: '^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+(?P<level>\w+)\s+(?P<thread>ThreadId\(\d+\))\s+(?P<message>.*)'
      - labels:
          level:
      - timestamp:
          source: timestamp
          format: "2006-01-02T15:04:05.999999999Z"
EOF

echo ""
echo "Generated: $WORKING_CONFIG"

# Generate systemd unit
UNIT_FILE="$OUTPUT_DIR/promtail.service"
cat > "$UNIT_FILE" <<EOF
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

echo "Generated: $UNIT_FILE"
echo ""
echo "To install, run:"
echo "  ./deploy/install-promtail.sh $OUTPUT_DIR"
