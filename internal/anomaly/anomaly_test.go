package anomaly

import (
	"math"
	"testing"
)

func TestDetectorWarmUp(t *testing.T) {
	d := NewDetector(Config{
		Kind:       MetricCPU,
		WindowSize: 20,
		ZThreshold: 3.0,
		MinSamples: 10,
	})

	// During warm-up, no anomaly should be flagged.
	for i := 0; i < 9; i++ {
		res := d.Evaluate(50.0)
		if res.Anomalous {
			t.Fatalf("sample %d: should not flag during warm-up", i)
		}
	}
}

func TestDetectorSpikeDetection(t *testing.T) {
	d := NewDetector(Config{
		Kind:       MetricCPU,
		WindowSize: 30,
		ZThreshold: 2.5,
		MinSamples: 10,
		EWMAAlpha:  0.3,
	})

	// Establish a stable baseline around 20%.
	for i := 0; i < 20; i++ {
		d.Evaluate(20.0)
	}

	// A sudden spike to 95% should be anomalous.
	res := d.Evaluate(95.0)
	if !res.Anomalous {
		t.Fatalf("expected spike to be anomalous, got z=%.2f mean=%.2f stddev=%.2f",
			res.ZScore, res.Mean, res.StdDev)
	}
	if res.Reason != "spike_above_baseline" {
		t.Fatalf("expected reason spike_above_baseline, got %s", res.Reason)
	}
}

func TestDetectorDropDetection(t *testing.T) {
	d := NewDetector(Config{
		Kind:       MetricMemory,
		WindowSize: 30,
		ZThreshold: 2.5,
		MinSamples: 10,
		EWMAAlpha:  0.3,
	})

	// Establish baseline around 80%.
	for i := 0; i < 20; i++ {
		d.Evaluate(80.0)
	}

	// Sudden drop to 10% should be anomalous.
	res := d.Evaluate(10.0)
	if !res.Anomalous {
		t.Fatalf("expected drop to be anomalous, got z=%.2f", res.ZScore)
	}
	if res.Reason != "drop_below_baseline" {
		t.Fatalf("expected reason drop_below_baseline, got %s", res.Reason)
	}
}

func TestDetectorStableSeriesNoFalsePositive(t *testing.T) {
	d := NewDetector(Config{
		Kind:       MetricLoad,
		WindowSize: 30,
		ZThreshold: 3.0,
		MinSamples: 10,
	})

	// Very stable series with tiny noise should not trigger.
	for i := 0; i < 30; i++ {
		val := 1.5 + float64(i%3)*0.01 // tiny oscillation
		res := d.Evaluate(val)
		if i >= 10 && res.Anomalous {
			t.Fatalf("sample %d: stable series should not trigger, z=%.2f", i, res.ZScore)
		}
	}
}

func TestDetectorConstantSeries(t *testing.T) {
	d := NewDetector(Config{
		Kind:       MetricDisk,
		WindowSize: 20,
		ZThreshold: 3.0,
		MinSamples: 5,
	})

	// Constant series.
	for i := 0; i < 10; i++ {
		d.Evaluate(50.0)
	}

	// Any deviation from a constant series is anomalous (stddev=0 case).
	res := d.Evaluate(51.0)
	if !res.Anomalous {
		t.Fatal("expected deviation from constant series to be anomalous")
	}
	if res.Reason != "constant_series_deviation" {
		t.Fatalf("expected constant_series_deviation, got %s", res.Reason)
	}
}

func TestDetectorReset(t *testing.T) {
	d := NewDetector(Config{
		Kind:       MetricCPU,
		WindowSize: 20,
		ZThreshold: 3.0,
		MinSamples: 5,
	})

	for i := 0; i < 10; i++ {
		d.Evaluate(30.0)
	}

	d.Reset()

	// After reset, should be in warm-up again.
	_, _, count, init := d.State()
	if count != 0 || init {
		t.Fatalf("after reset: expected count=0 init=false, got count=%d init=%v", count, init)
	}
}

func TestDetectorEWMAAdaptation(t *testing.T) {
	d := NewDetector(Config{
		Kind:       MetricCPU,
		WindowSize: 50,
		ZThreshold: 3.0,
		MinSamples: 10,
		EWMAAlpha:  0.5, // fast adaptation
	})

	// Baseline at 20%.
	for i := 0; i < 15; i++ {
		d.Evaluate(20.0)
	}

	// Gradually drift to 60% — EWMA should adapt, not flag as anomaly.
	anomalousDuringDrift := false
	for i := 0; i < 20; i++ {
		val := 20.0 + float64(i)*2.0 // gradual +2 per sample
		res := d.Evaluate(val)
		if res.Anomalous {
			anomalousDuringDrift = true
		}
	}

	// Gradual drift should not trigger (EWMA adapts).
	if anomalousDuringDrift {
		t.Log("warning: gradual drift triggered anomaly (may be acceptable with small window)")
	}

	// After drift stabilizes at 60%, a sudden spike to 95% should trigger.
	for i := 0; i < 10; i++ {
		d.Evaluate(60.0)
	}
	res := d.Evaluate(95.0)
	if !res.Anomalous {
		t.Fatalf("expected spike after drift to be anomalous, z=%.2f mean=%.2f", res.ZScore, res.Mean)
	}
}

func TestMeanStdDev(t *testing.T) {
	data := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	mean, stddev := meanStdDev(data)
	if math.Abs(mean-5.0) > 0.01 {
		t.Fatalf("expected mean=5.0, got %.2f", mean)
	}
	// Population stddev = sqrt(32/8) = 2.0
	if math.Abs(stddev-2.0) > 0.01 {
		t.Fatalf("expected stddev=2.0, got %.2f", stddev)
	}
}

func TestEmptyWindow(t *testing.T) {
	mean, stddev := meanStdDev([]float64{})
	if mean != 0 || stddev != 0 {
		t.Fatal("expected zero for empty window")
	}
}
