package web

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"time"

	"github.com/fastcarslowcar/fcsc-agent/internal/collector"
)

var funcMap = template.FuncMap{
	"statusColor": func(status string) string {
		switch status {
		case "healthy":
			return "#22c55e"
		case "degraded":
			return "#eab308"
		case "unhealthy":
			return "#ef4444"
		default:
			return "#6b7280"
		}
	},
	"boolColor": func(ok bool) string {
		if ok {
			return "#22c55e"
		}
		return "#ef4444"
	},
	"boolText": func(ok bool) string {
		if ok {
			return "UP"
		}
		return "DOWN"
	},
	"boolIcon": func(ok bool) string {
		if ok {
			return "●"
		}
		return "✕"
	},
	"rateColor": func(rate float64) string {
		if rate > 0 {
			return "#22c55e"
		}
		return "#ef4444"
	},
	"backlogColor": func(backlog int64) string {
		if backlog == 0 {
			return "#22c55e"
		} else if backlog < 100 {
			return "#eab308"
		}
		return "#ef4444"
	},
	"formatDuration": func(d time.Duration) string {
		if d == 0 {
			return "N/A"
		}
		if d < time.Second {
			return fmt.Sprintf("%dms", d.Milliseconds())
		}
		if d < time.Minute {
			return fmt.Sprintf("%.1fs", d.Seconds())
		}
		if d < time.Hour {
			return fmt.Sprintf("%.1fm", d.Minutes())
		}
		return fmt.Sprintf("%.1fh", d.Hours())
	},
	"sortedQueueNames": func(queues map[string]interface{}) []string {
		names := make([]string, 0, len(queues))
		for n := range queues {
			if n != "geyser_subscribe" {
				names = append(names, n)
			}
		}
		sort.Strings(names)
		return names
	},
}

