package collector

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/fastcarslowcar/fcsc-agent/internal/checks"
	"github.com/fastcarslowcar/fcsc-agent/internal/config"
	"github.com/fastcarslowcar/fcsc-agent/internal/discovery"
	"github.com/fastcarslowcar/fcsc-agent/internal/logparser"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	cfg    *config.Config
	tailer *logparser.Tailer

	// Prometheus descriptors
	botRunning      *prometheus.Desc
	botInfo         *prometheus.Desc
	geyserTotalIn   *prometheus.Desc
	geyserTotalOut  *prometheus.Desc
	geyserBacklog   *prometheus.Desc
	geyserSendRate  *prometheus.Desc
	geyserRecvRate  *prometheus.Desc
	queueSendRate   *prometheus.Desc
	queueRecvRate   *prometheus.Desc
	queueBacklog    *prometheus.Desc
	clickhouseUp    *prometheus.Desc
	rpcUp           *prometheus.Desc
	logLastLineTS   *prometheus.Desc

	// Cached check results (updated periodically)
	mu             sync.RWMutex
	cachedRPCUp    float64
	cachedCHUp     float64
	cachedBotUp    float64
	cachedBranch   string
	hostname       string
}

func New(cfg *config.Config, tailer *logparser.Tailer) *Collector {
	hostname, _ := os.Hostname()

	labels := []string{"vm"}

	c := &Collector{
		cfg:    cfg,
		tailer: tailer,
		hostname: hostname,

		botRunning:     prometheus.NewDesc("fcsc_bot_running", "Whether the arb bot process is running", labels, nil),
		botInfo:        prometheus.NewDesc("fcsc_bot_info", "Bot info with branch label", append(labels, "git_branch"), nil),
		geyserTotalIn:  prometheus.NewDesc("fcsc_geyser_total_in", "Geyser subscribe total messages in", labels, nil),
		geyserTotalOut: prometheus.NewDesc("fcsc_geyser_total_out", "Geyser subscribe total messages out", labels, nil),
		geyserBacklog:  prometheus.NewDesc("fcsc_geyser_backlog", "Geyser subscribe backlog", labels, nil),
		geyserSendRate: prometheus.NewDesc("fcsc_geyser_send_rate", "Geyser subscribe send rate msg/s", labels, nil),
		geyserRecvRate: prometheus.NewDesc("fcsc_geyser_recv_rate", "Geyser subscribe recv rate msg/s", labels, nil),
		queueSendRate:  prometheus.NewDesc("fcsc_queue_send_rate", "Queue send rate msg/s", append(labels, "queue"), nil),
		queueRecvRate:  prometheus.NewDesc("fcsc_queue_recv_rate", "Queue recv rate msg/s", append(labels, "queue"), nil),
		queueBacklog:   prometheus.NewDesc("fcsc_queue_backlog", "Queue backlog count", append(labels, "queue"), nil),
		clickhouseUp:   prometheus.NewDesc("fcsc_clickhouse_up", "ClickHouse reachable", labels, nil),
		rpcUp:          prometheus.NewDesc("fcsc_rpc_up", "Solana RPC reachable", labels, nil),
		logLastLineTS:  prometheus.NewDesc("fcsc_log_last_line_timestamp", "Unix timestamp of last parsed log line", labels, nil),

		cachedBranch: "unknown",
	}

	return c
}

// StartChecks runs periodic connectivity checks in a background goroutine.
func (c *Collector) StartChecks() {
	// Run initial checks immediately
	c.runChecks()
	c.refreshDiscovery()

	// Periodic connectivity checks
	go func() {
		ticker := time.NewTicker(c.cfg.CheckInterval)
		defer ticker.Stop()
		for range ticker.C {
			c.runChecks()
		}
	}()

	// Periodic log file + branch discovery
	go func() {
		ticker := time.NewTicker(c.cfg.LogScanInterval)
		defer ticker.Stop()
		for range ticker.C {
			c.refreshDiscovery()
		}
	}()
}

func (c *Collector) runChecks() {
	var rpc, ch, bot float64

	if c.cfg.IsCheckEnabled("rpc") {
		if checks.RPCReachable(c.cfg.RPCUrl) {
			rpc = 1
		}
	}

	if c.cfg.IsCheckEnabled("clickhouse") {
		if checks.ClickHouseReachable(c.cfg.ClickHouseURL) {
			ch = 1
		}
	}

	if c.cfg.IsCheckEnabled("process") {
		if checks.BotRunning(c.cfg.BotProcessName) {
			bot = 1
		}
	}

	c.mu.Lock()
	c.cachedRPCUp = rpc
	c.cachedCHUp = ch
	c.cachedBotUp = bot
	c.mu.Unlock()
}

func (c *Collector) refreshDiscovery() {
	// Find latest log file
	latestLog := discovery.FindLatestLogFile(c.cfg.LogScanPaths)
	if latestLog != "" && latestLog != c.tailer.CurrentFile() {
		c.tailer.SetFile(latestLog)
	}

	// Detect git branch
	repoDir := c.cfg.TradersGarageDir
	if repoDir == "" && latestLog != "" {
		repoDir = discovery.TradersGarageDirFromLogPath(latestLog)
	}
	branch := discovery.GitBranch(repoDir)

	c.mu.Lock()
	c.cachedBranch = branch
	c.mu.Unlock()

	if latestLog != "" {
		log.Printf("discovery: log=%s branch=%s", latestLog, branch)
	}
}

