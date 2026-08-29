// Package alertmanager implements production-grade alert management
// inspired by Prometheus Alertmanager and academic research on alert
// correlation.
//
// Key features:
//
//  1. Grouping: alerts with matching labels within a time window are
//     batched into a single notification, reducing alert fatigue.
//     Reference: Prometheus Alertmanager — group_by clusters related
//     alerts so one page covers all affected instances.
//
//  2. Inhibition: when a higher-severity "source" alert is active,
//     matching lower-severity "target" alerts are suppressed. For
//     example, a host-down alert inhibits all service-down alerts on
//     that host.
//     Reference: "Alarm reduction and root cause inference based on
//     association mining" (Frontiers in Computer Science, 2023) —
//     association-based reduction removes 62% of redundant alarms.
//
//  3. Escalation: critical alerts that remain unacknowledged past a
//     timeout are escalated (e.g. from webhook to email to pager).
//
//  4. Silences: planned maintenance windows suppress matching alerts,
//     with audit logging of who silenced what and why.
//
//  5. Deduplication: alerts are fingerprinted by labels + name;
//     identical active alerts are not re-dispatched.
//
// The manager is safe for concurrent use.
package alertmanager

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Severity mirrors notifier.Severity but is defined here to avoid an
// import cycle (notifier does not import alertmanager).
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityRecovery Severity = "recovery"
)

