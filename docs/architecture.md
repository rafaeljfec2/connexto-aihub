# AI Hub — Implemented Architecture (E01–E06; E07 Validation)

This document describes the current runtime through E06. E07 validates Codex
CLI as the first supported harness but adds no production runtime behavior or
durable AI Hub conversation state. The original product vision and proposed
future layers remain in `AI-Hub-Product-Architecture-Kickoff.md`.

## Runtime model

Single Go binary, modular monolith, local-first. No external database, queueing
service, cache, or other runtime service is required; E06 uses a bounded
in-process records queue.

```text
cmd/ai-hub/main.go
    |
    +--> app lifecycle: guarded providers -> catalog -> records -> HTTP/execution
    |
    +--> server/router --> access policy (E04) --> gateway (E02/E03/E06)
    |                                              |
    |                                              +--> execution.Manager (E05)
    |                                                       |
    |                                                       +--> OpenCode Go (E02)
    |                                                       |
    |                                                       +--> terminal observer
    |                                                             -> records (E06)
    +--> observability
```

Health endpoints are served by `health`; provider endpoints are injected as
additional routes into the same router.

Dependencies are explicit and acyclic. `cmd` wires every package and owns the
construction order; `app` depends only on the standard library (it knows the
`Module` contract, not the concrete modules); `server` depends on `config`,
`access` and `observability`; `config` reuses `access` validation helpers;
`access` is a standard-library-only leaf package; `health` and `observability`
depend on the standard library only.

## Packages

| Package                  | Responsibility                                                                                            |
| ------------------------ | --------------------------------------------------------------------------------------------------------- |
| `cmd/ai-hub`             | Process entrypoint, flags, signal handling, shutdown budget.                                              |
| `internal/app`           | `Module` contract, ordered start, reverse-order stop, rollback.                                           |
| `internal/config`        | Typed configuration, env defaults overrides, validation, provider settings.                               |
| `internal/server`        | Listener, exposure policy, `http.Server`, routing tree and HTTP middleware.                               |
| `internal/access`        | Static bearer authentication, Host/Origin policy, request admission and request identity (E04).           |
| `internal/health`        | Readiness registry and liveness/readiness handlers.                                                       |
| `internal/observability` | `slog` construction and correlation ID context propagation.                                               |
| `internal/providers`     | OpenCode Go adapter: credential handling, catalog descriptors, Responses execution, provider state (E02). |
| `internal/catalog`       | Model catalog domain, JSON snapshot, synchronization and queries (E03).                                   |
| `internal/gateway`       | Provider passthrough (E02), catalog (E03) and records query handlers (E06).                              |
| `internal/execution`     | Per-request execution lifecycle, cancellation, terminal arbitration and drain coordination (E05).         |
| `internal/records`       | Content-free execution records, bounded writer, JSONL persistence, retention and query index (E06).       |

`providers` is a concrete adapter, not an interface-driven layer: the spike
validated exactly one adapter shape (Responses in, Responses out); an
abstraction will be introduced only when a second provider exists.

## Lifecycle

1. `main` loads and validates configuration, including the mandatory gateway
   key and access policy. Invalid input aborts before any listener is created.
2. The logger is built from the validated log settings.
3. Modules are constructed explicitly and injected into `app.New`.
4. `App.Start` starts modules in order (`guarded providers` → `catalog` →
   `records` → `http/execution lifecycle`), stopping already started modules in
   reverse order if one fails.
5. The process waits for `SIGINT`/`SIGTERM` or a fatal serving error.
6. `App.Stop` runs with a context limited by `server.shutdown_timeout`. The
   HTTP/execution lifecycle closes execution admission and drains or cancels
   work; records then drains for at most two seconds; the guarded provider is
   released only after E05 confirms its executions are drained.

`server.Start` binds the listener synchronously, so port conflicts fail startup
instead of surfacing asynchronously. The bound address is checked after
`net.Listen`: loopback is always accepted, while a non-loopback listener is
accepted only after explicit allowed-host configuration; otherwise the listener
is closed and startup fails.

## Request pipeline

