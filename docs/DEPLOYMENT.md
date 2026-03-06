# Deployment Guide

Run all commands from the **Fastcar 2 host** (where `lxc` commands work).

## Prerequisites

- `lxc` CLI with access to all VMs
- This repo checked out on the Fastcar 2 host
- Go 1.26+ on the host (or use pre-built binary)

## 1. Build the agent

```bash
cd ~/monitor-project
./scripts/build.sh
```

Or pull the pre-built binary from a VM that has it:
```bash
mkdir -p build
lxc file pull dev-claude/home/developer/git/monitor-project/build/fcsc-agent build/fcsc-agent
chmod +x build/fcsc-agent
```

## 2. Deploy to all VMs

```bash
# All VMs at once (agent + promtail)
./deploy/rollout.sh

# Test with one VM first
./deploy/rollout.sh dev-ross

# Agent only
./deploy/rollout.sh --agent-only

# Promtail only
./deploy/rollout.sh --promtail-only
```

The rollout script will:
- Push the fcsc-agent binary and systemd unit to each VM
- Install and start the agent service
- Download promtail 3.2.1 (first deploy only), install config with hostname, start service

## 3. Update the monitoring stack on monitoring.lxd

The existing docker-compose on monitoring.lxd needs two changes:

### 3a. Add Prometheus targets volume mount

The existing Prometheus service mounts only the config file. We need to add a volume for the targets directory.

```bash
lxc exec monitoring -- bash
cd /home/galileo/traders-garage
```

Create the targets directory:
```bash
mkdir -p docker/prometheus/targets
```

Edit the docker-compose.yml and add this volume to the `prometheus` service:

```yaml
    volumes:
      - prometheus_data:/prometheus
      - ./docker/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - ./docker/prometheus/console_libraries:/etc/prometheus/console_libraries
      - ./docker/prometheus/consoles:/etc/prometheus/consoles
      - ./docker/prometheus/targets:/etc/prometheus/targets:ro    # <-- ADD THIS
```

### 3b. Copy our config files in

From inside monitoring.lxd:

```bash
# Clone or pull this repo
git clone <repo-url> /tmp/monitor-project
# Or if already cloned:
# cd /tmp/monitor-project && git pull

# Prometheus config (adds fcsc-agent scrape job with file_sd_configs)
cp /tmp/monitor-project/monitoring-stack/prometheus/prometheus.yml \
   /home/galileo/traders-garage/docker/prometheus/prometheus.yml

# Prometheus targets
cp /tmp/monitor-project/monitoring-stack/prometheus/targets/vm-health.yml \
   /home/galileo/traders-garage/docker/prometheus/targets/vm-health.yml

# Grafana dashboards (into a subfolder so they don't mix with existing dashboards)
mkdir -p /home/galileo/traders-garage/docker/grafana/dashboards/fcsc
cp /tmp/monitor-project/monitoring-stack/grafana/dashboards/*.json \
   /home/galileo/traders-garage/docker/grafana/dashboards/fcsc/

# Grafana provisioning — add our dashboard provider
# NOTE: This adds to existing provisioning, not replaces it.
# Check if there's already a dashboards.yml — if so, append our provider to it.
cp /tmp/monitor-project/monitoring-stack/grafana/provisioning/dashboards.yml \
   /home/galileo/traders-garage/docker/grafana/provisioning/dashboards-fcsc.yml
```

### 3c. Restart services

```bash
cd /home/galileo/traders-garage

# Restart prometheus (picks up new config + targets volume)
docker compose up -d prometheus

# Restart grafana (picks up new dashboards + provisioning)
docker compose restart grafana
```

Note: Prometheus has `--web.enable-lifecycle`, so after the first restart (to get the targets volume mounted), future config/target changes can be hot-reloaded:
```bash
curl -X POST http://localhost:9090/-/reload
```

## 4. Verify deployment

### Check all agents are running

From Fastcar 2 host:
```bash
for vm in $(grep -v '^#' deploy/vms.conf | grep -v '^$'); do
    echo -n "$vm: "
    lxc exec "$vm" -- curl -sf http://localhost:9100/health 2>/dev/null \
      | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['status'])" \
      2>/dev/null || echo "UNREACHABLE"
done
```

### Check Prometheus targets

```bash
lxc exec monitoring -- curl -s http://localhost:9090/api/v1/targets \
  | python3 -c "
import sys, json
data = json.load(sys.stdin)
for t in data['data']['activeTargets']:
    if 'fcsc' in t.get('labels', {}).get('job', ''):
        print(f\"{t['labels'].get('vm','?'):15s} {t['health']:8s} {t['lastScrape']}\")
"
```

### Check Grafana

Open in browser: `http://monitoring.lxd:33000`

Navigate to: **Dashboards > FCSC Monitoring** folder

Three dashboards should appear:
- Fleet Overview
- VM Detail
- Log Explorer

### Check logs are flowing to Loki

```bash
lxc exec monitoring -- curl -s \
  'http://localhost:3100/loki/api/v1/query?query={job="arb-bot"}&limit=5' \
  | python3 -m json.tool | head -20
```

## 5. Updating

### Update agent binary

```bash
./scripts/build.sh
./deploy/rollout.sh --agent-only
```

### Update promtail config

Edit `deploy/promtail-config.yaml`, then:
```bash
./deploy/rollout.sh --promtail-only
```

### Update Prometheus targets

Edit `monitoring-stack/prometheus/targets/vm-health.yml`, copy to monitoring.lxd, then:
```bash
lxc exec monitoring -- curl -X POST http://localhost:9090/-/reload
```

### Update Grafana dashboards

Copy new JSON files to monitoring.lxd's `docker/grafana/dashboards/`, then:
```bash
lxc exec monitoring -- docker restart grafana
```

## Troubleshooting

### Agent not starting
```bash
lxc exec <vm> -- journalctl -u fcsc-agent -n 50
```

### Promtail not shipping logs
```bash
lxc exec <vm> -- journalctl -u promtail -n 50
lxc exec <vm> -- curl -s http://localhost:9080/targets
```

### Prometheus not scraping
```bash
lxc exec monitoring -- curl -s http://localhost:9090/api/v1/targets | python3 -m json.tool
```

### Agent web UI
Open `http://<vm>.lxd:9100/` in a browser for the HTMX health dashboard.
