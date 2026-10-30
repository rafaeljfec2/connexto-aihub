# ADR-003 — Structured logging with slog and correlation IDs

- **Status:** accepted
- **Date:** 2026-10-30

## Decision

Use `log/slog` with JSON as the default format. Generate a 128-bit hexadecimal
correlation ID per request, or reuse a valid inbound `X-Request-Id`, and carry it
through `context.Context`.

## Rationale

- Stdlib, no dependency, structured by design.
- Validating the inbound header (8–128 characters, restricted charset) prevents
  log injection and header forgery.
- Context propagation keeps the identifier available to future providers and
  execution paths.

## Consequences

- Handlers must derive their logger or read the ID from the context.
- Only `X-Request-Id` is accepted; adding another header means a deliberate
  change to the middleware.
- Access logs never include headers, query strings or bodies.
