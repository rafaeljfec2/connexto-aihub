# ADR-017 — Execution records storage: append-only JSONL segments

- **Status:** accepted (E06 design, 2026-10-31)
- **Date:** 2026-10-31
- **Related:** SPEC E06, ADR-010, ADR-013

## Context

E06 must keep a bounded, queryable history of finished executions. The only
persistence precedent is the catalog snapshot (ADR-010): a whole-file rewrite
with atomic rename and quarantine. Rewriting the complete history for every
execution would be quadratic and would make a single bad write threaten the
whole file. The repository has no third-party dependency and one process owns
the data directory.

## Decision

1. Store execution records as **append-only JSONL segments** at
   `{AIHUB_DATA_DIR}/records/executions-%020d.jsonl`. Directory mode `0700`,
   files `0600`. Schema version is the first attribute of every line.
2. Keep an **in-memory index** rebuilt at startup from the valid lines of every
   segment. The index is the only query source; a record becomes query-visible
   only after its complete line write succeeds.
3. Bound the retained set with the settings in SPEC §10.2 (age, count, managed
   bytes, segment size, retention interval). Trim in the single writer
   goroutine, at startup and on the configured interval, and before an append
   that would cross a hard cap.
4. Rotate before an append would make a non-empty active segment exceed the
   configured segment size: sync and close the previous segment, create the next
   one exclusively, then best-effort sync the directory.
5. Retention boundaries inside a segment are compacted through a same-directory
   temporary file: write retained complete lines, sync, close, atomically
   rename, then best-effort sync the directory. A failure before the rename
   leaves the source byte-identical.
6. On the first invalid or incomplete line, isolate the source segment from
   appends and preserve its valid prefix in the index. Truncate the source to
   the last validated LF boundary and sync it. Never copy unvalidated bytes to
   a quarantine file. If truncation or sync cannot be confirmed, leave the
   source untouched where possible, disable writes, serve recovered records and
   report degraded history.
7. Per-record fsync is disabled. Sync the active segment at rotation and at
   orderly writer shutdown; sync the directory after segment creation, source
   truncation, and compaction rename. A successful `Write` means operating
   system acceptance, not stable-media durability.
8. Keep file operations behind an unexported package seam so short writes,
   write/truncate/sync/close/rotation failures and permission denials are
   deterministic in tests. No public multi-backend abstraction is introduced.

## Rationale

- JSONL matches immutable records, is inspectable, needs no dependency and
  keeps corruption localized to one line.
- The in-memory index makes the bounded recent history queryable without a
  database while respecting the single-process, local-first deployment.
- The retention caps bound memory, disk and startup work.

## Consequences

- Startup recovery is bounded by the configured storage caps, not by unbounded
  history; a lower cap configured later is enforced by the startup trim.
- A crash may lose appended bytes since the last successful sync. No
  exactly-once or transactional guarantee is claimed.
- A failed compaction or corrupt segment can make history read-only until an
  operator intervenes; E05 execution continues regardless.
- A future SQLite migration can replace the storage adapter without changing
  the record model, query contract or cursor format.
