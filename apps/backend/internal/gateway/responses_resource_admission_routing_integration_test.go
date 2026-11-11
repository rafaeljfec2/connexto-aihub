package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/providers"
	"aihub/internal/resources"
	"aihub/internal/resources/pgstore"
)

func TestRoutedResponsesResourceAdmissionWithPostgresIntegration(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	ctx := context.Background()
	if _, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope: resources.ResourceScope{Kind: resources.ScopeGlobal}, DailyLimit: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	var openerCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"ok"}`)
	}))
	t.Cleanup(upstream.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := newOpenCodeRoutingHandlerWithResources(t, upstream.URL, logger, time.Minute, service, &openerCalls, "routed-model")

	body := `{"input":[{"type":"input_text","text":"hello"}]}`
	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	firstReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(first, firstReq)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d body = %s", first.Code, first.Body.String())
	}
	if openerCalls.Load() != 1 {
		t.Fatalf("opener calls after allow = %d, want 1", openerCalls.Load())
	}

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	secondReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(second, secondReq)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", second.Code)
	}
	if openerCalls.Load() != 1 {
		t.Fatalf("opener calls after reject = %d, want still 1", openerCalls.Load())
	}
	if !strings.Contains(second.Body.String(), "resource_limit_exceeded") {
		t.Fatalf("second body = %s", second.Body.String())
	}
}

func TestRoutedResponsesPassesSelectedProviderModelToResourceAdmissionIntegration(t *testing.T) {
	pool := dbtest.Pool(t)
	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	service := resources.NewService(pgstore.New(pool), resources.WithClock(func() time.Time { return now }))
	ctx := context.Background()
	if _, err := service.CreatePolicy(ctx, resources.NewPolicyInput{
		Scope:      resources.ResourceScope{Kind: resources.ScopeModel, ProviderID: providers.OpenCodeGoID, ModelID: "routed-model"},
		DailyLimit: 5,
		Enabled:    true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := newOpenCodeRoutingHandlerWithResources(t, upstream.URL, logger, time.Minute, service, nil, "routed-model")

	body := `{"input":[{"type":"input_text","text":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	readCtx := context.Background()
	var modelID, providerID string
	err := pool.QueryRow(readCtx, `SELECT provider_id, model_id FROM resource_admissions ORDER BY created_at DESC LIMIT 1`).Scan(&providerID, &modelID)
	if err != nil {
		t.Fatalf("read admission row: %v", err)
	}
	if providerID != providers.OpenCodeGoID || modelID != "routed-model" {
		t.Fatalf("persisted target = %q/%q, want %q/routed-model", providerID, modelID, providers.OpenCodeGoID)
	}
}
