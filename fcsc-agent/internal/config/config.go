package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr      string        `yaml:"listen_addr"`
	RPCUrl          string        `yaml:"rpc_url"`
	ClickHouseURL   string        `yaml:"clickhouse_url"`
	LogScanPaths    []string      `yaml:"log_scan_paths"`
	LogScanInterval time.Duration `yaml:"log_scan_interval"`
	CheckInterval   time.Duration `yaml:"check_interval"`
	BotProcessName  string        `yaml:"bot_process_name"`
	TradersGarageDir string       `yaml:"traders_garage_dir"`
	EnableChecks    []string      `yaml:"enable_checks"`
}

func DefaultConfig() *Config {
	return &Config{
		ListenAddr:      ":9100",
		RPCUrl:          "http://64.130.40.37:8899",
		ClickHouseURL:   "http://monitoring.lxd:8123",
		LogScanPaths:    []string{"/home"},
		LogScanInterval: 60 * time.Second,
		CheckInterval:   5 * time.Second,
		BotProcessName:  "",
		TradersGarageDir: "",
		EnableChecks:    []string{"all"},
	}
}

func (c *Config) IsCheckEnabled(name string) bool {
	for _, check := range c.EnableChecks {
		if check == "all" || check == name {
			return true
		}
	}
	return false
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	var configFile string
	var logScanPaths string
	var enableChecks string
	var logScanInterval string
	var checkInterval string

	flag.StringVar(&configFile, "config", "", "Path to YAML config file")
	flag.StringVar(&cfg.ListenAddr, "listen-addr", cfg.ListenAddr, "Prometheus metrics listen address")
	flag.StringVar(&cfg.RPCUrl, "rpc-url", cfg.RPCUrl, "Solana RPC endpoint for health check")
	flag.StringVar(&cfg.ClickHouseURL, "clickhouse-url", cfg.ClickHouseURL, "ClickHouse endpoint for health check")
	flag.StringVar(&logScanPaths, "log-scan-paths", strings.Join(cfg.LogScanPaths, ","), "Comma-separated base paths to scan for metrics.log files")
	flag.StringVar(&logScanInterval, "log-scan-interval", cfg.LogScanInterval.String(), "How often to rescan for new log files")
	flag.StringVar(&checkInterval, "check-interval", cfg.CheckInterval.String(), "How often to run connectivity checks")
	flag.StringVar(&cfg.BotProcessName, "bot-process-name", cfg.BotProcessName, "Process name to detect (empty = auto-detect)")
	flag.StringVar(&cfg.TradersGarageDir, "traders-garage-dir", cfg.TradersGarageDir, "Path to traders-garage repo (empty = auto-detect)")
	flag.StringVar(&enableChecks, "enable-checks", strings.Join(cfg.EnableChecks, ","), "Comma-separated checks: geyser,clickhouse,rpc,process (or 'all')")
	flag.Parse()

	// Load YAML config file if specified (values are defaults for flags)
	if configFile != "" {
		if err := cfg.loadYAML(configFile); err != nil {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
	}

	// Re-parse flags to override YAML values (flags take precedence)
	// We do this by checking if flags were explicitly set
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "log-scan-paths":
			cfg.LogScanPaths = splitComma(logScanPaths)
		case "log-scan-interval":
			if d, err := time.ParseDuration(logScanInterval); err == nil {
				cfg.LogScanInterval = d
			}
		case "check-interval":
			if d, err := time.ParseDuration(checkInterval); err == nil {
				cfg.CheckInterval = d
			}
		case "enable-checks":
			cfg.EnableChecks = splitComma(enableChecks)
		}
	})

	// If flags weren't visited but we didn't load YAML, parse the string defaults
	if configFile == "" {
		cfg.LogScanPaths = splitComma(logScanPaths)
		cfg.EnableChecks = splitComma(enableChecks)
		if d, err := time.ParseDuration(logScanInterval); err == nil {
			cfg.LogScanInterval = d
		}
		if d, err := time.ParseDuration(checkInterval); err == nil {
			cfg.CheckInterval = d
		}
	}

	// Environment variable overrides
	cfg.applyEnvOverrides()

	return cfg, nil
}

func (c *Config) loadYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("FCSC_LISTEN_ADDR"); v != "" {
		c.ListenAddr = v
	}
	if v := os.Getenv("FCSC_RPC_URL"); v != "" {
		c.RPCUrl = v
	}
	if v := os.Getenv("FCSC_CLICKHOUSE_URL"); v != "" {
		c.ClickHouseURL = v
	}
	if v := os.Getenv("FCSC_LOG_SCAN_PATHS"); v != "" {
		c.LogScanPaths = splitComma(v)
	}
	if v := os.Getenv("FCSC_BOT_PROCESS_NAME"); v != "" {
		c.BotProcessName = v
	}
	if v := os.Getenv("FCSC_TRADERS_GARAGE_DIR"); v != "" {
		c.TradersGarageDir = v
	}
	if v := os.Getenv("FCSC_ENABLE_CHECKS"); v != "" {
		c.EnableChecks = splitComma(v)
	}
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
