# ADR-005 — Loopback binding without gateway authentication in E01

- **Status:** accepted, must be revisited in E04
- **Date:** 2026-10-30

## Decision

E01 binds to `127.0.0.1:8080` by default and does not implement gateway
authentication. Authentication is deferred to the API Gateway epic.

## Rationale

- Authentication is not part of the E01 scope and would expand the increment.
- Loopback binding is the safest available default for a local-first tool.

## Consequences and risk

- Loopback is **not** a trust boundary: any local process, and browser-based
  cross-origin or DNS-rebinding attacks, can reach the endpoint.
- Therefore no credential is stored or proxied yet. Provider credentials (E02)
  must not be exposed before a token-based gateway authentication and
  `Host`/`Origin` validation exist.
- Binding to a non-loopback host is possible by configuration and is explicitly
  unsupported until E04. The server logs a warning when the bound address is not
  loopback, derived from the actual listener rather than from configuration.
