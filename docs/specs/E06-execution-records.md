# SPEC E06 — Execution Records, Results & Usage

Status: **approved by the developer on 2026-10-31.**

Date: 2026-10-31.

Workflow stage: approved SPEC. Phase 3 — Review completed by explicit developer
approval of the refined and independently reviewed version. Phases 4–7 are
authorized under the project SDD workflow. Commit, push and publication remain
unauthorized.

Discovery basis: `docs/discovery/E06-discovery.md`, verified against commit
`b706aa8` (`feat(execution): add coordinated Responses execution lifecycle`),
clean working tree at the time of writing.

## 1. Objective

Introduce a bounded, persistent history of finished executions produced by the
E05 Execution Manager. E06 stores terminal classification and minimized
operational metadata. Provider-reported usage and billing metadata are not
available as structured input to this MVP; `usage` is always `null`, and any
response observation or parsing is explicitly deferred to a separately
approved future SPEC.

E06 consumes finished executions; it never runs, retries, reroutes, prices, or
authorizes them. E06 performs no local cost calculation. Schema v1 stores no
provider cost or billing figures; any future provider-reported values must be
stored verbatim and must not be presented as AI Hub billing truth.

## 2. Context

E05 gives every execution registered by the manager one immutable terminal
`Result{ExecutionID, Outcome, Phase, Cause, UpstreamStatus, Committed,
Duration}`. `execution.finish` closes the upstream body and joins its reader
before constructing the result; it logs the result and removes the active
registration before `Manager.Execute` returns. The Gateway is the only result
consumer today. No execution history is persisted.

Verified gaps that E06 addresses:

- `Result` does not carry request correlation, target identity, or wall-clock
  start time; those remain in the internal `Command` and `execution`.
- No seam delivers a terminal result to any consumer other than the Gateway.
- `Target.ModelID` is structurally supported but always empty on the current
  passthrough path; model identity is unknowable today without payload parsing
  (forbidden on that path by ADR-014).
- Usage exists only inside forwarded bytes (final `response.completed` SSE event
  or the trailing part of a non-streaming Responses object — spike §2.10);
  nothing parses or stores it.
- The persistence precedent is the catalog JSON snapshot (ADR-010). Its
  whole-file rewrite is unsuitable for per-execution history; ADR-010 says the
  catalog store can be replaced when E06 needs indexed history without changing
  the catalog domain. ADR-010 does not prescribe E06's storage format.

The repository has no third-party dependencies, one data directory
(`AIHUB_DATA_DIR`, one process owner), and an E04 identity model with a single
static gateway principal (no per-user attribution; ADR-013).

## 3. Scope

### 3.1 Positive scope

1. An `internal/records` package for the versioned record model, JSONL
   persistence, bounded in-memory query index, retention, and degradation
   diagnostics.
2. An additive terminal observer contract owned by `internal/execution` and
   wired by `cmd/ai-hub`. The immutable observation contains the existing
   terminal `Result`, request ID, target identity, and execution start time.
   It does not contain the provider session, request payload, response body, or
   credentials.
3. A bounded asynchronous writer: one in-process queue with capacity 256 and
   one append owner. Queue overflow drops the newest record without blocking an
   execution; the loss is counted and logged.
4. Append-only JSONL segments under `AIHUB_DATA_DIR/records`, plus bounded
   startup recovery, rotation, and retention compaction.
5. Authenticated query routes: `GET /v1/records/executions/{execution_id}` and
   `GET /v1/records/executions` with bounded filters and stable cursor
   pagination.
6. `usage: null` for every schema-v1 record. No response tap, SSE parsing,
   response buffering, or provider-specific usage type is implemented in E06.
7. A bounded record-writer drain after E05 execution drain and before provider
   release. A record flush timeout is logged as degradation and does not change
   E05's execution or shutdown result.
8. Unit and fake-upstream tests for observation, storage, query, retention,
   recovery, privacy, and shutdown behavior, including race testing.
9. Design-phase ADRs: ADR-017 for records persistence and ADR-018 for the
   E05-to-E06 observation contract. No ADR is created during this SPEC
   refinement.

### 3.2 Negative scope

- No response tap, SSE/JSON response parsing, usage capture, or population of
  `usage`; schema-v1 records always return `usage: null`.
- No billing, invoicing, pricing engine, currency conversion, or any arithmetic
  on cost fields. Schema v1 does not store provider-reported cost or billing
  fields; a future approved usage schema must define their provenance first.
- No quota, limit, allowance enforcement, or authorization policy (E09).
- No task classification, provider/model selection, routing, retry, or fallback
  (E05/E08 boundaries).
- No aggregation/rollup endpoints, dashboards, metrics export, or OpenTelemetry.
- No storage of provider session IDs, request payloads, response bodies,
  generated content, tool calls, HTTP headers, credentials, provider error
  bodies, or user identity data.
- No durable sessions, no conversation persistence, no cross-turn linkage
  beyond what correlation identifiers carry.
- No SQLite, PostgreSQL, or any new third-party dependency in E06.
- No multi-process data-directory support; same single-owner assumption as E03.
- No changes to E01–E05 execution or HTTP behavior: streaming semantics,
  byte-for-byte passthrough, status contract, error mapping, and provider guard
  remain unchanged. E06 adds the internal observation seam and a bounded
  two-second records flush after the E05 drain; it cannot weaken E05's clean
  shutdown requirements.

## 4. Functional requirements

FR-1. The manager invokes the configured observer at most once for each
registered execution, after the upstream body is closed, the reader is joined,
and the terminal result is immutable.

FR-2. Every schema-v1 record contains exactly the fields in §6. Unknown or
inapplicable values use JSON `null`; E06 invents no placeholders. Values
outside schema bounds are rejected as `invalid_record` and counted; rejection
never affects the E05 result.

FR-3. `model_id` is populated only when it was present in the execution target.
The current E05 passthrough path records `null`.

FR-4. `upstream_status` is present for received response headers, including
forwarded upstream `4xx`/`5xx`, and is `null` if no upstream headers arrived.

FR-5. Schema-v1 `usage` is always `null`. E06 performs no response inspection,
usage parsing, token estimation, cost arithmetic, or provenance assertion.

FR-6. Queries return complete records from the successfully written/recovered
index, apply the filters in §5, sort deterministically, and enforce the bounded
cursor page size. They do not expose queued records before append succeeds.

FR-7. Queue, write, sync, recovery, retention, or shutdown failures never
change an E05 result, its HTTP response, manager state, or provider-stop guard.
Failures have a stable class, an in-process counter, and a redacted structured
log entry. Previously indexed records remain queryable.

