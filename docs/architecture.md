# AI Hub — Implemented Architecture (E01)

This document describes what exists today. Product vision and future layers are
in `AI-Hub-Product-Architecture-Kickoff.md`; only E01 is implemented.

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
 internal/server (HTTP transport)
   |            |
   v            v
internal/     internal/health
observability
```

Dependencies are explicit and acyclic. `cmd` wires every package and owns the
construction order; `app` depends only on the standard library (it knows the
`Module` contract, not the concrete modules); `server` depends on `config` and
`observability`; `health` and `observability` depend on the standard library
only.

## Packages

| Package                   | Responsibility                                                        |
| ------------------------- | --------------------------------------------------------------------- |
| `cmd/ai-hub`              | Process entrypoint, flags, signal handling, shutdown budget.          |
| `internal/app`            | `Module` contract, ordered start, reverse-order stop, rollback.       |
| `internal/config`         | Typed configuration, strict JSON decoding, env overrides, validation. |
| `internal/server`         | Listener, `http.Server`, routing tree and HTTP middleware.            |
| `internal/health`         | Readiness registry and liveness/readiness handlers.                   |
| `internal/observability`  | `slog` construction and correlation ID context propagation.           |

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
http.ServeMux  ->  GET /healthz | GET /readyz
```

The access log intentionally excludes headers, query strings and bodies, so
credentials cannot leak through it. Query strings are excluded because they can
carry API keys. Recovery runs inside access logging, so panicked requests are
logged too; if the handler already committed a status, recovery logs the panic
without appending error text to the partial body.

## Configuration

Precedence is `defaults < file < environment`. Validation aggregates every
problem into one error. `server.port` must be 1–65535; timeouts must be positive;
`read_header_timeout` cannot exceed `read_timeout`.

Environment-only values are limited to the documented `AIHUB_*` variables; the
application never reads provider credentials yet.

## Health semantics

- `GET /healthz` answers liveness only. It does not touch dependencies.
- `GET /readyz` runs registered checks sequentially and returns `up`/`down`.
  Check error details are never serialized; only `ok`/`error` per check is
  exposed, to avoid leaking connection strings or credentials.

The registry is empty in E01 because no dependency exists. Checks are registered
as E02/E03 introduce provider and storage dependencies.

## Error handling

Errors are wrapped with context and propagated upward; `main` prints the failure
to stderr and exits non-zero. Runtime request errors use standard HTTP status
codes. Panics are recovered, logged, and answered with `500`.

## Testing strategy

- Table-driven unit tests for configuration validation and correlation IDs.
- `httptest` for handlers and middleware, including propagation, method handling,
  panic recovery and absence of credential material in logs.
- Real listener tests for startup failure, health serving and in-flight request
  completion during shutdown.
- `-race` enabled in the documented test command.
