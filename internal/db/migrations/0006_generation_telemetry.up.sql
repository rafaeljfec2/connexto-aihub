CREATE TABLE generation_telemetry (
    execution_id                text PRIMARY KEY CHECK (execution_id ~ '^[0-9a-f]{32}$'),
    provider_session_id         text CHECK (provider_session_id IS NULL OR octet_length(provider_session_id) <= 256),
    streaming                   boolean,
    provider_latency_ns         bigint CHECK (provider_latency_ns IS NULL OR provider_latency_ns >= 0),
    routing_overhead_ns         bigint CHECK (routing_overhead_ns IS NULL OR routing_overhead_ns >= 0),
    time_to_first_token_ns      bigint CHECK (time_to_first_token_ns IS NULL OR time_to_first_token_ns >= 0),
    output_stream_duration_ns   bigint CHECK (output_stream_duration_ns IS NULL OR output_stream_duration_ns >= 0),
    finish_reason               text CHECK (finish_reason IS NULL OR octet_length(finish_reason) <= 128),
    recorded_at                 timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE generation_provider_events (
    execution_id        text NOT NULL REFERENCES generation_telemetry (execution_id) ON DELETE CASCADE,
    sequence            smallint NOT NULL CHECK (sequence BETWEEN 0 AND 15),
    event_type          text NOT NULL CHECK (event_type IN (
                            'routing_selected', 'upstream_request_started',
                            'upstream_response_received', 'first_output', 'generation_finished'
                        )),
    observed_at         timestamptz NOT NULL,
    provider_id         text NOT NULL CHECK (octet_length(provider_id) BETWEEN 1 AND 128),
    model_id            text NOT NULL CHECK (octet_length(model_id) <= 4096),
    upstream_status     integer CHECK (upstream_status IS NULL OR upstream_status BETWEEN 100 AND 599),
    duration_ns         bigint CHECK (duration_ns IS NULL OR duration_ns >= 0),
    PRIMARY KEY (execution_id, sequence)
);

CREATE INDEX generation_provider_events_observed_at_idx
    ON generation_provider_events (observed_at);
