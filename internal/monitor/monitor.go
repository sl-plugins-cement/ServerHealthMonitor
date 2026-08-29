package monitor

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/Oxen112774/ServerHealthMonitor/internal/anomaly"
	"github.com/Oxen112774/ServerHealthMonitor/internal/collector"
	"github.com/Oxen112774/ServerHealthMonitor/internal/notifier"
)

// InstanceState tracks health state for one SCP:SL instance.
type InstanceState struct {
	Service         string
	Port            int
	FailureCount    int
	LastRestartTime int64 // unix
	LastState       string
}

// ResourceThresholds defines optional system-level alert thresholds.
// Zero values disable the corresponding check.
type ResourceThresholds struct {
	CPUPercent    float64 // system-wide CPU usage percent
	MemoryPercent float64 // memory usage percent
	DiskPercent   float64 // root disk usage percent
	LoadMultiple  float64 // load_1m / CPU cores ratio
	ProcessCount  int     // total process count
}

// Monitor runs health checks and performs auto-recovery.
type Monitor struct {
	mu        sync.Mutex
	states    map[string]*InstanceState
	collector *collector.Collector
	notifier  *notifier.Notifier

	failureThreshold  int
	restartCooldown   int
	socketWaitTimeout int

	resThresholds ResourceThresholds
	resActive     map[string]bool // resource alert latched until recovery

	// Adaptive anomaly detectors (optional, nil when disabled).
	// Reference: sliding-window Z-score + EWMA (arXiv:2308.00393).
	anomalyCPU     *anomaly.Detector
	anomalyMemory  *anomaly.Detector
	anomalyDisk    *anomaly.Detector
	anomalyLoad    *anomaly.Detector
	anomalyActive  map[string]bool // anomaly alert latched until recovery
}

// New creates a Monitor.
func New(c *collector.Collector, n *notifier.Notifier, failureThreshold, restartCooldown, socketWaitTimeout int) *Monitor {
	return &Monitor{
		states:            make(map[string]*InstanceState),
		collector:         c,
		notifier:          n,
		failureThreshold:  failureThreshold,
		restartCooldown:   restartCooldown,
		socketWaitTimeout: socketWaitTimeout,
		resActive:         make(map[string]bool),
		anomalyActive:     make(map[string]bool),
	}
}

// SetAnomalyDetectors attaches adaptive anomaly detectors for CPU, memory,
// disk, and load. Pass nil for any metric to disable anomaly detection on it.
func (m *Monitor) SetAnomalyDetectors(cpu, memory, disk, load *anomaly.Detector) {
	m.anomalyCPU = cpu
	m.anomalyMemory = memory
	m.anomalyDisk = disk
	m.anomalyLoad = load
}

// SetResourceThresholds enables system-level resource alerts.
func (m *Monitor) SetResourceThresholds(t ResourceThresholds) {
	m.resThresholds = t
}

// Check runs one health check cycle for all instances.
func (m *Monitor) Check() {
	metrics, instances := m.collector.Get()
	now := time.Now().Unix()

	for _, inst := range instances {
		m.checkInstance(inst, now)
	}

	m.checkResources(metrics)
	m.checkAnomalies(metrics)
}

