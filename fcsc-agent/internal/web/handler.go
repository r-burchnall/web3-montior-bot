package web

import (
	"encoding/json"
	"net/http"

	"github.com/fastcarslowcar/fcsc-agent/internal/collector"
)

type Handler struct {
	coll *collector.Collector
}

func NewHandler(coll *collector.Collector) *Handler {
	return &Handler{coll: coll}
}

// Register adds all web routes to the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleIndex)
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/htmx/status", h.handleStatusFragment)
	mux.HandleFunc("/htmx/geyser", h.handleGeyserFragment)
	mux.HandleFunc("/htmx/queues", h.handleQueuesFragment)
	mux.HandleFunc("/htmx/connectivity", h.handleConnectivityFragment)
}

// handleHealth returns structured JSON for curl / programmatic access.
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	snap := h.coll.Snapshot()

	resp := HealthResponse{
		Status:   snap.OverallStatus,
		Hostname: snap.Hostname,
		Bot: BotStatus{
			Running:   snap.BotRunning,
			GitBranch: snap.GitBranch,
			LogFile:   snap.LogFile,
			LogAgeSec: snap.LogAge.Seconds(),
		},
		Geyser: GeyserStatus{
			RecvRate: snap.GeyserRecvRate,
			SendRate: snap.GeyserSendRate,
			Backlog:  snap.GeyserBacklog,
			TotalIn:  snap.GeyserTotalIn,
			TotalOut: snap.GeyserTotalOut,
		},
		Connectivity: ConnectivityStatus{
			RPC:        snap.RPCUp,
			RPCURL:     snap.RPCUrl,
			ClickHouse: snap.ClickHouseUp,
			ClickHouseURL: snap.ClickHouseURL,
		},
	}

	resp.Queues = make(map[string]QueueStatus)
	for name, qs := range snap.Queues {
		if name == "geyser_subscribe" {
			continue
		}
		resp.Queues[name] = QueueStatus{
			SendRate: qs.SendRate,
			RecvRate: qs.RecvRate,
			Backlog:  qs.Backlog,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if snap.OverallStatus == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(resp)
}

// handleIndex serves the full HTML page with HTMX.
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	snap := h.coll.Snapshot()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderIndex(w, snap)
}

// HTMX fragment handlers — each returns a partial HTML snippet.
func (h *Handler) handleStatusFragment(w http.ResponseWriter, r *http.Request) {
	snap := h.coll.Snapshot()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderStatusFragment(w, snap)
}

func (h *Handler) handleGeyserFragment(w http.ResponseWriter, r *http.Request) {
	snap := h.coll.Snapshot()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderGeyserFragment(w, snap)
}

func (h *Handler) handleQueuesFragment(w http.ResponseWriter, r *http.Request) {
	snap := h.coll.Snapshot()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderQueuesFragment(w, snap)
}

func (h *Handler) handleConnectivityFragment(w http.ResponseWriter, r *http.Request) {
	snap := h.coll.Snapshot()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderConnectivityFragment(w, snap)
}

// JSON response types

type HealthResponse struct {
	Status       string                 `json:"status"`
	Hostname     string                 `json:"hostname"`
	Bot          BotStatus              `json:"bot"`
	Geyser       GeyserStatus           `json:"geyser"`
	Connectivity ConnectivityStatus     `json:"connectivity"`
	Queues       map[string]QueueStatus `json:"queues"`
}

type BotStatus struct {
	Running   bool    `json:"running"`
	GitBranch string  `json:"git_branch"`
	LogFile   string  `json:"log_file"`
	LogAgeSec float64 `json:"log_age_seconds"`
}

type GeyserStatus struct {
	RecvRate float64 `json:"recv_rate"`
	SendRate float64 `json:"send_rate"`
	Backlog  int64   `json:"backlog"`
	TotalIn  int64   `json:"total_in"`
	TotalOut int64   `json:"total_out"`
}

type ConnectivityStatus struct {
	RPC           bool   `json:"rpc"`
	RPCURL        string `json:"rpc_url"`
	ClickHouse    bool   `json:"clickhouse"`
	ClickHouseURL string `json:"clickhouse_url"`
}

type QueueStatus struct {
	SendRate float64 `json:"send_rate"`
	RecvRate float64 `json:"recv_rate"`
	Backlog  int64   `json:"backlog"`
}
