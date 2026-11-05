package gateway_test

import (
	"bytes"
	"context"
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

type inMemoryProviders struct {
	records          map[string]providers.Record
	sealed           map[string][]byte
	sealedManagement map[string][]byte
}

func (r *inMemoryProviders) taken(name, exceptID string) bool {
	for id, record := range r.records {
		if id != exceptID && record.Name == name {
			return true
		}
	}
	return false
}

func (r *inMemoryProviders) Create(_ context.Context, input providers.NewRecord) (providers.Record, error) {
	if _, ok := r.records[input.ID]; ok || r.taken(input.Name, "") {
		return providers.Record{}, providers.ErrConflict
	}
	now := time.Now()
	record := providers.Record{
		ID: input.ID, Name: input.Name, Type: input.Type, Endpoint: input.Endpoint,
		Enabled: input.Enabled, HasCredential: input.Credential != nil,
		HasManagementCredential: input.ManagementCredential != nil, CreatedAt: now, UpdatedAt: now,
	}
	r.records[input.ID] = record
	r.sealed[input.ID] = input.Credential
	r.sealedManagement[input.ID] = input.ManagementCredential
	return record, nil
}

func (r *inMemoryProviders) EnsureConfigured(ctx context.Context, input providers.NewRecord) (providers.Record, error) {
	return r.Create(ctx, input)
}

func (r *inMemoryProviders) Get(_ context.Context, id string) (providers.Record, error) {
	record, ok := r.records[id]
	if !ok {
		return providers.Record{}, providers.ErrNotFound
	}
	return record, nil
}

func (r *inMemoryProviders) List(_ context.Context) ([]providers.Record, error) {
	result := make([]providers.Record, 0, len(r.records))
	for _, record := range r.records {
		result = append(result, record)
	}
	return result, nil
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
	if changes.ManagementCredential != nil {
		r.sealedManagement[id] = changes.ManagementCredential
		record.HasManagementCredential = true
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

func (r *inMemoryProviders) SealedManagementCredential(_ context.Context, id string) ([]byte, error) {
	if _, ok := r.records[id]; !ok {
		return nil, providers.ErrNotFound
	}
	return r.sealedManagement[id], nil
}

func adminServer(t *testing.T) *httptest.Server {
	t.Helper()
	cipher, err := providers.NewCredentialCipher(bytes.Repeat([]byte{0x22}, config.CredentialKeyBytes))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}
	repo := &inMemoryProviders{
		records: map[string]providers.Record{}, sealed: map[string][]byte{},
		sealedManagement: map[string][]byte{},
	}
	service := providers.NewService(repo, cipher, providers.NewRegistry(), providers.ProbeEndpoint,
		[]string{"provider.example.test", "127.0.0.1"})
	admin := gateway.NewProviderAdmin(service, newAdminLogger())
	mux := http.NewServeMux()
	mux.Handle("GET /api/providers/available", admin.Available())
	mux.Handle("POST /api/providers", admin.Create())
	mux.Handle("GET /api/providers", admin.List())
	mux.Handle("GET /api/providers/{id}", admin.Get())
	mux.Handle("PATCH /api/providers/{id}", admin.Update())
	mux.Handle("POST /api/providers/{id}/validate", admin.Validate())
	mux.Handle("POST /api/providers/{id}/sync-models", admin.SyncModels())
	mux.Handle("GET /api/providers/{id}/status", admin.Status())
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
