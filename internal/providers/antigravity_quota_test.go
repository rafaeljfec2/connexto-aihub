package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAntigravityQuotaAdapter_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/providers/antigravity/usage" {
			t.Errorf("expected path /v1/providers/antigravity/usage, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Gateway-API-Key") != "secret-gateway-key" {
			t.Errorf("missing or invalid gateway key header: %s", r.Header.Get("X-Gateway-API-Key"))
		}

		frac1 := 0.85
		fracZero := 0.0
		email := "engineer@connexto.ai"
		tierName := "Google One AI Premium"
		reset1 := "2026-11-06T20:00:00Z"
		resetZero := "2026-11-06T21:00:00Z"

		resp := proxyUsageResponse{
			Provider: "antigravity",
			Status:   "available",
			Source:   "agy",
			Account: &proxyUsageAccount{
				Email:    &email,
				TierName: &tierName,
			},
			Quotas: []proxyUsageMetric{
				{
					ID:                "gemini-2.5-pro",
					Name:              ptrString("Gemini 2.5 Pro"),
					Nature:            "rate_limit",
					ResetAt:           &reset1,
					Scope:             ptrString("model"),
					RemainingFraction: &frac1,
				},
				{
					ID:                "gemini-2.5-flash",
					Name:              ptrString("Gemini 2.5 Flash"),
					Nature:            "rate_limit",
					ResetAt:           &resetZero,
					Scope:             ptrString("model"),
					RemainingFraction: &fracZero,
				},
			},
			FetchedAt: "2026-11-06T19:30:00Z",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewAntigravityQuotaAdapter(server.Client(), nil)
	snapshot, err := adapter.FetchQuota(context.Background(), server.URL+"/v1", "secret-gateway-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusAvailable {
		t.Fatalf("expected status available, got %s", snapshot.Status)
	}
	if snapshot.AccountLabel != "engineer@connexto.ai" {
		t.Errorf("expected account label engineer@connexto.ai, got %s", snapshot.AccountLabel)
	}
	if snapshot.WorkspaceName == nil || *snapshot.WorkspaceName != "Google One AI Premium" {
		t.Errorf("expected workspace Google One AI Premium, got %v", snapshot.WorkspaceName)
	}
	if len(snapshot.Quotas) != 2 {
		t.Fatalf("expected 2 quotas, got %d", len(snapshot.Quotas))
	}

	// Model 1: 0.85
	q1 := snapshot.Quotas[0]
	if q1.Metric != "gemini-2.5-pro" || q1.Label != "Gemini 2.5 Pro" {
		t.Errorf("unexpected metric 1 identity: %+v", q1)
	}
	if q1.RemainingFraction == nil || *q1.RemainingFraction != 0.85 {
		t.Errorf("expected remaining fraction 0.85, got %v", q1.RemainingFraction)
	}
	if q1.RemainingPercent == nil || *q1.RemainingPercent != 85.0 {
		t.Errorf("expected remaining percent 85.0, got %v", q1.RemainingPercent)
	}
	if q1.ConsumedPercent == nil || *q1.ConsumedPercent != 15.0 {
		t.Errorf("expected consumed percent 15.0, got %v", *q1.ConsumedPercent)
	}
	if q1.ResetsAt == nil || *q1.ResetsAt != "2026-11-06T20:00:00Z" {
		t.Errorf("expected reset timestamp 2026-11-06T20:00:00Z, got %v", q1.ResetsAt)
	}
	// Absolute fields must be nil
	if q1.Used != nil || q1.Limit != nil || q1.Remaining != nil {
		t.Errorf("absolute fields must be nil: used=%v, limit=%v, remaining=%v", q1.Used, q1.Limit, q1.Remaining)
	}

	// Model 2: 0.0 (Preserve explicit zero!)
	q2 := snapshot.Quotas[1]
	if q2.Metric != "gemini-2.5-flash" {
		t.Errorf("unexpected metric 2 identity: %+v", q2)
	}
	if q2.RemainingFraction == nil || *q2.RemainingFraction != 0.0 {
		t.Errorf("expected remaining fraction 0.0 preserved, got %v", q2.RemainingFraction)
	}
	if q2.RemainingPercent == nil || *q2.RemainingPercent != 0.0 {
		t.Errorf("expected remaining percent 0.0 preserved, got %v", q2.RemainingPercent)
	}
	if q2.ConsumedPercent == nil || *q2.ConsumedPercent != 100.0 {
		t.Errorf("expected consumed percent 100.0, got %v", q2.ConsumedPercent)
	}
}

func TestAntigravityQuotaAdapter_Unauthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		resp := proxyUsageResponse{
			Provider: "antigravity",
			Status:   "unauthenticated",
			Source:   "agy",
			Error: &proxyUsageError{
				Code:    "unauthenticated",
				Message: "No active local Antigravity session found.",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewAntigravityQuotaAdapter(server.Client(), nil)
	snapshot, err := adapter.FetchQuota(context.Background(), server.URL, "valid-gw-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot.Status != QuotaStatusUnauthorized {
		t.Fatalf("expected status unauthorized for unauthenticated session, got %s", snapshot.Status)
	}
	if snapshot.Error == nil || snapshot.Error.Code != "unauthenticated" {
		t.Fatalf("expected unauthenticated error code, got %+v", snapshot.Error)
	}
	if len(snapshot.Quotas) != 0 {
		t.Errorf("expected 0 quotas on unauthenticated, got %d", len(snapshot.Quotas))
	}
}

func TestAntigravityQuotaAdapter_MissingCredential(t *testing.T) {
	adapter := NewAntigravityQuotaAdapter(nil, nil)
	snapshot, err := adapter.FetchQuota(context.Background(), "http://127.0.0.1:9090", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot.Status != QuotaStatusUnconfiguredCredential {
		t.Fatalf("expected status unconfigured_credential, got %s", snapshot.Status)
	}
}

func TestAntigravityQuotaAdapter_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	adapter := NewAntigravityQuotaAdapter(server.Client(), nil)
	snapshot, err := adapter.FetchQuota(ctx, server.URL, "valid-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot.Status != QuotaStatusTimeout {
		t.Fatalf("expected timeout status, got %s", snapshot.Status)
	}
}

func TestAntigravityQuotaAdapter_HostValidation(t *testing.T) {
	adapter := NewAntigravityQuotaAdapter(nil, []string{"proxy.internal"})
	snapshot, err := adapter.FetchQuota(context.Background(), "http://malicious.evil.com/v1", "valid-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot.Status != QuotaStatusError || snapshot.Error.Code != "host_not_allowed" {
		t.Fatalf("expected host_not_allowed error, got %+v", snapshot)
	}
}

func ptrString(s string) *string {
	return &s
}
