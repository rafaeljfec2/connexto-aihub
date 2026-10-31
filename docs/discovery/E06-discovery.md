# Discovery E06 — Execution Records, Results & Usage

Status: discovery candidate — awaiting human approval. Nothing in this document
is implemented or approved as a SPEC.

Date: 2026-10-31.

Head at the time of this discovery: `b706aa8` (`feat(execution): add
coordinated Responses execution lifecycle`), on `main`, working tree clean.
Every code claim below was verified against this commit.

## 1. Executive Summary

E05 delivered a complete, in-memory execution lifecycle: every admitted
`POST /v1/responses` request becomes one logical execution with one immutable
terminal result. Nothing is recorded past the process run: the terminal result
is logged once, returned to the Gateway, and then discarded. Usage information
returned by OpenCode Go is currently invisible to AI Hub, because E05 forwards
the upstream body as opaque bytes and never parses it.

E06 must add a records domain that (a) receives the terminal result of every
execution, (b) persists a minimized, content-free execution record, (c) captures
provider-informed usage when it can be observed without violating the E05
no-buffering contract, and (d) exposes a minimal query API.

Key findings:

1. The E05 `Result` contract is not record-ready. It carries `ExecutionID`,
   `Outcome`, `Phase`, `Cause`, `UpstreamStatus`, `Committed`, `Duration` — but
   not the request ID, provider session, target identity, or wall-clock
   timestamps (`internal/execution/types.go`). The caller (Gateway) reads the
   result after `Manager.Execute` returns; the identity information exists only
   in the `Command` the Gateway itself built and in structured logs.
2. `Target.ModelID` is always empty on the current passthrough path
   (`internal/gateway/responses_handler.go`): the Gateway does not extract the
   model from the payload and E05 explicitly does not parse payload bytes
   (ADR-014 §2). Any execution record therefore has `model_id = unknown` today.
3. Usage exists only inside the upstream response body — inside the final
   `response.completed` SSE event, or the trailing part of a non-streaming JSON
   Responses object (validated in the E02 protocol spike §2.10). Observing it
   requires a new, explicit, narrowly scoped response-inspection capability.
   Without it, E06 can only record `usage = absent`.
4. The project's only persistence precedent (catalog, ADR-010) is a versioned
   JSON snapshot with atomic rename and quarantine. ADR-010 explicitly states
   that full-file rewrite is not the right shape for indexed execution history
   and anticipates replacing the store at E06 without changing the domain.
5. The repository has **zero third-party Go dependencies** (`go.mod`: module
   `aihub`, go 1.23, no requires). Any persistence choice that adds a dependency
   (e.g., SQLite driver) is an architectural event that needs explicit
   justification and approval.

Recommended posture for E06: a new `internal/records` package fed by an
explicit observer seam on the Execution Manager, persisting one minimized,
append-only record per finished execution to the existing data directory
(append-only JSONL), with a minimal ID/latest query API, best-effort durability
(part of the tail may be lost on crash — never double-counted), and usage
captured via a bounded, additive response tap that is proposed separately for
human approval because it touches the E05 no-buffering boundary.

## 2. Current system state (verified in code)

### 2.1 Runtime shape

- Single Go 1.23 binary `cmd/ai-hub`, modular monolith, local-first. No
  database, queue, cache, or external service (`docs/architecture.md`, `go.mod`).
- Modules are constructed in `cmd/ai-hub/main.go`: provider client, execution
  manager, catalog store + manager, access policy, router, HTTP server. Lifecycle
  composition lives in `cmd/ai-hub/lifecycle.go`.
- Data directory: `AIHUB_DATA_DIR` (default
  `${XDG_DATA_HOME:-~/.local/share}/ai-hub`). Currently contains only
  `catalog.json` (+ quarantine files). One process owns the directory (ADR-010).

### 2.2 E05 — Execution Manager (what exists)

Verified files: `internal/execution/types.go`, `execution.go`,
`execution_run.go`, `execution_terminal.go`, `execution_body.go`,
`manager.go`, `errors.go`; `internal/providers/client.go`;
`internal/gateway/responses_handler.go`.

Lifecycle (SPEC E05 §6, implemented as specified):

```text
accepted -> opening -> forwarding -> terminal (completed | failed | canceled | timed_out)
```

Contracts:

