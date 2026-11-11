package records

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestServiceReportsDropsOutsideCallback(t *testing.T) {
	var logs bytes.Buffer
	service := New(testOptions(t.TempDir()))
	service.logger = slog.New(slog.NewTextHandler(&logs, nil))
	service.now = func() time.Time { return at(0) }

	for n := 0; n < 3; n++ {
		service.counters.record(failureQueueOverflow)
	}
	service.counters.record(failureInvalidRecord)
	service.reportDrops()

	if count := strings.Count(logs.String(), "queue_overflow"); count != 1 {
		t.Fatalf("overflow warnings = %d, want 1: %s", count, logs.String())
	}
	if !strings.Contains(logs.String(), "dropped=3") {
		t.Fatalf("overflow warning did not report the drop count: %s", logs.String())
	}
	if count := strings.Count(logs.String(), "invalid_record"); count != 1 {
		t.Fatalf("invalid-record warnings = %d, want 1: %s", count, logs.String())
	}

	service.counters.record(failureQueueOverflow)
	service.now = func() time.Time { return at(0).Add(10 * time.Second) }
	service.reportDrops()
	if count := strings.Count(logs.String(), "queue_overflow"); count != 1 {
		t.Fatalf("overflow warnings inside the window = %d, want 1", count)
	}

	service.now = func() time.Time { return at(0).Add(dropLogInterval + time.Second) }
	service.reportDrops()
	if count := strings.Count(logs.String(), "queue_overflow"); count != 2 {
		t.Fatalf("overflow warnings after the window = %d, want 2", count)
	}
	if !strings.Contains(logs.String(), "class=queue_overflow dropped=1") {
		t.Fatalf("second overflow warning = %s, want the new drops only", logs.String())
	}
}
