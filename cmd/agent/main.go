package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Oxen112774/ServerHealthMonitor/internal/alertmanager"
	"github.com/Oxen112774/ServerHealthMonitor/internal/anomaly"
	"github.com/Oxen112774/ServerHealthMonitor/internal/collector"
	"github.com/Oxen112774/ServerHealthMonitor/internal/config"
	"github.com/Oxen112774/ServerHealthMonitor/internal/monitor"
	"github.com/Oxen112774/ServerHealthMonitor/internal/notifier"
	"github.com/Oxen112774/ServerHealthMonitor/internal/web"
)

func main() {
	// CLI flags
	configPath := flag.String("config", "/etc/server-health-monitor-agent.conf", "Path to config file")
	host := flag.String("host", "", "Listen address (overrides config)")
	port := flag.Int("port", 0, "Listen port (overrides config)")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// CLI overrides
	if *host != "" {
		cfg.Host = *host
	}
	if *port != 0 {
		cfg.Port = *port
	}

	// Validate intervals: NewTicker panics on non-positive, so clamp to safe bounds.
	if cfg.CollectInterval <= 0 {
		cfg.CollectInterval = 3
	}
	if cfg.CheckInterval < 3 || cfg.CheckInterval > 30 {
		cfg.CheckInterval = 5
	}

	log.Println("=" + repeat("=", 58))
	log.Println("  Server Health Monitor Agent (Go Edition)")
	log.Println("=" + repeat("=", 58))
	log.Printf("  Config:         %s", *configPath)
	log.Printf("  Listen:         %s:%d", cfg.Host, cfg.Port)
	log.Printf("  Collect every:  %ds", cfg.CollectInterval)
	log.Printf("  Check every:    %ds", cfg.CheckInterval)
	log.Printf("  Fail threshold: %d times", cfg.FailureThreshold)
	log.Printf("  Restart cooldown: %ds", cfg.RestartCooldown)
	if cfg.ServerChanKey != "" {
		log.Println("  ServerChan:     enabled")
	}
	if cfg.WebhookURL != "" {
		log.Println("  Webhook:        enabled")
	}
	log.Println("=" + repeat("=", 58))

	// Components
	c := collector.NewCollector(cfg.ConnectivityHost, cfg.ConnectivityPort, cfg.ConnectivityTimeout)

	smtpCfg := notifier.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		To:       cfg.SMTPTo,
	}
	n := notifier.NewNotifier(cfg.MaxAlerts, cfg.ServerChanKey, cfg.WebhookURL, cfg.NotifyCooldown, smtpCfg)

	// Alert manager: groups, inhibits, escalates, and silences alerts.
	// Dispatch bridges alertmanager groups back to the notifier channels.
	am := alertmanager.NewManager(alertmanager.Config{
		GroupWindow:   time.Duration(cfg.AlertGroupWindow) * time.Second,
		GroupLabels:   []string{"alertname", "severity", "instance"},
		EscalateAfter: time.Duration(cfg.AlertEscalateAfter) * time.Second,
		MaxAlerts:     cfg.MaxAlerts,
	}, func(group *alertmanager.Group) {
		for _, a := range group.Alerts {
			if a.Silenced || a.Inhibited {
				continue
			}
			sev := notifier.SeverityWarning
			switch a.Severity {
			case alertmanager.SeverityCritical:
				sev = notifier.SeverityCritical
			case alertmanager.SeverityRecovery:
				sev = notifier.SeverityRecovery
			}
			n.Push(notifier.Alert{
				Time:    a.Time.Format("2006-01-02 15:04:05"),
				Type:    sev,
				Title:   a.Title,
				Message: a.Message,
			})
		}
	})
	defer am.Stop()

	// Default inhibition rule: host-level critical alerts inhibit
	// service-level warnings on the same instance.
	am.AddInhibitRule(alertmanager.InhibitRule{
		Name:        "host_down_inhibits_services",
		SourceMatch: map[string]string{"alertname": "HostDown", "severity": "critical"},
		TargetMatch: map[string]string{"severity": "warning"},
		EqualLabels: []string{"instance"},
	})

	m := monitor.New(c, n, cfg.FailureThreshold, cfg.RestartCooldown, cfg.SocketWaitTimeout)
	m.SetResourceThresholds(monitor.ResourceThresholds{
		CPUPercent:    cfg.WarnCPU,
		MemoryPercent: cfg.WarnMemory,
		DiskPercent:   cfg.WarnDisk,
		LoadMultiple:  cfg.WarnLoad,
		ProcessCount:  cfg.WarnProcessCount,
	})

	// Attach adaptive anomaly detectors if enabled.
	// Reference: sliding-window Z-score + EWMA (arXiv:2308.00393).
	if cfg.AnomalyEnabled {
		baseCfg := anomaly.Config{
			WindowSize: cfg.AnomalyWindowSize,
			ZThreshold: cfg.AnomalyZThreshold,
			EWMAAlpha:  cfg.AnomalyEWMAAlpha,
			MinSamples: 10,
		}
		cpuCfg := baseCfg
		cpuCfg.Kind = anomaly.MetricCPU
		memCfg := baseCfg
		memCfg.Kind = anomaly.MetricMemory
		diskCfg := baseCfg
		diskCfg.Kind = anomaly.MetricDisk
		loadCfg := baseCfg
		loadCfg.Kind = anomaly.MetricLoad
		m.SetAnomalyDetectors(
			anomaly.NewDetector(cpuCfg),
			anomaly.NewDetector(memCfg),
			anomaly.NewDetector(diskCfg),
			anomaly.NewDetector(loadCfg),
		)
		log.Println("  Anomaly:        enabled (Z-score + EWMA)")
	}

	h := web.New(c, n, cfg.AuthUser, cfg.AuthPass, cfg.MetricsAuth)

	// First collection
	c.Collect()
	h.UpdatePrometheus()

	// Collection ticker
	collectTicker := time.NewTicker(time.Duration(cfg.CollectInterval) * time.Second)
	defer collectTicker.Stop()

	// Health check ticker
	checkTicker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Second)
	defer checkTicker.Stop()

	// Prometheus update ticker (aligned with collection)
	promTicker := time.NewTicker(time.Duration(cfg.CollectInterval) * time.Second)
	defer promTicker.Stop()

	// Start background workers
	go func() {
		for range collectTicker.C {
			c.Collect()
		}
	}()

	go func() {
		for range checkTicker.C {
			m.Check()
		}
	}()

	go func() {
		for range promTicker.C {
			h.UpdatePrometheus()
		}
	}()

	// HTTP server
	mux := http.NewServeMux()
	h.Register(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP(S) server in background.
	// TLS reference: NIST SP 800-52 Rev.2 — encrypt all management-plane
	// traffic; zero-trust principle: never assume the network is safe.
	go func() {
		log.Printf("HTTP server listening on %s", addr)
		log.Printf("  Dashboard:  http://<server-ip>:%d/", cfg.Port)
		log.Printf("  API:        http://<server-ip>:%d/api/status", cfg.Port)
		log.Printf("  Health:     http://<server-ip>:%d/api/health", cfg.Port)
		log.Printf("  Readiness:  http://<server-ip>:%d/api/ready", cfg.Port)
		log.Printf("  Prometheus: http://<server-ip>:%d/metrics", cfg.Port)
		if cfg.TLSEnabled {
			log.Printf("  TLS:        enabled (HTTPS)")
			if err := srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS server error: %v", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server error: %v", err)
			}
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown sequence:
	// 1. Mark as shutting down → readiness returns 503 → load balancers drain
	// 2. Stop tickers (no new collections / checks)
	// 3. Shutdown HTTP server with timeout (finish in-flight requests)
	// Reference: "Graceful Shutdown in Go: Properly Stopping Services Under Load"
	h.SetShuttingDown()
	collectTicker.Stop()
	checkTicker.Stop()
	promTicker.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server graceful shutdown error: %v", err)
		_ = srv.Close()
	}
	log.Println("Shutdown complete.")
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
