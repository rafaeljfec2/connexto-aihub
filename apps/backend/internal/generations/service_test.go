package generations

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/records"
)

type failingRepository struct{}

func (failingRepository) Insert(context.Context, TelemetryInput) error { return nil }

func (failingRepository) List(context.Context, []string) (map[string]Enrichment, error) {
	return nil, errors.New("database unavailable")
}

func (failingRepository) DeleteExcept(context.Context, []string) error { return nil }

func TestGetMarksDetailDegradedWhenEnrichmentReadFails(t *testing.T) {
	recordsService := records.New(records.Options{
		Dir:               t.TempDir(),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		MaxAge:            time.Hour,
		MaxCount:          10,
		MaxStorageBytes:   1 << 20,
		SegmentBytes:      1 << 16,
		RetentionInterval: time.Hour,
	})
	if err := recordsService.Start(context.Background()); err != nil {
		t.Fatalf("start records service: %v", err)
	}
	t.Cleanup(func() { _ = recordsService.Stop(context.Background()) })

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	finished := execution.FinishedExecution{
		Result: execution.Result{
			ExecutionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Outcome:     execution.OutcomeCompleted,
			Phase:       execution.PhaseForwarding,
			Duration:    time.Millisecond,
		},
		Target:    execution.Target{ProviderID: "opencode-go"},
		StartedAt: startedAt,
	}
	recordsService.OnExecutionFinished(finished)
	waitForRetainedRecord(t, recordsService, finished.Result.ExecutionID)

	service := NewService(recordsService, failingRepository{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, found, degraded, err := service.Get(context.Background(), finished.Result.ExecutionID)
	if err != nil {
		t.Fatalf("get generation: %v", err)
	}
	if !found {
		t.Fatal("generation was not found")
	}
	if !degraded {
		t.Fatal("detail must report degraded enrichment after repository failure")
	}
}

func waitForRetainedRecord(t *testing.T, service *records.Service, executionID string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, found, err := service.Get(executionID); err == nil && found {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("execution record was not retained")
}

type captureRepository struct {
	inserted []TelemetryInput
}

func (r *captureRepository) Insert(_ context.Context, input TelemetryInput) error {
	r.inserted = append(r.inserted, input)
	return nil
}

func (r *captureRepository) List(context.Context, []string) (map[string]Enrichment, error) {
	return map[string]Enrichment{}, nil
}

func (r *captureRepository) DeleteExcept(context.Context, []string) error { return nil }

func TestSafeClientApp(t *testing.T) {
	if got := SafeClientApp("  ai-hub-playground  "); got == nil || *got != "ai-hub-playground" {
		t.Fatalf("SafeClientApp trimmed = %v, want ai-hub-playground", got)
	}
	if got := SafeClientApp("bad\napp"); got != nil {
		t.Fatalf("SafeClientApp control char = %v, want nil", got)
	}
	if got := SafeClientApp("   "); got != nil {
		t.Fatalf("SafeClientApp empty = %v, want nil", got)
	}
	long := strings.Repeat("a", 150)
	if got := SafeClientApp(long); got == nil || len(*got) != 128 {
		t.Fatalf("SafeClientApp long len = %v, want 128", got)
	}
}

func TestRecordBestEffortSanitizesClientApp(t *testing.T) {
	repo := &captureRepository{}
	service := NewService(nil, repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.RecordBestEffort(context.Background(), TelemetryInput{
		ExecutionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ProviderID:  "cursor",
		ClientApp:   "  codex_cli_rs  ",
	})
	if len(repo.inserted) != 1 || repo.inserted[0].ClientApp != "codex_cli_rs" {
		t.Fatalf("inserted = %+v, want sanitized ClientApp", repo.inserted)
	}
}