FR-8. On corruption, isolate the source segment from appends and preserve its
valid prefix in the query index. Never copy unvalidated raw bytes into a
quarantine file. Truncate the source to the last valid line only after the
segment is isolated. If truncation cannot be confirmed, leave the source
untouched where possible, disable writes, serve recovered valid records, and
continue startup in a degraded state.

FR-9. Retention, rotation, and storage caps follow §10 at startup, at the
configured cleanup interval, and before an append would exceed a hard count or
storage cap. Only expired or oldest records may be removed.

FR-10. After E05 confirms that no registered execution can still invoke the
observer, close observer admission and drain the record queue for at most two
seconds. Flush expiry is a warning and does not fail an otherwise successful
application shutdown. If E05 cleanup itself remains unconfirmed, E05 continues
to report that failure and retains its provider-release guard; E06 does not
claim its writer is closed while callbacks may still arrive.

FR-11. Query routes use the existing E04 authentication, Host/Origin policy,
correlation middleware, and route composition. No public bypass, principal, or
new header is introduced.

FR-12. Every stored field, query response, and record log line obeys §10.1: no
payload, response body, generated content, provider session, HTTP header,
credential, or provider error text is persisted or logged.

### 4.1 Non-functional requirements

NFR-1. Zero third-party dependencies; standard library only.

NFR-2. File I/O never runs on the client-visible response path or the E05
terminal cleanup path. Observer invocation performs a bounded non-blocking
enqueue; one writer goroutine appends.

NFR-3. Concurrency-safe under `go test -race`: at-most-once notification per
execution, duplicate-ID rejection, and concurrent queries during append and
retention updates.

NFR-4. File permissions: records directory `0700`, record files `0600`
(matching ADR-010).

NFR-5. Enqueue never waits for capacity or disk. Record shutdown waits no more
than the fixed two-second flush window; it does not claim to cancel a blocked
operating-system write.

NFR-6. JSONL files stay inspectable; schema version is first-class on every
record line.

NFR-7. Storage load, corruption isolation, and retention failures do not
prevent application startup. Invalid E06 configuration may fail startup.

## 5. Query contract (MVP)

```text
GET /v1/records/executions/{execution_id}
  200 -> ExecutionRecord
  404 -> not_found, when no record with that ID is currently retained
  503 -> records_unavailable, only when the query index itself is unavailable

GET /v1/records/executions
      ?limit=N (default 50, max 200)
      &provider_id=exact
      &model_id=exact (matches stored non-null values only)
      &outcome=completed|failed|canceled|timed_out
      &started_after=RFC3339 (exclusive)
      &started_before=RFC3339 (exclusive)
      &cursor=opaque-page-boundary
  200 -> { "records": [...], "count": N, "next_cursor": string|null,
           "degraded": boolean }
  503 -> records_unavailable, only when no query index is available
```

The list sorts by `completed_at DESC, execution_id DESC`. `count` is the number of records in this page, not a total-history count. `next_cursor` is `null` at the end of the current result set. Its unpadded base64url payload is the canonical UTC `completed_at` timestamp in RFC3339Nano form, one LF separator, and the lowercase hexadecimal `execution_id` of the last row. The decoded payload must be at most 64 bytes and match that exact structure; the cursor is combined with the supplied filters. For descending order, the next page contains only tuples strictly less than the cursor tuple `(completed_at, execution_id)`. It carries no snapshot guarantee: retention can remove old rows between pages, and newly completed rows are read from a fresh first page.

Each query parameter may appear at most once; unknown parameters, malformed
cursor/ID, invalid enum, invalid timestamp, empty or over-limit provider/model filter (`provider_id`: 1–128 UTF-8 bytes; `model_id`: 1–4096 UTF-8 bytes), `started_after >= started_before`, or `limit` outside 1–200 return the existing `400 invalid_request` envelope. Timestamps accept RFC3339 and are normalized to UTC. `started_after` and `started_before` use strict comparisons. Exact provider/model filters are case-sensitive. Aggregation endpoints do not exist in E06.

The single-record response is the complete stored `ExecutionRecord`. A found
record remains queryable when history is degraded. `404` means only that no
matching record exists in the currently retained/queryable history; it does
not assert that the execution never existed. Retention, queue overflow, or a
past process crash can make historical absence unknowable. Ordinary storage
read/corruption failures that yield a partial or empty in-memory index return
the available result with `degraded: true`. `503` is reserved for failure to
construct any usable query index. The list returns
the recovered page and sets `degraded: true` when this process has evidence of
lost records or incomplete recovery; the flag is not a durable completeness
certificate across process restarts.

Both routes are registered through `server.Route` and the same global E04
policy as existing protected routes. E04 continues to own authentication,
Host/Origin validation, correlation IDs, and request logging.

## 6. Data model

```text
ExecutionRecord (schema_version = 1)
  schema_version      int             required; exactly 1 for this schema
  execution_id        string(32 lowercase hex)  required, unique
  request_id          string(8–128) | null  correlation only; null if unavailable
  provider_id         string(1–128 bytes)   required; current value is "opencode-go"
  model_id            string(1–4096 bytes) | null  null on current passthrough path
  started_at          RFC3339Nano UTC required
  completed_at        RFC3339Nano UTC required; started_at + duration_ns
  duration_ns         int64           required; non-negative elapsed nanoseconds
  outcome             completed|failed|canceled|timed_out  required
  phase               opening|forwarding                  required
  cause               string | null   null iff outcome = completed
  upstream_status     int | null      null when no upstream headers arrived
  response_committed  bool            required
  usage               null            required in schema version 1
```

`request_id`, `model_id`, `cause`, and `upstream_status` are JSON null when
unknown or inapplicable; `provider_session_id` is not part of schema version 1.
The current E04 request path supplies a correlation ID and E05 supplies
`provider_id`, but the record contract does not fabricate a value if a future
manager caller lacks correlation metadata. `cause` is null only for
`completed`; non-completed records use the exact E05 cause vocabulary.

A non-null `request_id` must satisfy E04's correlation grammar
(`[A-Za-z0-9._:-]{8,128}`). `provider_id` and `model_id` are valid UTF-8
strings within the byte limits shown above. The encoded JSON object, including
escaping and the trailing LF, must not exceed 16 KiB. A value or encoded record
outside these bounds is rejected by the records boundary, counted as
`invalid_record`, and never changes the E05 result; E06 does not truncate or
normalize identifiers.

`started_at` is the UTC wall-clock copy captured when E05 creates the execution.
`duration_ns` is copied from the immutable E05 result. `completed_at` is derived
as `started_at + duration_ns`, avoiding a second wall-clock read and clock-jump
ambiguity. The model ID remains null until the execution target itself carries
it; E06 does not inspect request payloads or infer a model.

