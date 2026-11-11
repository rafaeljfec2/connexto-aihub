ALTER TABLE provider_execution_usage
    DROP COLUMN IF EXISTS provider_list_cost;

DELETE FROM schema_migrations WHERE version = 8;
