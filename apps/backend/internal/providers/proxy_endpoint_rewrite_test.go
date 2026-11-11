package providers_test

import (
	"context"
	"testing"

	"aihub/internal/providers"
)

func TestWrapLoopbackProxyEndpoint(t *testing.T) {
	ctx := context.Background()
	baseRepo := newMemoryRepository()

	if _, err := baseRepo.Create(ctx, providers.NewRecord{
		ID:       providers.AntigravityID,
		Name:     "Antigravity",
		Type:     providers.TypeAntigravity,
		Endpoint: "http://127.0.0.1:9090/v1",
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := baseRepo.Create(ctx, providers.NewRecord{
		ID:       providers.CursorID,
		Name:     "Cursor",
		Type:     providers.TypeCursor,
		Endpoint: "http://localhost:9090/v1",
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := baseRepo.Create(ctx, providers.NewRecord{
		ID:       providers.OpenRouterID,
		Name:     "OpenRouter",
		Type:     providers.TypeOpenRouter,
		Endpoint: "http://127.0.0.1:18090/api/v1",
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}

	wrapped := providers.WrapLoopbackProxyEndpoint(baseRepo, "http://gateway:9090/v1")

	antigravity, err := wrapped.Get(ctx, providers.AntigravityID)
	if err != nil {
		t.Fatal(err)
	}
	if antigravity.Endpoint != "http://gateway:9090/v1" {
		t.Fatalf("antigravity endpoint = %q, want http://gateway:9090/v1", antigravity.Endpoint)
	}

	records, err := wrapped.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	endpoints := map[string]string{}
	for _, record := range records {
		endpoints[record.ID] = record.Endpoint
	}
	if endpoints[providers.CursorID] != "http://gateway:9090/v1" {
		t.Fatalf("cursor endpoint = %q, want http://gateway:9090/v1", endpoints[providers.CursorID])
	}
	if endpoints[providers.OpenRouterID] != "http://127.0.0.1:18090/api/v1" {
		t.Fatalf("openrouter endpoint = %q, must remain unchanged", endpoints[providers.OpenRouterID])
	}

	// Persisted repository record remains untouched for host-native execution.
	persisted, err := baseRepo.Get(ctx, providers.AntigravityID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Endpoint != "http://127.0.0.1:9090/v1" {
		t.Fatalf("persisted endpoint = %q, want untouched http://127.0.0.1:9090/v1", persisted.Endpoint)
	}
}

func TestServiceAllowHTTPEndpointValidation(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepository()
	if _, err := repo.Create(ctx, providers.NewRecord{
		ID:       providers.AntigravityID,
		Name:     "Antigravity",
		Type:     providers.TypeAntigravity,
		Endpoint: "http://127.0.0.1:9090/v1",
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}

	wrapped := providers.WrapLoopbackProxyEndpoint(repo, "http://gateway:9090/v1")
	cipher, err := providers.NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}

	registry := providers.NewRegistry()
	service := providers.NewServiceWithProbes(wrapped, cipher, registry, nil, []string{"gateway"})
	service.AllowHTTPEndpoint("http://gateway:9090/v1")

	if _, err := service.Bootstrap(ctx, providers.BootstrapInput{
		ID:       providers.AntigravityID,
		Name:     "Antigravity",
		Type:     providers.TypeAntigravity,
		Endpoint: "http://gateway:9090/v1",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("Bootstrap() with AllowHTTPEndpoint failed: %v", err)
	}

	if _, err := service.Create(ctx, providers.CreateInput{
		ID:       "other-proxy",
		Name:     "Other Proxy",
		Type:     providers.TypeCursor,
		Endpoint: "http://gateway:8080/v1",
		Enabled:  true,
	}); err == nil {
		t.Fatal("expected HTTP on non-registered port to remain rejected")
	}
}
