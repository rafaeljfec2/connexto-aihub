-- 0002_provider_management — provider configuration columns (Phase C1).
--
-- The providers table keeps its catalog sync state and its identity (id,
-- ADR-009). Management columns are added alongside it: a display name, the
-- adapter type (independent of the id), the upstream endpoint, the
-- enabled flag, adapter-specific configuration, the encrypted credential
-- (AES-256-GCM; the key lives outside PostgreSQL) and audit timestamps.
--
-- Sync-only rows written by the catalog keep these columns NULL; the
-- provider module seeds the management values for configured providers.

ALTER TABLE providers
    ADD COLUMN name                 text,
    ADD COLUMN type                 text,
    ADD COLUMN endpoint             text,
    ADD COLUMN enabled              boolean NOT NULL DEFAULT true,
    ADD COLUMN config               jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN credential_ciphertext bytea,
    ADD COLUMN created_at           timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN updated_at           timestamptz NOT NULL DEFAULT now(),
    ADD CONSTRAINT providers_name_unique UNIQUE (name),
    ADD CONSTRAINT providers_name_not_empty CHECK (name IS NULL OR btrim(name) <> ''),
    ADD CONSTRAINT providers_type_not_empty CHECK (type IS NULL OR btrim(type) <> ''),
    ADD CONSTRAINT providers_endpoint_not_empty CHECK (endpoint IS NULL OR btrim(endpoint) <> ''),
    ADD CONSTRAINT providers_config_object CHECK (jsonb_typeof(config) = 'object');
