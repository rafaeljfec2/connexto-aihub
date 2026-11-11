package providers_test

import (
	"context"
	"errors"
	"testing"

	"aihub/internal/providers"
)

func TestServiceValidatesProviderWithItsRegisteredProbe(t *testing.T) {
	repo := newMemoryRepository()
	var opencodeCalls, openRouterCalls int
	service := providers.NewServiceWithProbes(repo, testCipher(t), providers.NewRegistry(), map[string]providers.ProbeFunc{
		providers.TypeOpenCodeGo: func(context.Context, string, string) error { opencodeCalls++; return nil },
		providers.TypeOpenRouter: func(_ context.Context, endpoint, credential string) error {
			openRouterCalls++
			if endpoint != "https://openrouter.example.test/api/v1" || credential != testSecret {
				t.Errorf("OpenRouter probe received unexpected configuration")
			}
			return nil
		},
	}, []string{"openrouter.example.test"})
	if _, err := service.Bootstrap(context.Background(), providers.BootstrapInput{
		ID: providers.OpenRouterID, Name: "OpenRouter", Type: providers.TypeOpenRouter,
		Endpoint: "https://openrouter.example.test/api/v1", Credential: testSecret, Enabled: true,
	}); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if err := service.Validate(context.Background(), providers.OpenRouterID); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if opencodeCalls != 0 || openRouterCalls != 1 {
		t.Fatalf("probe calls = OpenCode %d, OpenRouter %d", opencodeCalls, openRouterCalls)
	}
}

func TestServiceDoesNotProbeProviderTypeWithoutAdapterSupport(t *testing.T) {
	fixture := newServiceFixture(t)
	if _, err := fixture.service.Create(context.Background(), providers.CreateInput{
		ID: "future-provider", Name: "Future Provider", Type: "future-adapter",
		Endpoint: testEndpoint, Credential: testSecret, Enabled: true,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := fixture.service.Validate(context.Background(), "future-provider"); !errors.Is(err, providers.ErrProbeUnavailable) {
		t.Fatalf("Validate() error = %v, want ErrProbeUnavailable", err)
	}
}
