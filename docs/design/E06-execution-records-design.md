# E06 — Execution Records, Results & Usage — Design

Status: design accepted under the approved SPEC E06 (2026-10-31).
Related: `docs/specs/E06-execution-records.md`, ADR-017, ADR-018.

## 1. Delivery boundaries (DEEPWORK)

The change crosses modules, has three independently shippable boundaries and a
partial delivery would be incoherent, so the work is split as follows. Each
boundary is validated before the next one starts; the independent reviewer runs
once at Phase 7.

| Boundary | Contents | Exit gate |
| --- | --- | --- |
| B1 — records domain | record model, bounds validation, cursor, in-memory index, query filtering | focused `internal/records` model/index tests green |
| B2 — JSONL storage | recovery, append, rotation, retention compaction, corruption isolation, fault seam | focused store/recovery/retention tests green |
| B3 — observation | `FinishedExecution`, `ResultObserver`, manager invocation, bounded queue, single writer, counters, shutdown drain | execution + service race tests green |
| B4 — composition | config, HTTP query API, router wiring, app/lifecycle ordering, docs | route, config, lifecycle tests green; full regression green |

## 2. Package layout

`internal/records` owns the domain and the storage adapter. Files stay focused
and under the repository's 300-line rule:

| File | Responsibility |
| --- | --- |
| `record.go` | schema-v1 `Record`, validation rules, vocabulary |
| `codec.go` | strict JSON encode/decode, LF framing, 16 KiB bound |
| `cursor.go` | opaque cursor encode/decode and tuple comparison |
| `index.go` | in-memory index and deterministic query ordering |
| `query.go` | `Filter`/`Page` types and page-limit constants |
| `store.go` | startup, append orchestration, cap checks, close |
| `segments.go` | segment open/rotate/rollback/close, physical size, `writeAll` |
| `retention.go` | age/count/storage trim and per-segment compaction |
| `convert.go` | projection of `FinishedExecution` onto the record |
| `files.go` | unexported file-operation seam used for fault injection |
| `service.go` | bounded queue, writer goroutine, counters, observer implementation |
| `counters.go` | process-local degradation counters and snapshot |

The HTTP contract lives in `internal/gateway/records_handler.go`; it imports
`internal/records` but not its storage internals.

## 3. Contracts

### 3.1 E05 → E06

```go
type FinishedExecution struct {
    Result    Result
    RequestID string
    Target    Target
    StartedAt time.Time
}

type ResultObserver interface {
    OnExecutionFinished(FinishedExecution)
}
```

`Manager` receives the observer through `execution.WithObserver` and invokes it
once in `execution.finish`, after `logFinished` and before `remove`. A panic is
recovered at that boundary. `StartedAt` is the internal wall clock captured when
the execution was registered; `CompletedAt` is derived as
`StartedAt.Add(Result.Duration)` so no second clock read is introduced.

### 3.2 Records service

```go
type Service struct { /* queue, pending set, store, counters */ }

func New(opts Options) *Service
func (s *Service) Start(ctx context.Context) error
func (s *Service) Stop(ctx context.Context) error
func (s *Service) OnExecutionFinished(execution.FinishedExecution)
func (s *Service) Get(id string) (Record, bool, error)
func (s *Service) List(filter Filter) (Page, error)
func (s *Service) Degraded() bool
func (s *Service) Counters() Counters
```

`Start` runs bounded recovery and starts the writer. `Stop` closes admission,
closes the queue and waits for the writer with the fixed two-second window; it
is idempotent. The service implements `app.Module` so the application runner can
roll it back on startup failure, and is declared before the HTTP/execution
lifecycle module so reverse shutdown flushes records after E05 drains.

### 3.3 Storage invariants

- One writer goroutine owns every mutation of segment state.
- The index is updated only after a complete line write succeeds.
- Duplicate detection covers both pending and retained identifiers.
- An ambiguous append is never retried: rollback to the pre-write offset; if
  rollback fails, writes are disabled and the bytes are left for startup
  recovery.
- File operations go through `fileOps`; the production implementation is
  `osFileOps`.

### 3.4 Query contract

`Filter` carries provider/model/outcome/time bounds, a decoded `Cursor` and the
page limit. `index.page` filters, applies strict tuple comparison for the
cursor, sorts by `completed_at DESC, execution_id DESC`, and returns at most
`limit` rows plus the cursor of the last returned row when more rows exist.

The gateway handler validates parameters before calling the domain, so the
domain assumes validated enum values and time bounds but still defends against
empty identifiers.

## 4. Implementation sequence

1. `record.go`, `codec.go`, `cursor.go`, `index.go`, `query.go` (B1).
2. `files.go`, `store.go`, `retention.go` (B2).
3. `counters.go`, `service.go`; E05 observer contract and invocation (B3).
4. Config fields/validation, gateway handlers, main/lifecycle wiring, docs (B4).

## 5. Test strategy mapped to acceptance criteria

| Boundary | Tests |
| --- | --- |
| B1 | schema validation (required/null/enums/bounds/timestamp equality), strict codec (unknown/missing fields, LF, 16 KiB), cursor encode/decode/tamper, filter and ordering |
| B2 | append round-trip, short write handling, write/truncate/sync/close failure injection, rotation boundary, restart recovery, corrupt tail/middle, truncation failure, age/count/storage trim, permissions |
| B3 | callback placement and at-most-once, panic isolation, nil observer, non-blocking overflow, duplicate rejection, FIFO, concurrent queries under `-race`, shutdown drain and timeout |
| B4 | real-router auth/Host/Origin/invalid parameters, single-record and list semantics, config defaults/ranges/cross-field errors, lifecycle order and flush failure isolation |

## 6. Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| Observer adds latency to execution cleanup | Callback only validates and performs a non-blocking enqueue; all I/O is in the writer |
| A blocked OS write outlives the two-second drain | Report writer state as unknown, never claim cancellation, let process exit end it |
| Corruption loses valid history | Preserve validated prefix, isolate the segment, truncate only after isolation |
| Retention deletes newer data | Remove expired/oldest only, by `completed_at ASC, execution_id ASC`, and never exceed the configured caps silently |
| Schema drift | Strict decoder rejects unknown/missing fields and unsupported versions |
| E05 regression | Additive option and observer call only; full E01–E05 suites remain unchanged |
