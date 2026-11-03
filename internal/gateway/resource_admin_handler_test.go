package gateway_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"aihub/internal/access"
	"aihub/internal/gateway"
	"aihub/internal/resources"
)

type inMemoryResources struct {
	policies map[int64]resources.ResourcePolicy
	nextID   int64
}

func (r *inMemoryResources) CreatePolicy(_ context.Context, input resources.NewPolicyInput) (resources.ResourcePolicy, error) {
	r.nextID++
	policy := resources.ResourcePolicy{
		ID: r.nextID, Scope: input.Scope, DailyLimit: input.DailyLimit, Enabled: input.Enabled,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	r.policies[policy.ID] = policy
	return policy, nil
}

func (r *inMemoryResources) UpdatePolicy(_ context.Context, id int64, input resources.UpdatePolicyInput) (resources.ResourcePolicy, error) {
	policy, ok := r.policies[id]
	if !ok {
		return resources.ResourcePolicy{}, resources.ErrNotFound
	}
	if input.DailyLimit != nil {
		policy.DailyLimit = *input.DailyLimit
	}
	policy.UpdatedAt = time.Now().UTC()
	r.policies[id] = policy
	return policy, nil
}

func (r *inMemoryResources) SetPolicyEnabled(_ context.Context, id int64, enabled bool) (resources.ResourcePolicy, error) {
	policy, ok := r.policies[id]
	if !ok {
		return resources.ResourcePolicy{}, resources.ErrNotFound
	}
	policy.Enabled = enabled
	policy.UpdatedAt = time.Now().UTC()
	r.policies[id] = policy
	return policy, nil
}

func (r *inMemoryResources) GetPolicy(_ context.Context, id int64) (resources.ResourcePolicy, error) {
	policy, ok := r.policies[id]
	if !ok {
		return resources.ResourcePolicy{}, resources.ErrNotFound
	}
	return policy, nil
}

func (r *inMemoryResources) ListPolicies(_ context.Context) ([]resources.ResourcePolicy, error) {
	out := make([]resources.ResourcePolicy, 0, len(r.policies))
	for _, policy := range r.policies {
		out = append(out, policy)
	}
	return out, nil
}

func (r *inMemoryResources) GetPolicyByScope(_ context.Context, scope resources.ResourceScope) (resources.ResourcePolicy, error) {
	for _, policy := range r.policies {
		if policy.Scope == scope {
			return policy, nil
		}
	}
	return resources.ResourcePolicy{}, resources.ErrNotFound
}

func (r *inMemoryResources) ListUsage(_ context.Context, query resources.UsageQuery) ([]resources.UsageSnapshot, error) {
	start, reset := resources.UsagePeriodBounds(query.PeriodStart)
	out := make([]resources.UsageSnapshot, 0, len(r.policies))
	for _, policy := range r.policies {
		out = append(out, resources.UsageSnapshot{
			Policy: policy, PeriodStart: start, ResetAt: reset, AdmittedCount: 0,
		})
	}
	return out, nil
}

func (r *inMemoryResources) Admit(_ context.Context, request resources.AdmissionRequest, now time.Time) (resources.AdmissionDecision, error) {
	start, reset := resources.UsagePeriodBounds(now)
	return resources.AdmissionDecision{AdmissionID: request.ID, Allowed: true, PeriodStart: start, ResetAt: reset}, nil
}

const (
	resourceAdminToken  = "adminKeyFixture0123456789abcdefGHIJ"
	resourceGatewayToken = "gatewayKeyFixture0123456789abcdefGH"
)

func resourceAdminServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := &inMemoryResources{policies: map[int64]resources.ResourcePolicy{}}
	service := resources.NewService(repo)
	admin := gateway.NewResourceAdmin(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	policy, err := access.NewPolicy(access.PolicyConfig{
		APIKey:       resourceGatewayToken,
		AdminAPIKey:  resourceAdminToken,
		MaxBodyBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewPolicy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /api/resource-policies", admin.List())
	mux.Handle("PUT /api/resource-policies", admin.Upsert())
	mux.Handle("GET /api/resource-usage", admin.Usage())
	mux.Handle("POST /api/resource-policies/{id}/enable", admin.Enable())
	mux.Handle("POST /api/resource-policies/{id}/disable", admin.Disable())

	server := httptest.NewServer(policy.Authorize(mux))
	t.Cleanup(server.Close)
	return server
}

func resourceAdminCall(t *testing.T, method, url, body, token string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	return resp.StatusCode, string(payload)
}

func TestResourceAdminUpsertListEnableDisable(t *testing.T) {
	server := resourceAdminServer(t)
	token := resourceAdminToken

	createBody := `{"scope":"global","daily_limit":100,"enabled":true}`
	status, payload := resourceAdminCall(t, http.MethodPut, server.URL+"/api/resource-policies", createBody, token)
	if status != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodPut, server.URL+"/api/resource-policies", createBody, token)
	if status != http.StatusOK {
		t.Fatalf("idempotent upsert status = %d body = %s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodGet, server.URL+"/api/resource-policies", "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"daily_limit":100`) {
		t.Fatalf("list = %d %s", status, payload)
	}

	var listed struct {
		Policies []struct {
			ID int64 `json:"id"`
		} `json:"policies"`
	}
	if err := json.Unmarshal([]byte(payload), &listed); err != nil || len(listed.Policies) != 1 {
		t.Fatalf("decode list: %v policies=%+v", err, listed.Policies)
	}

	disableURL := server.URL + "/api/resource-policies/" + strconv.FormatInt(listed.Policies[0].ID, 10) + "/disable"
	status, payload = resourceAdminCall(t, http.MethodPost, disableURL, "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"enabled":false`) {
		t.Fatalf("disable = %d %s", status, payload)
	}

	status, payload = resourceAdminCall(t, http.MethodGet, server.URL+"/api/resource-usage", "", token)
	if status != http.StatusOK || !strings.Contains(payload, `"remaining_requests":0`) {
		t.Fatalf("usage = %d %s", status, payload)
	}
}

func TestResourceAdminRejectsInvalidInputAndUnknownFields(t *testing.T) {
	server := resourceAdminServer(t)
	token := resourceAdminToken

	for name, body := range map[string]string{
		"unknown field": `{"scope":"global","daily_limit":1,"extra":1}`,
		"invalid scope": `{"scope":"tenant","daily_limit":1}`,
		"invalid limit": `{"scope":"global","daily_limit":0}`,
	} {
		status, _ := resourceAdminCall(t, http.MethodPut, server.URL+"/api/resource-policies", body, token)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", name, status)
		}
	}

	if status, _ := resourceAdminCall(t, http.MethodGet, server.URL+"/api/resource-usage?surprise=1", "", token); status != http.StatusBadRequest {
		t.Fatalf("unknown query status = %d, want 400", status)
	}
}

func TestResourceAdminRequiresAdminCredential(t *testing.T) {
	server := resourceAdminServer(t)
	if status, _ := resourceAdminCall(t, http.MethodGet, server.URL+"/api/resource-policies", "", ""); status != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want 401", status)
	}
}