var indexTemplate = template.Must(template.New("index").Funcs(funcMap).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>FCSC Agent — {{.Hostname}}</title>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
      background: #0f172a;
      color: #e2e8f0;
      min-height: 100vh;
    }
    .container { max-width: 900px; margin: 0 auto; padding: 24px 16px; }
    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid #1e293b;
    }
    header h1 { font-size: 1.5rem; font-weight: 600; }
    header h1 span { color: #94a3b8; font-weight: 400; font-size: 1rem; margin-left: 8px; }
    .badge {
      display: inline-block;
      padding: 4px 12px;
      border-radius: 9999px;
      font-size: 0.75rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }
    .grid { display: grid; gap: 16px; }
    .grid-2 { grid-template-columns: 1fr 1fr; }
    .grid-3 { grid-template-columns: 1fr 1fr 1fr; }
    @media (max-width: 640px) {
      .grid-2, .grid-3 { grid-template-columns: 1fr; }
    }
    .card {
      background: #1e293b;
      border-radius: 12px;
      padding: 20px;
      border: 1px solid #334155;
    }
    .card h2 {
      font-size: 0.8rem;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      color: #94a3b8;
      margin-bottom: 12px;
    }
    .stat-value {
      font-size: 2rem;
      font-weight: 700;
      line-height: 1;
      font-variant-numeric: tabular-nums;
    }
    .stat-label {
      font-size: 0.75rem;
      color: #64748b;
      margin-top: 4px;
    }
    .indicator {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 0;
    }
    .indicator + .indicator { border-top: 1px solid #334155; }
    .indicator .dot {
      width: 10px;
      height: 10px;
      border-radius: 50%;
      flex-shrink: 0;
    }
    .indicator .name { flex: 1; font-size: 0.875rem; }
    .indicator .value {
      font-size: 0.875rem;
      font-weight: 600;
      font-variant-numeric: tabular-nums;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.875rem;
    }
    th {
      text-align: left;
      padding: 8px 12px;
      color: #94a3b8;
      font-weight: 500;
      border-bottom: 1px solid #334155;
    }
    td {
      padding: 8px 12px;
      border-bottom: 1px solid #1e293b;
      font-variant-numeric: tabular-nums;
    }
    .meta {
      font-size: 0.75rem;
      color: #64748b;
      margin-top: 16px;
      text-align: center;
    }
    .meta code {
      background: #334155;
      padding: 2px 6px;
      border-radius: 4px;
      font-size: 0.7rem;
    }
    .pulse { animation: pulse 2s infinite; }
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
    .htmx-settling { opacity: 0.8; }
    .htmx-swapping { opacity: 0.5; }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <h1>fcsc-agent<span>{{.Hostname}}</span></h1>
      <div id="status-badge" hx-get="/htmx/status" hx-trigger="every 3s" hx-swap="innerHTML">
        {{template "status-badge" .}}
      </div>
    </header>

    <div class="grid grid-3" style="margin-bottom: 16px;">
      <div class="card">
        <h2>Bot</h2>
        <div class="stat-value" style="color: {{boolColor .BotRunning}}">{{boolText .BotRunning}}</div>
        <div class="stat-label">branch: {{.GitBranch}}</div>
      </div>
      <div class="card" id="geyser-card" hx-get="/htmx/geyser" hx-trigger="every 3s" hx-swap="innerHTML">
        {{template "geyser-card" .}}
      </div>
      <div class="card" id="connectivity-card" hx-get="/htmx/connectivity" hx-trigger="every 3s" hx-swap="innerHTML">
        {{template "connectivity-card" .}}
      </div>
    </div>

    <div class="grid grid-2" style="margin-bottom: 16px;">
      <div class="card">
        <h2>Geyser Detail</h2>
        <div class="indicator">
          <div class="name">Total In</div>
          <div class="value">{{.GeyserTotalIn}}</div>
        </div>
        <div class="indicator">
          <div class="name">Total Out</div>
          <div class="value">{{.GeyserTotalOut}}</div>
        </div>
        <div class="indicator">
          <div class="name">Backlog</div>
          <div class="value" style="color: {{backlogColor .GeyserBacklog}}">{{.GeyserBacklog}}</div>
        </div>
      </div>
      <div class="card">
        <h2>Log File</h2>
        <div class="indicator">
          <div class="name">File</div>
          <div class="value" style="font-size: 0.7rem; word-break: break-all;">{{if .LogFile}}{{.LogFile}}{{else}}not found{{end}}</div>
        </div>
        <div class="indicator">
          <div class="name">Last Line Age</div>
          <div class="value">{{formatDuration .LogAge}}</div>
        </div>
      </div>
    </div>

    <div class="card" id="queues-card" hx-get="/htmx/queues" hx-trigger="every 3s" hx-swap="innerHTML" style="margin-bottom: 16px;">
      {{template "queues-table" .}}
    </div>

    <div class="meta">
      Auto-refreshing every 3s via HTMX &middot;
      <code>curl {{.Hostname}}:9100/health</code> for JSON &middot;
      <code>/metrics</code> for Prometheus
    </div>
  </div>
</body>
</html>
`))

var statusBadgeTemplate = template.Must(template.New("status-badge").Funcs(funcMap).Parse(
	`<span class="badge" style="background: {{statusColor .OverallStatus}}; color: #0f172a;">{{.OverallStatus}}</span>`,
))

var geyserCardTemplate = template.Must(template.New("geyser-card").Funcs(funcMap).Parse(`<h2>Geyser</h2>
<div class="stat-value" style="color: {{rateColor .GeyserRecvRate}}">{{printf "%.0f" .GeyserRecvRate}}</div>
<div class="stat-label">msg/s recv rate</div>
<div style="margin-top: 8px; font-size: 0.8rem; color: #94a3b8;">
  send: {{printf "%.0f" .GeyserSendRate}} msg/s &middot; backlog: <span style="color: {{backlogColor .GeyserBacklog}}">{{.GeyserBacklog}}</span>
</div>`))

var connectivityCardTemplate = template.Must(template.New("connectivity-card").Funcs(funcMap).Parse(`<h2>Connectivity</h2>
<div class="indicator">
  <div class="dot" style="background: {{boolColor .RPCUp}}"></div>
  <div class="name">Solana RPC</div>
  <div class="value" style="color: {{boolColor .RPCUp}}">{{boolIcon .RPCUp}}</div>
</div>
<div class="indicator">
  <div class="dot" style="background: {{boolColor .ClickHouseUp}}"></div>
  <div class="name">ClickHouse</div>
  <div class="value" style="color: {{boolColor .ClickHouseUp}}">{{boolIcon .ClickHouseUp}}</div>
</div>`))

var queuesTableTemplate = template.Must(template.New("queues-table").Funcs(funcMap).Parse(`<h2>Queue Metrics</h2>
<table>
  <thead>
    <tr><th>Queue</th><th>Send Rate</th><th>Recv Rate</th><th>Backlog</th></tr>
  </thead>
  <tbody>
    {{range $name, $q := .Queues}}{{if ne $name "geyser_subscribe"}}
    <tr>
      <td>{{$name}}</td>
      <td>{{printf "%.0f" $q.SendRate}} msg/s</td>
      <td>{{printf "%.0f" $q.RecvRate}} msg/s</td>
      <td style="color: {{backlogColor $q.Backlog}}">{{$q.Backlog}}</td>
    </tr>
    {{end}}{{end}}
    {{if eq (len .Queues) 0}}
    <tr><td colspan="4" style="color: #64748b; text-align: center;">No queue data yet</td></tr>
    {{end}}
  </tbody>
</table>`))

func init() {
	// Parse the sub-templates into the index template so they can be invoked
	template.Must(indexTemplate.New("status-badge").Funcs(funcMap).Parse(
		`<span class="badge" style="background: {{statusColor .OverallStatus}}; color: #0f172a;">{{.OverallStatus}}</span>`,
	))
	template.Must(indexTemplate.New("geyser-card").Funcs(funcMap).Parse(`<h2>Geyser</h2>
<div class="stat-value" style="color: {{rateColor .GeyserRecvRate}}">{{printf "%.0f" .GeyserRecvRate}}</div>
<div class="stat-label">msg/s recv rate</div>
<div style="margin-top: 8px; font-size: 0.8rem; color: #94a3b8;">
  send: {{printf "%.0f" .GeyserSendRate}} msg/s &middot; backlog: <span style="color: {{backlogColor .GeyserBacklog}}">{{.GeyserBacklog}}</span>
</div>`))
	template.Must(indexTemplate.New("connectivity-card").Funcs(funcMap).Parse(`<h2>Connectivity</h2>
<div class="indicator">
  <div class="dot" style="background: {{boolColor .RPCUp}}"></div>
  <div class="name">Solana RPC</div>
  <div class="value" style="color: {{boolColor .RPCUp}}">{{boolIcon .RPCUp}}</div>
</div>
<div class="indicator">
  <div class="dot" style="background: {{boolColor .ClickHouseUp}}"></div>
  <div class="name">ClickHouse</div>
  <div class="value" style="color: {{boolColor .ClickHouseUp}}">{{boolIcon .ClickHouseUp}}</div>
</div>`))
	template.Must(indexTemplate.New("queues-table").Funcs(funcMap).Parse(`<h2>Queue Metrics</h2>
<table>
  <thead>
    <tr><th>Queue</th><th>Send Rate</th><th>Recv Rate</th><th>Backlog</th></tr>
  </thead>
  <tbody>
    {{range $name, $q := .Queues}}{{if ne $name "geyser_subscribe"}}
    <tr>
      <td>{{$name}}</td>
      <td>{{printf "%.0f" $q.SendRate}} msg/s</td>
      <td>{{printf "%.0f" $q.RecvRate}} msg/s</td>
      <td style="color: {{backlogColor $q.Backlog}}">{{$q.Backlog}}</td>
    </tr>
    {{end}}{{end}}
    {{if eq (len .Queues) 0}}
    <tr><td colspan="4" style="color: #64748b; text-align: center;">No queue data yet</td></tr>
    {{end}}
  </tbody>
</table>`))
}

func renderIndex(w io.Writer, snap *collector.HealthSnapshot) {
	indexTemplate.Execute(w, snap)
}

func renderStatusFragment(w io.Writer, snap *collector.HealthSnapshot) {
	statusBadgeTemplate.Execute(w, snap)
}

func renderGeyserFragment(w io.Writer, snap *collector.HealthSnapshot) {
	geyserCardTemplate.Execute(w, snap)
}

func renderQueuesFragment(w io.Writer, snap *collector.HealthSnapshot) {
	queuesTableTemplate.Execute(w, snap)
}

func renderConnectivityFragment(w io.Writer, snap *collector.HealthSnapshot) {
	connectivityCardTemplate.Execute(w, snap)
}
