package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aihub/internal/records"
)

func TestTrafficSummaryHandler_EmptyHistory(t *testing.T) {
	handler := NewTrafficSummaryHandler(nil, func() time.Time {
		return time.Date(2026, 11, 3, 20, 0, 0, 0, time.UTC)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/telemetry/traffic-summary", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res TrafficSummaryDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if res.TotalRequests != 0 {
		t.Errorf("expected total_requests 0, got %d", res.TotalRequests)
	}
	if res.HasSufficientHistory {
		t.Errorf("expected has_sufficient_history false, got true")
	}
	if len(res.Buckets) != 0 {
		t.Errorf("expected 0 buckets, got %d", len(res.Buckets))
	}
}

func TestTrafficSummaryHandler_RejectsUnsupportedRange(t *testing.T) {
	handler := NewTrafficSummaryHandler(nil, time.Now)

	req := httptest.NewRequest(http.MethodGet, "/api/telemetry/traffic-summary?range=7d", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestTrafficSummaryHandler_WithRecords(t *testing.T) {
	now := time.Date(2026, 11, 3, 20, 0, 0, 0, time.UTC)
	recService := records.New(records.Options{
		Dir: t.TempDir(),
	})
	if err := recService.Start(t.Context()); err != nil {
		t.Fatalf("failed to start records: %v", err)
	}
	t.Cleanup(func() { _ = recService.Stop(t.Context()) })

	handler := NewTrafficSummaryHandler(recService, func() time.Time {
		return now
	})

	req := httptest.NewRequest(http.MethodGet, "/api/telemetry/traffic-summary", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res TrafficSummaryDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if res.Range != "24h" {
		t.Errorf("expected range '24h', got '%s'", res.Range)
	}
}