```text
X-Request-Id validation/generation
        |
        v
access logging (method, path, status, bytes, duration, correlation id)
        |
        v
panic recovery
        |
        v
Access policy: Host/Origin -> bearer authentication
        |
        v
route admission: body bound, media/encoding and provider session
        |
        v
http.ServeMux ->  GET /healthz | GET /readyz
   |              | GET /v1/models        | GET /diagnostics/providers
   |              | POST /v1/responses    (streaming, byte-for-byte pass-through)
   |              | GET /v1/catalog/models | GET /v1/catalog/models/{provider}/{model...}
   |              | GET /v1/catalog/providers | POST /v1/catalog/sync
   |              | GET /v1/records/executions
   |              | GET /v1/records/executions/{execution_id}
```

The access log intentionally excludes headers, query strings and bodies, so
credentials cannot leak through it. Only exactly `GET /healthz` and
`GET /readyz` bypass authentication; both still enforce Host/Origin policy. Query strings are excluded because they can
carry API keys. Recovery runs inside access logging, so panicked requests are
logged too; if the handler already committed a status, recovery logs the panic
without appending error text to the partial body.

## Configuration

Precedence is `defaults < environment`. Validation aggregates every problem
into one error. `server.port` must be 1–65535; timeouts must be positive;
`read_header_timeout` cannot exceed `read_timeout`.

Provider settings (E02):

```env
AIHUB_PROVIDER_OPENCODEGO_BASE_URL=   # default https://opencode.ai/inference/go
AIHUB_PROVIDER_OPENCODEGO_API_KEY=    # empty => provider disabled
```

Catalog settings (E03):

```env
AIHUB_DATA_DIR=                       # default ${XDG_DATA_HOME:-~/.local/share}/ai-hub
AIHUB_CATALOG_STALE_AFTER=24h         # freshness threshold; must be > 0
AIHUB_CATALOG_SYNC_TIMEOUT=10s        # startup and manual sync deadline; must be > 0
```

An invalid base URL (scheme, host, query/fragment or embedded credentials)
aborts startup; a missing API key does not — the provider starts disabled.

Gateway settings (E04):

```env
AIHUB_GATEWAY_API_KEY=                # mandatory; 32-512 b64token characters, distinct from provider key
AIHUB_GATEWAY_ALLOWED_HOSTS=          # explicit for non-loopback; defaults to localhost/127.0.0.1/::1
AIHUB_GATEWAY_ALLOWED_ORIGINS=        # exact http/https origins; empty allows no browser Origin
AIHUB_GATEWAY_MAX_BODY_BYTES=33554432 # 1 KiB-128 MiB
```

## Gateway access control (E04)

Host and Origin checks run before authentication and body admission. Host
matching uses the request authority with port/brackets/case normalized and
never trusts forwarded headers. Origin is exact and canonicalized; absent
Origin is allowed, while `null`, wildcard, multiple values and malformed origins
are rejected with `403`. Rejections do not echo the offending value.

Authentication requires exactly one `Authorization: Bearer <key>` value. The key
is validated at startup, and the runtime access policy stores only a SHA-256
digest for constant-time comparison. The original key remains in the process
configuration and environment. Local errors use the stable
`{"error":{"type","message"}}` envelope with `Cache-Control: no-store`; `401` adds
`WWW-Authenticate: Bearer realm="ai-hub"`.

`POST /v1/responses` buffers at most the configured body limit plus one sentinel
byte, forwards the accepted bytes unchanged and never buffers the response.
Catalog sync keeps a 4 KiB cap. Non-identity content encodings are rejected, as
are duplicate media/session/authorization headers. Admission resolves
`x-opencode-session` before the E02 handler; valid supplied values are
preserved, while absent/empty values generate `aihub-ses-<32 hex>`.

`internal/access.Metadata` carries the authenticated principal, provider session
and accepted Host/Origin in the request context. It is never persisted or added
to the provider payload. `X-Request-Id` remains correlation-only.

## Health semantics

- `GET /healthz` answers liveness only. It does not touch dependencies.
- `GET /readyz` runs registered checks sequentially and returns `up`/`down`.
  Check error details are never serialized; only `ok`/`error` per check is
  exposed, to avoid leaking connection strings or credentials.

