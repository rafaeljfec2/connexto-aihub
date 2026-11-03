package gateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aihub/internal/execution"
	"aihub/internal/gateway"
	"aihub/internal/providers"
	"aihub/internal/resources"
)

type stubResourceAdmitter struct {
	allow   bool
	fail    error
	calls   atomic.Int32
	lastReq resources.AdmissionRequest
}

func (s *stubResourceAdmitter) Admit(_ context.Context, request resources.AdmissionRequest) (resources.AdmissionDecision, error) {
	s.calls.Add(1)
	s.lastReq = request
	if s.fail != nil {
		return resources.AdmissionDecision{}, s.fail
	}
	if !s.allow {
		reset := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
		return resources.AdmissionDecision{
			Allowed: false, Reason: resources.RejectionReasonLimitExceeded,
			ResetAt: reset,
		}, resources.ErrLimitExceeded
	}
	return resources.AdmissionDecision{Allowed: true, AdmissionID: request.ID}, nil
}

func TestResponsesResourceAdmissionAllowsExecutionIntegration(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\"}\n\n"))
	}))
	t.Cleanup(upstream.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var openerCalls atomic.Int32
	manager := execution.NewMultiProviderManager(logger, func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
		openerCalls.Add(1)
		return http.Get(upstream.URL)
	}, time.Minute)

	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, newEnabledClient(t, upstream.URL))
	admitter := &stubResourceAdmitter{allow: true}
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, admitter)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(`{"model":"m1","input":"hi"}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if admitter.calls.Load() != 1 {
		t.Fatalf("admitter calls = %d, want 1", admitter.calls.Load())
	}
	if openerCalls.Load() != 1 {
		t.Fatalf("opener calls = %d, want 1 after admission", openerCalls.Load())
	}
}

func TestResponsesResourceAdmissionRejectsBeforeExecutionIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var openerCalls atomic.Int32
	manager := execution.NewMultiProviderManager(logger, func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
		openerCalls.Add(1)
		return nil, nil
	}, time.Minute)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, newEnabledClient(t, "http://127.0.0.1:1"))
	admitter := &stubResourceAdmitter{allow: false}
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, admitter)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(`{"model":"m1","input":"hi"}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if openerCalls.Load() != 0 {
		t.Fatal("provider opener must not run when admission rejects")
	}
	var body struct {
		Error struct {
			Type    string `json:"type"`
			ResetAt string `json:"reset_at"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Type != "resource_limit_exceeded" || body.Error.ResetAt == "" {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestResponsesResourceAdmissionUnavailableBeforeExecutionIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var openerCalls atomic.Int32
	manager := execution.NewMultiProviderManager(logger, func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
		openerCalls.Add(1)
		return nil, nil
	}, time.Minute)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, newEnabledClient(t, "http://127.0.0.1:1"))
	admitter := &stubResourceAdmitter{fail: resources.ErrUnavailable}
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, admitter)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(`{"model":"m1","input":"hi"}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if openerCalls.Load() != 0 {
		t.Fatal("provider opener must not run when admission is unavailable")
	}
}

func TestResponsesResourceAdmissionInvalidInputReturnsBadRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager := execution.NewMultiProviderManager(logger, func(context.Context, execution.Target, []byte, string, string) (*http.Response, error) {
		return nil, nil
	}, time.Minute)
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, newEnabledClient(t, "http://127.0.0.1:1"))
	admitter := &stubResourceAdmitter{fail: resources.ErrInvalidInput}
	handler := gateway.NewResponsesHandlerWithRegistry(registry, manager, logger, time.Minute, nil, admitter)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(`{"model":"m1","input":"hi"}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid_request") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
