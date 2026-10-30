# ADR-006 — Health endpoint semantics

- **Status:** accepted
- **Date:** 2026-10-30

## Decision

Expose two endpoints: `GET /healthz` for liveness and `GET /readyz` for
readiness. Readiness returns `503` when any registered check fails and never
serializes check error details.

## Rationale

- Liveness and readiness answer different questions and will diverge once
  providers and storage exist.
- Suppressing error text avoids leaking connection strings, hosts or credentials
  through an unauthenticated endpoint.

## Consequences

- Diagnostics rely on logs, not on the HTTP response body.
- The readiness registry is empty in E01, so `/readyz` equals `/healthz` until
  dependencies are registered.
- Checks run sequentially; concurrency can be added when checks become slow.
