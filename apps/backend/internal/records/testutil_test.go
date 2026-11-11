package records

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func at(second int) time.Time {
	return time.Date(2026, 10, 31, 12, 0, second, 123456789, time.UTC)
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func executionID(n int) string {
	return fmt.Sprintf("%032x", n)
}

func completedRecord(id string, completedAt time.Time) Record {
	duration := 1500 * time.Millisecond
	requestID := "req-0123456789abcdef"

	return Record{
		SchemaVersion:     SchemaVersion,
		ExecutionID:       id,
		RequestID:         &requestID,
		ProviderID:        "opencode-go",
		StartedAt:         completedAt.Add(-duration),
		CompletedAt:       completedAt,
		DurationNS:        int64(duration),
		Outcome:           OutcomeCompleted,
		Phase:             PhaseForwarding,
		ResponseCommitted: true,
		UpstreamStatus:    intPtr(200),
	}
}

func failedRecord(id string, completedAt time.Time, cause string) Record {
	record := completedRecord(id, completedAt)
	record.Outcome = OutcomeFailed
	record.Cause = strPtr(cause)
	record.UpstreamStatus = nil

	return record
}

func eventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("condition was not satisfied before the timeout")
}

func storedLineCount(t *testing.T, dir string) int {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read records directory: %v", err)
	}

	total := 0
	for _, entry := range entries {
		if _, ok := parseSegmentName(entry.Name()); !ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read segment %s: %v", entry.Name(), err)
		}
		total += len(splitLines(data))
	}

	return total
}

func storedSegmentBytes(t *testing.T, dir string) int64 {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read records directory: %v", err)
	}

	total := int64(0)
	for _, entry := range entries {
		if _, ok := parseSegmentName(entry.Name()); !ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat segment %s: %v", entry.Name(), err)
		}
		total += info.Size()
	}

	return total
}

func splitLines(data []byte) []string {
	trimmed := strings.TrimSuffix(string(data), "\n")
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "\n")
}