- `Command{RequestID, Target{ProviderID, ModelID}, ProviderSessionID, Payload,
  ContentType}` — request correlation (ADR-013), provider-ready opaque payload
  (E02 byte-for-byte), provider session context.
- `Result{ExecutionID, Outcome, Phase, Cause, UpstreamStatus, Committed,
  Duration}` — published by `finish()` after the upstream body is closed exactly
  once and the reader goroutine is joined. Outcome set:
  `completed | failed(transport, upstream_read) | canceled(client, shutdown) |
  timed_out(opening, inactivity)`.
- `UpstreamStatus` is `0` when no upstream headers were received (transport
  failure before headers); it is the real upstream status even for upstream
  `4xx`/`5xx` responses that are forwarded untouched.
- The terminal arbiter reserves exactly one event per execution; the result is
  immutable once published. The execution is then removed from the active
  registry and a structured log line (`execution finished`) is emitted with
  execution_id, request_id, provider_id, model_id, outcome, phase, cause,
  upstream_status, duration.
- The `Result` returned to the Gateway is the only non-log consumer. After the
  handler maps opening-time transport failure to `502` and opening-timeout to
  `504`, the result is dropped. No artifact of the execution survives.

Identity facts (ADR-013/ADR-014):

- `execution_id`: fresh 16 random bytes, hex, in-process only ("not persisted"
  per E05 SPEC §5 — E06 changes this by decision).
- `request_id`: correlation-only, client-supplied (validated 8–128 restricted
  charset) or AI Hub generated 128-bit hex.
- `provider_session_id`: `x-opencode-session` value, client-supplied (1–128
  chars of `[A-Za-z0-9._:-]`) or generated `aihub-ses-<32 hex>`. Per-request
  provider context; never a durable AI Hub session; E04 never persists it.
- `provider_id`: fixed `opencode-go` today.
- `model_id`: structurally supported by `Target` and validated if non-empty,
  but the Gateway passes `Target{ProviderID: "opencode-go"}` **without**
  `ModelID`; extraction from the payload was deliberately not implemented
  (E05 SPEC §5.1). Nothing in the current path can state which model was used.

Timing facts:

- The execution holds `started time.Time` and reports `Duration` in the result.
  Wall-clock start/end are not exposed in the result contract.
- No total execution timeout exists (ADR-008); the stream may last arbitrarily
  long while progressing.

### 2.3 E02 provider — where usage lives today

`internal/providers/client.go` sends the Responses payload verbatim to
`{base}/openai/v1/responses` and returns the upstream `*http.Response` for
streaming. It never parses the response body.

Validated upstream behavior (protocol spike §2.8/§2.10):

- Streaming (`stream: true`, what Codex always sends): the SSE event family is
  forwarded, and the **final `response.completed` event carries the `usage`
  block**, including OpenAI-standard counters and vendor metadata:

  ```json
  "usage": {
    "input_tokens": 36, "output_tokens": 19, "total_tokens": 55, "cost": 0,
    "opencode": {
      "billing_source": "go", "list_cost": 0.0000168,
      "allowance": 0.0000126,
      "tokens": {"input": 36, "output": 19, "reasoning": 15, "cache_read": 0}
    }
  }
  ```

- Non-streaming: a complete Responses object whose `usage` sits near the end of
  the JSON body (position not contractually fixed upstream).
- `cost = 0` in the spike because the request was covered by the Go plan
  allowance; `allowance` reports what was consumed from the monthly limit.
  These are provider-reported figures, not an AI Hub billing truth (§4 below).

Because usage arrives **inside the byte stream E05 forwards opaquely**, every
usage scenario in the E06 brief must be answered against this reality:

| Scenario | Consequence under current contracts |
| --- | --- |
| Usage present at normal end | Inside final `response.completed` / trailing body bytes; not parsed by anything. |
| Usage absent | Store `usage = absent`; also depends on model/provider behavior. |
| Usage partial | Upstream controls fields; record only observed fields + provenance. |
| Usage only at end of streaming | Inherently true for Responses streaming; see tap options (§7.3). |
| Failure before usage | `failed(transport)`/`timed_out(opening)` → no body bytes → usage absent. |
| No final usage event (inactivity timeout mid-stream, client cancel) | The final event is never forwarded when the sink stops after a terminal reservation (`execution_run.go` stops forwarding on `hasReserved()`); usage absent. |
| Provider vs locally computed usage | Never computed locally in E06; no local token accounting exists in the codebase. |

