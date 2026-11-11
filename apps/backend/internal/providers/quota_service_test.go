package providers

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeQuotaRepo struct {
	records         []Record
	creds           map[string][]byte
	managementCreds map[string][]byte
}

func (f *fakeQuotaRepo) Create(ctx context.Context, r NewRecord) (Record, error) {
	return Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepo) EnsureConfigured(ctx context.Context, r NewRecord) (Record, error) {
	return Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepo) Get(ctx context.Context, id string) (Record, error) {
	for _, rec := range f.records {
		if rec.ID == id {
			return rec, nil
		}
	}
	return Record{}, ErrNotFound
}

func (f *fakeQuotaRepo) List(ctx context.Context) ([]Record, error) {
	return append([]Record(nil), f.records...), nil
}

func (f *fakeQuotaRepo) Update(ctx context.Context, id string, c Changes) (Record, error) {
	return Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepo) SetEnabled(ctx context.Context, id string, enabled bool) (Record, error) {
	return Record{}, errors.New("unimplemented")
}

func (f *fakeQuotaRepo) SealedCredential(ctx context.Context, id string) ([]byte, error) {
	return f.creds[id], nil
}

func (f *fakeQuotaRepo) SealedManagementCredential(ctx context.Context, id string) ([]byte, error) {
	return f.managementCreds[id], nil
}

type fakeMockAdapter struct {
	fn func(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error)
}

func (m *fakeMockAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	return m.fn(ctx, endpoint, credential)
}

func TestQuotaService_QueryAllAccounts(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher init error: %v", err)
	}

	sealedKey1, _ := cipher.Seal("openrouter-team-a", "key-a")
	sealedKey2, _ := cipher.Seal("openrouter-team-b", "key-b")

	repo := &fakeQuotaRepo{
		records: []Record{
			{
				ID:            "openrouter-team-a",
				Name:          "OpenRouter Team A",
				Type:          TypeOpenRouter,
				Endpoint:      "https://openrouter.ai/api/v1",
				Enabled:       true,
				HasCredential: true,
			},
			{
				ID:            "openrouter-team-b",
				Name:          "OpenRouter Team B",
				Type:          TypeOpenRouter,
				Endpoint:      "https://openrouter.ai/api/v1",
				Enabled:       true,
				HasCredential: true,
			},
			{
				ID:            "openai-prod",
				Name:          "OpenAI Produção",
				Type:          "openai",
				Endpoint:      "https://api.openai.com/v1",
				Enabled:       true,
				HasCredential: true,
			},
			{
				ID:            "anthropic-unconfigured",
				Name:          "Anthropic Dev",
				Type:          "anthropic",
				Endpoint:      "https://api.anthropic.com",
				Enabled:       true,
				HasCredential: false,
			},
			{
				ID:            "disabled-account",
				Name:          "Conta Inativa",
				Type:          TypeOpenRouter,
				Endpoint:      "https://openrouter.ai/api/v1",
				Enabled:       false,
				HasCredential: true,
			},
		},
		creds: map[string][]byte{
			"openrouter-team-a": sealedKey1,
			"openrouter-team-b": sealedKey2,
		},
	}

	adminService := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"openrouter.ai", "api.openai.com"})

	usd := "USD"
	limitA := 200.0
	usedA := 50.0

	openrouterAdapter := &fakeMockAdapter{
		fn: func(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
			if credential == "key-a" {
				return ProviderQuotaSnapshot{
					Status: QuotaStatusAvailable,
					Scope:  ScopeAPIKey,
					Quotas: []ProviderQuotaMetric{
						{
							Metric: "limit",
							Label:  "Limite da chave",
							Nature: NatureQuota,
							Limit:  &limitA,
							Unit:   &usd,
						},
						{
							Metric: "usage",
							Label:  "Consumo total",
							Nature: NatureCost,
							Used:   &usedA,
							Unit:   &usd,
						},
					},
					Source: ProviderQuotaSource{
						Type:      "provider_api",
						Name:      "openrouter_key_api",
						Scope:     ScopeAPIKey,
						FetchedAt: time.Now().UTC().Format(time.RFC3339),
					},
				}, nil
			}
			// key-b triggers unauthorized
			return ProviderQuotaSnapshot{
				Status: QuotaStatusUnauthorized,
				Scope:  ScopeAPIKey,
				Error: &ProviderQuotaError{
					Code:    "unauthorized",
					Message: "Chave inválida",
				},
			}, nil
		},
	}

	adapters := map[string]QuotaAdapter{
		TypeOpenRouter: openrouterAdapter,
	}

	service := NewQuotaService(adminService, adapters, nil)

	res, err := service.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Snapshots) != 5 {
		t.Fatalf("expected 5 snapshots, got %d", len(res.Snapshots))
	}

	snapshotsMap := make(map[string]ProviderQuotaSnapshot)
	for _, s := range res.Snapshots {
		snapshotsMap[s.AccountID] = s
	}

	// 1. Team A (OpenRouter) should be available with mapped metrics
	teamA := snapshotsMap["openrouter-team-a"]
	if teamA.Status != QuotaStatusAvailable {
		t.Errorf("team-a status expected available, got %s", teamA.Status)
	}
	if len(teamA.Quotas) != 2 {
		t.Errorf("team-a expected 2 quotas, got %d", len(teamA.Quotas))
	}

	// 2. Team B (OpenRouter) should be unauthorized without breaking Team A
	teamB := snapshotsMap["openrouter-team-b"]
	if teamB.Status != QuotaStatusUnauthorized {
		t.Errorf("team-b status expected unauthorized, got %s", teamB.Status)
	}

	// 3. OpenAI should be unsupported
	openAI := snapshotsMap["openai-prod"]
	if openAI.Status != QuotaStatusUnsupported {
		t.Errorf("openai expected unsupported, got %s", openAI.Status)
	}

	// 4. Anthropic should be unconfigured_credential
	anthropic := snapshotsMap["anthropic-unconfigured"]
	if anthropic.Status != QuotaStatusUnconfiguredCredential {
		t.Errorf("anthropic expected unconfigured_credential, got %s", anthropic.Status)
	}

	// 5. Disabled account should be unsupported
	disabled := snapshotsMap["disabled-account"]
	if disabled.Status != QuotaStatusUnsupported {
		t.Errorf("disabled account expected unsupported, got %s", disabled.Status)
	}
}

