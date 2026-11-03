package gateway

import (
	"testing"

	"aihub/internal/health"
)

func TestReadinessReportChanged(t *testing.T) {
	t.Parallel()

	previous := health.Report{Status: health.StatusUp, Checks: map[string]string{"database": "ok"}}
	current := health.Report{Status: health.StatusDown, Checks: map[string]string{"database": "error"}}

	if !readinessReportChanged(previous, current) {
		t.Fatal("expected readiness change when database check fails")
	}

	stable := health.Report{Status: health.StatusUp, Checks: map[string]string{"database": "ok"}}
	if readinessReportChanged(previous, stable) {
		t.Fatal("expected no change for identical readiness")
	}
}