### 2.4 E04 — identity and admission boundary

- Static bearer authentication (single gateway principal), Host/Origin policy,
  bounded and buffered body admission, media/encoding validation
  (`internal/access`). There is no per-user identity; ADR-013 §5 forbids
  implying durable sessions, users, roles, or tenants. E06 cannot attribute
  records to anything richer than correlation IDs and the gateway principal.

### 2.5 E03 — persistence precedent and expectations

`internal/catalog/store.go` (ADR-010) is the house pattern:

- Versioned JSON snapshot; write = temp file → file `fsync` → close → `rename`
  → best-effort directory `fsync`; previous file never truncated.
- Corrupt/unsupported snapshots are quarantined; failed quarantine disables
  writes for the session.
- Memory commits only after the persisted write succeeds.
- ADR-010 consequences (verbatim intent): every write rewrites the whole file,
  acceptable "until E06 needs indexed execution history, at which point the
  store can be replaced without changing the domain"; crash durability depends
  on directory `fsync`; multi-process access unsupported.

The catalog store is therefore **not** directly reusable for records (snapshot
rewrite per execution would be quadratic and destructive), but its directory,
permissions, fsync discipline, and its "store can be swapped, domain stays" rule
define E06's constraints.

### 2.6 Lifecycle and shutdown interplay (E05)

`cmd/ai-hub/lifecycle.go` composes: drain gate → server stop → `manager.Wait`
(graceful) → forced cancel at deadline → force-close → cleanup-window wait →
guarded provider release (`Drained()` must be true). Any E06 flush point must
slot into this sequence such that losing the storage does not change execution
semantics and losing a record tail is the accepted worst case (§8).

## 3. Architecture diagnosis

E05 is a deliberately terminal-state machine with no memory beyond the active
registry. That design is correct for its responsibility and E06 must not make
the manager durable. But three gaps separate E05 from any records domain:

1. **Identity gap.** `Result` does not carry who/what executed (request, session,
   provider, model) nor wall-clock time. The manager already holds all of it
   (`execution.command`, `execution.started`).
2. **Content-observation gap.** Usage is inside the forwarded byte stream.
   E05's invariants (no buffering, no parsing of request, forwarding stops at
   terminal reservation) mean usage observation needs a new, additive,
   contract-bound capability — it is the single largest design risk of E06.
3. **No delivery point.** There is no seam through which a terminal result
   reaches another interested package: the only consumer is the Gateway.

E06 also inherits constraints from the rest of the system:

- Zero-dependency posture (go.mod), local-first data dir, single process owner.
- E04 provides no user identity → records are attributed to request/session
  correlation only.
- The catalog teaches: persistence failure must not corrupt accepted state;
  failed writes degrade to in-memory diagnostics; durability is honest about
  directory-fsync uncertainty.

## 4. Problems E06 must solve

1. Represent a finished execution durably, minimized and content-free, without
   turning execution persistence failures into execution failures.
2. Decide the observability seam between `internal/execution` and records
   without coupling E05 to storage details or inventing a generic event bus.
3. Decide, explicitly, whether and how usage is captured given the byte-level
   passthrough — including the difference between "provider reported usage" and
   any local notion of cost.
4. Choose persistence shape and consistency guarantees that are honest about
   local-first, single-process, crash-loss windows.
5. Preserve full evolvability for E08/E09 (provider/model/attribution fields),
   E10 (learning) and dashboards without implementing any of them.

## 5. Existing contracts (inventory)

| Contract | Source today | Needed by E06 | Status |
| --- | --- | --- | --- |
| Execution ID | `exec.Result.ExecutionID` | record primary key | Available |
| Request ID | `Command.RequestID` (not in `Result`) | traceability field | Gap — must be added to the result/observation |
| Provider session | `Command.ProviderSessionID` (not in `Result`) | traceability field (policy decision, §9) | Gap — same |
| Provider ID | `Command.Target.ProviderID` (not in `Result`) | mandatory record field | Gap — same |
| Model ID | `Command.Target.ModelID` — **always empty today** | optional record field | Gap — how to represent absence |
| Terminal outcome/phase/cause | `Result.{Outcome,Phase,Cause}` | mandatory record fields | Available |
| Upstream HTTP status | `Result.UpstreamStatus` (0 = no headers) | conditional record field | Available |
| Response committed | `Result.Committed` | conditional field | Available |
| Elapsed duration | `Result.Duration` | mandatory field | Available |
| Start/end wall-clock | only internal `execution.started` | mandatory fields | Gap — must be exposed |
| Usage (tokens/cost/billing) | inside forwarded body bytes | stored usage | **No contract exists** |
| Delivery of result post-handling | none (log + discard) | new seam | **No contract exists** |

