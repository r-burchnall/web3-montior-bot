# Quickstart

Build, run, and test the fcsc-agent locally.

## Prerequisites

- Go 1.26+ installed
- Access to this repository

## Build

```bash
./scripts/build.sh
```

Produces `build/fcsc-agent` (~12MB static binary).

## Run locally

### Minimal (geyser log parsing only)

```bash
./build/fcsc-agent \
  --listen-addr :9100 \
  --enable-checks geyser \
  --log-scan-paths /home
```

### Full checks (RPC + ClickHouse + process + geyser)

```bash
./build/fcsc-agent \
  --listen-addr :9100 \
  --rpc-url http://64.130.40.37:8899 \
  --clickhouse-url http://monitoring.lxd:8123 \
  --enable-checks all \
  --log-scan-paths /home
```

### With explicit paths

```bash
./build/fcsc-agent \
  --listen-addr :9100 \
  --rpc-url http://64.130.40.37:8899 \
  --clickhouse-url http://monitoring.lxd:8123 \
  --enable-checks all \
  --log-scan-paths /home/ross,/home/matt \
  --traders-garage-dir /home/ross/traders-garage \
  --check-interval 10s
```

## Test endpoints

### Health check (structured JSON)

```bash
curl -s http://localhost:9100/health | python3 -m json.tool
```

Example response:

```json
{
    "status": "healthy",
    "hostname": "dev-ross",
    "bot": {
        "running": true,
        "git_branch": "main",
        "log_file": "/home/ross/traders-garage/logs/metrics.log.20260306.071454",
        "log_age_seconds": 2.5
    },
    "geyser": {
        "recv_rate": 40,
        "send_rate": 40,
        "backlog": 0,
        "total_in": 800,
        "total_out": 800
    },
    "connectivity": {
        "rpc": true,
        "rpc_url": "http://64.130.40.37:8899",
        "clickhouse": true,
        "clickhouse_url": "http://monitoring.lxd:8123"
    },
    "queues": {
        "quoter": { "send_rate": 250, "recv_rate": 249, "backlog": 2 },
        "searcher": { "send_rate": 150, "recv_rate": 150, "backlog": 0 },
        "executor": { "send_rate": 10, "recv_rate": 10, "backlog": 0 }
    }
}
```

Status codes:
- `200` — healthy or degraded
- `503` — unhealthy (bot down)

### Prometheus metrics

```bash
curl -s http://localhost:9100/metrics
```

### Web UI (browser)

Open `http://localhost:9100/` in a browser. The page auto-refreshes every 3s via HTMX.

### HTMX fragments (for testing partial updates)

```bash
curl -s http://localhost:9100/htmx/status
curl -s http://localhost:9100/htmx/geyser
curl -s http://localhost:9100/htmx/queues
curl -s http://localhost:9100/htmx/connectivity
```

## Quick smoke test script

```bash
# Build
./scripts/build.sh

# Start agent in background
./build/fcsc-agent --listen-addr :9100 --enable-checks geyser --log-scan-paths /home &
sleep 2

# Verify all endpoints
echo "=== Health ==="
curl -sf http://localhost:9100/health | python3 -m json.tool

echo "=== Metrics ==="
curl -sf http://localhost:9100/metrics | grep fcsc_

echo "=== Web UI ==="
curl -sf http://localhost:9100/ | head -5 && echo "...OK"

# Cleanup
kill %1
```

## Configure for deployment

Run the interactive wizard to generate systemd unit + config:

```bash
./scripts/configure-agent.sh
```

Then deploy:

```bash
# Single VM
lxc file push build/fcsc-agent <vm>/tmp/fcsc-agent
lxc file push build/fcsc-agent.service <vm>/tmp/fcsc-agent.service
lxc exec <vm> -- bash /tmp/install-agent.sh

# All VMs (edit deploy/vms.conf first)
./deploy/rollout.sh
```