// checkResources emits warnings when system resource thresholds are exceeded.
// Each alert latches until the resource recovers, avoiding notification storms.
func (m *Monitor) checkResources(metrics collector.Metrics) {
	t := m.resThresholds
	if t == (ResourceThresholds{}) {
		return // no thresholds configured
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	latch := func(key, title, message string) {
		if m.resActive[key] {
			return
		}
		m.resActive[key] = true
		m.notifier.Push(notifier.Alert{
			Time:    ts,
			Type:    notifier.SeverityWarning,
			Title:   title,
			Message: message,
		})
	}
	released := func(key string) {
		if m.resActive[key] {
			m.resActive[key] = false
		}
	}

	if t.CPUPercent > 0 {
		if metrics.CPUPercent >= t.CPUPercent {
			latch("cpu", "CPU 使用率过高",
				fmt.Sprintf("当前 %.1f%% ≥ 阈值 %.1f%%", metrics.CPUPercent, t.CPUPercent))
		} else {
			released("cpu")
		}
	}
	if t.MemoryPercent > 0 {
		if metrics.MemoryPercent >= t.MemoryPercent {
			latch("memory", "内存使用率过高",
				fmt.Sprintf("当前 %.1f%% ≥ 阈值 %.1f%%", metrics.MemoryPercent, t.MemoryPercent))
		} else {
			released("memory")
		}
	}
	if t.DiskPercent > 0 {
		if metrics.DiskPercent >= t.DiskPercent {
			latch("disk", "磁盘使用率过高",
				fmt.Sprintf("当前 %.1f%% ≥ 阈值 %.1f%%", metrics.DiskPercent, t.DiskPercent))
		} else {
			released("disk")
		}
	}
	if t.LoadMultiple > 0 && metrics.CPUCount > 0 {
		limit := t.LoadMultiple * float64(metrics.CPUCount)
		if metrics.Load1 >= limit {
			latch("load", "系统负载过高",
				fmt.Sprintf("1 分钟负载 %.2f ≥ 阈值 %.2f（%d 核 × %.1f）",
					metrics.Load1, limit, metrics.CPUCount, t.LoadMultiple))
		} else {
			released("load")
		}
	}
	if t.ProcessCount > 0 {
		if metrics.ProcessCount >= t.ProcessCount {
			latch("process", "进程数过多",
				fmt.Sprintf("当前 %d ≥ 阈值 %d", metrics.ProcessCount, t.ProcessCount))
		} else {
			released("process")
		}
	}
}

// checkAnomalies evaluates system metrics against adaptive anomaly detectors.
// Unlike static thresholds (checkResources), anomaly detection learns the
// normal baseline from recent history and flags statistically significant
// deviations (3-sigma rule). This catches slow leaks and sudden spikes that
// static thresholds would miss.
// Reference: "A Survey of Time Series Anomaly Detection Methods in the
// AIOps Domain" (arXiv:2308.00393); 3-sigma rule flags 99.7% outliers.
func (m *Monitor) checkAnomalies(metrics collector.Metrics) {
	if m.anomalyCPU == nil && m.anomalyMemory == nil && m.anomalyDisk == nil && m.anomalyLoad == nil {
		return
	}

	ts := time.Now().Format("2006-01-02 15:04:05")

	anomalyLatch := func(key, title, message string) {
		if m.anomalyActive[key] {
			return
		}
		m.anomalyActive[key] = true
		m.notifier.Push(notifier.Alert{
			Time:    ts,
			Type:    notifier.SeverityWarning,
			Title:   "[异常检测] " + title,
			Message: message + "\n（基于滑动窗口 Z-score 自适应阈值，非静态阈值）",
		})
	}
	anomalyReleased := func(key string) {
		if m.anomalyActive[key] {
			m.anomalyActive[key] = false
		}
	}

	if m.anomalyCPU != nil {
		res := m.anomalyCPU.Evaluate(metrics.CPUPercent)
		if res.Anomalous {
			anomalyLatch("anomaly_cpu", "CPU 使用率异常波动",
				fmt.Sprintf("当前 %.1f%%, 基线 %.1f%%, Z-score %.2f (%s)",
					res.Value, res.Mean, res.ZScore, res.Reason))
		} else {
			anomalyReleased("anomaly_cpu")
		}
	}

	if m.anomalyMemory != nil {
		res := m.anomalyMemory.Evaluate(metrics.MemoryPercent)
		if res.Anomalous {
			anomalyLatch("anomaly_memory", "内存使用率异常波动",
				fmt.Sprintf("当前 %.1f%%, 基线 %.1f%%, Z-score %.2f (%s)",
					res.Value, res.Mean, res.ZScore, res.Reason))
		} else {
			anomalyReleased("anomaly_memory")
		}
	}

	if m.anomalyDisk != nil {
		res := m.anomalyDisk.Evaluate(metrics.DiskPercent)
		if res.Anomalous {
			anomalyLatch("anomaly_disk", "磁盘使用率异常增长",
				fmt.Sprintf("当前 %.1f%%, 基线 %.1f%%, Z-score %.2f (%s)\n可能存在磁盘泄漏或日志暴涨",
					res.Value, res.Mean, res.ZScore, res.Reason))
		} else {
			anomalyReleased("anomaly_disk")
		}
	}

	if m.anomalyLoad != nil {
		res := m.anomalyLoad.Evaluate(metrics.Load1)
		if res.Anomalous {
			anomalyLatch("anomaly_load", "系统负载异常飙升",
				fmt.Sprintf("当前 1min 负载 %.2f, 基线 %.2f, Z-score %.2f (%s)",
					res.Value, res.Mean, res.ZScore, res.Reason))
		} else {
			anomalyReleased("anomaly_load")
		}
	}
}

func (m *Monitor) checkInstance(inst collector.Instance, now int64) {
	m.mu.Lock()
	st, ok := m.states[inst.Service]
	if !ok {
		st = &InstanceState{
			Service:   inst.Service,
			Port:      inst.Port,
			LastState: inst.State,
		}
		m.states[inst.Service] = st
	}
	m.mu.Unlock()

	isUnhealthy := false
	reason := ""

	// Check service state
	if inst.State == "inactive" || inst.State == "failed" {
		isUnhealthy = true
		reason = fmt.Sprintf("服务状态: %s", inst.State)
	} else if inst.UDP == "not-listening" {
		isUnhealthy = true
		reason = fmt.Sprintf("UDP 端口 %d 未监听", inst.Port)
	}

	ts := time.Now().Format("2006-01-02 15:04:05")

	if isUnhealthy {
		st.FailureCount++
		log.Printf("[WARN] %s 异常 (%s), 连续 %d/%d 次",
			inst.Service, reason, st.FailureCount, m.failureThreshold)

		// First failure: send alert
		if st.FailureCount == 1 {
			m.notifier.Push(notifier.Alert{
				Time:    ts,
				Type:    notifier.SeverityWarning,
				Title:   fmt.Sprintf("服务异常 - %s", inst.Service),
				Message: fmt.Sprintf("%s\n已开始计数，连续 %d 次后自动重启", reason, m.failureThreshold),
			})
		}

		// Reached threshold -> attempt restart
		if st.FailureCount >= m.failureThreshold {
			canRestart := (now - st.LastRestartTime) >= int64(m.restartCooldown)
			if canRestart {
				st.LastRestartTime = now
				st.FailureCount = 0
				go m.restartService(inst)
			} else {
				remaining := m.restartCooldown - int(now-st.LastRestartTime)
				log.Printf("[INFO] %s 在重启冷却中，剩余 %d 秒", inst.Service, remaining)
			}
		}
	} else {
		// Healthy
		if st.FailureCount > 0 {
			// Recovered
			log.Printf("[INFO] %s 已恢复正常", inst.Service)
			m.notifier.Push(notifier.Alert{
				Time:    ts,
				Type:    notifier.SeverityRecovery,
				Title:   fmt.Sprintf("服务恢复 - %s", inst.Service),
				Message: fmt.Sprintf("服务已恢复正常运行"),
			})
		}
		st.FailureCount = 0
	}

	st.LastState = inst.State
}

func (m *Monitor) restartService(inst collector.Instance) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[ACTION] 正在重启 %s ...", inst.Service)

	m.notifier.Push(notifier.Alert{
		Time:    ts,
		Type:    notifier.SeverityCritical,
		Title:   fmt.Sprintf("自动重启 - %s", inst.Service),
		Message: fmt.Sprintf("连续异常达到阈值，正在执行 systemctl restart"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "restart", inst.Service)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[ERROR] 重启 %s 失败: %v, output: %s", inst.Service, err, string(out))
		m.notifier.Push(notifier.Alert{
			Time:    time.Now().Format("2006-01-02 15:04:05"),
			Type:    notifier.SeverityCritical,
			Title:   fmt.Sprintf("重启失败 - %s", inst.Service),
			Message: fmt.Sprintf("systemctl restart 失败: %v\n%s", err, string(out)),
		})
		return
	}

	// Wait for UDP port to come back
	waited := 0
	for waited < m.socketWaitTimeout {
		time.Sleep(2 * time.Second)
		waited += 2
		if checkUDPFast(inst.Port) {
			log.Printf("[INFO] %s 重启成功，UDP 端口已恢复 (等待 %d 秒)", inst.Service, waited)
			m.notifier.Push(notifier.Alert{
				Time:    time.Now().Format("2006-01-02 15:04:05"),
				Type:    notifier.SeverityRecovery,
				Title:   fmt.Sprintf("重启成功 - %s", inst.Service),
				Message: fmt.Sprintf("UDP 端口 %d 已恢复监听，用时 %d 秒", inst.Port, waited),
			})
			return
		}
	}

	log.Printf("[WARN] %s 重启后 UDP 端口在 %d 秒内未恢复", inst.Service, m.socketWaitTimeout)
	m.notifier.Push(notifier.Alert{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Type:    notifier.SeverityCritical,
		Title:   fmt.Sprintf("重启后端口未恢复 - %s", inst.Service),
		Message: fmt.Sprintf("重启执行成功，但 UDP %d 在 %d 秒内未监听", inst.Port, m.socketWaitTimeout),
	})
}

func checkUDPFast(port int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ss", "-H", "-lun").Output()
	if err != nil {
		return false
	}
	return collector.UDPPortListening(string(out), port)
}
