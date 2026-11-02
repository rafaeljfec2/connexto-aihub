-- 0002_provider_management reverse migration. Drops the management columns of
-- providers; the catalog sync state and the models table are left intact.
-- Provider configuration and encrypted credentials are lost on rollback.

ALTER TABLE providers
    DROP CONSTRAINT IF EXISTS providers_config_object,
    DROP CONSTRAINT IF EXISTS providers_endpoint_not_empty,
    DROP CONSTRAINT IF EXISTS providers_type_not_empty,
    DROP CONSTRAINT IF EXISTS providers_name_not_empty,
    DROP CONSTRAINT IF EXISTS providers_name_unique,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS credential_ciphertext,
    DROP COLUMN IF EXISTS config,
    DROP COLUMN IF EXISTS enabled,
    DROP COLUMN IF EXISTS endpoint,
    DROP COLUMN IF EXISTS type,
    DROP COLUMN IF EXISTS name;
