# AI Hub — Implemented Architecture (E01–E02)

This document describes what exists today. Product vision and future layers are
in `AI-Hub-Product-Architecture-Kickoff.md`; E02 (first provider integration
plus gateway passthrough) is implemented alongside E01.

## Runtime model

Single Go binary, modular monolith, local-first. No database, queue, cache or
external service is required.

```text
cmd/ai-hub/main.go
         |
         v
    internal/app  (lifecycle)
         |
         v
  internal/providers (opencode-go adapter, E02)
         |
         v
 internal/server (HTTP transport)
   |    |         |
   v    v         v
internal/       internal/
gateway (E02)   observability
```

Health endpoints are served by `health`; provider endpoints are injected as
additional routes into the same router.

Dependencies are explicit and acyclic. `cmd` wires every package and owns the
construction order; `app` depends only on the standard library (it knows the
`Module` contract, not the concrete modules); `server` depends on `config` and
`observability`; `health` and `observability` depend on the standard library
only.

## Packages

| Package                   | Responsibility                                                        |
| ------------------------- | ------------------------------------------------------------------------ |
| `cmd/ai-hub`              | Process entrypoint, flags, signal handling, shutdown budget.          |
| `internal/app`            | `Module` contract, ordered start, reverse-order stop, rollback.       |
| `internal/config`         | Typed configuration, env defaults overrides, validation, provider settings. |
| `internal/server`         | Listener, `http.Server`, routing tree and HTTP middleware.            |
| `internal/health`         | Readiness registry and liveness/readiness handlers.                   |
| `internal/observability`  | `slog` construction and correlation ID context propagation.           |
| `internal/providers`      | OpenCode Go adapter: credential handling, catalog, Responses execution, provider state (E02). |
| `internal/gateway`        | Provider passthrough handlers: Responses, models catalog, provider diagnostics (E02). |

`providers` is a concrete adapter, not an interface-driven layer: the spike
validated exactly one adapter shape (Responses in, Responses out); an
abstraction will be introduced only when a second provider exists.

## Lifecycle

1. `main` loads and validates configuration. Invalid input aborts before any
   listener is created.
2. The logger is built from the validated log settings.
3. Modules are constructed explicitly and injected into `app.New`.
4. `App.Start` starts modules in order, stopping already started modules in
   reverse order if one fails.
5. The process waits for `SIGINT`/`SIGTERM` or a fatal serving error.
6. `App.Stop` runs with a context limited by `server.shutdown_timeout` and stops
   modules in reverse order, joining errors.

`server.Start` binds the listener synchronously, so port conflicts fail startup
instead of surfacing asynchronously.

## Request pipeline

```text
X-Request-Id validation/generation
        |
        v
panic recovery
        |
        v
access logging (method, path, status, bytes, duration, correlation id)
        |
        v
http.ServeMux ->  GET /healthz | GET /readyz
   |              | GET /v1/models        | GET /diagnostics/providers
   |              | POST /v1/responses    (streaming, byte-for-byte pass-through)
```

The access log intentionally excludes headers, query strings and bodies, so
credentials cannot leak through it. Query strings are excluded because they can
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

An invalid base URL (scheme, host, query/fragment or embedded credentials)
aborts startup; a missing API key does not — the provider starts disabled.

## Health semantics

- `GET /healthz` answers liveness only. It does not touch dependencies.
- `GET /readyz` runs registered checks sequentially and returns `up`/`down`.
  Check error details are never serialized; only `ok`/`error` per check is
  exposed, to avoid leaking connection strings or credentials.

The registry is empty as long as no dependency is registered. Provider
availability deliberately does **not** flip readiness: a disabled or unhealthy
provider is a diagnostic state exposed at `GET /diagnostics/providers`, never a
decision about whether the application itself is fine (see E02 SPEC decision
D-R).

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