func TestQuotaService_QueryAntigravityReturnsUnsupportedNotUnconfigured(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher init error: %v", err)
	}
	sealedGw, _ := cipher.SealManagement("antigravity", "gw-key")
	repo := &fakeQuotaRepo{
		records: []Record{
			{
				ID:                      "antigravity",
				Name:                    "Antigravity",
				Type:                    TypeAntigravity,
				Endpoint:                "http://127.0.0.1:9090/v1",
				Enabled:                 true,
				HasCredential:           false,
				HasManagementCredential: true,
			},
		},
		managementCreds: map[string][]byte{"antigravity": sealedGw},
	}
	admin := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"127.0.0.1"})
	svc := NewQuotaService(admin, map[string]QuotaAdapter{}, nil)
	res, err := svc.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(res.Snapshots))
	}
	got := res.Snapshots[0]
	if got.Status != QuotaStatusUnsupported {
		t.Fatalf("status = %s, want unsupported", got.Status)
	}
	if got.Error == nil || got.Error.Code != "quota_not_applicable" {
		t.Fatalf("error = %#v, want quota_not_applicable", got.Error)
	}
}

type fakeAntigravityQuotaAdapter struct {
	snapshot ProviderQuotaSnapshot
	err      error
}

func (f *fakeAntigravityQuotaAdapter) FetchQuota(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
	if f.err != nil {
		return ProviderQuotaSnapshot{}, f.err
	}
	return f.snapshot, nil
}

