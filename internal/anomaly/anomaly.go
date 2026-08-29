// Package anomaly implements adaptive threshold anomaly detection for
// server metrics. It combines two well-established techniques from the
// AIOps literature:
//
//  1. Sliding-window Z-score (3-sigma rule): flags a sample as anomalous
//     when it deviates more than Z standard deviations from the rolling mean.
//     Reference: "Detecting Anomalies in Systems for AI Using Hardware
//     Telemetry" (arXiv:2510.26008) — flags windows exceeding the 99th
//     percentile of mean absolute Z-score.
//
//  2. EWMA (Exponentially Weighted Moving Average): assigns exponentially
//     decaying weights to older observations, allowing the baseline to adapt
//     to slow drift (e.g. gradual memory leak, diurnal load patterns) while
//     remaining sensitive to sudden spikes.
//     Reference: "A Survey of Time Series Anomaly Detection Methods in the
//     AIOps Domain" (arXiv:2308.00393); "Real-Time Outlier Detection in
//     Fast-Moving Data Streams" — EWMA outperforms simple MA in dynamic
//     environments for rapid anomaly identification.
//
// The detector is safe for concurrent use.
package anomaly

import (
	"math"
	"sync"
)

// MetricKind identifies which system metric a detector tracks.
type MetricKind string

const (
	MetricCPU     MetricKind = "cpu"
	MetricMemory  MetricKind = "memory"
	MetricDisk    MetricKind = "disk"
	MetricLoad    MetricKind = "load"
	MetricProcess MetricKind = "process"
)

// Result is the outcome of evaluating one sample against a detector.
type Result struct {
	Kind      MetricKind `json:"kind"`
	Value     float64    `json:"value"`
	Mean      float64    `json:"mean"`       // EWMA-smoothed baseline
	StdDev    float64    `json:"stddev"`     // rolling window standard deviation
	ZScore    float64    `json:"z_score"`    // |value - mean| / stddev
	Anomalous bool       `json:"anomalous"`  // true if z_score > threshold
	Reason    string     `json:"reason,omitempty"`
}

// Detector tracks one metric series and evaluates incoming samples.
type Detector struct {
	mu sync.Mutex

	kind   MetricKind
	window []float64 // sliding window of raw samples
	maxLen int

	// EWMA state
	ewma   float64 // current EWMA value
	alpha  float64 // smoothing factor (0 < alpha <= 1)
	init   bool    // whether EWMA has been initialised

	// Z-score threshold (default 3.0 = 3-sigma)
	zThreshold float64

	// MinSamples is the minimum number of samples required before anomaly
	// detection is active. Prevents false positives during warm-up.
	minSamples int
}

// Config holds detector parameters.
type Config struct {
	Kind        MetricKind
	WindowSize  int     // number of samples in the sliding window (default 60)
	ZThreshold  float64 // Z-score threshold (default 3.0)
	EWMAAlpha   float64 // EWMA smoothing factor (default 0.2)
	MinSamples  int     // minimum samples before detection (default 10)
}

// NewDetector creates a new anomaly detector with the given configuration.
// Zero values are replaced with sensible defaults.
func NewDetector(cfg Config) *Detector {
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 60
	}
	if cfg.ZThreshold <= 0 {
		cfg.ZThreshold = 3.0
	}
	if cfg.EWMAAlpha <= 0 || cfg.EWMAAlpha > 1 {
		cfg.EWMAAlpha = 0.2
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = 10
	}
	return &Detector{
		kind:        cfg.Kind,
		window:      make([]float64, 0, cfg.WindowSize),
		maxLen:      cfg.WindowSize,
		alpha:       cfg.EWMAAlpha,
		zThreshold:  cfg.ZThreshold,
		minSamples:  cfg.MinSamples,
	}
}

// Evaluate ingests one sample and returns an anomaly detection result.
// The sample is added to the sliding window and the EWMA baseline is
// updated before the Z-score is computed.
func (d *Detector) Evaluate(value float64) Result {
	d.mu.Lock()
	defer d.mu.Unlock()

	res := Result{
		Kind:  d.kind,
		Value: value,
	}

	// Initialise EWMA on first sample.
	if !d.init {
		d.ewma = value
		d.init = true
	}

	// Update EWMA: new_ewma = alpha * value + (1 - alpha) * old_ewma
	d.ewma = d.alpha*value + (1-d.alpha)*d.ewma
	res.Mean = round(d.ewma, 2)

	// Add to sliding window.
	d.window = append(d.window, value)
	if len(d.window) > d.maxLen {
		d.window = d.window[1:]
	}

	// Not enough samples yet — return baseline info without flagging.
	if len(d.window) < d.minSamples {
		res.Reason = "warming_up"
		return res
	}

	// Compute rolling mean and population standard deviation.
	mean, stddev := meanStdDev(d.window)
	res.StdDev = round(stddev, 2)

	// If stddev is zero (constant series), any deviation is anomalous,
	// but we avoid division by zero by treating it as a small epsilon.
	if stddev < 1e-9 {
		if math.Abs(value-mean) > 1e-9 {
			res.Anomalous = true
			res.ZScore = math.Inf(1)
			res.Reason = "constant_series_deviation"
		}
		return res
	}

	z := math.Abs(value-mean) / stddev
	res.ZScore = round(z, 2)

	if z > d.zThreshold {
		res.Anomalous = true
		if value > mean {
			res.Reason = "spike_above_baseline"
		} else {
			res.Reason = "drop_below_baseline"
		}
	}

	return res
}

// Reset clears all state, returning the detector to its initial condition.
// Useful when a known maintenance event has occurred and historical baseline
// is no longer representative.
func (d *Detector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.window = d.window[:0]
	d.ewma = 0
	d.init = false
}

// State returns a snapshot of the detector's internal state for debugging.
func (d *Detector) State() (mean, stddev float64, sampleCount int, initialized bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.window) == 0 {
		return 0, 0, 0, d.init
	}
	m, s := meanStdDev(d.window)
	return m, s, len(d.window), d.init
}

// --- helpers ---

func meanStdDev(data []float64) (mean, stddev float64) {
	if len(data) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean = sum / float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	// Population standard deviation (we observe the full window, not a sample).
	stddev = math.Sqrt(variance / float64(len(data)))
	return mean, stddev
}

func round(v float64, decimals int) float64 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return v
	}
	mult := math.Pow10(decimals)
	return math.Round(v*mult) / mult
}
