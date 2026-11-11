package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

func errf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

const (
	postgresDeleteAbsentCatalogProviders = `
DELETE FROM providers WHERE name IS NULL AND NOT (id = ANY($1::text[]))`

	postgresResetAbsentManagedProviders = `
UPDATE providers SET last_attempt_at = NULL, last_success_at = NULL, outcome = 'never',
       error_class = 'none', upstream_status = 0, discovered_count = 0, added_count = 0,
       updated_count = 0, unchanged_count = 0, missing_count = 0
WHERE name IS NOT NULL AND NOT (id = ANY($1::text[]))`

	// The upsert touches only sync-state columns: provider management
	// columns (name, type, endpoint, credentials) belong to the provider
	// module and must survive every catalog snapshot save.
	postgresInsertProvider = `
INSERT INTO providers (id, last_attempt_at, last_success_at, outcome, error_class,
                       upstream_status, discovered_count, added_count, updated_count,
                       unchanged_count, missing_count)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET
    last_attempt_at  = EXCLUDED.last_attempt_at,
    last_success_at  = EXCLUDED.last_success_at,
    outcome          = EXCLUDED.outcome,
    error_class      = EXCLUDED.error_class,
    upstream_status  = EXCLUDED.upstream_status,
    discovered_count = EXCLUDED.discovered_count,
    added_count      = EXCLUDED.added_count,
    updated_count    = EXCLUDED.updated_count,
    unchanged_count  = EXCLUDED.unchanged_count,
    missing_count    = EXCLUDED.missing_count`

	postgresInsertModel = `
INSERT INTO models (provider_id, model_id, canonical_id, name, type, family, modalities,
                    limits_context, limits_input, limits_output, capabilities,
                    provider_metadata, first_seen_at, last_seen_at, missing_since)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
)

// readProviderSnapshot loads the provider sync state rows.
func (s *PostgresStore) readProviderSnapshot(ctx context.Context, tx pgx.Tx) (map[string]ProviderSnapshot, error) {
	rows, err := tx.Query(ctx, postgresSelectProviders)
	if err != nil {
		return nil, errf("read providers: %w", err)
	}

	defer rows.Close()

	providers := make(map[string]ProviderSnapshot)

	for rows.Next() {
		var (
			id                           string
			lastAttemptAt, lastSuccessAt *time.Time
			outcome, errorClass          string
			upstreamStatus               int
			counts                       SyncCounts
		)

		if err := rows.Scan(&id, &lastAttemptAt, &lastSuccessAt, &outcome, &errorClass,
			&upstreamStatus, &counts.Discovered, &counts.Added, &counts.Updated,
			&counts.Unchanged, &counts.Missing); err != nil {
			return nil, errf("scan provider %d: %w", len(providers)+1, err)
		}

		providers[id] = ProviderSnapshot{
			Sync: ProviderSyncState{
				LastAttemptAt:  lastAttemptAt,
				LastSuccessAt:  lastSuccessAt,
				Outcome:        SyncOutcome(outcome),
				ErrorClass:     SyncErrorClass(errorClass),
				UpstreamStatus: upstreamStatus,
				Counts:         counts,
			},
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errf("read providers: %w", err)
	}

	return providers, nil
}

// readModelRows groups the model rows by provider, keeping the deterministic
// order (provider_id, model_id).
func (s *PostgresStore) readModelRows(ctx context.Context, tx pgx.Tx) (map[string][]ModelEntry, error) {
	rows, err := tx.Query(ctx, postgresSelectModels)
	if err != nil {
		return nil, errf("read models: %w", err)
	}

	defer rows.Close()

	models := make(map[string][]ModelEntry)

	for rows.Next() {
		entry, providerID, scanErr := scanModelRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}

		models[providerID] = append(models[providerID], entry)
	}

	if err := rows.Err(); err != nil {
		return nil, errf("read models: %w", err)
	}

	return models, nil
}

func scanModelRow(rows pgx.Rows) (ModelEntry, string, error) {
	var (
		entry                        ModelEntry
		providerID                   string
		canonicalID, modelType       *string
		modelFamily                  *string
		limitsContext, limitsInput   *int64
		limitsOutput                 *int64
		modalitiesJSON, metadataJSON []byte
		capabilitiesJSON             []byte
	)

	if err := rows.Scan(&providerID, &entry.ModelID, &canonicalID, &entry.Name,
		&modelType, &modelFamily, &modalitiesJSON,
		&limitsContext, &limitsInput, &limitsOutput,
		&capabilitiesJSON, &metadataJSON, &entry.FirstSeenAt, &entry.LastSeenAt,
		&entry.MissingSince); err != nil {
		return entry, "", errf("scan model row: %w", err)
	}

	entry.ProviderID = providerID
	entry.CanonicalID = nullableString(canonicalID)
	entry.Type = nullableString(modelType)
	entry.Family = nullableString(modelFamily)
	entry.Limits = Limits{Context: limitsContext, Input: limitsInput, Output: limitsOutput}

	if len(modalitiesJSON) > 0 {
		modalities := Modalities{}
		if err := json.Unmarshal(modalitiesJSON, &modalities); err != nil {
			return entry, providerID, errf("decode modalities for model %q: %w", entry.ModelID, err)
		}

		entry.Modalities = &modalities
	}

	capabilities := make(map[string]CapabilityClaim, len(capabilityVocabulary))
	if err := json.Unmarshal(capabilitiesJSON, &capabilities); err != nil {
		return entry, providerID, errf("decode capabilities for model %q: %w", entry.ModelID, err)
	}

	entry.Capabilities = capabilities

	if len(metadataJSON) > 0 {
		entry.ProviderMetadata = json.RawMessage(metadataJSON)
	}

	entry.FirstSeenAt = entry.FirstSeenAt.UTC()
	entry.LastSeenAt = entry.LastSeenAt.UTC()
	if entry.MissingSince != nil {
		*entry.MissingSince = entry.MissingSince.UTC()
	}

	return entry, providerID, nil
}

// releaseAbsentProviders applies the full-replace semantics to providers that
// the snapshot no longer contains. Catalog-only rows (no management name) are
// deleted. Configured providers keep their management columns and only have
// their sync state reset to the never-synced state.
func (s *PostgresStore) releaseAbsentProviders(ctx context.Context, tx pgx.Tx, providers map[string]ProviderSnapshot) error {
	ids := sortedStringKeys(providers)

	if _, err := tx.Exec(ctx, postgresDeleteAbsentCatalogProviders, ids); err != nil {
		return errf("remove absent providers: %w", err)
	}

	if _, err := tx.Exec(ctx, postgresResetAbsentManagedProviders, ids); err != nil {
		return errf("reset absent managed providers: %w", err)
	}

	return nil
}

func (s *PostgresStore) writeProviders(ctx context.Context, tx pgx.Tx, providers map[string]ProviderSnapshot) error {
	batch := &pgx.Batch{}

	ids := sortedStringKeys(providers)

	for _, id := range ids {
		state := providers[id].Sync
		batch.Queue(postgresInsertProvider, id, state.LastAttemptAt, state.LastSuccessAt,
			string(state.Outcome), string(state.ErrorClass), state.UpstreamStatus,
			state.Counts.Discovered, state.Counts.Added, state.Counts.Updated,
			state.Counts.Unchanged, state.Counts.Missing)
	}

	return s.flushBatch(ctx, tx, batch, "write providers")
}

func (s *PostgresStore) writeModels(ctx context.Context, tx pgx.Tx, providers map[string]ProviderSnapshot) error {
	batch := &pgx.Batch{}

	for _, id := range sortedStringKeys(providers) {
		for index := range providers[id].Models {
			entry := providers[id].Models[index]

			// The snapshot groups models by provider; the stored identity
			// comes from the map key, not from the entry itself.
			entry.ProviderID = id

			args, encodeErr := encodeModelEntry(entry)

			if encodeErr != nil {
				return encodeErr
			}

			batch.Queue(postgresInsertModel, args...)
		}
	}

	return s.flushBatch(ctx, tx, batch, "write models")
}

func encodeModelEntry(entry ModelEntry) ([]any, error) {
	entry.FirstSeenAt = entry.FirstSeenAt.UTC()
	entry.LastSeenAt = entry.LastSeenAt.UTC()

	// The local copy keeps the caller's snapshot untouched; the pointee of
	// MissingSince is shared, so a new value replaces the pointer result.
	missingSince := entry.MissingSince
	if missingSince != nil {
		value := missingSince.UTC()
		missingSince = &value
	}

	capabilitiesJSON, err := rawJSON(entry.Capabilities)
	if err != nil {
		return nil, errf("encode capabilities for model %q: %w", entry.ModelID, err)
	}

	var modalitiesJSON []byte
	if entry.Modalities != nil {
		if modalitiesJSON, err = rawJSON(entry.Modalities); err != nil {
			return nil, errf("encode modalities for model %q: %w", entry.ModelID, err)
		}
	}

	metadataJSON := normalizeProviderMetadata(entry.ProviderMetadata)

	optionalInt64 := func(value *int64) any {
		if value == nil {
			return nil
		}

		return *value
	}

	return []any{
		entry.ProviderID, entry.ModelID, entry.CanonicalID, entry.Name,
		entry.Type, entry.Family, nullableJSON(modalitiesJSON),
		optionalInt64(entry.Limits.Context), optionalInt64(entry.Limits.Input), optionalInt64(entry.Limits.Output),
		capabilitiesJSON, nullableJSON(metadataJSON),
		entry.FirstSeenAt, entry.LastSeenAt, missingSince,
	}, nil
}

func normalizeProviderMetadata(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return nil
	}

	return []byte(raw)
}

func nullableJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}

	return value
}

func (s *PostgresStore) flushBatch(ctx context.Context, tx pgx.Tx, batch *pgx.Batch, operation string) error {
	if batch.Len() == 0 {
		return nil
	}

	results := tx.SendBatch(ctx, batch)

	if err := results.Close(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return errf("%s: %w", operation, err)
	}

	return nil
}

func nullableString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func sortedStringKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
