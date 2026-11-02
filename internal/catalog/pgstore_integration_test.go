package catalog

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aihub/internal/db"
	"aihub/internal/db/dbtest"
)

// The conformance suite proves that the PostgreSQL snapshot store is
// equivalent to the versioned JSON file store (ADR-020): the same fixture
// round-trips through both and the loaded states match field by field.

func truncateMicro(value time.Time) time.Time {
	return value.UTC().Truncate(time.Microsecond)
}

func int64ptr(value int64) *int64 { return &value }

func conformanceFixture(t *testing.T) Snapshot {
	t.Helper()

	base := truncateMicro(time.Date(2026, 11, 1, 12, 0, 0, 999000, time.UTC))
	missingSince := truncateMicro(base.Add(2 * time.Hour))

	limits := Limits{Context: int64ptr(131072), Input: int64ptr(131000), Output: int64ptr(8192)}
	modalities := &Modalities{Input: []string{"text", "image"}, Output: []string{"text"}}

	fullEntry := ModelEntry{
		ModelID:     "deepseek-v4.1-flash",
		CanonicalID: "deepseek-ai/deepseek-v4.1-flash",
		Name:        "DeepSeek V4.1 Flash",
		Type:        "chat",
		Family:      "deepseek",
		Modalities:  modalities,
		Limits:      limits,
		Capabilities: map[string]CapabilityClaim{
			capabilityTextGeneration:   supportedClaim(ConfidenceExplicit),
			capabilityStreaming:        supportedClaim(ConfidenceExplicit),
			capabilityFunctionTools:    supportedClaim(ConfidenceDerived),
			capabilityCustomTools:      unsupportedClaim(),
			capabilityReasoning:        supportedClaim(ConfidenceDerived),
			capabilityImageInput:       unsupportedClaim(),
			capabilityAudioInput:       unknownClaim(),
			capabilityAudioOutput:      unknownClaim(),
			capabilityStructuredOutput: unknownClaim(),
		},
		ProviderMetadata: json.RawMessage(`{"owned_by":"opencode","context_length":131072,"extra":["a","b"]}`),
		FirstSeenAt:      base,
		LastSeenAt:       base,
	}

	minimalEntry := ModelEntry{
		ModelID: "kimi-k2.5-x",
		Name:    "Kimi K2.5 X",
		Capabilities: map[string]CapabilityClaim{
			capabilityTextGeneration:   supportedClaim(ConfidenceExplicit),
			capabilityStreaming:        supportedClaim(ConfidenceExplicit),
			capabilityFunctionTools:    unknownClaim(),
			capabilityCustomTools:      unknownClaim(),
			capabilityReasoning:        unknownClaim(),
			capabilityImageInput:       unknownClaim(),
			capabilityAudioInput:       unknownClaim(),
			capabilityAudioOutput:      unknownClaim(),
			capabilityStructuredOutput: unknownClaim(),
		},
		FirstSeenAt:  base,
		LastSeenAt:   base,
		MissingSince: &missingSince,
	}

	snapshot := Snapshot{
		Version: SnapshotVersion,
		SavedAt: missingSince,
		Providers: map[string]ProviderSnapshot{
			"opencode-go": {
				Sync: ProviderSyncState{
					LastAttemptAt:  &missingSince,
					LastSuccessAt:  &base,
					Outcome:        SyncOutcomeSuccess,
					ErrorClass:     SyncErrorNone,
					UpstreamStatus: 200,
					Counts:         SyncCounts{Discovered: 34, Added: 22, Updated: 9, Unchanged: 2, Missing: 1},
				},
				Models: []ModelEntry{fullEntry, minimalEntry},
			},
			"openai": {
				Sync: ProviderSyncState{
					LastAttemptAt:  &base,
					Outcome:        SyncOutcomeRejectedEmpty,
					ErrorClass:     SyncErrorEmptyCatalog,
					UpstreamStatus: 200,
				},
				Models: []ModelEntry{},
			},
		},
	}

	// Mirror the file store's Load normalization: model entries carry their
	// provider identity inside the provider snapshot.
	for providerID, provider := range snapshot.Providers {
		for index := range provider.Models {
			provider.Models[index].ProviderID = providerID
		}

		snapshot.Providers[providerID] = provider
	}

	return snapshot
}

