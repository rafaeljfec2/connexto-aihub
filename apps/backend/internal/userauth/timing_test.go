package userauth

import (
	"testing"
	"time"
)

func TestMitigateLoginTimingRunsArgonWork(t *testing.T) {
	MitigateLoginTiming("probe-password")
	hash := dummyHashForTiming()
	if hash == "" {
		t.Fatal("expected dummy hash")
	}

	start := time.Now()
	MitigateLoginTiming("another-probe")
	elapsed := time.Since(start)
	if elapsed < time.Millisecond {
		t.Fatalf("expected measurable argon work, got %v", elapsed)
	}
}
