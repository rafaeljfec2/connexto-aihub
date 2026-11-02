-- 0001_init — PostgreSQL persistence foundation (ADR-020).
--
-- Creates the catalog persistence: providers (provider-scoped sync state)
-- and models (provider-scoped mode identity per ADR-009). Management columns
-- for providers (name, type, endpoint, credentials) arrive with a later
-- migration in the Provider Management phase; executions in the Execution
-- History phase.

CREATE TABLE IF NOT EXISTS providers (
    id               text PRIMARY KEY,
    last_attempt_at  timestamptz,
    last_success_at  timestamptz,
    outcome          text NOT NULL CHECK (outcome IN ('never', 'success', 'rejected_empty', 'failed', 'skipped')),
    error_class      text NOT NULL CHECK (error_class IN (
                         'none', 'provider_disabled', 'provider_unhealthy', 'transport', 'timeout',
                         'http_status', 'invalid_payload', 'empty_catalog', 'persistence')),
    upstream_status  integer NOT NULL CHECK (upstream_status >= 0),
    discovered_count integer NOT NULL DEFAULT 0 CHECK (discovered_count >= 0),
    added_count      integer NOT NULL DEFAULT 0 CHECK (added_count >= 0),
    updated_count    integer NOT NULL DEFAULT 0 CHECK (updated_count >= 0),
    unchanged_count  integer NOT NULL DEFAULT 0 CHECK (unchanged_count >= 0),
    missing_count    integer NOT NULL DEFAULT 0 CHECK (missing_count >= 0)
);

CREATE TABLE IF NOT EXISTS models (
    provider_id       text NOT NULL REFERENCES providers (id) ON DELETE CASCADE,
    model_id          text NOT NULL,
    canonical_id      text,
    name              text NOT NULL,
    type              text,
    family            text,
    modalities        jsonb,
    limits_context    bigint CHECK (limits_context IS NULL OR limits_context > 0),
    limits_input      bigint CHECK (limits_input IS NULL OR limits_input > 0),
    limits_output     bigint CHECK (limits_output IS NULL OR limits_output > 0),
    capabilities      jsonb NOT NULL,
    provider_metadata jsonb,
    first_seen_at     timestamptz NOT NULL,
    last_seen_at      timestamptz NOT NULL,
    missing_since     timestamptz,
    PRIMARY KEY (provider_id, model_id)
);

CREATE TABLE IF NOT EXISTS catalog_metadata (
    id       integer PRIMARY KEY CHECK (id = 1),
    saved_at timestamptz NOT NULL
);
