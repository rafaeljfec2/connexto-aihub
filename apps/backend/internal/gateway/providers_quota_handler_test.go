package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/providers"
)

type fakeQuotaRepoForHandler struct {
	records []providers.Record
	creds   map[string][]byte
}

func (f *fakeQuotaRepoForHandler) Create(ctx context.Context, r providers.NewRecord) (providers.Record, error) {
	return providers.Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepoForHandler) EnsureConfigured(ctx context.Context, r providers.NewRecord) (providers.Record, error) {
	return providers.Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepoForHandler) Get(ctx context.Context, id string) (providers.Record, error) {
	return providers.Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepoForHandler) List(ctx context.Context) ([]providers.Record, error) {
	return f.records, nil
}

func (f *fakeQuotaRepoForHandler) Update(ctx context.Context, id string, c providers.Changes) (providers.Record, error) {
	return providers.Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepoForHandler) SetEnabled(ctx context.Context, id string, enabled bool) (providers.Record, error) {
	return providers.Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepoForHandler) Delete(ctx context.Context, id string) error {
	return errors.New("unimplemented")
}

func (f *fakeQuotaRepoForHandler) SealedCredential(ctx context.Context, id string) ([]byte, error) {
	return f.creds[id], nil
}

func (f *fakeQuotaRepoForHandler) SealedManagementCredential(ctx context.Context, id string) ([]byte, error) {
	return nil, nil
}

func TestProviderQuotasHandler_Success(t *testing.T) {
	cipher, _ := providers.NewCredentialCipher([]byte("01234567890123456789012345678901"))
	sealed, _ := cipher.Seal("openrouter-1", "test-key")

	serverMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"label":"sk-provider-label-marker","usage":12.5,"limit":50.0,"limit_remaining":37.5}}`))
	}))
	defer serverMock.Close()

	repo := &fakeQuotaRepoForHandler{
		records: []providers.Record{
			{
				ID:            "openrouter-1",
				Name:          "OpenRouter Primary",
				Type:          providers.TypeOpenRouter,
				Endpoint:      serverMock.URL,
				Enabled:       true,
				HasCredential: true,
			},
		},
		creds: map[string][]byte{
			"openrouter-1": sealed,
		},
	}

	adminService := providers.NewServiceWithProbes(repo, cipher, providers.NewRegistry(), nil, []string{"openrouter.ai", "127.0.0.1"})

	quotaAdapter := providers.NewOpenRouterQuotaAdapter(serverMock.Client(), []string{"127.0.0.1", "localhost"})

	quotaService := providers.NewQuotaService(adminService, map[string]providers.QuotaAdapter{
		providers.TypeOpenRouter: quotaAdapter,
	}, nil)

	handler := NewProviderQuotasHandler(quotaService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/ui/providers/quotas?sync=true", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "sk-provider-label-marker") || strings.Contains(rec.Body.String(), "test-key") {
		t.Fatal("quota response exposed provider key details or a credential")
	}

	var resp providers.ProviderQuotasResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Snapshots) != 2 {
		t.Fatalf("expected one key and one account snapshot, got %d", len(resp.Snapshots))
	}

	s := resp.Snapshots[0]
	if s.AccountID != "openrouter-1" {
		t.Errorf("expected account_id openrouter-1, got %s", s.AccountID)
	}
	if s.AccountLabel != "OpenRouter Primary" {
		t.Errorf("expected account_label 'OpenRouter Primary', got %s", s.AccountLabel)
	}
	if s.Status != providers.QuotaStatusAvailable {
		t.Errorf("expected status available, got %s", s.Status)
	}
	if s.Scope != providers.ScopeAPIKey || resp.Snapshots[1].Scope != providers.ScopeAccount {
		t.Fatalf("snapshot scopes = %q and %q, want api_key and account", s.Scope, resp.Snapshots[1].Scope)
	}
	if resp.Snapshots[1].Status != providers.QuotaStatusUnconfiguredCredential {
		t.Fatalf("account snapshot status = %q, want unconfigured credential", resp.Snapshots[1].Status)
	}

	// 2. Subsequent default GET /api/ui/providers/quotas returns cached snapshots immediately without sync query.
	reqCached := httptest.NewRequest(http.MethodGet, "/api/ui/providers/quotas", nil)
	recCached := httptest.NewRecorder()
	handler.ServeHTTP(recCached, reqCached)
	if recCached.Code != http.StatusOK {
		t.Fatalf("expected status 200 on cached read, got %d", recCached.Code)
	}
	var respCached providers.ProviderQuotasResponse
	if err := json.Unmarshal(recCached.Body.Bytes(), &respCached); err != nil {
		t.Fatalf("failed to decode cached response: %v", err)
	}
	if len(respCached.Snapshots) != 2 {
		t.Fatalf("expected 2 cached snapshots, got %d", len(respCached.Snapshots))
	}
	if respCached.Snapshots[0].Status != providers.QuotaStatusAvailable {
		t.Errorf("cached snapshot status = %q, want available", respCached.Snapshots[0].Status)
	}
}

func TestProviderQuotasHandler_MethodNotAllowed(t *testing.T) {
	handler := NewProviderQuotasHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/ui/providers/quotas", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}
