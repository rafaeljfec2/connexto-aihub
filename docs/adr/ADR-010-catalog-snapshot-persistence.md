# ADR-010 — Catalog persistence: versioned JSON snapshot with quarantine

- **Status:** accepted (E03 design, 2026-10-30)
- **Date:** 2026-10-30
- **Related:** E03 model catalog, SPEC E03 rev 5

## Decision

The model catalog is persisted as a **versioned JSON snapshot** at
`{AIHUB_DATA_DIR}/catalog.json` (default XDG data home), directory `0700`, file
`0600`.

- **Atomic replacement:** write a temp file in the same directory, `fsync` it,
  `Close` it, `rename` it over the target, then best-effort `fsync` the
  directory. The target is never opened for truncation.
- **Quarantine:** a corrupt or unsupported snapshot is `rename`d to a uniquely
  named file (`catalog.corrupt-<RFC3339Nano>.json`,
  `catalog.unsupported-v<N>-<RFC3339Nano>.json`) before the application starts
  empty. Quarantine files are never read, rewritten, reused or deleted.
- **Preservation invariants:**
  - **P1:** while a valid snapshot exists, a failure before the committing
    `rename` (create, write, file `fsync`, `Close`, `rename`) leaves it
    byte-identical and loadable.
  - **P2:** a successfully quarantined corrupt snapshot keeps its original bytes
    in quarantine, unaffected by later failures.
- **Failure handling:** a read error or a failed quarantine disables writes for
  the session; no write targets the unreadable or corrupt file.
- **Commit ordering:** accepted catalog state (entries, accepted counters,
  `last_success_at`) is committed to memory only after the snapshot write
  succeeds. If persistence fails, no candidate entries or counters are
  published and `last_success_at` is unchanged. Operational attempt metadata
  (`outcome`, `error_class`, `upstream_status`, `last_attempt_at`) is updated
  in memory to record the failure; it is not durable and may be lost on
  restart.
- **Durability:** atomic visibility comes from `rename`; crash durability
  requires the file `fsync` **and** the directory `fsync`. A directory-`fsync`
  failure is reported as completed-but-not-durable, never as data loss.
- **Concurrency:** a single in-process writer guarded by a mutex. Multiple
  processes sharing a data directory are not supported in E03.

No database and no external dependency are introduced.

## Rationale

- The catalog is small (28 models ≈ 34 KB), derived data, and read whole on
  startup; a single JSON file is inspectable, portable and dependency-free.
- The repository currently has no persistence layer and no external Go
  dependencies; adding SQLite for one derived snapshot would add operational and
  migration cost with no query need.
- Quarantine plus atomic replacement gives a recoverable failure mode without a
  database transaction.
- A versioned envelope allows a future schema change to be detected instead of
  misread.

## Consequences

- Every write rewrites the whole file; acceptable until E06 needs indexed
  execution history, at which point the store can be replaced without changing
  the domain.
- Crash durability depends on filesystem directory `fsync`; the limitation is
  explicit in SPEC §8.3.
- Multi-process access to the same data directory is unsupported; the fixed
  loopback port already makes a second instance unusual.
- Quarantine files accumulate indefinitely; E03 never deletes them.
- The snapshot contains public model metadata and sync state only — never
  credentials, headers, URLs or upstream bodies.
- A sync attempt that fails to persist is recorded in memory as
  `outcome=failed`, `error_class=persistence` (SPEC rev 5, approved amendment
  A1) and is lost on restart; the accepted catalog state remains the last
  persisted one.
