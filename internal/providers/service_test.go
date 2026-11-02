package providers_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"aihub/internal/providers"
)

const testEndpoint = "https://provider.example.test/inference/go"

type serviceFixture struct {
	service  *providers.Service
	repo     *memoryRepository
	adapter  *recordingAdapter
	probeErr error
	probed   []string
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()

	fixture := &serviceFixture{repo: newMemoryRepository(), adapter: &recordingAdapter{}}
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, fixture.adapter)

	cipher := testCipher(t)
	fixture.service = providers.NewService(fixture.repo, cipher, registry,
		func(_ context.Context, endpoint, credential string) error {
			fixture.probed = append(fixture.probed, endpoint+"|"+credential)

			return fixture.probeErr
		})

	return fixture
}

func (f *serviceFixture) bootstrap(t *testing.T, credential string, enabled bool) providers.Record {
	t.Helper()

	record, err := f.service.Bootstrap(context.Background(), providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: testEndpoint, Credential: credential, Enabled: enabled,
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	return record
}

func TestCreateRejectsInvalidConfigurations(t *testing.T) {
	fixture := newServiceFixture(t)
	ctx := context.Background()

	cases := map[string]providers.CreateInput{
		"invalid id":           {ID: "Bad ID", Name: "n", Type: providers.TypeOpenCodeGo, Endpoint: testEndpoint},
		"unknown type":         {ID: "p1", Name: "n", Type: "unknown", Endpoint: testEndpoint},
		"endpoint credentials": {ID: "p2", Name: "n", Type: providers.TypeOpenCodeGo, Endpoint: "https://user:pw@host.example"},
		"empty name":           {ID: "p3", Name: "   ", Type: providers.TypeOpenCodeGo, Endpoint: testEndpoint},
		"blank credential":     {ID: "p4", Name: "n", Type: providers.TypeOpenCodeGo, Endpoint: testEndpoint, Credential: "   "},
	}

	for name, input := range cases {
		if _, err := fixture.service.Create(ctx, input); !errors.Is(err, providers.ErrInvalid) {
			t.Fatalf("%s: error = %v, want ErrInvalid", name, err)
		}
	}
}

func TestCreateStoresConfigurationWithoutExposingCredential(t *testing.T) {
	fixture := newServiceFixture(t)

	record, err := fixture.service.Create(context.Background(), providers.CreateInput{
		ID: "opencode-secondary", Name: "OpenCode Secondary", Type: providers.TypeOpenCodeGo,
		Endpoint: testEndpoint, Credential: testSecret, Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !record.HasCredential {
		t.Fatal("record does not report the stored credential")
	}

	stored, err := fixture.repo.SealedCredential(context.Background(), "opencode-secondary")
	if err != nil || bytes.Contains(stored, []byte(testSecret)) {
		t.Fatalf("credential stored in plaintext or unreadable: %v", err)
	}
}

func TestUpdateAndToggleReconfigureTheBoundRuntime(t *testing.T) {
	fixture := newServiceFixture(t)
	ctx := context.Background()
	fixture.bootstrap(t, testSecret, true)

	newEndpoint := "https://other.example.test/inference/go"
	newCredential := "sk-rotated-credential-0002"

	if _, err := fixture.service.Update(ctx, providers.OpenCodeGoID, providers.UpdateInput{
		Endpoint: &newEndpoint, Credential: &newCredential,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	applied := fixture.adapter.last()
	if applied.BaseURL != newEndpoint || applied.APIKey != newCredential || !applied.Enabled {
		t.Fatalf("runtime after update = %+v", applied)
	}

	if _, err := fixture.service.Disable(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	if fixture.adapter.last().Enabled {
		t.Fatal("disabled configuration still enabled at runtime")
	}

	if _, err := fixture.service.Enable(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	if !fixture.adapter.last().Enabled {
		t.Fatal("enabled configuration not applied to runtime")
	}
}

func TestBootstrapKeepsPersistedValuesOnRestart(t *testing.T) {
	fixture := newServiceFixture(t)
	ctx := context.Background()
	fixture.bootstrap(t, testSecret, true)

	if _, err := fixture.service.Update(ctx, providers.OpenCodeGoID, providers.UpdateInput{
		Credential: stringPtr("sk-admin-credential-0003"),
	}); err != nil {
		t.Fatalf("Update credential: %v", err)
	}

	if _, err := fixture.service.Disable(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	restarted := newServiceFixtureWithRepo(t, fixture.repo)
	record, err := restarted.service.Bootstrap(ctx, providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: "https://env.example.test", Credential: testSecret, Enabled: true,
	})
	if err != nil {
		t.Fatalf("Bootstrap after restart: %v", err)
	}

	if record.Enabled {
		t.Fatal("persisted disabled state was overwritten by the environment")
	}

	if record.Endpoint != testEndpoint {
		t.Fatalf("endpoint = %q, want persisted %q", record.Endpoint, testEndpoint)
	}

	applied := restarted.adapter.last()
	if applied.APIKey != "sk-admin-credential-0003" || applied.Enabled {
		t.Fatalf("restarted runtime = %+v, want persisted credential and disabled state", applied)
	}
}

func TestValidateUsesStoredEndpointAndCredentialWithoutChangingState(t *testing.T) {
	fixture := newServiceFixture(t)
	ctx := context.Background()
	fixture.bootstrap(t, testSecret, true)

	if err := fixture.service.Validate(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if len(fixture.probed) != 1 || fixture.probed[0] != testEndpoint+"|"+testSecret {
		t.Fatalf("probe received %v", fixture.probed)
	}
}

func TestValidateReportsMissingCredentialAndUpstreamFailure(t *testing.T) {
	fixture := newServiceFixture(t)
	ctx := context.Background()
	fixture.bootstrap(t, "", true)

	if err := fixture.service.Validate(ctx, providers.OpenCodeGoID); !errors.Is(err, providers.ErrNoCredential) {
		t.Fatalf("Validate without credential = %v, want ErrNoCredential", err)
	}

	fixture.bootstrap(t, testSecret, true)
	fixture.probeErr = errors.New("connectivity check returned status 401")

	if err := fixture.service.Validate(ctx, providers.OpenCodeGoID); err == nil {
		t.Fatal("Validate accepted an upstream failure")
	}
}

func TestAdministrativeOnlyConfigurationIsNotBoundToRuntime(t *testing.T) {
	fixture := newServiceFixture(t)

	if _, err := fixture.service.Create(context.Background(), providers.CreateInput{
		ID: "opencode-secondary", Name: "Secondary", Type: providers.TypeOpenCodeGo,
		Endpoint: testEndpoint, Enabled: false,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if len(fixture.adapter.applied) != 0 {
		t.Fatal("creating an unbound configuration reconfigured the runtime")
	}
}

func TestServiceErrorsDoNotLeakCredentials(t *testing.T) {
	fixture := newServiceFixture(t)

	_, err := fixture.service.Create(context.Background(), providers.CreateInput{
		ID: "p9", Name: "n", Type: "unknown", Endpoint: testEndpoint, Credential: testSecret,
	})
	if err == nil || strings.Contains(err.Error(), testSecret) {
		t.Fatalf("error = %v, want a message without the credential", err)
	}
}

func newServiceFixtureWithRepo(t *testing.T, repo *memoryRepository) *serviceFixture {
	t.Helper()

	fixture := &serviceFixture{repo: repo, adapter: &recordingAdapter{}}
	registry := providers.NewRegistry()
	registry.Bind(providers.OpenCodeGoID, fixture.adapter)
	fixture.service = providers.NewService(repo, testCipher(t), registry,
		func(context.Context, string, string) error { return nil })

	return fixture
}

func stringPtr(value string) *string { return &value }

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
