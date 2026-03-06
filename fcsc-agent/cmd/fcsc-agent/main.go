package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fastcarslowcar/fcsc-agent/internal/collector"
	"github.com/fastcarslowcar/fcsc-agent/internal/config"
	"github.com/fastcarslowcar/fcsc-agent/internal/logparser"
	"github.com/fastcarslowcar/fcsc-agent/internal/web"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	hostname, _ := os.Hostname()
	log.Printf("fcsc-agent starting on %s, listen=%s", hostname, cfg.ListenAddr)
	log.Printf("  rpc-url=%s clickhouse-url=%s", cfg.RPCUrl, cfg.ClickHouseURL)
	log.Printf("  log-scan-paths=%v scan-interval=%s", cfg.LogScanPaths, cfg.LogScanInterval)
	log.Printf("  checks=%v check-interval=%s", cfg.EnableChecks, cfg.CheckInterval)

	tailer := logparser.NewTailer()
	go tailer.Run()

	coll := collector.New(cfg, tailer)
	coll.StartChecks()

	reg := prometheus.NewRegistry()
	reg.MustRegister(coll)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Web UI and health API
	webHandler := web.NewHandler(coll)
	webHandler.Register(mux)

	log.Printf("serving on %s (web UI: /, health JSON: /health, metrics: /metrics)", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("http server error: %v", err)
	}
}
