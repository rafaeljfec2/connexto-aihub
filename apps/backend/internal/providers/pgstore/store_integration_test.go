package pgstore_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/config"
	"aihub/internal/db"
	"aihub/internal/db/dbtest"
	"aihub/internal/providers"
	"aihub/internal/providers/pgstore"
)

const (
	testSecret   = "sk-integration-secret-0001"
	testEndpoint = "https://provider.example.test/inference/go"
)

func migratedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool := dbtest.Pool(t)

	if _, _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return pool
}

// newService simulates a process start: every call builds fresh in-process
// state, so only the database carries configuration across the "restart".
func newService(t *testing.T, pool *pgxpool.Pool, adapter providers.Adapter) *providers.Service {
	t.Helper()

	cipher, err := providers.NewCredentialCipher(bytes.Repeat([]byte{0x33}, config.CredentialKeyBytes))
	if err != nil {
		t.Fatalf("NewCredentialCipher: %v", err)
	}

	registry := providers.NewRegistry()
	if adapter != nil {
		registry.Bind(providers.OpenCodeGoID, adapter)
	}

	return providers.NewService(pgstore.New(pool), cipher, registry,
		func(context.Context, string, string) error { return nil }, []string{"provider.example.test", "other.example.test", "admin.example.test", "127.0.0.1", "localhost"})
}

type captureAdapter struct {
	last providers.Settings
	seen bool
}

func (a *captureAdapter) Reconfigure(settings providers.Settings) error {
	a.last = settings
	a.seen = true

	return nil
}

func (a *captureAdapter) Enabled() bool { return a.last.Enabled }

func bootstrapInput(credential string, endpoint string, enabled bool) providers.BootstrapInput {
	return providers.BootstrapInput{
		ID: providers.OpenCodeGoID, Name: "OpenCode Go", Type: providers.TypeOpenCodeGo,
		Endpoint: endpoint, Credential: credential, Enabled: enabled,
	}
}

func TestProviderConfigurationCRUDIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	service := newService(t, pool, nil)

	created, err := service.Create(ctx, providers.CreateInput{
		ID: "opencode-secondary", Name: "OpenCode Secondary", Type: providers.TypeOpenCodeGo,
		Endpoint: testEndpoint, Credential: testSecret, Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !created.HasCredential || !created.Enabled || created.CreatedAt.IsZero() {
		t.Fatalf("created record = %+v", created)
	}

	if _, err := service.Create(ctx, providers.CreateInput{
		ID: "opencode-secondary", Name: "Other", Type: providers.TypeOpenCodeGo, Endpoint: testEndpoint,
	}); !errors.Is(err, providers.ErrConflict) {
		t.Fatalf("duplicate id error = %v, want ErrConflict", err)
	}

	if _, err := service.Create(ctx, providers.CreateInput{
		ID: "opencode-third", Name: "OpenCode Secondary", Type: providers.TypeOpenCodeGo, Endpoint: testEndpoint,
	}); !errors.Is(err, providers.ErrConflict) {
		t.Fatalf("duplicate name error = %v, want ErrConflict", err)
	}

	renamed := "OpenCode Renamed"
	if _, err := service.Update(ctx, "opencode-secondary", providers.UpdateInput{Name: &renamed}); err != nil {
		t.Fatalf("Update name: %v", err)
	}

	before, err := pgstore.New(pool).SealedCredential(ctx, "opencode-secondary")
	if err != nil {
		t.Fatalf("SealedCredential: %v", err)
	}

	rotated := "sk-rotated-integration-0002"
	if _, err := service.Update(ctx, "opencode-secondary", providers.UpdateInput{Credential: &rotated}); err != nil {
		t.Fatalf("Update credential: %v", err)
	}

	after, err := pgstore.New(pool).SealedCredential(ctx, "opencode-secondary")
	if err != nil || bytes.Equal(before, after) || bytes.Contains(after, []byte(rotated)) {
		t.Fatalf("credential rotation not persisted sealed: err %v", err)
	}

	if _, err := service.Disable(ctx, "opencode-secondary"); err != nil {
		t.Fatalf("Disable: %v", err)
	}

	got, err := service.Get(ctx, "opencode-secondary")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != renamed || got.Enabled || !got.HasCredential {
		t.Fatalf("persisted record = %+v", got)
	}

	list, err := service.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %d records, err %v; want 1", len(list), err)
	}

	if _, err := service.Get(ctx, "missing"); !errors.Is(err, providers.ErrNotFound) {
		t.Fatalf("Get missing = %v, want ErrNotFound", err)
	}
}

func TestCredentialIsStoredSealedIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	service := newService(t, pool, nil)

	if _, err := service.Create(ctx, providers.CreateInput{
		ID: "opencode-secondary", Name: "Secondary", Type: providers.TypeOpenCodeGo,
		Endpoint: testEndpoint, Credential: testSecret, Enabled: true,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT credential_ciphertext FROM providers WHERE id = 'opencode-secondary'`).Scan(&raw); err != nil {
		t.Fatalf("read raw column: %v", err)
	}

	if len(raw) == 0 || bytes.Contains(raw, []byte(testSecret)) {
		t.Fatal("credential column holds plaintext or nothing")
	}
}

func TestBootstrapSeedsOnceAndPersistedStateSurvivesRestartIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()

	first := &captureAdapter{}
	if _, err := newService(t, pool, first).Bootstrap(ctx,
		bootstrapInput(testSecret, testEndpoint, true)); err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}

	if first.last.APIKey != testSecret || !first.last.Enabled || first.last.BaseURL != testEndpoint {
		t.Fatalf("first runtime = %+v", first.last)
	}

	adminEndpoint := "https://admin.example.test/inference/go"
	if _, err := newService(t, pool, nil).Update(ctx, providers.OpenCodeGoID,
		providers.UpdateInput{Endpoint: &adminEndpoint}); err != nil {
		t.Fatalf("admin Update: %v", err)
	}

	if _, err := newService(t, pool, nil).Disable(ctx, providers.OpenCodeGoID); err != nil {
		t.Fatalf("admin Disable: %v", err)
	}

	restarted := &captureAdapter{}
	record, err := newService(t, pool, restarted).Bootstrap(ctx,
		bootstrapInput("sk-env-other-0002", testEndpoint, true))
	if err != nil {
		t.Fatalf("restart Bootstrap: %v", err)
	}

	if record.Enabled || record.Endpoint != adminEndpoint {
		t.Fatalf("restart record = %+v, want persisted disabled state and endpoint", record)
	}

	if restarted.last.APIKey != testSecret || restarted.last.Enabled || restarted.last.BaseURL != adminEndpoint {
		t.Fatalf("restart runtime = %+v, want persisted state", restarted.last)
	}

	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM providers WHERE name IS NOT NULL`).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("managed rows = %d, err %v; want exactly one seeded row", rows, err)
	}
}

func TestBootstrapFillsCatalogOnlyRowWithoutOverwritingSyncStateIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `INSERT INTO providers (id, outcome, error_class, upstream_status, last_attempt_at)
		VALUES ('opencode-go', 'success', 'none', 200, now())`); err != nil {
		t.Fatalf("insert catalog-only row: %v", err)
	}

	if _, err := newService(t, pool, nil).Bootstrap(ctx, bootstrapInput(testSecret, testEndpoint, true)); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	var outcome string
	if err := pool.QueryRow(ctx, `SELECT outcome FROM providers WHERE id = 'opencode-go'`).Scan(&outcome); err != nil {
		t.Fatalf("read outcome: %v", err)
	}

	if outcome != "success" {
		t.Fatalf("seeding overwrote catalog sync state: outcome = %q", outcome)
	}
}

func TestUnboundConfigurationsAreNotRoutedAtRuntimeIntegration(t *testing.T) {
	pool := migratedPool(t)
	ctx := context.Background()
	adapter := &captureAdapter{}

	service := newService(t, pool, adapter)
	if _, err := service.Create(ctx, providers.CreateInput{
		ID: "opencode-secondary", Name: "Secondary", Type: providers.TypeOpenCodeGo, Endpoint: testEndpoint,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if adapter.seen {
		t.Fatal("an unbound configuration reached the bound runtime")
	}
}