func testPoolForCatalog(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// dbtest.Pool skips when AIHUB_TEST_DATABASE_URL is unset and creates a
	// throwaway database for this test; schema migrations are this helper's
	// own responsibility.
	pool := dbtest.Pool(t)

	bootstrapCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, _, err := db.Migrate(bootstrapCtx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// dbtest migrates the schema; reset the catalog data for a reproducible
	// start without touching the bookkeeping tables.
	if _, err := pool.Exec(cleanupCtx, "TRUNCATE providers, models, catalog_metadata"); err != nil {
		t.Fatalf("reset catalog tables: %v", err)
	}

	return pool
}

func freshPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return NewPostgresStore(pool, nil)
}

func findEntry(entries []ModelEntry, modelID string) *ModelEntry {
	for index := range entries {
		if entries[index].ModelID == modelID {
			return &entries[index]
		}
	}

	return nil
}

func assertSnapshotsEqual(t *testing.T, want, got Snapshot) {
	t.Helper()

	if got.Version != want.Version {
		t.Fatalf("version = %d, want %d", got.Version, want.Version)
	}

	if !got.SavedAt.Equal(want.SavedAt) {
		t.Fatalf("saved_at = %v, want %v", got.SavedAt, want.SavedAt)
	}

	if len(got.Providers) != len(want.Providers) {
		t.Fatalf("provider count = %d, want %d", len(got.Providers), len(want.Providers))
	}

	for providerID, wantProvider := range want.Providers {
		gotProvider, present := got.Providers[providerID]
		if !present {
			t.Fatalf("provider %q missing from the loaded snapshot", providerID)
		}

		assertSyncStateEqual(t, providerID, wantProvider.Sync, gotProvider.Sync)

		if len(gotProvider.Models) != len(wantProvider.Models) {
			t.Fatalf("provider %q model count = %d, want %d",
				providerID, len(gotProvider.Models), len(wantProvider.Models))
		}

		for _, wantEntry := range wantProvider.Models {
			gotEntry := findEntry(gotProvider.Models, wantEntry.ModelID)
			if gotEntry == nil {
				t.Fatalf("provider %q is missing model %q", providerID, wantEntry.ModelID)
			}

			assertEntriesEqual(t, *gotEntry, wantEntry)
		}
	}
}

// assertSyncStateEqual compares the provider sync state; time instants are
// compared by value so the PostgreSQL scan timezone never masks equality.
func assertSyncStateEqual(t *testing.T, providerID string, want, got ProviderSyncState) {
	t.Helper()

	if got.Outcome != want.Outcome {
		t.Fatalf("provider %q outcome = %q, want %q", providerID, got.Outcome, want.Outcome)
	}

	if got.ErrorClass != want.ErrorClass {
		t.Fatalf("provider %q error_class = %q, want %q", providerID, got.ErrorClass, want.ErrorClass)
	}

	if got.UpstreamStatus != want.UpstreamStatus {
		t.Fatalf("provider %q upstream_status = %d, want %d", providerID, got.UpstreamStatus, want.UpstreamStatus)
	}

	if got.Counts != want.Counts {
		t.Fatalf("provider %q counts = %+v, want %+v", providerID, got.Counts, want.Counts)
	}

	if !timePointerEqual(got.LastAttemptAt, want.LastAttemptAt) {
		t.Fatalf("provider %q last_attempt_at = %v, want %v", providerID, got.LastAttemptAt, want.LastAttemptAt)
	}

	if !timePointerEqual(got.LastSuccessAt, want.LastSuccessAt) {
		t.Fatalf("provider %q last_success_at = %v, want %v", providerID, got.LastSuccessAt, want.LastSuccessAt)
	}
}

func timePointerEqual(got, want *time.Time) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}

	return got.Equal(*want)
}