`usage` is explicitly null in every v1 JSON object. There is no separate
`usage_provenance` field: no provenance exists when there is no usage value. A
future approved schema may represent a usage object with its provenance inside
that object and must increment `schema_version`; E06 v1 does not reserve or
populate that shape.

The domain record is independent of JSONL filenames and rotation metadata.
Unknown schema versions are never silently reinterpreted. Schema evolution
requires a new version and explicit compatibility/recovery rules before it is
implemented.

## 7. Contracts

### 7.1 E05 → E06 observation contract (new; ADR-018)

`internal/execution` gains an additive immutable `FinishedExecution` value and
one observer interface. The value contains the existing terminal `Result`,
`request_id`, `Target{ProviderID, ModelID}`, and the UTC wall-clock `started_at`
copied from the registered execution. It omits `ProviderSessionID`, payload,
content type, response body, credentials, and HTTP headers.

The conceptual interface is:

```text
type ResultObserver interface {
    // Called once after E05 cleanup and terminal-result construction.
    // Implementations must return promptly and must not mutate the value.
    OnExecutionFinished(FinishedExecution)
}
```

`Manager.finish` invokes the observer after the body has closed, the sole reader
has joined, and the terminal result has been fixed, but before it removes the
execution from the active registry and before `Manager.Execute` returns the
result to the Gateway. The observer callback is synchronous only for the
bounded enqueue operation; it performs no file I/O and does not wait for queue
capacity. Therefore a successful `Manager.Wait` proves that every completed
callback returned; it does not prove that a queued record reached disk. The
callback runs at most once per registered execution and is not retried. The
observer cannot change the result returned to the Gateway.

The records implementation enqueues using a bounded queue without waiting for
capacity or doing file I/O. Queue metadata synchronization is limited to a
short critical section; the writer releases queue synchronization before any
filesystem operation, so producer progress does not depend on disk progress.
Capacity is 256 records; on overflow it drops the newest notification,
increments `queue_overflow`, and marks history degraded. The callback does not
log each drop; a recorder diagnostic path emits a rate-limited warning outside
the callback, and shutdown reports any still-pending loss count.
This is the explicit backpressure policy: no producer blocking and no unbounded
memory growth. The single writer consumes accepted notifications FIFO in queue
admission order. Concurrent executions have no causal global order; query
ordering is defined independently in §5.

The records service rejects a duplicate `execution_id` already pending or in
the retained index, keeps the first accepted notification, and increments
`duplicate_observation`. E05 remains the primary at-most-once source. A nil
observer is a no-op for isolated E05 callers/tests; production composition
must always inject the records service. If storage is unavailable, the service
counts/drops notifications and keeps E05 running. Degradation counters and
the API `degraded` flag are process-local diagnostics; they are not persisted
and cannot certify that older history is complete after restart.

`Manager` recovers an observer panic at the callback boundary, logs only a
stable observer-failure class and execution correlation, and continues result
publication/removal. It does not retry a panicking callback. Record failures
belong to E06 diagnostics and never become execution failures.

Dependency direction: `internal/execution` defines the interface; records
implements it; `cmd` wires it. `internal/execution` must not import
`internal/records`.

### 7.2 Delivery alternatives and choice

| Alternative | Simplicity | Lifecycle and failure impact | Decision |
| --- | --- | --- | --- |
| Synchronous append in the observer callback | Fewest moving parts and immediate append result | Disk latency and a blocked filesystem call would delay `Manager.Execute` returning and keep the execution registered; persistence failure can be isolated but cannot be bounded by a Go context | Rejected because storage must not gate E05 cleanup or the client response |
| Asynchronous single writer with a bounded queue | Adds one queue, writer, and close/drain contract | Enqueue is non-blocking; queue-full drops the newest notification and marks history degraded; writer I/O remains off E05; drain has a fixed two-second wait | Selected for MVP |

The capacity is 256 records. At the 16 KiB encoded-line limit, the maximum
payload memory represented by a full queue is 4 MiB, excluding small Go object
overhead. Accepted notifications are written FIFO by the order in which
enqueue operations linearize; concurrent completion has no stronger global
ordering guarantee. There is no producer backpressure, retry, or durable queue.
The only shutdown flush boundary is specified in §11; expiry does not prove
that an in-flight operating-system write stopped.

### 7.3 Records storage boundary

`internal/records` owns the record model, query index, and JSONL implementation.
The JSONL file operations remain behind an unexported package-local seam used
for deterministic fault injection. E06 does not introduce a public
multi-backend repository framework for one implementation. The domain model
does not contain filenames, segment numbers, offsets, or JSONL-specific state;
a separately approved SQLite migration can replace the local storage adapter
without changing the record or query contract.

### 7.4 Usage boundary — excluded from E06

The OpenCode Go protocol spike (§2.8 and §2.10) observed a final
`response.completed` event with `usage` in one real integration and documented
an example of OpenCode billing metadata. This is evidence for that observed
provider/model/response, not a universal Responses contract or a guarantee that
all providers, models, errors, or streams return the same fields.

Schema-v1 writes and returns `usage: null` for every record. E06 does not add a
reader tap, inspect SSE event framing, parse non-streaming JSON, retain trailing
bytes, delay forwarding, or alter the opaque E02/E05 response path. A future
usage feature requires a separately approved SPEC and tests proving byte-
identical passthrough for supported cases; provenance must be embedded in the
future usage value as `provider_reported`, not inferred from local calculation.

### 7.5 Record processing facts

The internal processing stages are `observed → queued → appended → synced`, or
one of `dropped_queue_full`, `append_failed`, `sync_uncertain`,
`close_uncertain`, or `lost_at_shutdown`. These are storage diagnostics, never E05 execution states
and never mutable fields on an API-visible record. A record enters the query
index only after a full JSONL line write succeeds. A successful file write is
not a durability confirmation; §8 defines the separate sync boundary.

## 8. Persistence

### 8.1 JSONL and SQLite alternatives

