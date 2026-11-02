-- 0001_init reverse migration. Full rollback of the persistence foundation
-- tables; the schema_migrations bookkeeping table itself is dropped too.

DROP TABLE IF EXISTS catalog_metadata;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS schema_migrations;
