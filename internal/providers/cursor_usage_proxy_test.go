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

func TestStartCursorProxyAuth_OK(t *testing.T) {
	var sawPath, sawKey, hubAccount string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawKey = r.Header.Get("X-Gateway-API-Key")
		hubAccount = r.Header.Get("X-Hub-Account-Id")
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), "acct-cursor") {
			t.Fatalf("expected accountId in body, got %q", string(b))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"authentication_pending","sessionId":"sid-1","expiresAt":"2026-11-06T13:00:00Z","interactionUrl":"http://hub.example/interact"}`))
	}))
	t.Cleanup(server.Close)

	repo := newMemoryRepository()
	cipher := testCipher(t)
	service := providers.NewService(repo, cipher, providers.NewRegistry(),
		func(context.Context, string, string) error { return nil },
		[]string{"127.0.0.1"})

	proxyBase := strings.TrimRight(server.URL, "/") + "/v1"
	_, err := service.Create(context.Background(), providers.CreateInput{
		ID:                   "acct-cursor",
		Name:                 "Cursor",
		Type:                 providers.TypeCursor,
		Endpoint:             proxyBase,
		ManagementCredential: "gw-test-key",
		Enabled:              true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := service.StartCursorProxyAuth(context.Background(), "acct-cursor")
	if err != nil {
		t.Fatalf("StartCursorProxyAuth = %v", err)
	}
	if got.SessionID != "sid-1" {
		t.Fatalf("sessionId = %q", got.SessionID)
	}
	if sawPath != "/v1/providers/cursor/auth" {
		t.Fatalf("path = %q", sawPath)
	}
	if sawKey != "gw-test-key" {
		t.Fatalf("gateway key = %q", sawKey)
	}
	if hubAccount != "acct-cursor" {
		t.Fatalf("X-Hub-Account-Id = %q", hubAccount)
	}
}
