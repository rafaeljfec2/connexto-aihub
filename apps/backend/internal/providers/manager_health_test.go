package providers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"aihub/internal/providers"
)

func TestServiceValidateUpdatesOnlyTheRequestedProviderState(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepository()
	registry := providers.NewRegistry()
	openCode := &modelListTestAdapter{routedTestAdapter: &routedTestAdapter{}, status: http.StatusOK}
	openRouter := &modelListTestAdapter{routedTestAdapter: &routedTestAdapter{}, status: http.StatusOK}
	registry.Bind(providers.OpenCodeGoID, openCode)
	registry.Bind(providers.OpenRouterID, openRouter)
	manager := providers.NewRegistryManager(nil, registry)
	probeCalls := map[string]int{}
	openCodeFailure := errors.New("sanitized upstream failure")
	service := providers.NewServiceWithProbes(repo, testCipher(t), registry, map[string]providers.ProbeFunc{
		providers.TypeOpenCodeGo: func(context.Context, string, string) error {
			probeCalls[providers.TypeOpenCodeGo]++
			return openCodeFailure
		},
		providers.TypeOpenRouter: func(context.Context, string, string) error {
			probeCalls[providers.TypeOpenRouter]++
			return nil
		},
	}, []string{"opencode.example.test", "openrouter.example.test"})

	for _, input := range []providers.BootstrapInput{
		{ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo, Endpoint: "https://opencode.example.test/api/v1", Credential: "opencode-test-key", Enabled: true},
		{ID: providers.OpenRouterID, Name: "OpenRouter", Type: providers.TypeOpenRouter, Endpoint: "https://openrouter.example.test/api/v1", Credential: "openrouter-test-key", Enabled: true},
	} {
		if _, err := service.Bootstrap(ctx, input); err != nil {
			t.Fatalf("Bootstrap(%s): %v", input.ID, err)
		}
	}
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := service.Validate(ctx, providers.OpenRouterID); err != nil {
		t.Fatalf("Validate OpenRouter: %v", err)
	}
	assertManagerProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnknown)
	assertManagerProviderState(t, manager, providers.OpenRouterID, providers.StateHealthy)
	if probeCalls[providers.TypeOpenCodeGo] != 0 || probeCalls[providers.TypeOpenRouter] != 1 {
		t.Fatalf("explicit validation calls = OpenCode %d, OpenRouter %d", probeCalls[providers.TypeOpenCodeGo], probeCalls[providers.TypeOpenRouter])
	}

	if err := service.Validate(ctx, providers.OpenCodeGoID); !errors.Is(err, openCodeFailure) {
		t.Fatalf("Validate OpenCode error = %v, want upstream failure", err)
	}
	assertManagerProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnhealthy)
	assertManagerProviderState(t, manager, providers.OpenRouterID, providers.StateHealthy)
}

func TestServiceValidatePropagatesCancellationAndUpdatesHealth(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepository()
	registry := providers.NewRegistry()
	adapter := &modelListTestAdapter{routedTestAdapter: &routedTestAdapter{}, status: http.StatusOK}
	registry.Bind(providers.OpenCodeGoID, adapter)
	manager := providers.NewRegistryManager(nil, registry)
	service := providers.NewServiceWithProbes(repo, testCipher(t), registry, map[string]providers.ProbeFunc{
		providers.TypeOpenCodeGo: func(ctx context.Context, _, _ string) error { return ctx.Err() },
	}, []string{"opencode.example.test"})
	if _, err := service.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: "https://opencode.example.test/api/v1", Credential: "opencode-test-key", Enabled: true,
	}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	checkCtx, cancel := context.WithCancel(ctx)
	cancel()

	if err := service.Validate(checkCtx, providers.OpenCodeGoID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate error = %v, want context cancellation", err)
	}
	assertManagerProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnknown)
}

func TestServiceValidatePropagatesTimeoutAndMarksProviderUnhealthy(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepository()
	registry := providers.NewRegistry()
	adapter := &routedTestAdapter{}
	registry.Bind(providers.OpenCodeGoID, adapter)
	manager := providers.NewRegistryManager(nil, registry)
	service := providers.NewServiceWithProbes(repo, testCipher(t), registry, map[string]providers.ProbeFunc{
		providers.TypeOpenCodeGo: func(ctx context.Context, _, _ string) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}, []string{"opencode.example.test"})
	if _, err := service.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: "https://opencode.example.test/api/v1", Credential: "opencode-test-key", Enabled: true,
	}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	if err := service.Validate(checkCtx, providers.OpenCodeGoID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Validate error = %v, want deadline exceeded", err)
	}
	assertManagerProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnhealthy)
}

func TestServiceValidateDoesNotRestoreHealthAfterCredentialRotation(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepository()
	registry := providers.NewRegistry()
	adapter := &routedTestAdapter{}
	registry.Bind(providers.OpenCodeGoID, adapter)
	manager := providers.NewRegistryManager(nil, registry)
	probeEntered := make(chan struct{})
	releaseProbe := make(chan struct{})
	service := providers.NewServiceWithProbes(repo, testCipher(t), registry, map[string]providers.ProbeFunc{
		providers.TypeOpenCodeGo: func(context.Context, string, string) error {
			close(probeEntered)
			<-releaseProbe
			return nil
		},
	}, []string{"opencode.example.test"})
	if _, err := service.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: "https://opencode.example.test/api/v1", Credential: "opencode-old-key", Enabled: true,
	}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	validationDone := make(chan error, 1)
	go func() {
		validationDone <- service.Validate(ctx, providers.OpenCodeGoID)
	}()
	select {
	case <-probeEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("validation probe did not start")
	}
	newCredential := "opencode-rotated-key"
	if _, err := service.Update(ctx, providers.OpenCodeGoID, providers.UpdateInput{Credential: &newCredential}); err != nil {
		t.Fatalf("Update credential: %v", err)
	}
	close(releaseProbe)
	if err := <-validationDone; err != nil {
		t.Fatalf("Validate after rotation: %v", err)
	}
	assertManagerProviderState(t, manager, providers.OpenCodeGoID, providers.StateUnknown)
}

func assertManagerProviderState(t *testing.T, manager *providers.Manager, providerID string, want providers.State) {
	t.Helper()
	for _, status := range manager.Statuses() {
		if status.ID == providerID {
			if status.State != want {
				t.Fatalf("provider %s state = %q, want %q", providerID, status.State, want)
			}
			return
		}
	}
	t.Fatalf("provider %s missing from manager statuses", providerID)
}
