ALTER TABLE providers
    DROP COLUMN IF EXISTS management_credential_ciphertext;

DELETE FROM schema_migrations WHERE version = 7;