| Concern | Append-only JSONL (MVP choice) | SQLite |
| --- | --- | --- |
| Local-first and dependencies | Standard library, inspectable files, no service | Local file, but requires selecting and maintaining a driver; CGO/build portability and migrations become project concerns |
| Write concurrency | One in-process writer serializes appends; multiple processes are unsupported | Transactions serialize writers and coordinate concurrent readers |
| Lookup and filters | Rebuild a bounded in-memory index at startup; simple scans over at most the configured retained set | Indexed point lookup and filtered ordering are natural |
| Crash and partial writes | A line may be torn; the valid prefix is retained, the source is isolated from writes, and the unvalidated suffix is truncated without a raw copy; no transaction guarantee | Journal/WAL recovery and transactions provide stronger record-level consistency, subject to driver and filesystem guarantees |
| Corruption | Line-level recovery keeps records before the first invalid line queryable; if truncation cannot be confirmed the source remains read-only and writes are disabled | Page-level recovery is delegated to SQLite; corruption tooling and recovery differ |
| Rotation and retention | Explicit segments and compaction are required; operational behavior stays visible | Database compaction and row deletion replace segment management |
| Schema evolution | Every line has schema_version; loaders must reject unsupported versions | Migrations and transactionally upgraded schema are required |
| Memory and query cost | Bounded by 10,000 default records and 128 MiB managed storage; index is rebuilt at startup | Index can keep memory lower for large histories and support larger query ranges |
| Testability | Standard-library fault-injected file operations cover partial writes, rename, sync, and corruption isolation | SQL behavior and driver-specific migration/locking need additional test fixtures |
| Expected AI Hub use | Sufficient for a single local operator and a bounded recent history; no aggregates | Preferable when history grows beyond the caps, multiple processes/readers are required, or indexed analytics become a real requirement |

JSONL is selected for the MVP because the product is a single-process local
gateway, E06 needs a bounded recent history rather than analytics, and the
repository has no database dependency. A SQLite migration is justified only by
measured startup/query cost at the configured cap, a requirement for concurrent
process access, transactional multi-record updates, or a separately approved
aggregation/history horizon that JSONL cannot meet. This SPEC does not add a
public backend interface ahead of that evidence.

### 8.2 Files and line format

- Root: ${AIHUB_DATA_DIR}/records; segment names are
  executions-%020d.jsonl, with increasing numeric sequence and one active
  highest-numbered segment. Corrupt source segments are isolated from writes
  during recovery; E06 never copies their unvalidated bytes to another file.
- Each record is one UTF-8 JSON object encoded from schema-v1 fields, followed
  by exactly one LF byte. A line is at most 16 KiB. Unknown fields, a missing
  required field, invalid enum/value, unsupported schema version, invalid JSON,
  missing final LF, or an over-limit line is invalid.
- The single writer opens the active segment with append semantics. It writes
  the encoded line and newline, handling short writes until complete. Only a
  complete successful write is added to the in-memory index. The append path
  never rewrites the record history.
- If a write fails after writing any bytes, the writer attempts to truncate
  back to the pre-write offset. If rollback succeeds, the failed record is not
  indexed and is not retried; later records may be attempted. If rollback fails,
  the store disables writes for the process and leaves the uncertain bytes for
  startup recovery. An ambiguous write is never blindly retried.
- At runtime, duplicate execution_id values already queued or retained are
  rejected before append. During recovery, the first valid occurrence wins;
  later valid duplicate lines are excluded from the query index and counted as
  duplicate_record without deleting other valid lines.

### 8.3 Startup recovery and corruption isolation

- Enumerate numeric segments oldest to newest and validate complete lines in
  order, rebuilding the retained in-memory index. Recovery respects the age,
  count, and storage caps in §10; no unbounded full-history index is kept.
- On the first invalid or incomplete line in a segment, mark that source
  segment read-only for recovery and retain only records through its last
  validated LF boundary in the query index. Emit a rate-limited diagnostic with
  a stable failure class, segment basename, and byte offset; never log raw
  bytes or field values from the invalid line.
- After isolating the segment from append, truncate it to the last validated
  LF boundary and sync it. No raw quarantine copy is made. If truncation or
  sync cannot be confirmed, leave the source untouched where possible, disable
  all writes for the process, keep the valid prefix queryable, mark history
  degraded, and continue startup. Never append after unvalidated bytes.
- If the process crashes before truncation, recovery repeats from the same
  valid prefix on the next start. A stale compaction temp file is ignored only
  when its source segment remains intact; startup may remove that known
  uncommitted temp file.
- A read error also disables writes and marks history degraded. The application
  starts; the query service returns recovered records and reports degraded
  history as defined in §5.

### 8.4 Rotation, compaction, and durability

- Rotate before a line would make a non-empty active segment exceed the
  configured segment size. Sync and close the prior segment, create the next
  segment with exclusive create and mode 0600, then best-effort sync the
  records directory. A rotation failure rejects that append and degrades the
  record service; it does not affect E05.
- JSONL segments are append-only between retention operations. If a retention
  boundary falls inside a segment, compact only that affected segment through
  a same-directory temporary file: write retained complete lines, sync and
  close the temporary file, then atomically rename it over the segment and
  best-effort sync the directory. Before rename, any failure leaves the source
  byte-identical. After rename, a directory-sync failure means the new segment
  is visible but its crash durability is uncertain.
- A close error on the active segment or compaction temporary file increments
  `close_failed`, marks the writer state unknown, and disables further writes
  for the process. A compaction temp file is never renamed after a close error;
  the original segment remains authoritative. Completed writes already in the
  index remain queryable, with durability determined by the last successful
  file `Sync`.
- Per-record fsync is disabled in the MVP. A complete Write means the line was
  accepted by the operating system and is query-visible after index update; it
  does not mean stable-media durability. Sync the active segment on rotation
  and orderly writer shutdown, and sync the directory after segment creation,
  source truncation, or compaction rename. A successful Sync is the application's
  durability confirmation, subject to filesystem/device guarantees.
- A crash can lose any appended bytes since the last successful file sync and
  may lose a recent directory entry if directory sync failed. No exactly-once,
  transaction, or hardware-level persistence guarantee is claimed.
- Directory permissions are 0700 and segment/temporary files are 0600. If
  permissions cannot be enforced, the record store is unavailable for queries
  and writes, but application startup and E05 execution continue.
- The data directory remains single-process-owned. E06 does not add an
  inter-process lock; concurrent processes using the same directory are
  unsupported.

## 9. Consistency and failure semantics

### 9.1 The four distinct milestones

1. **Execution result:** E05 fixes one immutable terminal result after body
   closure and reader join. This remains the authority for execution outcome.
2. **Observer notification:** E05 invokes the configured observer once before
   registry removal. The callback either enqueues one immutable value, drops it
   under the stated queue/storage policy, or is isolated after panic. Callback
   return is not persistence.
3. **Append/index visibility:** one writer writes a complete JSONL line. Only a
   successful complete write makes it visible to queries in the in-memory
   index. A queued item is not query-visible.
4. **Durability confirmation:** file Sync (and directory Sync after namespace
   changes) is a separate boundary. A successful Write alone only confirms
   operating-system acceptance; even Sync depends on filesystem/device
   guarantees.

### 9.2 At-most-once and loss windows

At-most-once means E05 does not invoke a second observer callback for the same
execution, the recorder rejects a duplicate ID while it is pending or retained,
and the writer does not retry an ambiguous append. Therefore an execution
produces zero or one query-visible record in the retained history. It does not
mean every execution is recorded, durable delivery, or exactly-once processing.
There is no replay/outbox after process restart.

