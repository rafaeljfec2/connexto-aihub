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

func TestClearAntigravityProxyRuntimeSessions_OK(t *testing.T) {
	var sawMethod, sawPath, sawKey string
	bodyCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawMethod = r.Method
		sawPath = r.URL.Path
		sawKey = r.Header.Get("X-Gateway-API-Key")
		b, _ := io.ReadAll(r.Body)
		bodyCh <- string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cleared":true,"namespace":"antigravity","mapped_acp_ids_count":0,"listed_acp_sessions_count":2,"acp_sessions_closed":2,"acp_close_failures":0,"warnings":[]}`))
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

	res, err := service.ClearAntigravityProxyRuntimeSessions(context.Background(), providers.AntigravityID)
	if err != nil {
		t.Fatalf("ClearAntigravityProxyRuntimeSessions = %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.ListedAcpSessionsCount == nil || *res.ListedAcpSessionsCount != 2 {
		t.Fatalf("listed count = %v", res.ListedAcpSessionsCount)
	}
	if sawMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", sawMethod)
	}
	if sawPath != "/v1/runtime/sessions/clear" {
		t.Fatalf("path = %q, want /v1/runtime/sessions/clear", sawPath)
	}
	if sawKey != "gw-test-key" {
		t.Fatalf("X-Gateway-API-Key = %q", sawKey)
	}
	body := <-bodyCh
	if !strings.Contains(body, `"namespace":"antigravity"`) {
		t.Fatalf("body = %q", body)
	}
}

func TestClearAntigravityProxyRuntimeSessions_RejectsNonAntigravity(t *testing.T) {
	repo := newMemoryRepository()
	cipher := testCipher(t)
	service := providers.NewService(repo, cipher, providers.NewRegistry(),
		func(context.Context, string, string) error { return nil },
		[]string{"127.0.0.1"})

	_, err := service.Create(context.Background(), providers.CreateInput{
		ID:         "cursor",
		Name:       "Cursor",
		Type:       providers.TypeCursor,
		Endpoint:   "http://127.0.0.1:1/v1",
		Credential: strings.Repeat("a", 32),
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = service.ClearAntigravityProxyRuntimeSessions(context.Background(), "cursor")
	if err == nil || !strings.Contains(err.Error(), "session clearing applies only to antigravity") {
		t.Fatalf("error = %v, want invalid type", err)
	}
}
