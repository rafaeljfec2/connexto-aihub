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
	"testing"
	"time"

	"aihub/internal/config"
	"aihub/internal/gateway"
	"aihub/internal/providers"
)

const adminSecret = "oc_sk_admin_fixture_credential_0001"

// inMemoryProviders satisfies providers.Repository for handler tests.
type inMemoryProviders struct {
	records map[string]providers.Record
	sealed  map[string][]byte
}

func (r *inMemoryProviders) taken(name, exceptID string) bool {
	for id, record := range r.records {
		if id != exceptID && record.Name == name {
			return true
		}
	}

	return false
}

func (r *inMemoryProviders) Create(_ context.Context, in providers.NewRecord) (providers.Record, error) {
	if _, ok := r.records[in.ID]; ok || r.taken(in.Name, "") {
		return providers.Record{}, providers.ErrConflict
	}

	record := providers.Record{ID: in.ID, Name: in.Name, Type: in.Type, Endpoint: in.Endpoint,
		Enabled: in.Enabled, HasCredential: in.Credential != nil, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	r.records[in.ID] = record
	r.sealed[in.ID] = in.Credential

	return record, nil
}

func (r *inMemoryProviders) EnsureConfigured(ctx context.Context, in providers.NewRecord) (providers.Record, error) {
	return r.Create(ctx, in)
}

func (r *inMemoryProviders) Get(_ context.Context, id string) (providers.Record, error) {
	record, ok := r.records[id]
	if !ok {
		return providers.Record{}, providers.ErrNotFound
	}

	return record, nil
}

func (r *inMemoryProviders) List(_ context.Context) ([]providers.Record, error) {
	out := make([]providers.Record, 0, len(r.records))
	for _, record := range r.records {
		out = append(out, record)
	}

	return out, nil
}

func (r *inMemoryProviders) Update(_ context.Context, id string, changes providers.Changes) (providers.Record, error) {
	record, ok := r.records[id]
	if !ok {
		return providers.Record{}, providers.ErrNotFound
	}

	if changes.Name != nil {
		if r.taken(*changes.Name, id) {
			return providers.Record{}, providers.ErrConflict
		}

		record.Name = *changes.Name
	}

	if changes.Endpoint != nil {
		record.Endpoint = *changes.Endpoint
	}

	if changes.Credential != nil {
		r.sealed[id] = changes.Credential
		record.HasCredential = true
	}

	r.records[id] = record

	return record, nil
}

func (r *inMemoryProviders) SetEnabled(_ context.Context, id string, enabled bool) (providers.Record, error) {
	record, ok := r.records[id]
	if !ok {
		return providers.Record{}, providers.ErrNotFound
	}

	record.Enabled = enabled
	r.records[id] = record

	return record, nil
}

func (r *inMemoryProviders) SealedCredential(_ context.Context, id string) ([]byte, error) {
	if _, ok := r.records[id]; !ok {
		return nil, providers.ErrNotFound
	}

	return r.sealed[id], nil
}

func adminServer(t *testing.T) *httptest.Server {
	t.Helper()

	cipher, err := providers.NewCredentialCipher(bytes.Repeat([]byte{0x22}, config.CredentialKeyBytes))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}

	repo := &inMemoryProviders{records: map[string]providers.Record{}, sealed: map[string][]byte{}}
	service := providers.NewService(repo, cipher, providers.NewRegistry(), providers.ProbeEndpoint,
		[]string{"provider.example.test", "127.0.0.1"})
	admin := gateway.NewProviderAdmin(service, newAdminLogger())

	mux := http.NewServeMux()
	mux.Handle("POST /api/providers", admin.Create())
	mux.Handle("GET /api/providers", admin.List())
	mux.Handle("GET /api/providers/{id}", admin.Get())
	mux.Handle("PATCH /api/providers/{id}", admin.Update())
	mux.Handle("POST /api/providers/{id}/validate", admin.Validate())
	mux.Handle("POST /api/providers/{id}/enable", admin.Enable())
	mux.Handle("POST /api/providers/{id}/disable", admin.Disable())

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func newAdminLogger() *slog.Logger {
	logger, _ := newTestLogger()

	return logger
}

func adminCall(t *testing.T, method, url, body string) (int, string) {
	t.Helper()

	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if strings.Contains(string(payload), adminSecret) {
		t.Fatalf("response leaked the credential: %s", payload)
	}

	return response.StatusCode, string(payload)
}

func TestProviderAdminCreateReturnsConfigurationWithoutCredential(t *testing.T) {
	server := adminServer(t)
	body := `{"id":"opencode-go","name":"OpenCode Go","type":"opencode-go","endpoint":"https://provider.example.test","credential":"` + adminSecret + `"}`

	status, payload := adminCall(t, http.MethodPost, server.URL+"/api/providers", body)
	if status != http.StatusCreated {
		t.Fatalf("status = %d body %s, want 201", status, payload)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if response["credential_configured"] != true || response["enabled"] != true {
		t.Fatalf("response = %v", response)
	}

	if _, leaked := response["credential"]; leaked {
		t.Fatal("response carries a credential field")
	}
}

func TestProviderAdminRejectsMalformedAndUnknownFields(t *testing.T) {
	server := adminServer(t)

	for name, body := range map[string]string{
		"malformed json": `{"id":`,
		"unknown field":  `{"id":"p1","name":"n","type":"opencode-go","endpoint":"https://e.test","surprise":1}`,
		"trailing data":  `{"id":"p1"} {"id":"p2"}`,
	} {
		if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", body); status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", name, status)
		}
	}
}

func TestProviderAdminMapsNotFoundConflictAndState(t *testing.T) {
	server := adminServer(t)
	create := `{"id":"p1","name":"Primary","type":"opencode-go","endpoint":"https://provider.example.test"}`

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", create); status != http.StatusCreated {
		t.Fatalf("create status = %d", status)
	}

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers", create); status != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", status)
	}

	if status, _ := adminCall(t, http.MethodGet, server.URL+"/api/providers/missing", ""); status != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", status)
	}

	if status, body := adminCall(t, http.MethodPost, server.URL+"/api/providers/p1/disable", ""); status != http.StatusOK || !strings.Contains(body, `"enabled":false`) {
		t.Fatalf("disable = %d %s", status, body)
	}

	if status, body := adminCall(t, http.MethodPatch, server.URL+"/api/providers/p1", `{"name":"Renamed"}`); status != http.StatusOK || !strings.Contains(body, "Renamed") {
		t.Fatalf("patch = %d %s", status, body)
	}
}

func TestProviderAdminValidateReportsMissingCredentialAndUnreachableUpstream(t *testing.T) {
	server := adminServer(t)

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers",
		`{"id":"p1","name":"Primary","type":"opencode-go","endpoint":"http://127.0.0.1:1"}`); status != http.StatusCreated {
		t.Fatalf("create status = %d", status)
	}

	if status, _ := adminCall(t, http.MethodPost, server.URL+"/api/providers/p1/validate", ""); status != http.StatusConflict {
		t.Fatalf("validate without credential = %d, want 409", status)
	}

	if status, _ := adminCall(t, http.MethodPatch, server.URL+"/api/providers/p1",
		`{"credential":"`+adminSecret+`"}`); status != http.StatusOK {
		t.Fatalf("patch credential = %d", status)
	}

	if status, body := adminCall(t, http.MethodPost, server.URL+"/api/providers/p1/validate", ""); status != http.StatusBadGateway || !strings.Contains(body, "provider_unreachable") {
		t.Fatalf("validate unreachable = %d %s, want 502", status, body)
	}
}
