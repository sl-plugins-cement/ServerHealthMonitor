package diagnostics

import (
	"context"
	"testing"
)

func TestCollectAndBuildPlan(t *testing.T) {
	report, err := Collect(context.Background())
	if err != nil {
		t.Fatalf("collect report: %v", err)
	}
	if report.ID == "" || report.Environment.OS == "" || report.Environment.Architecture == "" {
		t.Fatalf("incomplete environment report: %+v", report)
	}
	if report.RiskScore < 0 || report.RiskScore > 100 {
		t.Fatalf("risk score out of range: %d", report.RiskScore)
	}
	plan := BuildPlan(Report{Findings: []Finding{{ID: "x", Title: "test", Risk: RiskHigh}}})
	if len(plan) != 1 || !plan[0].RequiresConfirm || plan[0].Executable {
		t.Fatalf("high-risk plan must require confirmation and remain preview-only: %+v", plan)
	}
}
