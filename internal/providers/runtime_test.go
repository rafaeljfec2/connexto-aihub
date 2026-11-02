package providers_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/config"
	"aihub/internal/providers"
)

func newRuntimeClient(t *testing.T, baseURL, apiKey string) *providers.Client {
	t.Helper()

	client, err := providers.NewClient(config.ProviderOpenCodeGo{BaseURL: baseURL, APIKey: apiKey}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client
}

func TestClientEnabledRequiresEnabledStateAndCredential(t *testing.T) {
	client := newRuntimeClient(t, "https://provider.example.test", testSecret)

	if !client.Enabled() {
		t.Fatal("configured client reports disabled")
	}

	if err := client.Reconfigure(providers.Settings{BaseURL: "https://provider.example.test", APIKey: testSecret, Enabled: false}); err != nil {
		t.Fatalf("Reconfigure: %v", err)
	}

	if client.Enabled() {
		t.Fatal("disabled configuration still reports enabled")
	}

	if err := client.Reconfigure(providers.Settings{BaseURL: "https://provider.example.test", APIKey: "", Enabled: true}); err != nil {
		t.Fatalf("Reconfigure: %v", err)
	}

	if client.Enabled() {
		t.Fatal("client without credential reports enabled")
	}
}

func TestClientReconfigureRoutesNewRequestsToTheNewEndpoint(t *testing.T) {
	var hits int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++

		if got := r.Header.Get("Authorization"); got != "Bearer "+testSecret {
			t.Errorf("authorization header mismatch")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newRuntimeClient(t, "https://unused.example.test", "sk-old-0000")

	if err := client.Reconfigure(providers.Settings{BaseURL: server.URL, APIKey: testSecret, Enabled: true}); err != nil {
		t.Fatalf("Reconfigure: %v", err)
	}

	response, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	_ = response.Body.Close()

	if hits != 1 {
		t.Fatalf("new endpoint hits = %d, want 1", hits)
	}
}

func TestClientReconfigureRejectsInvalidEndpointAndKeepsState(t *testing.T) {
	client := newRuntimeClient(t, "https://provider.example.test", testSecret)

	err := client.Reconfigure(providers.Settings{BaseURL: "https://user:pw@provider.example.test", APIKey: "sk-other", Enabled: true})
	if err == nil {
		t.Fatal("endpoint with embedded credentials was accepted at runtime")
	}

	if !client.Enabled() {
		t.Fatal("rejected reconfiguration changed the client state")
	}
}

func TestProbeEndpointSucceedsOnlyWhenUpstreamAcceptsTheCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testSecret {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := providers.ProbeEndpoint(context.Background(), server.URL, testSecret); err != nil {
		t.Fatalf("ProbeEndpoint with valid credential: %v", err)
	}

	err := providers.ProbeEndpoint(context.Background(), server.URL, "sk-wrong-0000")
	if err == nil {
		t.Fatal("ProbeEndpoint accepted a rejected credential")
	}

	if strings.Contains(err.Error(), "sk-wrong-0000") {
		t.Fatalf("probe error leaked the credential: %v", err)
	}
}

func TestManagerReportsRuntimeDisableImmediately(t *testing.T) {
	client := newRuntimeClient(t, "https://provider.example.test", testSecret)
	manager := providers.NewManager(slog.New(slog.NewTextHandler(io.Discard, nil)), client)

	if err := client.Reconfigure(providers.Settings{BaseURL: "https://provider.example.test", APIKey: testSecret, Enabled: false}); err != nil {
		t.Fatalf("Reconfigure: %v", err)
	}

	statuses := manager.Statuses()
	if len(statuses) != 1 || statuses[0].Enabled || statuses[0].State != providers.StateDisabled {
		t.Fatalf("statuses after runtime disable = %+v", statuses)
	}
}

func TestConfigModuleWarnsWhenEnvironmentIsOverriddenWithoutLoggingSecrets(t *testing.T) {
	repo := newMemoryRepository()
	cipher := testCipher(t)

	seed := func(endpoint, credential string) (*bytes.Buffer, error) {
		buffer := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buffer, nil))
		service := providers.NewService(repo, cipher, providers.NewRegistry(),
			func(context.Context, string, string) error { return nil }, []string{"provider.example.test", "other.example.test", "admin.example.test", "127.0.0.1", "localhost"})
		module := providers.NewConfigModule(service, providers.BootstrapInput{
			ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
			Endpoint: endpoint, Credential: credential, Enabled: true,
		}, logger)

		return buffer, module.Start(context.Background())
	}

	if _, err := seed(testEndpoint, testSecret); err != nil {
		t.Fatalf("first start: %v", err)
	}

	quiet, err := seed(testEndpoint, "")
	if err != nil || strings.Contains(quiet.String(), "ignored") {
		t.Fatalf("matching environment warned: %v %q", err, quiet.String())
	}

	overridden, err := seed("https://other.example.test", "sk-env-override-0009")
	if err != nil {
		t.Fatalf("second start: %v", err)
	}

	if !strings.Contains(overridden.String(), "environment provider settings ignored") ||
		strings.Contains(overridden.String(), "sk-env-override-0009") {
		t.Fatalf("log = %q; want a warning without the credential", overridden.String())
	}
}
