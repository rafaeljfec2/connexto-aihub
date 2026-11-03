package gateway_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aihub/internal/access"
	"aihub/internal/concurrency"
	"aihub/internal/gateway"
)

type inMemoryConcurrency struct {
	policies map[int64]concurrency.Policy
	nextID   int64
}

func (m *inMemoryConcurrency) CreatePolicy(_ context.Context, input concurrency.NewPolicyInput) (concurrency.Policy, error) {
	m.nextID++
	policy := concurrency.Policy{
		ID: m.nextID, Scope: input.Scope, ConcurrencyLimit: input.ConcurrencyLimit, Enabled: input.Enabled,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	m.policies[policy.ID] = policy
	return policy, nil
}

func (m *inMemoryConcurrency) UpdatePolicy(_ context.Context, id int64, input concurrency.UpdatePolicyInput) (concurrency.Policy, error) {
	policy, ok := m.policies[id]
	if !ok {
		return concurrency.Policy{}, concurrency.ErrNotFound
	}
	if input.ConcurrencyLimit != nil {
		policy.ConcurrencyLimit = *input.ConcurrencyLimit
	}
	m.policies[id] = policy
	return policy, nil
}

func (m *inMemoryConcurrency) ConfigurePolicy(_ context.Context, id int64, limit int, enabled bool) (concurrency.Policy, error) {
	policy, ok := m.policies[id]
	if !ok {
		return concurrency.Policy{}, concurrency.ErrNotFound
	}
	policy.ConcurrencyLimit = limit
	policy.Enabled = enabled
	m.policies[id] = policy
	return policy, nil
}

func (m *inMemoryConcurrency) SetPolicyEnabled(_ context.Context, id int64, enabled bool) (concurrency.Policy, error) {
	policy, ok := m.policies[id]
	if !ok {
		return concurrency.Policy{}, concurrency.ErrNotFound
	}
	policy.Enabled = enabled
	m.policies[id] = policy
	return policy, nil
}

func (m *inMemoryConcurrency) GetPolicy(_ context.Context, id int64) (concurrency.Policy, error) {
	policy, ok := m.policies[id]
	if !ok {
		return concurrency.Policy{}, concurrency.ErrNotFound
	}
	return policy, nil
}

func (m *inMemoryConcurrency) GetPolicyByScope(_ context.Context, scope concurrency.Scope) (concurrency.Policy, error) {
	for _, policy := range m.policies {
		if policy.Scope == scope {
			return policy, nil
		}
	}
	return concurrency.Policy{}, concurrency.ErrNotFound
}

func (m *inMemoryConcurrency) ListPolicies(_ context.Context) ([]concurrency.Policy, error) {
	out := make([]concurrency.Policy, 0, len(m.policies))
	for _, policy := range m.policies {
		out = append(out, policy)
	}
	return out, nil
}

func (m *inMemoryConcurrency) ListUsage(_ context.Context, query concurrency.UsageQuery) ([]concurrency.UsageSnapshot, error) {
	out := make([]concurrency.UsageSnapshot, 0, len(m.policies))
	for _, policy := range m.policies {
		out = append(out, concurrency.UsageSnapshot{Policy: policy})
	}
	return out, nil
}

func (m *inMemoryConcurrency) Acquire(_ context.Context, _ concurrency.AcquireRequest, _ time.Time, _ time.Duration) (concurrency.AcquireDecision, error) {
	return concurrency.AcquireDecision{Allowed: true}, nil
}

func (m *inMemoryConcurrency) Release(_ context.Context, _ string) error { return nil }

func (m *inMemoryConcurrency) RenewLease(_ context.Context, _ string, _ time.Time, _ time.Duration) error {
	return nil
}

func concurrencyAdminServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := &inMemoryConcurrency{policies: map[int64]concurrency.Policy{}}
	service := concurrency.NewService(repo)
	admin := gateway.NewConcurrencyAdmin(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:       resourceGatewayToken,
		AdminAPIKey:  resourceAdminToken,
		MaxBodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/concurrency-policies", admin.List())
	mux.Handle("PUT /api/concurrency-policies", admin.Upsert())
	mux.Handle("GET /api/concurrency-usage", admin.Usage())
	mux.Handle("POST /api/concurrency-policies/{id}/enable", admin.Enable())
	mux.Handle("POST /api/concurrency-policies/{id}/disable", admin.Disable())

	server := httptest.NewServer(policy.Authorize(mux))
	t.Cleanup(server.Close)
	return server
}

func TestConcurrencyAdminUpsertAndUsageQueryValidation(t *testing.T) {
	server := concurrencyAdminServer(t)
	token := resourceAdminToken

	body := `{"scope":"global","concurrency_limit":3,"enabled":true}`
	status, payload := resourceAdminCall(t, http.MethodPut, server.URL+"/api/concurrency-policies", body, token)
	if status != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodPut, server.URL+"/api/concurrency-policies", body, token)
	if status != http.StatusOK {
		t.Fatalf("update status = %d body = %s", status, payload)
	}

	if status, _ := resourceAdminCall(t, http.MethodGet, server.URL+"/api/concurrency-usage?unknown=1", "", token); status != http.StatusBadRequest {
		t.Fatalf("unknown query status = %d, want 400", status)
	}

	status, payload = resourceAdminCall(t, http.MethodGet, server.URL+"/api/concurrency-usage", "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"concurrency_limit":3`) {
		t.Fatalf("usage = %d %s", status, payload)
	}
}
