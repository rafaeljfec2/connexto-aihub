# ADR-001 — Fiber HTTP server with net/http-compatible handlers

- **Status:** superseded by the Fiber v2 transport migration
- **Date:** 2026-10-30
- **Context:** E01 needs routing, middleware, timeouts and graceful shutdown.

## Decision

Use Fiber v2 as the only network HTTP server. Existing application handlers and
middleware continue to use the `net/http` contracts behind an explicit adapter
so the migration does not rewrite gateway or domain behavior.

## Rationale

- Fiber provides the required network server and graceful shutdown primitives.
- The adapter preserves the established handler, authentication and error
  contracts while allowing the transport to migrate incrementally.
- Streaming uses Fiber/fasthttp's body stream writer rather than the generic
  net/http adapter, which would buffer response bodies.

## Consequences

- The adapter must never retain a `fiber.Ctx` after a request returns.
- Streaming and cancellation require real-listener tests; unit tests alone are
  insufficient evidence.
