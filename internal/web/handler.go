package web

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Oxen112774/ServerHealthMonitor/internal/collector"
	"github.com/Oxen112774/ServerHealthMonitor/internal/notifier"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler serves the dashboard, API, and Prometheus metrics.
type Handler struct {
	collector *collector.Collector
	notifier  *notifier.Notifier
	authUser  string
	authPass  string

	// metricsAuth requires Basic Auth on /metrics when true.
	// Prometheus scrape config must then supply basic_auth credentials.
	metricsAuth bool

	// isShuttingDown is set true during graceful shutdown so that
	// readiness probes return 503 and load balancers stop sending traffic.
	// Reference: "Graceful Shutdown in Go: Properly Stopping Services Under Load"
	isShuttingDown atomic.Bool

	// history
	mu       sync.Mutex
	histCPU  []float64
	histMem  []float64
	histDisk []float64
	histMax  int

	// Prometheus metrics
	promCPU       prometheus.Gauge
	promMemory    prometheus.Gauge
	promDisk      prometheus.Gauge
	promLoad1     prometheus.Gauge
	promUptime    prometheus.Gauge
	promInstances *prometheus.GaugeVec
	promLabels    map[string]struct{} // "service|port" currently exposed
}

// New creates a new web Handler.
func New(c *collector.Collector, n *notifier.Notifier, authUser, authPass string, metricsAuth bool) *Handler {
	h := &Handler{
		collector:   c,
		notifier:    n,
		authUser:    authUser,
		authPass:    authPass,
		metricsAuth: metricsAuth,
		histMax:     60,
		promLabels:  make(map[string]struct{}),
		promCPU: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scpsl_cpu_percent",
			Help: "Current CPU usage percentage",
		}),
		promMemory: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scpsl_memory_percent",
			Help: "Current memory usage percentage",
		}),
		promDisk: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scpsl_disk_percent",
			Help: "Current disk usage percentage",
		}),
		promLoad1: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scpsl_load_1min",
			Help: "1-minute load average",
		}),
		promUptime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scpsl_system_uptime_seconds",
			Help: "System uptime in seconds",
		}),
		promInstances: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scpsl_instance_up",
				Help: "SCP:SL instance status (1=active, 0=inactive)",
			},
			[]string{"service", "port"},
		),
	}

	prometheus.MustRegister(h.promCPU)
	prometheus.MustRegister(h.promMemory)
	prometheus.MustRegister(h.promDisk)
	prometheus.MustRegister(h.promLoad1)
	prometheus.MustRegister(h.promUptime)
	prometheus.MustRegister(h.promInstances)

	return h
}

// UpdatePrometheus refreshes Prometheus gauge values from current metrics.
func (h *Handler) UpdatePrometheus() {
	m, instances := h.collector.Get()

	h.promCPU.Set(m.CPUPercent)
	h.promMemory.Set(m.MemoryPercent)
	h.promDisk.Set(m.DiskPercent)
	h.promLoad1.Set(m.Load1)
	h.promUptime.Set(float64(m.Uptime))

	seen := make(map[string]struct{}, len(instances))
	for _, inst := range instances {
		val := 0.0
		if inst.State == "active" {
			val = 1
		}
		h.promInstances.WithLabelValues(inst.Service, fmt.Sprintf("%d", inst.Port)).Set(val)
		seen[inst.Service+"|"+strconv.Itoa(inst.Port)] = struct{}{}
	}

	// Delete gauges for instances that no longer exist (renamed/removed units).
	for key := range h.promLabels {
		if _, ok := seen[key]; !ok {
			parts := strings.SplitN(key, "|", 2)
			if len(parts) == 2 {
				h.promInstances.DeleteLabelValues(parts[0], parts[1])
			}
		}
	}
	h.promLabels = seen

	// Update history
	h.mu.Lock()
	h.histCPU = appendFloat(h.histCPU, m.CPUPercent, h.histMax)
	h.histMem = appendFloat(h.histMem, m.MemoryPercent, h.histMax)
	h.histDisk = appendFloat(h.histDisk, m.DiskPercent, h.histMax)
	h.mu.Unlock()
}

func appendFloat(arr []float64, v float64, max int) []float64 {
	arr = append(arr, v)
	if len(arr) > max {
		arr = arr[len(arr)-max:]
	}
	return arr
}