The registry is empty as long as no dependency is registered. Provider
availability deliberately does **not** flip readiness: a disabled or unhealthy
provider is a diagnostic state exposed at `GET /diagnostics/providers`, never a
decision about whether the application itself is fine (see E02 SPEC decision
D-R). Catalog synchronization has the same stance: a failed sync or a stale
catalog never affects `/readyz`.

## Provider integration (E02)

The first provider is `opencode-go`, integrated exactly as validated in
`docs/research/opencode-go-protocol-spike.md`:

- AI Hub talks OpenAI Responses to the upstream; it never translates protocols.
- The Responses request body is forwarded byte-for-byte; the gateway injects
  only transport headers (`Authorization: Bearer`, `x-opencode-session`).
- `previous_response_id` is never emitted. Translated models such as
  `deepseek-v4.1-flash` require full conversation replay per request.
- Streaming: every SSE chunk is copied, flushed and followed by a re-armed
  `http.ResponseController` write deadline plus a read watchdog, so streams are
  bounded by a two-minute inactivity window instead of the server-wide
  `write_timeout` (ADR-008). Ordinary endpoints keep E01 timeout semantics.
- Sessions: an incoming `x-opencode-session` is preserved; otherwise a fresh
  `aihub-ses-…` ID is generated per request. `X-Request-Id` never derives it,
  and E02 does not implement cross-turn session continuity.
- Errors: upstream status and body pass through; transport failures become
  `502` with a generic body, provider-less callers get `503`.
- Confidentiality: credentials live only in `Authorization` headers, never in
  logs, error envelopes or mirrored response headers (only `Content-Type` is
  mirrored).
- A real smoke test exists behind `AIHUB_LIVE_SMOKE=1` plus
  `AIHUB_PROVIDER_OPENCODEGO_API_KEY` and is always skipped otherwise; mock
  green tests never count as real integration success.

## Model catalog (E03)

The catalog is a derived, provider-scoped view of the models a provider exposes.
Identity is `(provider_id, model_id)`; `canonical_id` is vendor metadata and is
never a key. Each entry carries the closed nine-capability vocabulary with
`state`, `provenance`, `confidence` and `verification`. Modality enumerations
only produce positive claims: an absent modality is `unknown`, never
`unsupported`. `provider_metadata` preserves the upstream object losslessly.

Listing is derived per request from four independent dimensions: `presence`
(last accepted sync), `freshness` (`stale_after`), `provider_state` (E02) and
`execution_verification` (always `unverified` in E03). `listing_state=current`
is a catalog condition, not proof that a model can execute a request.

Synchronization:

- At startup the module loads the snapshot, then runs one asynchronous sync
  only when the provider is enabled and healthy. Disabled, unhealthy or
  `unknown` providers are recorded as `skipped`; startup never fails.
- `POST /v1/catalog/sync` runs synchronously with `catalog.sync_timeout`,
  single-flight, and rejects an empty upstream payload as `rejected_empty`
  without deleting known models.
- An accepted sync is persisted before it is committed to memory. If the write
  fails, the accepted entries/counters/`last_success_at` remain unchanged; the
  operational attempt is recorded in memory as `failed/persistence` and is lost
  on restart. A directory-`fsync` failure is reported as
  completed-but-not-durable, not as a failed write.
- Missing models are retained with `missing_since`; `missing` counts only the
  present→absent transition of the accepted sync.

Persistence is a versioned JSON snapshot at `{AIHUB_DATA_DIR}/catalog.json`
(`0600`, directory `0700`). Writes use a temp file, file `fsync`, `rename` and a
best-effort directory `fsync`; the previous snapshot is never truncated. A
corrupt or unsupported snapshot is quarantined under a unique name before the
catalog starts empty; if quarantine fails, writes are disabled for the session
so recoverable data is never overwritten. One process owns the data directory.

## Execution lifecycle and records (E05/E06)

