CREATE TABLE provider_execution_usage (
    execution_id            text PRIMARY KEY,
    provider_id             text NOT NULL,
    model_id                text NOT NULL,
    input_tokens            bigint,
    output_tokens           bigint,
    total_tokens            bigint,
    cached_input_tokens     bigint,
    provider_reported_cost  numeric(24, 12),
    currency                text,
    usage_status            text NOT NULL CHECK (usage_status IN ('reported', 'unavailable', 'partial', 'invalid')),
    usage_source            text NOT NULL,
    recorded_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX provider_execution_usage_recorded_at_idx
    ON provider_execution_usage (recorded_at);

CREATE INDEX provider_execution_usage_provider_recorded_at_idx
    ON provider_execution_usage (provider_id, recorded_at);

CREATE INDEX provider_execution_usage_model_recorded_at_idx
    ON provider_execution_usage (provider_id, model_id, recorded_at);

CREATE INDEX provider_execution_usage_status_recorded_at_idx
    ON provider_execution_usage (usage_status, recorded_at);