A record may be absent when storage is unavailable at startup, the observer
queue is full, the recorder is closed, a write/rollback/rotation fails, a known
identifier exceeds schema bounds, a process exits before append, the record is
lost between the last successful Sync and a crash, corruption recovery discards
an unvalidated suffix after preserving the valid prefix, or the bounded
shutdown flush expires. These cases increment the corresponding
in-process counter and emit a redacted diagnostic when the logger is available.
The queue can hold at most 256 records; the shutdown flush wait is at most two
seconds. OS filesystem writes themselves are not cancellable by Go context, so
a pathological blocked filesystem operation can remain in the writer until
process exit; E06 never claims it was canceled or cleanly closed.

Persistence errors never alter an E05 result, client response, active registry,
or provider release guard. Existing records in the query index remain
available. Incomplete history is marked degraded. A missing ID returns `404`
when the index is available, meaning only that it is absent from the currently
retained/queryable set; `503` is reserved for a wholly unavailable index.

Concurrent executions are admitted to the FIFO queue in the order their
non-blocking enqueue operations linearize. This is not a claim that the queue
matches wall-clock terminal order across goroutines. API ordering is always
completed_at DESC, execution_id DESC, independent of append order.

### 9.3 Failure handling

| Failure | Record behavior | E05/application behavior |
| --- | --- | --- |
| Queue full or service closed | Drop the notification; increment a cause-specific counter; mark history degraded | Return the unchanged terminal result immediately |
| Duplicate execution ID | Keep first pending/retained value; reject duplicate; increment duplicate_observation | No result change |
| Value outside schema bounds | Drop the record; increment invalid_record; do not truncate or normalize the identifier | No result change |
| Directory/read/permission failure at startup | Keep records recovered so far; disable writes if safety cannot be established; query list reports degraded | Continue application startup and E05 execution |
| Complete-line append failure | Do not index or retry; attempt truncate to prior offset; if rollback fails, disable writes | No response/result change |
| File Sync or directory Sync failure | Keep the successful append query-visible; mark durability uncertain and history degraded | Shutdown continues |
| File Close failure | Keep completed full writes query-visible; increment `close_failed`, mark history degraded and writer state unknown, mark unsynced writes durability-uncertain, disable further writes, and do not retry | Warn and continue shutdown; do not claim the writer or descriptor closed |
| Disk full or storage cap reached | Trim only by §10; if insufficient, drop new record and increment storage_limit_reached | No response/result change |
| Rotation/retention rename failure | Preserve source segment before rename; mark degraded and reject affected write | No response/result change |
| Corrupt or incomplete line | Isolate source from writes; preserve valid prefix; truncate unvalidated suffix without copying raw bytes; disable writes if truncation/sync is unconfirmed | Start degraded; serve recovered history |
| Duplicate valid ID during recovery | First valid line remains in index; skip later duplicate and increment duplicate_record | Start degraded only if the duplicate signals history integrity loss |
| Flush deadline expires | Log pending count and flush_timeout; do not claim all records synced or writer stopped | Continue shutdown without returning a records flush error |

## 10. Privacy, security, and retention

### 10.1 Privacy boundary

- Stored vocabulary is limited to execution/request correlation, provider/model
  target IDs, timing, terminal classification, response-commit status, and
  schema-v1 usage:null.
- Never store or log provider_session_id, prompts, request payloads, response
  bodies, generated content, reasoning, tool calls/arguments/results, HTTP
  headers, IP/Host/Origin, credentials, provider error text, or fabricated
  user/principal identity.
- request_id is correlation only under ADR-013. It is bounded to the existing
  E04 grammar/length where supplied and is not treated as authentication,
  authorization, session identity, or a durable conversation link.
- Query routes inherit E04 bearer authentication and Host/Origin policy.
  Responses contain only stored schema fields and the list degradation marker.
  Logs include stable failure classes and counts, never record contents.
- Directory mode is 0700; JSONL and temporary compaction files are 0600. E06
  creates no raw-content quarantine file. Failure to enforce these modes
  disables record queries/writes while leaving E05 available.

### 10.2 Concrete retention and storage defaults

| Setting | Environment variable | Default | Valid range |
| --- | --- | --- | --- |
| Maximum record age | AIHUB_RECORDS_MAX_AGE | 720h (30 days) | 1h–8760h |
| Maximum record count | AIHUB_RECORDS_MAX_COUNT | 10000 | 1–100000 |
| Managed storage cap | AIHUB_RECORDS_MAX_STORAGE_BYTES | 134217728 (128 MiB) | 16 MiB–1 GiB |
| Segment rotation size | AIHUB_RECORDS_SEGMENT_BYTES | 8388608 (8 MiB) | 64 KiB–32 MiB |
| Retention cleanup interval | AIHUB_RECORDS_RETENTION_INTERVAL | 1h | 1m–24h |

Cross-field validation requires segment size to be at least 64 KiB, at least
16 KiB smaller than the storage cap, and no greater than half that cap. The
16 KiB maximum record line and 256-record observer queue are fixed MVP bounds,
not environment settings. Per-record fsync is disabled; file/directory Sync
occurs at rotation, namespace changes, and orderly shutdown. The record flush
window is fixed at two seconds.

The storage cap counts active and rotated JSONL segments. Corrupt suffix bytes
are never copied to a second file. Compaction temporarily requires at most one
segment of additional temporary space, so the managed physical peak is bounded
by the configured cap plus one segment. If source truncation cannot be
confirmed, the existing segment remains read-only and counts against the cap;
new records are dropped until the storage issue is resolved.

### 10.3 Retention execution

- Run trim once at startup and then on the configured interval in the writer
  goroutine. Before append, trim synchronously within that same background
  writer only if count or storage would cross its hard cap.
- Remove records older than max age first. If count or storage still exceeds
  the cap, remove oldest records by completed_at ASC, execution_id ASC until
  all configured limits are met. No newer record is deleted while an older
  retained record remains eligible for removal.
- A boundary inside a segment uses the atomic per-segment compaction in §8.4.
  The in-memory index changes only after the corresponding file operation is
  committed. A failed trim leaves its source segment intact where possible,
  marks history degraded, and prevents an append that would exceed the hard
  cap. E05 continues regardless.
- Rotated segments remain queryable until their records age out or become the
  oldest rows required to satisfy count/storage caps. Cleanup frequency and
  moment are fixed by the settings above; no dashboard or manual cleanup API is
  added.
- The configured count/age/storage values bound the retained query index.
  Operators choosing smaller limits accept a shorter history; retention is not
  a promise that every record remains available for the full maximum age.

## 11. E05 compatibility and shutdown integration

