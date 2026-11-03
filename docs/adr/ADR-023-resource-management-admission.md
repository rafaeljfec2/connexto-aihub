# ADR-023 — Resource admission and quota accounting

- **Status:** accepted — project lead approved 2026-11-03 (E09.1 authorized; later increments require separate approval)
- **Date:** 2026-11-03
- **Related:** SPEC E09, ADR-009, ADR-012, ADR-014, ADR-017, ADR-018, ADR-020, ADR-021, ADR-022

## Context

E09 needs a durable, concurrency-safe decision after E08 has selected an exact provider/model target and before E05 starts execution. E06 schema v1 deliberately leaves `usage` null, and the current provider adapters do not extract normalized token or cost data. The gateway also has one shared client principal, so it cannot assign quotas to individual users or tenants.

ADR-012 uses “admission” for bounded HTTP request-body validation before dispatch. E09 resource admission is a separate operation: it evaluates consumption policy for the routed target and does not replace or alter E04 body admission.

## Decision

1. **Placement.** Run Resource Admission after successful E08 routing and before `execution.Manager.Execute`. It receives the selected `(provider_id, model_id)` pair. A routing rejection or failed admission does not start E05 or call a provider.
2. **Accounting unit.** Count one Responses execution admitted by the Hub. This unit does not assert that the provider received the request, inference completed, or any tokens or money were consumed.
3. **Period.** Use UTC calendar days. Rolling windows and provider-specific billing periods are outside this decision.
4. **Scopes.** Support a global scope, a provider scope, and a provider/model scope. Model identity always includes its provider. Every enabled applicable policy must allow admission; a successful request increments all applicable scopes.
5. **Persistence.** PostgreSQL is required when E09 resource enforcement is enabled. Persist policies and period counters in PostgreSQL and restore them from that source after process restart; never reconstruct quota state from E06 or memory. File persistence retains its current behavior and does not provide E09 quota enforcement. PostgreSQL may remain local to preserve the local-first deployment model.
6. **Atomicity and concurrency.** Check applicable policy capacity and increment every corresponding period counter in one database transaction. A rejection or transaction failure must leave all counters unchanged. Concurrent requests for the last available unit must not exceed the configured limit. The concrete SQL, lock ordering, and transaction isolation are implementation decisions, subject to these guarantees.
7. **Consumption and lifecycle.** Consume the unit when the admission transaction commits. Do not refund it after provider failure, timeout, cancellation, or stream failure. A process stop after the commit but before E05 starts also leaves the unit consumed. Policy changes apply to later admissions; already admitted executions continue without interruption.
8. **Fail closed.** If an enabled policy cannot be evaluated safely because persistence is unavailable, do not bypass it. Return `503 resource_admission_unavailable`. When an applicable limit is exhausted, return `429 resource_limit_exceeded`. Keep both in the existing local error envelope and do not expose database details.
9. **Internal transaction idempotency.** Before its first database attempt, assign an internal stable admission identifier. Persist that identifier atomically with the admission decision and all counter increments. Any retry of the same admission transaction must reuse the identifier; if a prior commit succeeded but its acknowledgement was lost, the retry must return the committed decision without incrementing again. `X-Request-Id` remains correlation data and is not an idempotency key. This decision adds no public idempotency API.
10. **Boundaries.** Do not change E05 lifecycle ownership, E06 schema-v1 records or `usage: null`, E08 selection, or provider request/response payloads. Token and monetary limits, per-user/tenant policies, concurrency ceilings, refunds, provider retries/fallback, and distributed infrastructure remain deferred.

## Rationale

- The Hub can enforce admission counts with current evidence but cannot reliably enforce token or monetary limits without parsing and validating provider usage.
- PostgreSQL transactions provide the existing durable coordination mechanism for concurrent admissions and restart recovery without adding infrastructure.
- Counting at commit avoids refunds based on incomplete or best-effort E05/E06 observations. Naming the unit as a Hub admission makes its limits explicit.
- A separate internal identifier handles ambiguous commit acknowledgement without changing the meaning of the externally supplied correlation ID.

## Consequences

- A committed admission can consume a daily unit even when no provider invocation is confirmed. This is deliberate and must be visible in usage responses and operator documentation.
- File-only deployments retain their current behavior and do not expose E09 policy administration or enforcement.
- Repeated client requests, including requests that reuse an `X-Request-Id`, are separate admissions. The MVP has no provider retry or fallback behavior.
- E06 remains an independent, best-effort execution history and is not used to reconstruct quota counters.
- Implementation must select a schema and cleanup policy for stable internal admission identifiers, and prove the atomicity and idempotency guarantees with isolated PostgreSQL integration tests. Identifier retention must cover every possible retry of its admission. It must not weaken the guarantees in this ADR.

## Alternatives considered

| Alternative | Decision |
| --- | --- |
| Count provider calls, tokens, or cost from the current adapters | Rejected; those facts are not extracted as normalized, reliable usage values. |
| Reconstruct quota usage from E06 records | Rejected; E06 is asynchronous, bounded, at-most-once, and schema v1 has `usage: null`. |
| Use in-memory counters or file-mode counters for E09 | Rejected for the MVP; they do not provide the selected durable cross-request transaction guarantee. |
| Refund a unit when execution fails or is canceled | Rejected; terminal observation and provider outcomes cannot reliably establish billable usage or undo a committed admission. |
| Treat `X-Request-Id` as the idempotency key | Rejected; it is a caller-visible correlation value, not an execution identity or retry contract. |
| Add a public idempotency API | Deferred; not needed for internal database transaction retry safety. |

## Approval boundary

Project lead approved this ADR together with [SPEC E09](../specs/E09-resource-management.md) on 2026-11-03. Only increment **E09.1 — Resource Admission Core** is authorized; E09.2 and later increments require separate approval.