// SetShuttingDown marks the handler as shutting down. Readiness probes
// will return 503 after this is called, signalling load balancers to
// drain traffic before the process exits.
func (h *Handler) SetShuttingDown() {
	h.isShuttingDown.Store(true)
}

// Register routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.authWrap(h.handleDashboard))
	mux.HandleFunc("/index.html", h.authWrap(h.handleDashboard))
	mux.HandleFunc("/api/status", h.authWrap(h.handleAPIStatus))
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/ready", h.handleReady)
	if h.metricsAuth {
		mux.Handle("/metrics", h.authWrap(promhttp.Handler().ServeHTTP))
	} else {
		mux.Handle("/metrics", promhttp.Handler())
	}
}

func (h *Handler) authWrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")

		if h.authUser != "" {
			user, pass, ok := r.BasicAuth()
			// Constant-time comparison to prevent timing attacks.
			// Reference: NIST SP 800-63B — verifiers SHALL use constant-time
			// comparison when comparing secrets.
			userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(h.authUser)) == 1
			passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(h.authPass)) == 1
			if !ok || !userMatch || !passMatch {
				w.Header().Set("WWW-Authenticate", `Basic realm="SCP:SL Monitor"`)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (h *Handler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(dashboardHTML))
}

type apiResponse struct {
	Timestamp string                 `json:"timestamp"`
	Server    apiServer              `json:"server"`
	Metrics   apiMetrics             `json:"metrics"`
	Instances []collector.Instance   `json:"instances"`
	Alerts    []notifier.Alert       `json:"alerts"`
	History   map[string][]float64   `json:"history"`
}

type apiServer struct {
	Hostname  string `json:"hostname"`
	Uptime    int    `json:"uptime"`
	CPUCount  int    `json:"cpu_count"`
}

type apiMetrics struct {
	CPU     cpuMetric     `json:"cpu"`
	Memory  memoryMetric  `json:"memory"`
	Disk    diskMetric    `json:"disk"`
	Load    loadMetric    `json:"load"`
	Network string        `json:"network"`
	Process int           `json:"process_count"`
}

type cpuMetric struct {
	Percent float64 `json:"percent"`
}

type memoryMetric struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Percent   float64 `json:"percent"`
}

type diskMetric struct {
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	Free    uint64  `json:"free"`
	Percent float64 `json:"percent"`
}

type loadMetric struct {
	Load1  float64 `json:"1min"`
	Load5  float64 `json:"5min"`
	Load15 float64 `json:"15min"`
}

func (h *Handler) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	m, instances := h.collector.Get()
	alerts := h.notifier.Get()

	h.mu.Lock()
	hist := map[string][]float64{
		"cpu":    append([]float64(nil), h.histCPU...),
		"memory": append([]float64(nil), h.histMem...),
		"disk":   append([]float64(nil), h.histDisk...),
	}
	h.mu.Unlock()

	resp := apiResponse{
		Timestamp: m.Timestamp,
		Server: apiServer{
			Hostname: m.Hostname,
			Uptime:   m.Uptime,
			CPUCount: m.CPUCount,
		},
		Metrics: apiMetrics{
			CPU:    cpuMetric{Percent: m.CPUPercent},
			Memory: memoryMetric{Total: m.MemoryTotal, Used: m.MemoryUsed, Available: m.MemoryAvail, Percent: m.MemoryPercent},
			Disk:   diskMetric{Total: m.DiskTotal, Used: m.DiskUsed, Free: m.DiskFree, Percent: m.DiskPercent},
			Load:   loadMetric{Load1: m.Load1, Load5: m.Load5, Load15: m.Load15},
			Network: m.Network,
			Process: m.ProcessCount,
		},
		Instances: instances,
		Alerts:    alerts,
		History:   hist,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleReady is the readiness probe. Returns 200 when the agent is ready
// to serve traffic, 503 during graceful shutdown so load balancers / k8s
// endpoints controllers remove this pod before connections are dropped.
// Reference: "Why Graceful Shutdown Matters in Kubernetes" (alikhil.dev, 2025)
func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.isShuttingDown.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "shutting_down",
			"ready":  false,
			"time":   time.Now().Format(time.RFC3339),
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ready",
		"ready":  true,
		"time":   time.Now().Format(time.RFC3339),
	})
}
