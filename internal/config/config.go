package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration.
type Config struct {
	// Network
	Host string
	Port int

	// Authentication
	AuthUser string
	AuthPass string

	// Alert thresholds
	WarnCPU           float64
	WarnMemory        float64
	WarnDisk          float64
	WarnLoad          float64
	WarnProcessCount  int

	// Network connectivity check
	ConnectivityHost    string
	ConnectivityPort    int
	ConnectivityTimeout int

	// Collection / check intervals
	CollectInterval   int // seconds between metric updates
	CheckInterval     int // seconds between health checks (5-10)
	MaxAlerts         int

	// Auto-recovery
	FailureThreshold   int // consecutive failures before restart
	RestartCooldown    int // seconds between restarts
	SocketWaitTimeout  int // seconds to wait for UDP port after restart

	// Notifications
	ServerChanKey string // ServerChan send key (方糖)
	WebhookURL    string
	NotifyCooldown int // seconds between notifications per instance

	// SCP:SL service filter
	ServiceName string // empty = auto-discover all scpsl-*.service
	ServicePort string // explicit port when single-instance

	// -----------------------------------------------------------------------
	// TLS / HTTPS (zero-trust: encrypt all control-plane traffic)
	// -----------------------------------------------------------------------
	TLSEnabled  bool   // enable HTTPS on the agent listener
	TLSCertFile string // path to PEM-encoded certificate
	TLSKeyFile  string // path to PEM-encoded private key

	// -----------------------------------------------------------------------
	// Metrics endpoint authentication (Prometheus scrape auth)
	// -----------------------------------------------------------------------
	MetricsAuth bool // require Basic Auth on /metrics (default false for backward compat)

	// -----------------------------------------------------------------------
	// Adaptive anomaly detection (sliding-window Z-score + EWMA)
	// Reference: "A Survey of Time Series Anomaly Detection Methods in the
	// AIOps Domain" (arXiv:2308.00393); EWMA assigns exponentially
	// decaying weights to older observations, adapting to slow drift.
	// -----------------------------------------------------------------------
	AnomalyEnabled    bool    // enable adaptive threshold alerts
	AnomalyWindowSize int     // sliding window size (number of samples)
	AnomalyZThreshold float64 // Z-score threshold (default 3.0 = 3-sigma rule)
	AnomalyEWMAAlpha  float64 // EWMA smoothing factor 0<alpha<=1 (default 0.2)

	// -----------------------------------------------------------------------
	// Alert management (grouping / inhibition / escalation / silence)
	// Reference: Prometheus Alertmanager best practices;
	// "Alarm reduction and root cause inference based on association mining"
	// (Frontiers in Computer Science, 2023) — reduces 62% redundant alerts.
	// -----------------------------------------------------------------------
	AlertGroupWindow    int // seconds to group related alerts before dispatching (default 60)
	AlertEscalateAfter  int // seconds before a critical alert escalates to next channel (default 900)
	AlertSilenceEnabled bool // enable maintenance/silence windows

	// -----------------------------------------------------------------------
	// Email notification channel (SMTP)
	// -----------------------------------------------------------------------
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPTo       string // comma-separated recipients
}

// Defaults returns a Config populated with sensible defaults.
func Defaults() *Config {
	return &Config{
		Host:                "0.0.0.0",
		Port:                8080,
		AuthUser:            "",
		AuthPass:            "",
		WarnCPU:             90,
		WarnMemory:          90,
		WarnDisk:            85,
		WarnLoad:            4,
		WarnProcessCount:    500,
		ConnectivityHost:    "",
		ConnectivityPort:    443,
		ConnectivityTimeout: 5,
		CollectInterval:     3,
		CheckInterval:       5,
		MaxAlerts:           100,
		FailureThreshold:    3,
		RestartCooldown:     300,
		SocketWaitTimeout:   30,
		ServerChanKey:       "",
		WebhookURL:          "",
		NotifyCooldown:      600,
		ServiceName:         "",
		ServicePort:         "",

		// TLS defaults (disabled by default, enable for production)
		TLSEnabled:  false,
		TLSCertFile: "",
		TLSKeyFile:  "",

		// Metrics auth (disabled by default for backward compatibility)
		MetricsAuth: false,

		// Adaptive anomaly detection defaults
		AnomalyEnabled:    false, // opt-in: requires enough historical samples
		AnomalyWindowSize: 60,    // 60 samples × 3s collect = 3 minutes
		AnomalyZThreshold: 3.0,   // 3-sigma rule (99.7% confidence)
		AnomalyEWMAAlpha:  0.2,   // standard EWMA smoothing factor

		// Alert management defaults
		AlertGroupWindow:    60,  // group alerts within 60s
		AlertEscalateAfter:  900, // escalate after 15 min unacknowledged
		AlertSilenceEnabled: true,

		// SMTP defaults (disabled when host is empty)
		SMTPHost:     "",
		SMTPPort:     587,
		SMTPUser:     "",
		SMTPPassword: "",
		SMTPFrom:     "",
		SMTPTo:       "",
	}
}

