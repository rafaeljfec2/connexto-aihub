package providers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aihub/internal/providers"
)

func TestGetAntigravityProxyAuthStatus_OK(t *testing.T) {
	var sawPath, sawKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawKey = r.Header.Get("X-Gateway-API-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authenticated":true,"account":"u@example.com","lastCheckedAt":"2026-11-06T12:00:00Z","viabilityGate":"SUPPORTED","limitations":[]}`))
	}))
	t.Cleanup(server.Close)

	repo := newMemoryRepository()
	cipher := testCipher(t)
	service := providers.NewService(repo, cipher, providers.NewRegistry(),
		func(context.Context, string, string) error { return nil },
		[]string{"127.0.0.1"})

	proxyBase := strings.TrimRight(server.URL, "/") + "/v1"
	_, err := service.Create(context.Background(), providers.CreateInput{
		ID:                   providers.AntigravityID,
		Name:                 "Antigravity",
		Type:                 providers.TypeAntigravity,
		Endpoint:             proxyBase,
		ManagementCredential: "gw-test-key",
		Enabled:              true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := service.GetAntigravityProxyAuthStatus(context.Background(), providers.AntigravityID)
	if err != nil {
		t.Fatalf("GetAntigravityProxyAuthStatus = %v", err)
	}
	if got == nil || !got.Authenticated {
		t.Fatalf("authenticated = %v", got)
	}
	if sawPath != "/v1/providers/antigravity/auth/status" {
		t.Fatalf("path = %q", sawPath)
	}
	if sawKey != "gw-test-key" {
		t.Fatalf("gateway key header = %q", sawKey)
	}
}

func TestStartAntigravityProxyAuth_OK(t *testing.T) {
	var sawMethod, sawPath, sawKey string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		sawPath = r.URL.Path
		sawKey = r.Header.Get("X-Gateway-API-Key")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"operationId":"op-1","status":"pending","authUrl":"https://oauth.example/authorize","state":"st","expiresAt":"2026-11-06T13:00:00Z"}`))
	}))
	t.Cleanup(server.Close)

	repo := newMemoryRepository()
	cipher := testCipher(t)
	service := providers.NewService(repo, cipher, providers.NewRegistry(),
		func(context.Context, string, string) error { return nil },
		[]string{"127.0.0.1"})

	proxyBase := strings.TrimRight(server.URL, "/") + "/v1"
	_, err := service.Create(context.Background(), providers.CreateInput{
		ID:                   providers.AntigravityID,
		Name:                 "Antigravity",
		Type:                 providers.TypeAntigravity,
		Endpoint:             proxyBase,
		ManagementCredential: "gw-test-key",
		Enabled:              true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	redir := "https://hub.example/cb"
	got, err := service.StartAntigravityProxyAuth(context.Background(), providers.AntigravityID, &providers.AntigravityProxyAuthStartBody{RedirectURI: &redir})
	if err != nil {
		t.Fatalf("StartAntigravityProxyAuth = %v", err)
	}
	if got.AuthURL == "" {
		t.Fatal("empty auth URL")
	}
	if sawMethod != http.MethodPost || sawPath != "/v1/providers/antigravity/auth/start" {
		t.Fatalf("method/path = %s %q", sawMethod, sawPath)
	}
	if sawKey != "gw-test-key" {
		t.Fatalf("gateway key = %q", sawKey)
	}
	if !strings.Contains(body, "redirectUri") {
		t.Fatalf("body = %q", body)
	}
}
