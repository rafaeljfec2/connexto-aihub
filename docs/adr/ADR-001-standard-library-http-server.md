# ADR-001 — Standard library HTTP server

- **Status:** accepted
- **Date:** 2026-10-30
- **Context:** E01 needs routing, middleware, timeouts and graceful shutdown.

## Decision

Use `net/http` with `http.ServeMux` and Go 1.22+ method patterns. No HTTP
framework is introduced.

## Rationale

- Zero dependencies, matching the project rule to minimise them.
- Timeouts, header limits, shutdown and middleware are native.
- The routing surface in E01 is two endpoints; a framework would add no value.

## Consequences

- More explicit middleware wiring than framework-provided helpers.
- The router can be replaced without touching domain code if E04 requires
  framework-grade routing.
