# ADR-020 — PostgreSQL persistence foundation: pgx, embedded migrations and catalog persistence modes

- **Status:** accepted (Management Foundation Phase B, 2026-11-02)
- **Related:** ADR-002 (environment configuration), ADR-009 (model identity),
  ADR-010 (catalog snapshot persistence), ADR-017 (records storage), ADR-019

## Context

The Management Foundation designates PostgreSQL as the primary database. The
first bounded slice is the model catalog: its current persistence is the
versioned JSON snapshot of ADR-010 and must remain available for replacement
only after equivalence is validated. The backend so far has zero external Go
dependencies, hand-rolled persistence and a strict local-first lifecycle (ADR-004). Provider/credential management and execution history arrive
in later phases and are out of this migration.

## Decision

1. **Driver.** Use `github.com/jackc/pgx/v5` (native API + `pgxpool`) as the
   only database driver. No ORM. Repositories and stores use explicit,
   typed SQL statements held as constants next to the code that uses them.
2. **Migrations.** Versioned SQL files
   (`internal/db/migrations/NNNN_name.{up,down}.sql`) embedded with
   `go:embed`, applied at startup by a minimal runner:
   - one file = one transaction, recorded in `schema_migrations`;
   - `pg_advisory_lock` serializes concurrent runners;
   - startup fails closed on unreachable server, failed migration or a
     hidden-gap/unknown-newer version error.
   No external migration tool is introduced yet; the SQL files stay portable.
3. **Persistence modes for the catalog.**
   - `postgres`: catalog snapshot is persisted in PostgreSQL
     (`providers`, `models`, `catalog_metadata`, migration `0001_init`).
   - `file`: the ADR-010 JSON snapshot, unchanged.
   - `AIHUB_CATALOG_PERSISTENCE` selects the mode explicitly; the default is
     `postgres` when `AIHUB_DATABASE_URL` is set and `file` otherwise.
   - The file mechanism stays the default when no database is configured:
     local-first operation (and every existing test) keeps working with
     PostgreSQL entirely absent.
4. **Connection policy.** Explicit pool settings from validated environment
   variables: `AIHUB_DATABASE_URL` (URL-form DSN only), `AIHUB_DATABASE_MAX_CONNS`
   (1–64, default 4), `AIHUB_DATABASE_MIN_CONNS` (0–32, must be < max, default 0)
   and `AIHUB_DATABASE_CONNECT_TIMEOUT` (100 ms–60 s, default 5 s). Startup
   in postgres mode connects, migrates and pings with the configured timeout,
   then fails closed: the URL is an explicit operator opt-in, and a silently
   empty catalog is worse than a clear startup error.
5. **Snapshot equivalence.** `catalog` gains a `SnapshotStore` interface with
   two implementations. The PostgreSQL store mirrors the file store contract:
   `Load` never returns an error (it degrades to an empty snapshot with a
   warning) and never disables writes on read failure (transient network
   errors must not suspend synchronization), while `Save` is a single
   full-replace transaction — delete + insert — that can only commit
   atomically, preserving `SavedAt` and every snapshot field.
6. **Readiness.** In postgres mode a `database` readiness check (pooled ping
   with a short timeout) is registered. The E02 stance is unchanged: provider
   health never flips readiness; an operator-configured database is a
   dependency and does.
7. **Schema ownership.** Migration `0001_init` creates `providers`
   (provider-scoped sync state; management columns arrive with Phase C) and
   `models` with primary key `(provider_id, model_id)` per ADR-009. JSONB is
   used only where the closed-vocabulary map, optional modalities and the
   lossless `provider_metadata` genuinely need flexible structure;
   timestamps, limits and counters are typed columns. `executions` and
   `execution_contents` are deliberately deferred to the Phase D migration.

## Rationale

- pgx is the de facto standard PostgreSQL driver for Go, has a small
  dependency surface (two modules) and keeps `database/sql` coupling out.
- A hand-rolled embedded-migrations runner keeps the artifact chain
  reproducible without pulling a general-purpose migration framework, and the
  advisory lock makes a double-scheduler start safe rather than corrupting.
- Full-replace `Save` matches the snapshot semantics the manager already
  encodes (every persist receives the complete accepted state), avoids
  diff/merge complexity and preserves restart recovery trivially.

## Consequences

- `go.mod` gains its first external dependencies; `go test ./...` keeps
  running without a database in file mode and with integration tests skipped
  unless `AIHUB_TEST_DATABASE_URL` is exported.
- Timestamps persist with microsecond precision (PostgreSQL `timestamptz`),
  so snapshot fields that used nanosecond JSON precision lose that precision
  in postgres mode.
- A corrupt catalog state in PostgreSQL cannot be quarantined like a file;
  validation failure loads empty and lets the next accepted sync rewrite the
  store — a documented, intentional divergence from ADR-010.
- Operators opt into DSN secrets through the environment; DSN is redacted in
  logs and errors and the credential never leaves the pool configuration.