func assertEntriesEqual(t *testing.T, got, want ModelEntry) {
	t.Helper()

	if got.ProviderID != want.ProviderID ||
		got.ModelID != want.ModelID ||
		got.CanonicalID != want.CanonicalID ||
		got.Name != want.Name ||
		got.Type != want.Type ||
		got.Family != want.Family {
		t.Fatalf("entry identity/header differs:\n got  %+v\n want %+v", got, want)
	}

	if !reflect.DeepEqual(got.Modalities, want.Modalities) {
		t.Fatalf("model %q modalities:\n got  %+v\n want %+v", want.ModelID, got.Modalities, want.Modalities)
	}

	if !reflect.DeepEqual(got.Capabilities, want.Capabilities) {
		t.Fatalf("model %q capabilities:\n got  %+v\n want %+v", want.ModelID, got.Capabilities, want.Capabilities)
	}

	if !metadataEqual(got.ProviderMetadata, want.ProviderMetadata) {
		t.Fatalf("model %q metadata differs: %s vs %s", want.ModelID, got.ProviderMetadata, want.ProviderMetadata)
	}

	if !reflect.DeepEqual(got.Limits, want.Limits) {
		t.Fatalf("model %q limits:\n got  %+v\n want %+v", want.ModelID, got.Limits, want.Limits)
	}

	if !got.FirstSeenAt.Equal(want.FirstSeenAt) {
		t.Fatalf("model %q first_seen_at = %v, want %v", want.ModelID, got.FirstSeenAt, want.FirstSeenAt)
	}

	if !got.LastSeenAt.Equal(want.LastSeenAt) {
		t.Fatalf("model %q last_seen_at = %v, want %v", want.ModelID, got.LastSeenAt, want.LastSeenAt)
	}

	if !timePointerEqual(got.MissingSince, want.MissingSince) {
		t.Fatalf("model %q missing_since = %v, want %v", want.ModelID, got.MissingSince, want.MissingSince)
	}
}

// metadataEqual compares JSON payloads semantically: JSONB normalizes key
// order and whitespace, so raw bytes may differ entirely.
func metadataEqual(want, got json.RawMessage) bool {
	if len(want) == 0 || len(got) == 0 {
		return len(want) == 0 && len(got) == 0
	}

	var decodedWant, decodedGot any

	if err := json.Unmarshal(want, &decodedWant); err != nil {
		return false
	}

	if err := json.Unmarshal(got, &decodedGot); err != nil {
		return false
	}

	return reflect.DeepEqual(decodedWant, decodedGot)
}

// metadataEqual compares JSON payloads semantically: JSONB normalizes key
// order and whitespace, so raw bytes may differ entirely.
func TestPostgresStoreMatchesFileStoreConformanceIntegration(t *testing.T) {
	pool := testPoolForCatalog(t)
	ctx := context.Background()

	fixture := conformanceFixture(t)

	fileStore := NewStore(t.TempDir(), nil)
	if _, err := fileStore.Save(ctx, fixture); err != nil {
		t.Fatalf("save file snapshot: %v", err)
	}

	fileLoaded, fileResult := fileStore.Load(ctx)
	if fileResult.WritesDisabled || fileResult.QuarantinedPath != "" {
		t.Fatalf("file load degraded unexpectedly: %+v", fileResult)
	}

	pgStore := freshPostgresStore(pool)
	if _, err := pgStore.Save(ctx, fixture); err != nil {
		t.Fatalf("save postgres snapshot: %v", err)
	}

	pgLoaded, _ := pgStore.Load(ctx)
	assertSnapshotsEqual(t, fileLoaded, pgLoaded)
}

func TestPostgresStoreSurvivesRestartIntegration(t *testing.T) {
	pool := testPoolForCatalog(t)
	ctx := context.Background()

	fixture := conformanceFixture(t)

	store := freshPostgresStore(pool)
	if _, err := store.Save(ctx, fixture); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	loaded, _ := NewPostgresStore(pool, nil).Load(ctx)

	assertSnapshotsEqual(t, fixture, loaded)
}

func TestPostgresStoreReplacesRemovedProvidersIntegration(t *testing.T) {
	pool := testPoolForCatalog(t)
	ctx := context.Background()

	store := freshPostgresStore(pool)
	if _, err := store.Save(ctx, conformanceFixture(t)); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	remaining := conformanceFixture(t).Providers["opencode-go"]
	shrunk := newEmptySnapshot()
	shrunk.SavedAt = truncateMicro(time.Now().UTC())
	shrunk.Providers = map[string]ProviderSnapshot{"opencode-go": remaining}

	if _, err := store.Save(ctx, shrunk); err != nil {
		t.Fatalf("save shrunk snapshot: %v", err)
	}

	loaded, _ := store.Load(ctx)
	if _, removed := loaded.Providers["openai"]; removed {
		t.Fatalf("removed provider openai survived the full-replace save")
	}

	if len(loaded.Providers["opencode-go"].Models) == 0 {
		t.Fatalf("remaining provider lost its models")
	}
}