// Alert is an enriched alert event with labels for grouping/inhibition.
type Alert struct {
	// Fingerprint is the stable deduplication key, derived from Name + Labels.
	Fingerprint string `json:"fingerprint"`

	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Severity  Severity          `json:"severity"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Time      time.Time         `json:"time"`
	StartsAt  time.Time         `json:"starts_at"`
	EndsAt    time.Time         `json:"ends_at,omitempty"` // zero = still firing
	Status    AlertStatus       `json:"status"`
	AckedBy   string            `json:"acked_by,omitempty"`
	AckedAt   time.Time         `json:"acked_at,omitempty"`
	Escalated bool              `json:"escalated"`
	Silenced  bool              `json:"silenced"`
	Inhibited bool              `json:"inhibited"`
}

// AlertStatus represents the lifecycle state of an alert.
type AlertStatus string

const (
	StatusActive   AlertStatus = "active"
	StatusAcked    AlertStatus = "acknowledged"
	StatusResolved AlertStatus = "resolved"
	StatusSilenced AlertStatus = "silenced"
)

// InhibitRule defines a suppression rule: when a source alert matching
// SourceMatchers is active, target alerts matching TargetMatchers are
// inhibited. EqualLabels must match between source and target (e.g.
// ["instance"] ensures only alerts on the same host are inhibited).
type InhibitRule struct {
	Name          string            `json:"name"`
	SourceMatch   map[string]string `json:"source_match"`
	TargetMatch   map[string]string `json:"target_match"`
	EqualLabels   []string          `json:"equal_labels"`
}

// Silence is a time-bounded suppression of alerts matching Matchers.
type Silence struct {
	ID        string            `json:"id"`
	Matchers  map[string]string `json:"matchers"`
	StartsAt  time.Time         `json:"starts_at"`
	EndsAt    time.Time         `json:"ends_at"`
	CreatedBy string            `json:"created_by"`
	Comment   string            `json:"comment"`
}

// Group is a batch of related alerts dispatched together.
type Group struct {
	Key       string  `json:"key"`
	Alerts    []Alert `json:"alerts"`
	FirstSeen time.Time `json:"first_seen"`
	Dispatched bool    `json:"dispatched"`
}

// DispatchFunc is called when a group of alerts should be delivered.
// Implementations send to webhook, email, pager, etc.
type DispatchFunc func(group *Group)

// Config holds alertmanager parameters.
type Config struct {
	// GroupWindow is how long to wait after the first alert in a group
	// before dispatching, allowing related alerts to batch together.
	GroupWindow time.Duration

	// GroupLabels defines which label keys form a group key. Alerts
	// sharing the same values for these labels are grouped together.
	GroupLabels []string

	// EscalateAfter is how long a critical alert can remain
	// unacknowledged before being marked escalated.
	EscalateAfter time.Duration

	// MaxAlerts is the maximum number of active alerts to retain.
	MaxAlerts int
}

// Manager is the central alert management engine.
type Manager struct {
	mu sync.RWMutex

	cfg Config

	alerts    map[string]*Alert // fingerprint -> alert
	inhibits  []InhibitRule
	silences  []Silence
	groups    map[string]*Group // group key -> group

	dispatch DispatchFunc

	// stop channel for background goroutines
	stop chan struct{}
}

// NewManager creates a new alert manager with the given config and
// dispatch function. Zero config values are replaced with defaults.
func NewManager(cfg Config, dispatch DispatchFunc) *Manager {
	if cfg.GroupWindow <= 0 {
		cfg.GroupWindow = 60 * time.Second
	}
	if len(cfg.GroupLabels) == 0 {
		cfg.GroupLabels = []string{"alertname", "severity"}
	}
	if cfg.EscalateAfter <= 0 {
		cfg.EscalateAfter = 15 * time.Minute
	}
	if cfg.MaxAlerts <= 0 {
		cfg.MaxAlerts = 500
	}

	m := &Manager{
		cfg:      cfg,
		alerts:   make(map[string]*Alert),
		groups:   make(map[string]*Group),
		dispatch: dispatch,
		stop:     make(chan struct{}),
	}

	go m.run()
	return m
}

// Stop shuts down the background processing goroutine.
func (m *Manager) Stop() {
	close(m.stop)
}

// AddInhibitRule registers an inhibition rule.
func (m *Manager) AddInhibitRule(rule InhibitRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inhibits = append(m.inhibits, rule)
}

// AddSilence registers a silence window. Returns the silence ID.
func (m *Manager) AddSilence(s Silence) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.ID == "" {
		s.ID = fmt.Sprintf("sil-%d", time.Now().UnixNano())
	}
	m.silences = append(m.silences, s)
	return s.ID
}

// RemoveSilence deletes a silence by ID.
func (m *Manager) RemoveSilence(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, s := range m.silences {
		if s.ID == id {
			m.silences = append(m.silences[:i], m.silences[i+1:]...)
			return true
		}
	}
	return false
}

// ListSilences returns all active (non-expired) silences.
func (m *Manager) ListSilences() []Silence {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now()
	result := make([]Silence, 0, len(m.silences))
	for _, s := range m.silences {
		if now.Before(s.EndsAt) {
			result = append(result, s)
		}
	}
	return result
}

// Process ingests an alert. It applies deduplication, silencing,
// inhibition, grouping, and may trigger a dispatch.
// Returns the processed alert (with Fingerprint, Status, etc. populated).
func (m *Manager) Process(a Alert) *Alert {
	m.mu.Lock()
	defer m.mu.Unlock()

	if a.Fingerprint == "" {
		a.Fingerprint = fingerprint(a.Name, a.Labels)
	}
	if a.Time.IsZero() {
		a.Time = time.Now()
	}
	if a.StartsAt.IsZero() {
		a.StartsAt = a.Time
	}
	a.Status = StatusActive

	now := time.Now()

	// 1. Check silences.
	for _, s := range m.silences {
		if now.After(s.StartsAt) && now.Before(s.EndsAt) && matchLabels(a.Labels, s.Matchers) {
			a.Status = StatusSilenced
			a.Silenced = true
			// Still record the alert so it shows in the UI as silenced.
			m.alerts[a.Fingerprint] = &a
			m.prune()
			return &a
		}
	}

	// 2. Check inhibition: is there an active source alert that should
	//    inhibit this one?
	for _, rule := range m.inhibits {
		if matchLabels(a.Labels, rule.TargetMatch) {
			// Look for an active source alert.
			for _, src := range m.alerts {
				if src.Status != StatusActive && src.Status != StatusAcked {
					continue
				}
				if !matchLabels(src.Labels, rule.SourceMatch) {
					continue
				}
				// Check equal labels match between source and target.
				if equalLabelsMatch(src.Labels, a.Labels, rule.EqualLabels) {
					a.Inhibited = true
					a.Status = StatusActive // still active but won't dispatch
					m.alerts[a.Fingerprint] = &a
					m.prune()
					return &a
				}
			}
		}
	}

	// 3. Deduplication: if already active and not resolved, update but
	//    don't re-dispatch.
	if existing, ok := m.alerts[a.Fingerprint]; ok {
		if existing.Status == StatusActive || existing.Status == StatusAcked {
			// Update timestamp and message but keep status.
			existing.Time = a.Time
			existing.Message = a.Message
			existing.EndsAt = time.Time{} // still firing
			return existing
		}
	}

	// 4. Store the alert.
	m.alerts[a.Fingerprint] = &a
	m.prune()

	// 5. Add to group for batched dispatch.
	groupKey := m.groupKey(a)
	g, ok := m.groups[groupKey]
	if !ok {
		g = &Group{
			Key:       groupKey,
			FirstSeen: now,
		}
		m.groups[groupKey] = g
	}
	g.Alerts = append(g.Alerts, a)

	return &a
}

// Acknowledge marks an alert as acknowledged, preventing escalation.
func (m *Manager) Acknowledge(fingerprint, ackedBy string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.alerts[fingerprint]
	if !ok {
		return false
	}
	a.Status = StatusAcked
	a.AckedBy = ackedBy
	a.AckedAt = time.Now()
	return true
}

// Resolve marks an alert as resolved (firing stopped).
func (m *Manager) Resolve(fingerprint string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.alerts[fingerprint]
	if !ok {
		return false
	}
	a.Status = StatusResolved
	a.EndsAt = time.Now()
	return true
}

// ListActive returns all currently active (non-resolved) alerts,
// most recent first.
func (m *Manager) ListActive() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Alert, 0, len(m.alerts))
	for _, a := range m.alerts {
		if a.Status != StatusResolved {
			result = append(result, *a)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.After(result[j].Time)
	})
	return result
}

// ListAll returns all alerts (including resolved), most recent first,
// capped at MaxAlerts.
func (m *Manager) ListAll() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Alert, 0, len(m.alerts))
	for _, a := range m.alerts {
		result = append(result, *a)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.After(result[j].Time)
	})
	if len(result) > m.cfg.MaxAlerts {
		result = result[:m.cfg.MaxAlerts]
	}
	return result
}

// --- internal ---

func (m *Manager) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.processTick()
		}
	}
}

func (m *Manager) processTick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Dispatch groups whose window has elapsed.
	for key, g := range m.groups {
		if !g.Dispatched && now.Sub(g.FirstSeen) >= m.cfg.GroupWindow {
			g.Dispatched = true
			if m.dispatch != nil {
				// Copy to avoid holding lock during dispatch.
				gCopy := *g
				go m.dispatch(&gCopy)
			}
			delete(m.groups, key)
		}
	}

	// Escalate unacknowledged critical alerts.
	for _, a := range m.alerts {
		if a.Severity == SeverityCritical &&
			a.Status == StatusActive &&
			!a.Escalated &&
			now.Sub(a.StartsAt) >= m.cfg.EscalateAfter {
			a.Escalated = true
			// Re-dispatch as escalated.
			if m.dispatch != nil {
				escAlert := *a
				escAlert.Title = "[ESCALATED] " + a.Title
				g := &Group{
					Key:       "escalate-" + a.Fingerprint,
					Alerts:    []Alert{escAlert},
					FirstSeen: now,
				}
				go m.dispatch(g)
			}
		}
	}

	// Expire old resolved alerts (keep last 100 resolved).
	resolvedCount := 0
	for fp, a := range m.alerts {
		if a.Status == StatusResolved {
			resolvedCount++
			if resolvedCount > 100 {
				delete(m.alerts, fp)
			}
		}
	}
}

func (m *Manager) prune() {
	if len(m.alerts) <= m.cfg.MaxAlerts {
		return
	}
	// Remove oldest resolved alerts first.
	type entry struct {
		fp string
		t  time.Time
	}
	var resolved []entry
	var active []entry
	for fp, a := range m.alerts {
		if a.Status == StatusResolved {
			resolved = append(resolved, entry{fp, a.Time})
		} else {
			active = append(active, entry{fp, a.Time})
		}
	}
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].t.Before(resolved[j].t)
	})
	overflow := len(m.alerts) - m.cfg.MaxAlerts
	for i := 0; i < overflow && i < len(resolved); i++ {
		delete(m.alerts, resolved[i].fp)
	}
}

func (m *Manager) groupKey(a Alert) string {
	parts := make([]string, 0, len(m.cfg.GroupLabels))
	for _, label := range m.cfg.GroupLabels {
		parts = append(parts, a.Labels[label])
	}
	// Sort for deterministic key.
	sort.Strings(parts)
	key := ""
	for i, p := range parts {
		if i > 0 {
			key += "|"
		}
		key += p
	}
	return key
}

// --- helpers ---

func fingerprint(name string, labels map[string]string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte{'\n'})
	// Sort label keys for deterministic fingerprint.
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{'='})
		h.Write([]byte(labels[k]))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func matchLabels(labels, matchers map[string]string) bool {
	for k, v := range matchers {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func equalLabelsMatch(src, tgt map[string]string, equalLabels []string) bool {
	for _, label := range equalLabels {
		if src[label] != tgt[label] {
			return false
		}
	}
	return true
}
