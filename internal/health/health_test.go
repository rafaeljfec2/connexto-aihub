package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/health"
)

func decodeReport(t *testing.T, body []byte) health.Report {
	t.Helper()

	var report health.Report
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatalf("decode report: %v (body %q)", err, body)
	}

	return report
}

func TestLivenessReturnsUp(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	health.Live().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if got, want := recorder.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}

	if got, want := recorder.Header().Get("Cache-Control"), "no-store"; got != want {
		t.Fatalf("Cache-Control = %q, want %q", got, want)
	}

	report := decodeReport(t, recorder.Body.Bytes())
	if report.Status != health.StatusUp {
		t.Fatalf("status = %q, want %q", report.Status, health.StatusUp)
	}
}

func TestReadinessWithoutChecksIsUp(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	health.NewRegistry().Ready().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	report := decodeReport(t, recorder.Body.Bytes())
	if report.Status != health.StatusUp {
		t.Fatalf("status = %q, want %q", report.Status, health.StatusUp)
	}

	if len(report.Checks) != 0 {
		t.Fatalf("checks = %v, want none", report.Checks)
	}
}

func TestReadinessWithPassingCheckIsUp(t *testing.T) {
	registry := health.NewRegistry()
	if err := registry.Register("self", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	registry.Ready().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	report := decodeReport(t, recorder.Body.Bytes())
	if report.Checks["self"] != "ok" {
		t.Fatalf("checks = %v, want self=ok", report.Checks)
	}
}

func TestReadinessWithFailingCheckIsDown(t *testing.T) {
	registry := health.NewRegistry()
	if err := registry.Register("dependency", func(context.Context) error {
		return errors.New("connection refused at postgres://user:secret@host")
	}); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	registry.Ready().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if got, want := recorder.Code, http.StatusServiceUnavailable; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	report := decodeReport(t, recorder.Body.Bytes())
	if report.Status != health.StatusDown {
		t.Fatalf("status = %q, want %q", report.Status, health.StatusDown)
	}

	if report.Checks["dependency"] != "error" {
		t.Fatalf("checks = %v, want dependency=error", report.Checks)
	}

	body := recorder.Body.String()
	if strings.Contains(body, "secret") || strings.Contains(body, "postgres://") {
		t.Fatalf("readiness response leaked check error details: %q", body)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	registry := health.NewRegistry()

	if err := registry.Register("", func(context.Context) error { return nil }); err == nil {
		t.Fatal("Register() must reject an empty name")
	}

	if err := registry.Register("check", nil); err == nil {
		t.Fatal("Register() must reject a nil check")
	}
}