// HealthSnapshot holds a point-in-time view of all health data for the web UI.
type HealthSnapshot struct {
	Hostname       string
	GitBranch      string
	BotRunning     bool
	RPCUp          bool
	ClickHouseUp   bool
	LogFile        string
	LogAge         time.Duration
	GeyserRecvRate float64
	GeyserSendRate float64
	GeyserBacklog  int64
	GeyserTotalIn  int64
	GeyserTotalOut int64
	Queues         map[string]*logparser.QueueStats
	LastLineTime   time.Time
	CheckInterval  time.Duration
	EnabledChecks  []string
	RPCUrl         string
	ClickHouseURL  string
	OverallStatus  string // "healthy", "degraded", "unhealthy"
}

// Snapshot returns a HealthSnapshot for use by the web UI and JSON API.
func (c *Collector) Snapshot() *HealthSnapshot {
	c.mu.RLock()
	rpcUp := c.cachedRPCUp
	chUp := c.cachedCHUp
	botUp := c.cachedBotUp
	branch := c.cachedBranch
	c.mu.RUnlock()

	state := c.tailer.State()

	snap := &HealthSnapshot{
		Hostname:      c.hostname,
		GitBranch:     branch,
		BotRunning:    botUp == 1,
		RPCUp:         rpcUp == 1,
		ClickHouseUp:  chUp == 1,
		LogFile:       c.tailer.CurrentFile(),
		LastLineTime:  state.LastLineTime,
		Queues:        state.Queues,
		CheckInterval: c.cfg.CheckInterval,
		EnabledChecks: c.cfg.EnableChecks,
		RPCUrl:        c.cfg.RPCUrl,
		ClickHouseURL: c.cfg.ClickHouseURL,
	}

	if !state.LastLineTime.IsZero() {
		snap.LogAge = time.Since(state.LastLineTime)
	}

	if gs, ok := state.Queues["geyser_subscribe"]; ok {
		snap.GeyserRecvRate = gs.RecvRate
		snap.GeyserSendRate = gs.SendRate
		snap.GeyserBacklog = gs.Backlog
		snap.GeyserTotalIn = gs.TotalIn
		snap.GeyserTotalOut = gs.TotalOut
	}

	// Determine overall status
	snap.OverallStatus = "healthy"
	if !snap.BotRunning && c.cfg.IsCheckEnabled("process") {
		snap.OverallStatus = "unhealthy"
	} else if snap.GeyserRecvRate == 0 && c.cfg.IsCheckEnabled("geyser") {
		snap.OverallStatus = "degraded"
	} else if (!snap.RPCUp && c.cfg.IsCheckEnabled("rpc")) || (!snap.ClickHouseUp && c.cfg.IsCheckEnabled("clickhouse")) {
		snap.OverallStatus = "degraded"
	}

	return snap
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.botRunning
	ch <- c.botInfo
	ch <- c.geyserTotalIn
	ch <- c.geyserTotalOut
	ch <- c.geyserBacklog
	ch <- c.geyserSendRate
	ch <- c.geyserRecvRate
	ch <- c.queueSendRate
	ch <- c.queueRecvRate
	ch <- c.queueBacklog
	ch <- c.clickhouseUp
	ch <- c.rpcUp
	ch <- c.logLastLineTS
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	rpcUp := c.cachedRPCUp
	chUp := c.cachedCHUp
	botUp := c.cachedBotUp
	branch := c.cachedBranch
	c.mu.RUnlock()

	vm := c.hostname
	state := c.tailer.State()

	// Bot status
	if c.cfg.IsCheckEnabled("process") {
		ch <- prometheus.MustNewConstMetric(c.botRunning, prometheus.GaugeValue, botUp, vm)
	}
	ch <- prometheus.MustNewConstMetric(c.botInfo, prometheus.GaugeValue, 1, vm, branch)

	// Geyser metrics (from the "geyser_subscribe" queue)
	if c.cfg.IsCheckEnabled("geyser") {
		if gs, ok := state.Queues["geyser_subscribe"]; ok {
			ch <- prometheus.MustNewConstMetric(c.geyserTotalIn, prometheus.GaugeValue, float64(gs.TotalIn), vm)
			ch <- prometheus.MustNewConstMetric(c.geyserTotalOut, prometheus.GaugeValue, float64(gs.TotalOut), vm)
			ch <- prometheus.MustNewConstMetric(c.geyserBacklog, prometheus.GaugeValue, float64(gs.Backlog), vm)
			ch <- prometheus.MustNewConstMetric(c.geyserSendRate, prometheus.GaugeValue, gs.SendRate, vm)
			ch <- prometheus.MustNewConstMetric(c.geyserRecvRate, prometheus.GaugeValue, gs.RecvRate, vm)
		}
	}

	// Other queue metrics
	for name, qs := range state.Queues {
		if name == "geyser_subscribe" {
			continue // already reported above
		}
		ch <- prometheus.MustNewConstMetric(c.queueSendRate, prometheus.GaugeValue, qs.SendRate, vm, name)
		ch <- prometheus.MustNewConstMetric(c.queueRecvRate, prometheus.GaugeValue, qs.RecvRate, vm, name)
		ch <- prometheus.MustNewConstMetric(c.queueBacklog, prometheus.GaugeValue, float64(qs.Backlog), vm, name)
	}

	// Connectivity checks
	if c.cfg.IsCheckEnabled("clickhouse") {
		ch <- prometheus.MustNewConstMetric(c.clickhouseUp, prometheus.GaugeValue, chUp, vm)
	}
	if c.cfg.IsCheckEnabled("rpc") {
		ch <- prometheus.MustNewConstMetric(c.rpcUp, prometheus.GaugeValue, rpcUp, vm)
	}

	// Last log line timestamp
	if !state.LastLineTime.IsZero() {
		ch <- prometheus.MustNewConstMetric(c.logLastLineTS, prometheus.GaugeValue, float64(state.LastLineTime.Unix()), vm)
	}
}
