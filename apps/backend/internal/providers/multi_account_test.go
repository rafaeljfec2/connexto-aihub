package providers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aihub/internal/providers"
)

func TestServiceCreateGeneratesUniqueSlugAndBindsAdapterFactory(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"resp_1"}`))
	}))
	defer upstream.Close()

	repo := newMemoryRepository()
	cipher, err := providers.NewCredentialCipher([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	registry := providers.NewRegistry()
	service := providers.NewServiceWithProbes(repo, cipher, registry, nil, []string{"127.0.0.1"})
	service.AllowHTTPEndpoint(upstream.URL)
	service.SetDefaultEndpoint(providers.OpenRouterID, upstream.URL)
	service.SetAdapterFactory(providers.TypeOpenRouter, func(record providers.Record) (providers.Adapter, error) {
		return providers.NewOpenRouterClient(providers.Settings{
			BaseURL:           record.Endpoint,
			Enabled:           record.Enabled,
			ProviderType:      record.Type,
			LogicalProviderID: record.ID,
		}, nil)
	})

	ctx := context.Background()
	first, err := service.Create(ctx, providers.CreateInput{
		TemplateID: providers.OpenRouterID,
		Name:       "OpenRouter Trabalho",
		Endpoint:   upstream.URL,
		Credential: "sk-or-work",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if first.ID != "openrouter-trabalho" {
		t.Fatalf("first.ID = %q, want openrouter-trabalho", first.ID)
	}

	second, err := service.Create(ctx, providers.CreateInput{
		TemplateID: providers.OpenRouterID,
		Name:       "OpenRouter Trabalho!",
		Endpoint:   upstream.URL,
		Credential: "sk-or-personal",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.ID != "openrouter-trabalho-2" {
		t.Fatalf("second.ID = %q, want openrouter-trabalho-2", second.ID)
	}

	for _, id := range []string{first.ID, second.ID} {
		resp, err := registry.OpenTargetResponses(ctx, providers.ResponsesTarget{
			ProviderID: id,
			ModelID:    "anthropic/claude-haiku-4.5",
		}, []byte(`{}`), "application/json", "")
		if err != nil {
			t.Fatalf("OpenTargetResponses(%s): %v", id, err)
		}
		_ = resp.Body.Close()
	}

	if err := service.Delete(ctx, second.ID); err != nil {
		t.Fatalf("delete second: %v", err)
	}
	if _, bound := registry.Resolve(second.ID); bound {
		t.Fatalf("expected %s to be unbound after delete", second.ID)
	}
	if _, err := service.Get(ctx, second.ID); err != providers.ErrNotFound {
		t.Fatalf("get after delete err = %v, want ErrNotFound", err)
	}
}