### 11.1 Compatibility boundary

E05 remains the sole owner of admission to execution, terminal arbitration,
cancellation, timeouts, upstream-body ownership, reader joining, and the
immutable terminal `Result`. E06 observes a finished result and cannot change
E05 state, HTTP status/body/headers, client-visible streaming, or provider
release decisions. E05 continues to own no persistent history.

The additive E05 contract is limited to `FinishedExecution` and
`ResultObserver` in §7.1. It exposes copies of already-held request/target
metadata and `started_at`, and contains no payload, provider session, response
content, credential, or HTTP header. The observer is called after resource
cleanup and result construction, before active deregistration and before
`Execute` returns. This is a deliberate internal E05 contract amendment;
ADR-018 must record it in Design before implementation. This SPEC refinement
does not modify E05 files or claim that ADR-018 already exists.

E02/E03/E04 external contracts remain unchanged. The existing Gateway continues
to supply the direct OpenCode Go target. A future resolved-command caller may
use the same manager and observer, but E06 introduces no routing or model
mapping behavior.

### 11.2 Shutdown order

The integration must preserve the sequence established by ADR-015 and the
current `cmd/ai-hub/lifecycle.go` composition:

1. The HTTP/execution lifecycle closes the manager's admission gate before
   calling `http.Server.Shutdown` with the configured graceful-drain context.
2. Registered executions may finish. A nil return or `ErrCleanupUnconfirmed`
   from `Manager.Wait` means the active registry is empty and every observer
   callback has returned. A context error means the registry did not become
   empty within that wait window.
3. If graceful shutdown or its wait expires, E05 reserves shutdown
   cancellation, cancels upstream contexts, closes bodies, force-closes HTTP
   connections, and waits in its separate cleanup window. E05 reports an error
   and retains the provider-release guard if execution cleanup is unconfirmed.
4. Once `Manager.Wait` establishes an empty registry (including the
   `ErrCleanupUnconfirmed` case, which reports a cleanup error after the
   registry empties), E06 closes observer admission and drains the record queue
   for at most two seconds. This bounded records wait is separate from E05's
   graceful and cleanup windows and may add at most two seconds to shutdown.
5. A successful records drain writes all accepted queued records, syncs and
   closes the active segment, and closes the writer. A timeout or storage
   failure logs a warning and does not change the E05 result, the E05 cleanup
   verdict, or the provider-release guard. It does not claim an in-flight
   filesystem operation was canceled or that the writer stopped.
6. The existing guarded provider module runs after the HTTP/execution
   lifecycle. It releases provider resources only when E05 `Drained()` is
   true. Records flush failure is not an input to that guard.

If E05's final cleanup wait expires while registrations remain, E06 cannot
prove that no observer callback can arrive and must not claim a clean recorder
close. E05 returns the shutdown error and the provider guard keeps the provider
open. Process termination may then discard queued records or leave an
in-progress operating-system write with an unknown outcome. This is an
explicit failure case, not a successful flush.

## 12. Dependencies and roadmap

| Direction | Dependency | Contract |
| --- | --- | --- |
| E06 → E05 | Finished-execution observer and read-only context enrichment | Required additive internal contract; ADR-018 in Design; no behavior change to E05 result or HTTP response |
| E06 → E04 | Existing bearer policy, Host/Origin checks, correlation, and admission | Reuse unchanged; query routes receive no bypass |
| E06 → ADR-010 | Local data directory and file-permission precedent | Reuse the data-directory setting and permissions; do not copy the catalog snapshot rewrite |
| E06 → protocol spike | Observed OpenCode usage shape | Evidence for deferring usage, not a universal contract; no usage type is populated |
| E09 → E06 (future) | Possible history/usage consumer | E06 v1 exposes no usage or user attribution; E09 must not assume either is available |
| E08 → E06 (future) | Resolved execution target and record consumer | E06 records only target metadata supplied to E05; it does not select or rewrite targets |
| E10 → E06 (future) | Possible learning/history consumer | Out of scope; no aggregates or learning contract are introduced |

Recommended delivery order is **E06 before E07**. E06 has no technical dependency on the harness; delivering it first allows E07's integrated harness validation to verify that record capture remains transparent. This is roadmap guidance, not a runtime dependency or a scope addition to E07. E09 eligibility must still constrain any later E08 automatic selection, as recorded by the existing roadmap. E06 introduces no third-party dependency, external service, database, or migration.

## 13. Decision register

These are the resolved positions approved by the developer on 2026-10-31 together with this SPEC. They bind Design and implementation.

| Decision | Candidate position | Rationale / consequence |
| --- | --- | --- |
| Usage observation | Excluded; every schema-v1 record has `usage: null` | Preserves opaque E02/E05 passthrough; no speculative parser or provider-specific assumption |
| Provider session persistence | Excluded from schema v1 and all record logs | Preserves ADR-013 and minimizes sensitive correlation data; request ID remains the record correlation |
| Observer delivery | Synchronous, non-blocking enqueue into one bounded 256-record queue; single FIFO writer | Isolates disk I/O from E05 cleanup; newest event is dropped on overflow and degradation is observable |
| Observer placement | After body close, reader join, and result construction; before registry removal and `Execute` return | Makes E05 `Wait` cover completion of all observer callbacks while retaining one terminal owner |
| Storage | Append-only JSONL segments, in-memory bounded query index, no public backend abstraction | Standard-library fit for a single-process local gateway and bounded recent history |
| Durability | Per-record sync disabled; sync at rotation, namespace changes, and orderly writer close | Keeps write cost bounded and states crash-loss windows honestly |
| Retention | 30 days, 10,000 records, 128 MiB managed data, 8 MiB segments, hourly trim; see §10 for configuration names/ranges | Bounds local disk and index growth with predictable defaults |
| Query API | `GET /v1/records/executions` and `GET /v1/records/executions/{execution_id}` behind E04 | Minimal point lookup and filtered list; no analytics or public bypass |
| Missing ID | `404` means not present in currently retained/queryable history; `503` only when the query index is unavailable | Avoids claiming that an execution never existed after retention or a loss window |
| Shutdown flush | Close observer admission after E05 confirms the active registry is empty; drain for at most two seconds; warn and continue on E06 failure | Preserves E05 ordering and provider guard; does not claim cancellation of blocked filesystem calls |
| Roadmap | Recommend E06 before E07; no technical dependency | Gives E07 the opportunity to validate end-to-end transparency without making E07 part of E06 |
| Schema | Version 1 contains only the fields in §6; future meaning changes require a new version | Prevents silent reinterpretation and speculative reserved fields |

## 14. Invariants

