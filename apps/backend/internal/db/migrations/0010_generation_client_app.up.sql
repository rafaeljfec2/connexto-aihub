ALTER TABLE generation_telemetry
    ADD COLUMN IF NOT EXISTS client_app text
    CHECK (client_app IS NULL OR octet_length(client_app) BETWEEN 1 AND 128);

