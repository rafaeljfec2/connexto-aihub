package providers_test

import (
	"context"
	"errors"
	"testing"

	"aihub/internal/providers"
)

func TestAdministrativeEndpointRequiresHTTPSExceptForLoopback(t *testing.T) {
	fixture := newServiceFixture(t)
	ctx := context.Background()
	rejected := []string{"http://provider.example.test/inference/go", "http://10.0.0.5/inference/go"}
	for index, endpoint := range rejected {
		_, err := fixture.service.Create(ctx, providers.CreateInput{
			ID: "plain" + string(rune('a'+index)), Name: "Plain " + string(rune('a'+index)),
			Type: providers.TypeOpenCodeGo, Endpoint: endpoint,
		})
		if !errors.Is(err, providers.ErrInvalid) {
			t.Fatalf("endpoint %q: error = %v, want ErrInvalid", endpoint, err)
		}
	}
	accepted := []string{"https://provider.example.test/inference/go", "http://127.0.0.1:8099", "http://localhost:8099"}
	for index, endpoint := range accepted {
		if _, err := fixture.service.Create(ctx, providers.CreateInput{
			ID: "ok" + string(rune('a'+index)), Name: "Ok " + string(rune('a'+index)),
			Type: providers.TypeOpenCodeGo, Endpoint: endpoint,
		}); err != nil {
			t.Fatalf("endpoint %q rejected: %v", endpoint, err)
		}
	}
}

func TestEndpointHostMustBeInAllowlist(t *testing.T) {
	fixture := newServiceFixture(t)
	ctx := context.Background()
	if _, err := fixture.service.Create(ctx, providers.CreateInput{
		ID: "outsider", Name: "Outsider", Type: providers.TypeOpenCodeGo,
		Endpoint: "https://attacker.example.test/inference/go",
	}); !errors.Is(err, providers.ErrInvalid) {
		t.Fatalf("Create with host outside the allowlist: error = %v, want ErrInvalid", err)
	}
	fixture.bootstrap(t, testSecret, true)
	redirected := "https://attacker.example.test/inference/go"
	if _, err := fixture.service.Update(ctx, providers.OpenCodeGoID, providers.UpdateInput{Endpoint: &redirected}); !errors.Is(err, providers.ErrInvalid) {
		t.Fatalf("Update redirecting the credential: error = %v, want ErrInvalid", err)
	}
	if fixture.adapter.last().BaseURL != testEndpoint {
		t.Fatal("rejected redirect still reached the runtime")
	}
	upper := "https://PROVIDER.example.test/inference/go"
	if _, err := fixture.service.Update(ctx, providers.OpenCodeGoID, providers.UpdateInput{Endpoint: &upper}); err != nil {
		t.Fatalf("allowlisted host in another letter case rejected: %v", err)
	}
}

func TestProviderWithoutAdapterRemainsAdministrable(t *testing.T) {
	fixture := newServiceFixture(t)
	record, err := fixture.service.Create(context.Background(), providers.CreateInput{
		ID: "future-provider", Name: "Future Provider", Type: "future-adapter",
		Endpoint: testEndpoint, Credential: testSecret, Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create provider without adapter: %v", err)
	}
	if !record.HasCredential {
		t.Fatal("administrative configuration lost its protected credential")
	}
	if len(fixture.adapter.applied) != 0 {
		t.Fatal("unbound provider was routed to another runtime")
	}
}

func TestBootstrapIgnoresUnusedEnvironmentEndpointButRechecksPersistedOne(t *testing.T) {
	repo := newMemoryRepository()
	cipher := testCipher(t)
	ctx := context.Background()
	first := providers.NewService(repo, cipher, providers.NewRegistry(),
		func(context.Context, string, string) error { return nil }, []string{"provider.example.test"})
	if _, err := first.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: testEndpoint, Credential: testSecret, Enabled: true,
	}); err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}
	later := providers.NewService(repo, cipher, providers.NewRegistry(),
		func(context.Context, string, string) error { return nil }, []string{"provider.example.test"})
	if _, err := later.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: "https://attacker.example.test", Enabled: true,
	}); err != nil {
		t.Fatalf("unused environment endpoint blocked startup: %v", err)
	}
	narrowed := providers.NewService(repo, cipher, providers.NewRegistry(),
		func(context.Context, string, string) error { return nil }, []string{"other.example.test"})
	if _, err := narrowed.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: testEndpoint, Enabled: true,
	}); !errors.Is(err, providers.ErrInvalid) {
		t.Fatalf("persisted endpoint outside the allowlist: error = %v, want ErrInvalid", err)
	}
}