E05 creates one in-process execution for each request admitted to
`POST /v1/responses`. The Execution Manager owns upstream-body reading,
cancellation, timeout classification, terminal arbitration and shutdown drain;
the Gateway remains the HTTP writer and preserves progressive streaming. E05
executes an explicit target and opaque provider-ready payload: it does not
classify tasks, choose providers/models, retry, or persist conversations.

E06 observes the immutable terminal result through a narrow execution-owned
observer. The callback only performs a bounded, non-blocking enqueue; a single
writer persists minimized schema-v1 records as append-only JSONL under
`AIHUB_DATA_DIR/records`. Records contain operational metadata only—no payloads,
responses, provider sessions, headers, or credentials. `usage` is always `null`
in schema v1. Delivery is best-effort and at-most-once: queue/storage failures
cannot change the execution result, and orderly shutdown gives the writer a
fixed two-second drain before guarded provider release.

## First harness validation (E07)

E07 validates Codex CLI through the existing Responses route; it adds no
production adapter, protocol translation, retry policy, or durable AI Hub
session. The validated support boundary is Codex CLI
`0.164.0-alpha.11.1` on Linux `amd64`, using OpenCode Go through AI Hub. Full
conversation context is replayed by the harness; `previous_response_id` is not
used. AC-14 is `OUT OF SCOPE`. The final E2E matrix and historical failures are
in [`docs/validation/E01-E07-execution-history.md`](validation/E01-E07-execution-history.md)
and the [E07 validation report](validation/E07-codex-first-harness-integration.md).

## Error handling

Errors are wrapped with context and propagated upward; `main` prints the failure
to stderr and exits non-zero. Runtime request errors use standard HTTP status
codes. Panics are recovered, logged, and answered with `500`.

## Testing strategy

- Table-driven unit tests for configuration validation and correlation IDs.
- `httptest`-based fake upstreams for the provider adapter and gateway,
  including byte-for-byte request/response preservation, upstream error
  pass-through and credential-redaction assertions.
- Real listeners with short write timeouts to prove the streaming inactivity
  deadline policy (survives, aborts on silence, aborts on client disconnect).
- `-race` enabled in the documented test command.
- `internal/providers/live_smoke_test.go` covers the real upstream, strictly
  opt-in via environment variables (never part of ordinary test runs).
- Catalog tests cover the capability mapping, listing matrix, sync counters,
  snapshot round-trip, quarantine, P1/preservation and fault-injected write,
  close and directory-`fsync` failures, plus handler contracts.
- `internal/catalog/live_smoke_test.go` performs a real catalog sync when
  `AIHUB_LIVE_SMOKE=1` and a credential are present.
- `internal/execution` and `internal/records` tests cover terminal arbitration,
  cancellation, shutdown, bounded enqueue, JSONL recovery, retention and query
  contracts. The E07 live validator is separately gated and exercises the real
  Codex/OpenCode Go route only under its approved execution plan.
- E04 tests cover key grammar and bearer forms, Host/Origin normalization,
  middleware precedence, body limits and chunked overflow, media/encoding
  rejection, session generation/preservation, listener refusal, response
  streaming and E02/E03 compatibility through the real router.

# Strict Code Architecture Rules

## File Length Limits (Token Optimization)

- **Hard Limit:** No source code file may exceed **300 physical lines** under any circumstances.
- **Pre-Refactoring Trigger:** If a modification or a new feature causes a file to exceed 250 lines, you MUST stop and plan a refactoring before writing any more code.
- **Definition of Done:** A task is NOT complete if any file modified or created contains more than 300 lines.

## Automated Refactoring Protocol

If a file hits the line limit, you are strictly required to:

1. **Apply SRP (Single Responsibility Principle):** Isolate core logic from auxiliary logic.
2. **Extract Modules:** Split utility functions, sub-components, types/interfaces, or constants into separate files within the same domain directory.
3. **Orchestrate Only:** Keep the main file as a clean orchestrator, ideally below 200 lines after the split.
4. **Notify the User:** Explicitly list the new files created and explain the breakdown strategy applied.

_Failure to comply with this limit will result in token waste and context degradation. Prioritize modularity over convenience._
