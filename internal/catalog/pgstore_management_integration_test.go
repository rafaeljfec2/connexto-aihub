package catalog

import (
	"context"
	"testing"
	"time"
)

// A catalog snapshot save must never touch provider management columns: the
// configuration (name, endpoint, enabled flag, encrypted credential) is owned
// by the provider module (Phase C1).
func TestPostgresStoreLoadExcludesAdministrativeOnlyProvidersIntegration(t *testing.T) {
	pool := testPoolForCatalog(t)
	ctx := context.Background()

	store := freshPostgresStore(pool)
	if _, err := store.Save(ctx, conformanceFixture(t)); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO providers (id, outcome, error_class, upstream_status, name, type, endpoint)
		 VALUES ('admin-only', 'never', 'none', 0, 'Admin Only', 'opencode-go', 'https://provider.invalid')`,
	); err != nil {
		t.Fatalf("insert administrative provider: %v", err)
	}

	loaded, _ := store.Load(ctx)
	if _, listed := loaded.Providers["admin-only"]; listed {
		t.Fatalf("administrative-only provider appeared as catalog state")
	}

	if _, listed := loaded.Providers["opencode-go"]; !listed {
		t.Fatalf("synchronized provider was dropped from the catalog snapshot")
	}
}

func TestPostgresStorePreservesProviderManagementIntegration(t *testing.T) {
	pool := testPoolForCatalog(t)
	ctx := context.Background()

	store := freshPostgresStore(pool)
	if _, err := store.Save(ctx, conformanceFixture(t)); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	sealed := []byte("sealed-credential")
	if _, err := pool.Exec(ctx,
		`UPDATE providers SET name = 'OpenCode Go', type = 'opencode-go',
		        endpoint = 'https://provider.invalid', enabled = false,
		        credential_ciphertext = $1
		  WHERE id = 'opencode-go'`, sealed); err != nil {
		t.Fatalf("seed management columns: %v", err)
	}

	if _, err := store.Save(ctx, conformanceFixture(t)); err != nil {
		t.Fatalf("save snapshot again: %v", err)
	}

	var (
		name, endpoint string
		enabled        bool
		credential     []byte
	)

	err := pool.QueryRow(ctx,
		`SELECT name, endpoint, enabled, credential_ciphertext FROM providers WHERE id = 'opencode-go'`,
	).Scan(&name, &endpoint, &enabled, &credential)
	if err != nil {
		t.Fatalf("read management columns: %v", err)
	}

	if name != "OpenCode Go" || endpoint != "https://provider.invalid" || enabled {
		t.Fatalf("management columns changed: name=%q endpoint=%q enabled=%v", name, endpoint, enabled)
	}

	if string(credential) != string(sealed) {
		t.Fatalf("encrypted credential was altered by a catalog save")
	}
}

// A configured provider absent from a catalog snapshot keeps its management
// columns and only loses its sync state; its models leave with the snapshot.
func TestPostgresStoreResetsAbsentConfiguredProviderIntegration(t *testing.T) {
	pool := testPoolForCatalog(t)
	ctx := context.Background()

	store := freshPostgresStore(pool)
	if _, err := store.Save(ctx, conformanceFixture(t)); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`UPDATE providers SET name = 'OpenCode Go', type = 'opencode-go',
		        endpoint = 'https://provider.invalid', enabled = true
		  WHERE id = 'opencode-go'`); err != nil {
		t.Fatalf("seed management columns: %v", err)
	}

	if _, err := store.Save(ctx, newEmptySnapshot()); err != nil {
		t.Fatalf("save empty snapshot: %v", err)
	}

	var (
		name, endpoint, outcome string
		lastAttempt             *time.Time
		models                  int
	)

	if err := pool.QueryRow(ctx,
		`SELECT name, endpoint, outcome, last_attempt_at FROM providers WHERE id = 'opencode-go'`,
	).Scan(&name, &endpoint, &outcome, &lastAttempt); err != nil {
		t.Fatalf("configured provider was deleted: %v", err)
	}

	if name != "OpenCode Go" || endpoint != "https://provider.invalid" {
		t.Fatalf("management columns changed: name=%q endpoint=%q", name, endpoint)
	}

	if outcome != "never" || lastAttempt != nil {
		t.Fatalf("sync state not reset: outcome=%q last_attempt=%v", outcome, lastAttempt)
	}

	if err := pool.QueryRow(ctx, `SELECT count(*) FROM models WHERE provider_id = 'opencode-go'`).Scan(&models); err != nil || models != 0 {
		t.Fatalf("models of the absent provider = %d, err %v; want 0", models, err)
	}
}