1. A registered execution has one E05 terminal result. E05 invokes the configured observer at most once; E06 creates zero or one retained record because notification, queue, storage, or process failure can lose it. Exactly-once delivery or durability is never claimed.
2. An accepted record is immutable. Schema changes use a new `schema_version` and explicit compatibility/recovery rules.
3. Persisted content is limited to the schema-v1 fields in §6. No payload, response body, generated content, provider session, header, credential, provider error text, or invented user identity is persisted or logged.
4. Observer, queue, append, sync, recovery, retention, or flush failure cannot change the E05 result, client response, manager state, or provider-release guard.
5. `usage` is always null in schema v1. E06 performs no usage inference, cost calculation, currency conversion, pricing, or allowance arithmetic.
6. `internal/execution` defines the observer contract and never imports `internal/records`; `cmd/ai-hub` composes the records implementation.
7. Storage failures degrade record history but do not prevent application startup or E05 request execution. Invalid E06 configuration may fail startup before serving requests.
8. Query routes use the existing E04 policy. `404` only describes the current retained/queryable set; `degraded` is process-local and not a durable completeness claim.
9. Record capture does not read or alter request or response payload bytes. E02/E05 passthrough and streaming remain byte-for-byte unchanged.
10. Record append order is the order of successful serialized writer work; API order is `completed_at DESC, execution_id DESC`. Concurrent terminal events have no stronger wall-clock ordering guarantee.

## 15. Acceptance criteria

1. On the successful path, every execution whose notification is accepted by the queue and whose append succeeds has exactly one record with the same `execution_id`. Concurrent terminal races never create a duplicate. Queue overflow, schema rejection, and append failures produce no query-visible record for the affected execution and increment the documented process-local degradation counter. If append succeeded but `Sync` fails, the record remains query-visible and its durability is marked uncertain.
2. A completed execution record contains every mandatory schema-v1 field; `upstream_status` and `response_committed` match E05's result; timestamps are UTC, `started_at <= completed_at`, and `duration_ns` equals the E05 elapsed duration. The equality `completed_at = started_at + duration_ns` holds at schema precision.
3. Fake-upstream cases for transport failure, opening timeout, client cancellation, shutdown cancellation, inactivity timeout, upstream read failure, and upstream `4xx`/`5xx` preserve E05 outcome/phase/cause/status/committed values exactly. `usage` is null in every case.
4. Upstream `4xx`/`5xx` responses retain their original status and body; E06 does not replace or modify the response.
5. Observer invocation occurs after body closure, reader join, and immutable result construction, at most once, before active deregistration. Observer panic, nil observer in isolated tests, unavailable storage, and queue-full behavior do not alter the returned E05 result. A full queue does not block the execution caller.
6. `go test -race` observer and writer tests prove one writer, FIFO enqueue order, duplicate-ID rejection, concurrent query safety, no query visibility before a complete append, and no mutation of an indexed record.
7. Injected short write, write error, rollback/truncate failure, sync error, close error, disk-full simulation, permission denial, and rotation failure preserve prior complete records, apply the specified degradation behavior, and do not change execution or HTTP results. A completed append remains query-visible after a close error; durability reflects the last successful `Sync`, and further writes stop when writer state is unknown.
8. Startup/restart tests recover valid records from all supported segments; a partial final line and a corrupt middle line preserve the valid prefix without copying the raw suffix. Failed truncation leaves the source unchanged where possible, disables writes, marks history degraded, and still permits startup with recovered records queryable.
9. Rotation occurs before an append would exceed the segment size. Retention tests cover age/count/storage boundaries, preserve records within limits, trim only eligible oldest records, and verify that query results span retained segments. Failed retention does not block E05 and does not permit a write beyond a hard storage cap.
10. Query tests cover ID lookup, missing ID semantics, unavailable index, every filter, invalid/repeated/unknown parameters, page bounds, stable ordering, cursor validation, and traversal without duplicate rows for a fixed dataset. They assert the documented lack of a cross-page snapshot guarantee under concurrent writes or retention.
11. Router tests prove both query routes retain E04 bearer, Host/Origin, correlation, and admission behavior, and that record query responses expose only schema-v1 fields plus the list `degraded` marker.
12. Shutdown tests prove the existing E05 sequence remains gate → server shutdown → manager wait → forced cancellation/close and cleanup wait when needed → records queue close/flush → guarded provider release. Successful flush proves accepted queued records were written, synced, and closed; timeout or close failure emits a warning and never claims the writer stopped. An E05 cleanup failure still prevents provider release regardless of E06.
13. Existing E01–E05 regression suites pass without changing their asserted public behavior. Byte-for-byte streaming tests prove record capture never inspects or changes payload bytes.
14. The implementation gate runs focused and full `-race` tests, `go vet`, `staticcheck`, `golangci-lint`, `go build`, `gofmt -d`, and `git diff --check`; created or modified source files stay within the repository's 300-line rule. No result is claimed by this SPEC refinement.
15. An opt-in real Codex CLI/OpenCode Go run may provide additional manual evidence for the record and query path when credentials are available; it is not a substitute for deterministic tests and is not required to approve this SPEC.

### 15.1 Requirement traceability matrix

| Requirement | Decision | Acceptance criterion | Evidence expected during implementation |
| --- | --- | --- | --- |
| FR-1–FR-4 | One immutable schema-v1 record from the finished value; unknown model/status remain null | AC-1–3 | E05 observer contract tests; record round-trip assertions for every terminal class |
| FR-5 and privacy invariant | `usage` null; provider session and content excluded | AC-3, AC-11, AC-13 | JSON serialization tests; forbidden-field and router response inspection |
| FR-6 | Query retained, confirmed records only; stable cursor order; bounded page size | AC-10–11 | Handler tests over fixed and concurrent datasets; cursor/filter boundary cases |
| FR-7, NFR-2, NFR-3 | Non-blocking observer enqueue; one writer; storage failure isolation | AC-5–7 | Fault-injected writer tests and race tests asserting unchanged E05 result/HTTP response |
| FR-8, NFR-7 | Preserve valid prefix; isolate source from writes; truncate invalid suffix without copying it; degrade to read-only/startup on failure | AC-8 | Corrupt-tail/middle-file restart tests and truncation failure injection |
| FR-9, storage bounds | Fixed defaults and validated limits; trim oldest eligible records | AC-9 | Deterministic clock and file-size tests for age/count/storage/rotation boundaries |
| FR-10 | Flush only after E05 callbacks finish; two-second bound; warning on failure | AC-12 | Lifecycle tests with pending queue, sync failure, deadline expiry, and provider guard assertions |
| FR-11 | Query endpoints use E04 route policy unchanged | AC-11 | Real-router auth, Host/Origin, and admission tests |
| FR-12, NFR-4, NFR-6 | Closed schema, restrictive file modes, schema version per JSONL line | AC-7–8, AC-11 | File permission assertions, line validation, and response/log redaction tests |
| NFR-1, NFR-5 | Standard library only; bounded recorder shutdown wait | AC-12, AC-14 | Dependency review plus timeout-controlled lifecycle tests |
| E05 compatibility | Observer is additive and cannot alter terminal result, streaming, or release guard | AC-2–5, AC-12–13 | Existing E01–E05 regressions plus E05 lifecycle/observer integration tests |