## 6. Gaps between E05 and E06 (explicit)

| # | Gap | Required evolution | Boundary preserved |
| --- | --- | --- | --- |
| G1 | `Result` lacks execution identity/time context | Enrich the published result (or accompany it) with request ID, provider session, target, started-at — read-only copies of command fields | E05 continues to own state; fields are copies, never mutated |
| G2 | No delivery seam | One explicit terminal observer registered at composition (`cmd`), invoked once per execution after terminal publication and registry removal | E05 must not import records; interface owned by `internal/execution`, implemented by records |
| G3 | Model ID absent on passthrough | Record represents `model_id = unknown` honestly; a future payload-mapping contract (E08) is the only path to populating it | E05 still does not parse payloads (ADR-014 §2) |
| G4 | Usage invisible | New optional response tap with a strict additive contract (§7.3); otherwise E06 ships with `usage = absent` | Forwarded bytes and client-visible streaming are untouched; tap never buffers the complete response |
| G5 | Wall-clock timing not exposed | Expose `started_at` (already held internally); ended = started + published duration, or expose both | No timeout semantics change (ADR-008 intact) |

## 7. Alternatives evaluated

### 7.1 Integration between E05 and records

| Alternative | Assessment |
| --- | --- |
| **A. Manager-owned observer (recommended)** — `internal/execution` defines `ResultObserver` interface with a read-only finished-execution value; records implements it; `cmd` wires it. Invoked exactly once per execution after terminal publication and deregistration, before removal from the active map finishes, so `Wait()` already covers record hand-off. | Explicit contract; dependency points E05→interface only; no import of records; the drain gate sequence guarantees the observer ran for every finished execution before provider release; testable; matches ADR-014's "manager publishes result once" design. Observer must be failure-isolated and non-blocking (its own duration must not gate execution cleanup — see consistency decision in §8). |
| B. Gateway-driven recording (handler calls records after `Execute` returns) | No new seam in E05, but duplicates the composition point on the gateway side and misses executions in future non-gateway callers (E08's resolved command will still end in the manager, not a direct handler call). Weaker invariant: "every finished execution is observed". |
| C. Internal event bus | Rejected. Anticipatory generic abstraction; ordering and failure semantics become a new subsystem. The repository rule "no abstractions ahead of evidence" (providers/catalag stance) applies. |
| D. E05 persists directly | Rejected. Violates SPEC E05 §3.2 ("no persistence") and ADR-014's boundary; would couple execution to storage. |

### 7.2 Persistence shape

| Alternative | Assessment |
| --- | --- |
| **A. Append-only JSONL, one record per line, in the data dir (recommended MVP)** | Append-only matches records: immutable finished executions. One `O_APPEND` write with `fsync` option; file rotation at a size bound; startup indexes in memory; simple filters (limit/provider/outcome/time) evaluated over a bounded in-memory window. Zero dependencies; inspectable; crash-tail loss is a clear, honest limit. Scales fine for a local-first gateway (hundreds–low thousands of records/day). Aggregation over long horizons later re-reads the file (or a future store replaces it behind the repository interface, per ADR-010's own doctrine). |
| B. Snapshot rewrite (reuse catalog store shape) | Rejected: rereads/rewrites the whole history every execution; O(n²) growth; destructive failure modes on the entire history instead of losing one record tail. |
| C. SQLite | Strong long-term candidate (indexed queries, aggregation). Rejected for the MVP: it introduces the project's first third-party dependency (driver without CGO / cgo toolchain), a schema-migration story, new failure modes, and a data-format firewall around the "one process owns the dir" rule. Recommended as the **documented future migration** when aggregation/volume evidence appears — the domain/repository split required by E06 makes the swap local. Also needs a decision about which driver family before approval. |
| D. PostgreSQL / external services | Rejected: contradicts local-first (no DB, no external service — architecture.md). |
| E. No persistence (memory only + richer logs) | Rejected: dies at process exit; E06's core purpose is the consultable history. |

### 7.3 Usage observation

| Alternative | Assessment |
| --- | --- |
| **A. Ship E06 without usage capture (MVP-minimal, recommended if approval for content inspection is withheld)** | Execute the rest of the domain with `usage = absent`. Zero risk to E05 streaming invariants. Usage observational capability is added later. |
| **B. Bounded final-event tap in the read path (recommended when usage is approved)** | A read-only tap inspects forwarded chunks only to detect the **final** Responses event (`response.completed` data payload for SSE; consumption of the trailing window for non-streaming JSON). Constraints: never modifies forwarded bytes; never delays or blocks forwarding; keeps at most a small bounded tail buffer (e.g., 64 KiB) that is discarded at terminal reservation; parse errors are silent for the record (`usage = absent` + diagnostic); usage becomes available only at completion. Risk: a new parsing behavior lives in the hot path of the byte-for-byte contract; must be covered by E02 regression tests (bytes unchanged with tap enabled and disabled). This is a genuine contract event on E05/E02 and needs explicit human approval. |
| C. Full response buffering/storage, parse at end | Rejected: violates no-buffering invariant; stores sensitive content; unnecessary for a milestone that wants only usage. |
| D. Post-hoc upstream re-query (`GET /v1/responses/{id}`) if upstream ever offers retrieval | Rejected now: progress-dependent, not in spike evidence for Go; merely recorded as a future option. |

### 7.4 Consistency / delivery strategy

| Alternative | Assessment |
| --- | --- |
| **A. In-memory registry first, synchronous durable append after (recommended)** | Terminal result remains authoritative in memory; the observer enqueues a record and the recorder performs one append (with bounded queue, single-writer). Persist failure does not alter the execution result: it increments a degradation counter and is logged — mirroring ADR-010's commit-order doctrine, inverted correctly: execution state was already committed when the record write runs. Loss semantics are explicit: a crash between terminal publication and flush loses the record (at-most-once per execution; never duplicated). No outbox/queue infrastructure is justified by current evidence. |
| B. Synchronous blocking write inside the observer | Simple but couples execution cleanup to disk latency (hundreds of executions with fsync would slow drain); rejected as the primary mode; keep the write off the critical path. |
| C. Transactional outbox / durable queue | Rejected: no infrastructure is acceptable, and exactly-once cannot be demonstrated without it — so E06 promises the honest weaker terms instead of pretending. |
| D. In-memory buffer with batched/periodic flush | Loss window becomes time-bounded rather than event-bounded; batch complexity without an evidenced throughput need; rejected in favor of per-record append with an explicit, small batching decision deferred. |

## 8. Consistency analysis (straight answers)

For each failure case in the brief:

1. **Execution ends and the record persists:** normal path. The record becomes
   visible to queries after the append commit. In-memory (current-process)
   availability may precede durable confirmation; query responses can label
   record state honestly (`persisted` / `pending`).
2. **Execution ends, persistence fails:** the execution result, the client
   response, and the manager are unaffected. The failure is logged with the
   classification, a degradation counter increments, and (if the data dir is
   unusable, catalog-style) the session may disable record writes after
   quarantine-style diagnostics. Guarantee: history is incomplete, execution is
   never misreported.
3. **Application shuts down during persistence:** the lifecycle composition
   grants a bounded flush window after the manager drains (before or alongside
   provider release). Unflushed tail records are lost, stated loudly. A shutdown
   flush timeout must not turn a clean stop into a failed stop; it reports a
   warn-level mismatch instead (contra E05's `ErrCleanupUnconfirmed`, which is
   about executions, not records).
4. **Process dies before registering the result:** the record does not exist.
   Worst case accepted by design: at-most-once, loss on crash before flush. No
   recovery protocol can conjure a terminal result that was never produced.
5. **Usage arrives after streaming started:** inherent to Responses; the tap
   approach observes only the final event. Until then no usage exists for the
   record.
6. **Client connection ends before completion:** execution terminal is
   `canceled(client)`; per current code the final event is not forwarded after a
   reservation wins, so usage is typically absent. Record stores the honest
   terminal state; a record with outcome `canceled` and absent usage is valid.

The **guarantee statement for the SPEC**: every finished execution produces at
most one record; a record is either absent (crash/failed write) or byte-stable;
no execution can produce two records; no failed record write can ever change an
execution's outcome, its HTTP behavior, or its logs. E06 must not claim
exactly-once durable delivery — it claims at-most-once with a bounded, explicit
loss window.

## 9. Data minimization, privacy, retention

What may be stored (closed vocabulary, schema-versioned):

- Identifiers: `execution_id`, `request_id`, `provider_id`, `model_id`
  (may be unknown), `provider_session_id` (see decision below).
- Timing: `started_at`, `ended_at`/`duration`.
- Terminal classification: outcome, phase, cause, `upstream_status`,
  `response_committed`.
- Usage: provider-reported figures only, stored verbatim with provenance
  `provider_reported` and an `observed_at`-implied linkage to the record.

What must never be stored (aligned with E02/E05 log laws: "no request bodies,
generated content, credentials, or raw provider response bodies"):

- Request payload, response body or any generated content, including reasoning
  items, tool call names/arguments/outputs.
- Any HTTP headers beyond nothing at all (no `Authorization`, no user agents),
  no IP/host/origin data that E04 already refuses to persist.
- Credentials in any form. There is nothing today that could leak them into a
  record: the provider key never enters execution state.
- Provider error bodies: E05 deliberately carries no provider error text in
  `Result` (ADR-014 §3) — E06 inherits that; classification fields only.

`provider_session_id` is the only sensitive field with a nuance: it can be
client-supplied. It is grammar-restricted (ADR-013) and is the key the provider
uses for routing, so it has real traceability value. Sessions never appear in
logs today (E04 does not log them). Two defensible options: store as-is
(recommended for traceability, restricted charset already enforced) or store a
redacted prefix only. This stays an open question for approval because ADR-013
says the value "is not persisted" — E06 would change that sentence deliberately.

Retention (proposal, needs approval): record count cap and/or age cap in
configuration, enforced opportunistically at startup (trim oldest), default
generous (e.g., 30 days) so no user has data hoarded behind their back; a
record file exceeding a size bound rotates. Quarantine-style behavior for a
corrupt line: stop at the bad offset, quarantine the remainder, keep valid
prefix — one bad line never destroys the whole history.

Access control: record query endpoints sit behind the existing E04 bearer
policy on loopback; no additional identity layer is introduced (E09 owns
per-principal permissions later).

## 10. Proposed data model (conceptual, not implemented)

```text
ExecutionRecord (schema_version = 1)
  execution_id        string(32 hex)  — required, unique
  request_id          string          — required (correlation)
  provider_id         string          — required ("opencode-go" today)
  model_id            string | null   — optional; null = unknown (current passthrough)
  provider_session_id string          — required pending §9 decision
  started_at          RFC3339 UTC     — required  (E05 evolution G5)
  ended_at            RFC3339 UTC     — required  (started_at + duration)
  outcome             completed|failed|canceled|timed_out — required
  phase               opening|forwarding                  — required
  cause               ""|transport|upstream_read|client|shutdown|opening|inactivity
                      — required; empty iff outcome = completed
  upstream_status     int | null      — null when no headers received (0 today -> null)
  response_committed  bool            — required
  usage               UsageReport | null — optional

UsageReport (all fields "provider_reported"; any subset may be present)
  input_tokens        int | null
  output_tokens       int | null
  total_tokens        int | null
  reasoning_tokens    int | null     — opencode.tokens.reasoning
  cache_read_tokens   int | null     — opencode.tokens.cache_read
  provider_cost       float | null   — usage.cost
  billing_source      string | null  — opencode.billing_source
  list_cost           float | null   — opencode.list_cost
  allowance           float | null   — opencode.allowance
```

Field classes:

- **Mandatory:** execution_id, request_id, provider_id, started_at, ended_at,
  outcome, phase, cause, response_committed, schema_version.
- **Optional:** model_id, upstream_status, provider_session_id (policy), usage.
- **Conditional:** usage well-formedness — when present it must state its
  provenance; `upstream_status` present iff any response was opened.

Cost-wise agreements this model encodes:

- `provider_cost`, `list_cost`, `allowance` are **facts reported by the
  provider**, stored verbatim. They are not AI Hub billing, not invoicing, not
  "true cost", and E06 performs zero arithmetic on them (no multiplication,
  no currency conversion, no catalog-price lookup). Effective billed cost is
  unknowable to a client-side gateway; store what was told, nothing else.
- Any future estimated/calculated cost belongs to E09 (limits/quota context) as
  a separate field family, in a separately approved contract — E06's schema
  reserves nothing speculative today beyond explicit nulls.

## 11. State transitions

Two disjoint state spaces must never be merged:

E05 execution states (as implemented, unchanged by E06):

```text
accepted → opening → forwarding → completed
                                ↘ failed(transport|upstream_read)
                                ↘ canceled(client|shutdown)
                                ↘ timed_out(opening|inactivity)
(terminal states are immutable; exactly one per execution)
```

E06 record persistence states (new, proposed):

```text
                 ┌─────────────────────────────────────────────┐
 terminal result │                                              v
 published ──────┤ enqueue ──► pending ── persisted ──────────► visible
                 │                ├── persist_failed (loss, degradation counter)
                 │                └── flushed_at_shutdown / lost (bounded tail)
                 └── never created (crash before enqueue)
(pending → persisted happens at most once; no record is ever mutated after persisted)
```

Record states are process-local diagnostics: a record that exists is final;
"pending" is only visible to the writing pipeline, never to queries for longer
than the flush window.

## 12. Queries (MVP proposal)

Minimal useful contract:

- `GET /v1/records/executions/{execution_id}` — single record with usage.
- `GET /v1/records/executions?limit=&provider_id=&model_id=&outcome=&started_after=&started_before=`
  — recent-first bounded list (default `limit=50`, max e.g. 200).
- Aggregation endpoints (totals, per-model usage, cost rollups) — deferred.
  Evidence for a real need is expected from E09/E11; adding rollups now would
  force storage/index decisions a JSONL MVP cannot make well.

## 13. Privacy and retention defaults (summary of §9)

- No content, ever. Records are metadata + provider-reported usage only.
- Records inherit "one process owns the data dir"; file permissions 0600,
  directory 0700 (same as catalog).
- All record-bearing logs keep the E05 redaction rules (request/session/target/
  outcome/duration only).
- Retention: explicit config keys, startup trim, size-bound rotation. Exact
  defaults approved by the developer (open question).

## 14. Roadmap and future dependencies (E08/E09, E10/E11)

- **E09 (eligibility, limits):** will need per-record usage consumed against
  allowances/quota windows. E06's non-negotiable data preservation for that:
  (`provider_id`, `model_id`, timing, outcome, full fidelity usage with
  provenance, and request correlation). E06 must not compute quota deltas or
  gate anything — that stays E09.
- **E08 (classification/selection):** will select provider/model *before*
  execution; E06 records *after*. Nothing in E06's contract may assume the
  gateway is the only caller: the observer seam (option A) survives E08's
  resolved-command callers naturally; option B (gateway-driven) would not.
- **Per-user attribution:** blocked by E04 (no identity). If E09/E04 introduce a
  principal expansion, records gain a new field family with a schema bump —
  E06 must not fabricate a `principal` field now.
- **E10/E11:** the record is the learning/observability substrate; aggregation
  belongs to later consumers, not here. The repository interface must survive a
  storage swap (ADR-010 doctrine) so analytics can later pick SQLite without
  domain changes.

## 15. Risks

| Risk | Mitigation in E06 proposal |
| --- | --- |
| Usage tap silently violates byte-for-byte passthrough | Tap is additive-only; regression tests assert forwarded bytes identical with tap on/off; MVP ships tap only under explicit approval; `usage = absent` fallback otherwise |
| Records write latency degrades drain/shutdown | Write off the execution critical path (§7.4 A); bounded flush window at shutdown; timeout degrades to warn, never fails the stop |
| History grows unbounded | Retention caps + size-bound rotation from day one |
| Future principal/identity fields become a breaking schema change | Explicit `schema_version`, append-only design, honest nulls; note in SPEC invariants |
| Confusion between "cost fields in usage" and "AI Hub billing" | Closed vocabulary, provenance `provider_reported`, scope exclusion (§17); documentation states the boundary explicitly |
| JSONL corrupt tail on crash | On load, stop at first bad line, quarantine remainder (catalog pattern), history minus tail remains valid; never rewrite the fused file blindly |
| Multiple processes sharing data dir (unsupported, as E03) | Same assumption as ADR-010; documented limitation |

## 16. Defects / observations about prior epics (recorded, not fixed)

1. **`Result` contract not self-contained** (G1, G5): identity and wall-clock
   data live in `Command`/`execution` but not in the published `Result`; the
   Gateway reconstructs the context only because it built the command. Not a
   defect per E05 scope — evolution item for E06.
2. **Model identity absent everywhere** (G3): the passthrough route cannot
   answer "which model was used" — from payload, provider, or anywhere. This is
   a known, approved E05 stance; flagged here because record usefulness is
   materially limited by it.
3. **Usage observability is structurally zero**: nothing in the codebase can
   currently observe usage even for diagnostics. E02/E05 choices made this a
   feature of the passthrough, not a bug; E06 must introduce the capability
   deliberately or accept absence.
4. Nothing here may be "corrected" inside E06 without separate approval (per
   Discovery constraints; the fixes are contract evolutions, not defects).

## 17. Recommended decisions (for the SPEC and the developer)

| # | Decision | Recommendation |
| --- | --- | --- |
| D1 | Delivery seam | Observer interface in `internal/execution`, implemented by `internal/records`, wired in `cmd`; invoked once post-terminal, pre-deregistration-complete |
| D2 | Persistence | Append-only JSONL in `AIHUB_DATA_DIR` (e.g. `records/executions-<n>.jsonl`), single-writer mutex, per-record write, fsync-per-record as a config choice; no DB in MVP |
| D3 | Record model | As in §10, schema_version 1, in package `internal/records` |
| D4 | Usage | MVP: record `usage = null` with provenance framework in place; enable the bounded final-event tap **only** as an explicitly approved sub-scope of E06 (otherwise usage capability is a follow-up epic) |
| D5 | Failure semantics | At-most-once; persist failure never affects executions or HTTP responses; degradation counter + structured log |
| D6 | Retention | Config: max age and max records; startup trim; rotation by size; exact defaults = approval question |
| D7 | Session ID storage | Store `provider_session_id` as-is (grammar-restricted) pending explicit approval because ADR-013 states non-persistence |
| D8 | Query API | GET single + bounded filtered list behind E04 policy; no aggregation in MVP |
| D9 | ADR set | ADR-017 (records store: append-only JSONL + retention), ADR-018 (E05→E06 terminal observation contract and Result enrichment), and — only if D4 tap is approved — ADR-019 (additive response usage tap) |

## 18. Open questions requiring human approval

1. Approve/withhold the usage tap (§7.3 B) as part of E06, keeping in mind it
   touches the streaming passthrough contract that E02/E05 froze for byte
   fidelity.
2. Approve storing `provider_session_id` (relaxing ADR-013's "not persisted"
   sentence), and if yes: stored verbatim, redacted prefix, or omitted?
3. Retention defaults (age/count/size and whether default fsync per record is
   acceptable given expected local volume).
4. Public route names (`/v1/records/...`) and whether usage figures may appear
   in query responses from day one.
5. Confirm E06 precedes or follows E07 (harness integration) in delivery order;
   no conflict exists, but recording it fixes the roadmap.

## 19. MVP scope recommendation

In: Execution Record persistence (JSONL, append-only), observer seam + minimal
E05 contract enrichment (G1/G5), minimal query API (§12), retention config,
schema versioning, failure/degradation semantics, unit + fake-upstream tests
with the catalog-style failure matrix, documentation (SPEC + ADRs).

Out (explicit): usage tap unless approved (D4), aggregation/rollups, dashboards,
billing/cost mathematics, sessions/conversation storage, retries, model
selection support, multi-process support, second-provider abstractions.

## References

- `docs/specs/E05-execution.md` (approved 2026-10-31)
- `docs/specs/E02-provider-management.md`, `docs/specs/E03-model-catalog.md`,
  `docs/specs/E04-gateway-hardening-access-control.md`
- `docs/adr/ADR-003-slog-and-correlation-ids.md` (identity)
- `docs/adr/ADR-008-streaming-inactivity-deadline.md` (stream lifetime)
- `docs/adr/ADR-010-catalog-snapshot-persistence.md` (store doctrine)
- `docs/adr/ADR-013-request-and-session-identity.md` (identity semantics)
- `docs/adr/ADR-014-execution-lifecycle-and-command-contract.md`,
  `docs/adr/ADR-015-coordinated-execution-shutdown.md`,
  `docs/adr/ADR-016-execution-timeout-and-error-mapping.md`
- `docs/research/opencode-go-protocol-spike.md` (usage fields §2.10)
- `AI-Hub-Product-Architecture-Kickoff.md` §13 (E06 = Usage & Observability)