// Load reads configuration from a file. Missing keys keep their defaults.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.Trim(strings.TrimSpace(line[idx+1:]), "\"'")

		switch strings.ToLower(k) {
		case "host":
			cfg.Host = v
		case "port":
			cfg.Port = parseInt(v, cfg.Port)
		case "auth_user":
			cfg.AuthUser = v
		case "auth_pass":
			cfg.AuthPass = v
		case "warn_cpu":
			cfg.WarnCPU = parseFloat(v, cfg.WarnCPU)
		case "warn_memory":
			cfg.WarnMemory = parseFloat(v, cfg.WarnMemory)
		case "warn_disk":
			cfg.WarnDisk = parseFloat(v, cfg.WarnDisk)
		case "warn_load":
			cfg.WarnLoad = parseFloat(v, cfg.WarnLoad)
		case "warn_process_count":
			cfg.WarnProcessCount = parseInt(v, cfg.WarnProcessCount)
		case "connectivity_host":
			cfg.ConnectivityHost = v
		case "connectivity_port":
			cfg.ConnectivityPort = parseInt(v, cfg.ConnectivityPort)
		case "connectivity_timeout":
			cfg.ConnectivityTimeout = parseInt(v, cfg.ConnectivityTimeout)
		case "collect_interval":
			cfg.CollectInterval = parseInt(v, cfg.CollectInterval)
		case "check_interval":
			cfg.CheckInterval = parseInt(v, cfg.CheckInterval)
		case "max_alerts":
			cfg.MaxAlerts = parseInt(v, cfg.MaxAlerts)
		case "failure_threshold":
			cfg.FailureThreshold = parseInt(v, cfg.FailureThreshold)
		case "restart_cooldown":
			cfg.RestartCooldown = parseInt(v, cfg.RestartCooldown)
		case "socket_wait_timeout":
			cfg.SocketWaitTimeout = parseInt(v, cfg.SocketWaitTimeout)
		case "serverchan_key":
			cfg.ServerChanKey = v
		case "webhook_url":
			cfg.WebhookURL = v
		case "notify_cooldown":
			cfg.NotifyCooldown = parseInt(v, cfg.NotifyCooldown)
		case "service":
			cfg.ServiceName = v
		case "service_port":
			cfg.ServicePort = v

		// --- TLS ---
		case "tls_enabled":
			cfg.TLSEnabled = parseBool(v, cfg.TLSEnabled)
		case "tls_cert_file":
			cfg.TLSCertFile = v
		case "tls_key_file":
			cfg.TLSKeyFile = v

		// --- Metrics auth ---
		case "metrics_auth":
			cfg.MetricsAuth = parseBool(v, cfg.MetricsAuth)

		// --- Anomaly detection ---
		case "anomaly_enabled":
			cfg.AnomalyEnabled = parseBool(v, cfg.AnomalyEnabled)
		case "anomaly_window_size":
			cfg.AnomalyWindowSize = parseInt(v, cfg.AnomalyWindowSize)
		case "anomaly_z_threshold":
			cfg.AnomalyZThreshold = parseFloat(v, cfg.AnomalyZThreshold)
		case "anomaly_ewma_alpha":
			cfg.AnomalyEWMAAlpha = parseFloat(v, cfg.AnomalyEWMAAlpha)

		// --- Alert management ---
		case "alert_group_window":
			cfg.AlertGroupWindow = parseInt(v, cfg.AlertGroupWindow)
		case "alert_escalate_after":
			cfg.AlertEscalateAfter = parseInt(v, cfg.AlertEscalateAfter)
		case "alert_silence_enabled":
			cfg.AlertSilenceEnabled = parseBool(v, cfg.AlertSilenceEnabled)

		// --- SMTP ---
		case "smtp_host":
			cfg.SMTPHost = v
		case "smtp_port":
			cfg.SMTPPort = parseInt(v, cfg.SMTPPort)
		case "smtp_user":
			cfg.SMTPUser = v
		case "smtp_password":
			cfg.SMTPPassword = v
		case "smtp_from":
			cfg.SMTPFrom = v
		case "smtp_to":
			cfg.SMTPTo = v
		}
	}

	return cfg, scanner.Err()
}

func parseInt(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

func parseFloat(s string, def float64) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return n
}

func parseBool(s string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		return def
	}
}
