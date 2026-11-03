CREATE TABLE resource_policies (
    id           bigserial PRIMARY KEY,
    scope        text NOT NULL CHECK (scope IN ('global', 'provider', 'model')),
    provider_id  text,
    model_id     text,
    daily_limit  integer NOT NULL CHECK (daily_limit > 0),
    enabled      boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT resource_policies_scope_fields CHECK (
        (scope = 'global' AND provider_id IS NULL AND model_id IS NULL)
        OR (scope = 'provider' AND provider_id IS NOT NULL AND model_id IS NULL)
        OR (scope = 'model' AND provider_id IS NOT NULL AND model_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX resource_policies_global_key
    ON resource_policies (scope) WHERE scope = 'global';

CREATE UNIQUE INDEX resource_policies_provider_key
    ON resource_policies (scope, provider_id) WHERE scope = 'provider';

CREATE UNIQUE INDEX resource_policies_model_key
    ON resource_policies (scope, provider_id, model_id) WHERE scope = 'model';

CREATE TABLE resource_period_counters (
    policy_id      bigint NOT NULL REFERENCES resource_policies (id) ON DELETE CASCADE,
    period_start   date NOT NULL,
    admitted_count integer NOT NULL DEFAULT 0 CHECK (admitted_count >= 0),
    PRIMARY KEY (policy_id, period_start)
);

CREATE TABLE resource_admissions (
    admission_id text PRIMARY KEY,
    provider_id  text NOT NULL,
    model_id     text NOT NULL,
    period_start date NOT NULL,
    allowed      boolean NOT NULL,
    reason       text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX resource_admissions_created_at_idx ON resource_admissions (created_at);