## 16. Test strategy

- **Record model and JSONL:** schema validation, null semantics, round trip, line bounds, duplicate IDs, short writes, rollback, sync/close failures, rotation, compaction, permissions, and restart reconstruction.
- **Recovery and retention:** incomplete tail, malformed middle line, unsupported schema, repeated recovery after a crash, truncation failure/read-only behavior, age/count/storage boundaries, and retained-index consistency.
- **Observer and concurrency:** callback placement and at-most-once behavior; terminal race orderings; panic isolation; nil observer in isolated callers; non-blocking queue overflow; FIFO enqueue sequencing; concurrent queries and retention under `-race`.
- **Fake-upstream/router integration:** every E05 terminal classification, upstream status passthrough, response commitment, query routes and filters, and unchanged E04 admission errors.
- **Shutdown integration:** graceful drain, forced cancel, completed observer callbacks, flush success, flush timeout, sync/close failure, and proof that E06 failures do not alter E05 cleanup errors or the provider-release guard.
- **Regression:** existing E01–E05 test suites remain unchanged and green; byte-exact stream tests exercise the same passthrough with recording enabled.
- No parser, usage tap, complete response buffer, real external service, or provider credential is required for the deterministic test suite.

## 17. Validation and Evidence

No implementation tests or runtime validations were run during SPEC refinement. Implementation evidence must follow this path:

1. Focused: `go test ./internal/records/... ./internal/execution/... ./internal/gateway/... -race` and lifecycle package tests.
2. Full: `go test ./... -count=1 -race`.
3. Quality: `go vet ./...`, `staticcheck ./...`, `golangci-lint run ./...`, `go build ./...`, `gofmt -d`, and `git diff --check`.
4. Manual: opt-in Codex CLI/OpenCode Go validation only when credentials and the service are available; redact secrets and verify no content leakage.
5. Independent Phase 7 `senior-code-review-judge` review with the claim and actual evidence results; correct material findings and rerun affected gates.

### Evidence block

- **Claim:** Every completed E05 execution is offered to the records domain at most once after E05 cleanup; accepted events become immutable queryable records when storage succeeds, while all E05 execution and streaming behavior remains unchanged.
- **Success type:** `test`, `build`, `lint`, and `manual`.
- **Evidence path:** observer/lifecycle race tests; JSONL failure/recovery and retention tests; real-router query tests; E01–E05 regression gates; optional redacted local E2E.
- **Budget:** focused execution/records/gateway/lifecycle tests plus the full Go regression and quality gates; no performance benchmark or live service is required for deterministic acceptance.
- **Affordance:** injected filesystem operations, controlled fake upstream, deterministic clock, queue completion signal, and writer-state signal where current seams cannot expose these outcomes without timing sleeps.

## 18. Residual risks and limitations

| Risk or limitation | Accepted consequence / mitigation |
| --- | --- |
| Queue overflow, storage failure, crash, or flush expiry loses history | At-most-once only; expose process-local counters and degraded diagnostics; no replay or exactly-once claim |
| A filesystem operation blocks beyond the two-second close wait | Go context cannot guarantee cancellation of an in-progress OS file operation; report writer state as unknown, do not claim a clean recorder close, and let process termination end it |
| `degraded` state is not persisted | After restart, prior queue drops or write losses may be unknowable; API describes current process evidence and retained data only |
| Retention removes old history | `404` means absent from the currently retained/queryable set, not that the execution never occurred |
| JSONL filtering scans a bounded in-memory index | Adequate for 10,000 default records; SQLite reconsidered only with measured startup/query pressure, multi-process need, or approved longer history/analytics |
| Per-record sync is disabled | A process or machine crash can lose successful writes since the last sync; directory-entry persistence also depends on successful directory sync and filesystem guarantees |
| One-process ownership is not enforced by a cross-process lock | Multiple processes sharing `AIHUB_DATA_DIR` are unsupported and can corrupt append/rotation assumptions |
| Source truncation fails during corruption recovery | The original unvalidated suffix remains on disk but is neither copied, queried, nor logged; the segment and then the store are read-only until operator remediation, and the bytes count against the storage cap |
| Model identity is missing in the current passthrough path | Record `model_id` remains null until an approved caller supplies target metadata; E06 never parses payloads |
| No provider usage is stored | E06 cannot support token/cost reporting or downstream quotas from usage values; a future usage SPEC must define provider compatibility and byte-fidelity tests |
| Internal observer addition touches E05 lifecycle | ADR-018 and additive E05 tests must prove callback placement, panic isolation, and unchanged cleanup/shutdown behavior before implementation is accepted |

## 19. ADRs for Design

No ADR is created or approved in this SPEC refinement. Design must produce:

- **ADR-017 — Execution records storage:** append-only JSONL, segment and compaction rules, retention, corruption isolation/recovery, file modes, sync/close points, failure semantics, and the reasons SQLite remains deferred.
- **ADR-018 — E05 terminal observation:** immutable finished-execution value, callback placement before registry removal, bounded non-blocking delivery, panic/failure isolation, composition wiring, and shutdown/drain ordering.

There is no ADR-019 in E06 because response usage observation is explicitly outside the MVP and requires a separately approved future SPEC.

## References

- `docs/discovery/E06-discovery.md` (Discovery evidence; approved decisions in this SPEC supersede its open recommendations)
- `docs/specs/E05-execution.md` (approved execution boundary)
- `docs/specs/E02-provider-management.md`, `docs/specs/E03-model-catalog.md`, `docs/specs/E04-gateway-hardening-access-control.md`
- `docs/adr/ADR-010-catalog-snapshot-persistence.md` (data directory and durability precedent; snapshot mechanics are not copied)
- `docs/adr/ADR-013-request-and-session-identity.md` (correlation and provider-session semantics)
- `docs/adr/ADR-014-execution-lifecycle-and-command-contract.md`
- `docs/adr/ADR-015-coordinated-execution-shutdown.md`
- `docs/adr/ADR-016-execution-timeout-and-error-mapping.md`
- `docs/research/opencode-go-protocol-spike.md` §2.8/§2.10 (observed usage shapes; not a universal provider contract)
- `AI-Hub-Product-Architecture-Kickoff.md` (E06/E07/E08/E09/E10 positioning)
