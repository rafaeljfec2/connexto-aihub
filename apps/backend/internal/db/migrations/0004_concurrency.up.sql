CREATE TABLE concurrency_policies (
    id                  bigserial PRIMARY KEY,
    scope               text NOT NULL CHECK (scope IN ('global', 'provider', 'model')),
    provider_id         text,
    model_id            text,
    concurrency_limit   integer NOT NULL CHECK (concurrency_limit > 0),
    enabled             boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT concurrency_policies_scope_fields CHECK (
        (scope = 'global' AND provider_id IS NULL AND model_id IS NULL)
        OR (scope = 'provider' AND provider_id IS NOT NULL AND model_id IS NULL)
        OR (scope = 'model' AND provider_id IS NOT NULL AND model_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX concurrency_policies_global_key
    ON concurrency_policies (scope) WHERE scope = 'global';

CREATE UNIQUE INDEX concurrency_policies_provider_key
    ON concurrency_policies (scope, provider_id) WHERE scope = 'provider';

CREATE UNIQUE INDEX concurrency_policies_model_key
    ON concurrency_policies (scope, provider_id, model_id) WHERE scope = 'model';

CREATE TABLE concurrency_leases (
    execution_id text PRIMARY KEY,
    provider_id  text NOT NULL,
    model_id     text NOT NULL,
    acquired_at  timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);

CREATE INDEX concurrency_leases_expires_at_idx ON concurrency_leases (expires_at);
CREATE INDEX concurrency_leases_provider_idx ON concurrency_leases (provider_id);
CREATE INDEX concurrency_leases_provider_model_idx ON concurrency_leases (provider_id, model_id);
