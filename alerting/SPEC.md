# Alerting Specification (Deferred)

## Delivery: Grafana Native Telegram Contact Point

Grafana supports Telegram natively as a contact point. No custom bot code required.

### Setup Steps
1. Create Telegram bot via @BotFather, obtain bot token
2. Add bot to team channel, obtain chat_id
3. Grafana > Alerting > Contact Points > New > Telegram
4. Configure bot_token and chat_id

### Alert Rules

| Rule | Condition | Severity | For |
|------|-----------|----------|-----|
| Geyser Down | `fcsc_geyser_recv_rate == 0` | Critical | 60s |
| Geyser Degraded | `fcsc_geyser_recv_rate == 0` | Warning | 15s |
| Bot Down | `fcsc_bot_running == 0` | Critical | 15s |
| ClickHouse Unreachable | `fcsc_clickhouse_up == 0` | Critical | 15s |
| RPC Unreachable | `fcsc_rpc_up == 0` | Critical | 15s |
| Log Stale | `time() - fcsc_log_last_line_timestamp > 120` | Warning | 60s |
| Geyser Backlog | `fcsc_geyser_backlog > 1000` | Warning | 30s |

### Notification Template
```
{{ .Status | toUpper }}: {{ .Labels.alertname }}
VM: {{ .Labels.vm }}
Branch: {{ .Labels.git_branch }}
Value: {{ .Values }}
```