func TestQuotaService_QueryAntigravitySuccessWithAdapter(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher init error: %v", err)
	}
	sealedGw, _ := cipher.SealManagement("antigravity", "gw-key")
	repo := &fakeQuotaRepo{
		records: []Record{
			{
				ID:                      "antigravity",
				Name:                    "Antigravity",
				Type:                    TypeAntigravity,
				Endpoint:                "http://127.0.0.1:9090/v1",
				Enabled:                 true,
				HasCredential:           false,
				HasManagementCredential: true,
			},
		},
		managementCreds: map[string][]byte{"antigravity": sealedGw},
	}
	admin := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"127.0.0.1"})

	remFrac := 0.75
	fakeAdapter := &fakeAntigravityQuotaAdapter{
		snapshot: ProviderQuotaSnapshot{
			Status:       QuotaStatusAvailable,
			AccountLabel: "engineer@connexto.ai",
			Quotas: []ProviderQuotaMetric{
				{
					Metric:            "gemini-2.5-pro",
					Label:             "Gemini 2.5 Pro",
					Nature:            NatureRateLimit,
					RemainingFraction: &remFrac,
				},
			},
		},
	}

	svc := NewQuotaService(admin, map[string]QuotaAdapter{TypeAntigravity: fakeAdapter}, nil)
	res, err := svc.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(res.Snapshots))
	}
	got := res.Snapshots[0]
	if got.Status != QuotaStatusAvailable {
		t.Fatalf("status = %s, want available", got.Status)
	}
	if got.AccountLabel != "engineer@connexto.ai" {
		t.Fatalf("account label = %s, want engineer@connexto.ai", got.AccountLabel)
	}
	if len(got.Quotas) != 1 || got.Quotas[0].Metric != "gemini-2.5-pro" {
		t.Fatalf("unexpected quotas: %+v", got.Quotas)
	}
}

func TestQuotaService_IsolationBetweenProviders(t *testing.T) {
	cipher, err := NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher init error: %v", err)
	}
	sealedGw, _ := cipher.SealManagement("antigravity", "gw-key")
	sealedOR, _ := cipher.Seal("openrouter", "sk-or-test")
	repo := &fakeQuotaRepo{
		records: []Record{
			{
				ID:                      "antigravity",
				Name:                    "Antigravity",
				Type:                    TypeAntigravity,
				Endpoint:                "http://127.0.0.1:9090/v1",
				Enabled:                 true,
				HasCredential:           false,
				HasManagementCredential: true,
			},
			{
				ID:            "openrouter",
				Name:          "OpenRouter",
				Type:          TypeOpenRouter,
				Endpoint:      "https://openrouter.ai/api/v1",
				Enabled:       true,
				HasCredential: true,
			},
		},
		creds:           map[string][]byte{"openrouter": sealedOR},
		managementCreds: map[string][]byte{"antigravity": sealedGw},
	}
	admin := NewServiceWithProbes(repo, cipher, NewRegistry(), nil, []string{"127.0.0.1", "openrouter.ai"})

	failingAntigravity := &fakeAntigravityQuotaAdapter{
		snapshot: ProviderQuotaSnapshot{
			Status: QuotaStatusError,
			Error:  &ProviderQuotaError{Code: "upstream_error", Message: "Proxy down"},
		},
	}
	workingOpenRouter := &fakeMockAdapter{
		fn: func(ctx context.Context, endpoint, credential string) (ProviderQuotaSnapshot, error) {
			return ProviderQuotaSnapshot{
				Status: QuotaStatusAvailable,
				Quotas: []ProviderQuotaMetric{
					{Metric: "usage", Label: "Uso"},
				},
			}, nil
		},
	}

	svc := NewQuotaService(admin, map[string]QuotaAdapter{
		TypeAntigravity: failingAntigravity,
		TypeOpenRouter:  workingOpenRouter,
	}, nil)

	res, err := svc.QueryAllAccounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(res.Snapshots))
	}

	var agSnapshot, orSnapshot ProviderQuotaSnapshot
	for _, snap := range res.Snapshots {
		if snap.ProviderID == TypeAntigravity {
			agSnapshot = snap
		} else if snap.ProviderID == TypeOpenRouter {
			orSnapshot = snap
		}
	}

	if agSnapshot.Status != QuotaStatusError {
		t.Errorf("antigravity status = %s, want error", agSnapshot.Status)
	}
	if orSnapshot.Status != QuotaStatusAvailable {
		t.Errorf("openrouter status = %s, want available", orSnapshot.Status)
	}
	if len(orSnapshot.Quotas) != 1 {
		t.Errorf("expected OpenRouter to have 1 quota metric unaffected by Antigravity failure")
	}
}
