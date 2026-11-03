# AI Hub Product Roadmap

**Status date:** 2026-11-03

**Repository snapshot:** main at 9338c538eed923c380ed2b9197b8ac6522d946ee

This document consolidates the implementation and validation state recorded in
the repository and AI Memory. Historical reports remain authoritative for the
specific evidence and limitations of each increment.

## Delivered

| Area | Status | Evidence and boundary |
| --- | --- | --- |
| E01–E07 — Backend foundation and harness | Implemented | Go modular monolith, gateway, model catalog, access control, execution lifecycle, content-free execution records, and Codex CLI harness. E07 acceptance is limited to the documented Codex CLI/Linux environment; AC-14 remains out of scope. |
| Phase B — PostgreSQL foundation | Implemented and validated | PostgreSQL pool, embedded migrations, persistence configuration, migration serialization, and isolated integration-test harness. See [ADR-020](../adr/ADR-020-postgresql-persistence-foundation.md) and [workspace guide](../development-workspace.md). |
| C1 — Provider administration | Implemented | PostgreSQL-backed provider configuration, sealed credentials, administration routes, runtime enable/disable and validation. Provider credentials are not returned by the API. |
| C2 — Multi-provider runtime | Implemented and accepted | Provider-ID-bound OpenCode Go and OpenRouter runtimes, persistence recovery, catalog and runtime diagnostics. A later acceptance decision records all 15 targeted PostgreSQL integration tests passing. |
| E08 — Intelligent routing | E08.1–E08.5 accepted | Deterministic routing, eligibility, Responses integration, explainable decisions and end-to-end coverage. E08.6–E08.8 are deferred and unauthorized. |
| E09 — Resource management | Closed | Resource admission, quota and concurrency policies, usage analytics, and provider usage metering are integrated. Ten audit findings were addressed and the closure was published in commit 9338c538. |

The detailed runtime overview is in [architecture](../architecture.md).
Specifications, ADRs, and validation reports contain the authoritative
contracts and evidence for individual capabilities.

## Evidence quality and limits

Keep these evidence types distinct when describing delivery:

- **Automated verification:** The C2 acceptance decision records 15 PostgreSQL
  integration tests passing in the isolated test-database harness, superseding
  the earlier C2.6 checkpoint that had no configured test database. E09's published closure
  records 982 passing tests and 6 skipped live/external tests, with full Go
  suites both with and without the race detector, plus vet and build checks.
- **Operational validation:** C2 reports record provider administration,
  readiness, diagnostics, configuration recovery, and explicit provider
  validation. The OpenRouter diagnostic state was observed as unknown after a
  successful probe, then reported healthy after a Hub-only restart and another
  explicit probe. The exact cause of the earlier process state was not
  established.
- **Live provider calls:** C2.5 records one OpenCode Go Responses stream and one
  authorized OpenRouter catalog sync and Responses stream. The OpenRouter
  inference returned HTTP 200 SSE and the selected model. These calls establish
  those specific invocations; they do not generalize to all OpenRouter models
  or account conditions.
- **E09 database validation:** The E09 closure records PostgreSQL-backed suites
  running through isolated per-test databases. The existing ai_hub application
  database was not used for test fixtures or migrations, and the test
  databases were removed by the test harness. Six live/external tests remained
  skipped.

## E08 — Deterministic routing boundary

E08.1–E08.5 are accepted. For the MVP:

- Respect explicit provider and model selection.
- Base eligibility on reliable catalog and runtime availability data.
- Keep routing decisions deterministic and explainable.
- Do not silently switch providers after an execution failure.
- Do not add an LLM classifier, learning, or automatic optimization.

E08.6–E08.8 remain deferred and are not authorized. Revisit them only through a
separate product decision and approved specification. See
[E08 MVP closure](../decisions/e08-mvp-closure-2026-11-03.md),
[E08.6 assessment](../decisions/e08.6-scope-assessment-2026-11-03.md), and
[ADR-022](../adr/ADR-022-deterministic-intelligent-routing.md).

## Next authorized development front: Frontend MVP

> **NEXT AUTHORIZED DEVELOPMENT FRONT**

The next development front is a small, incremental frontend that makes the
existing backend usable and observable. The frontend MVP is not a commitment to
deliver every administrative surface or the complete E11 dashboard.

The first recommended vertical increment is to initialize the frontend,
connect through a server-side same-origin proxy or BFF that keeps credentials
out of browser code, and show real service health and provider states with
loading, empty, connection-error, and success states. The current backend has
no browser session service or CORS/preflight support; the integration boundary
must be designed before a browser calls protected routes.

See the [Frontend MVP preparation](../frontend/FRONTEND-MVP.md) for the current
workspace, endpoint inventory, security constraints, architecture proposal,
and pending decisions. This document authorizes the frontend as the next
development front; implementation details and any contract changes remain
subject to their own design and approved specification.

## Future work

### E10 — Learning Engine

E10 remains future work. There are not yet real-user signals sufficient to
justify automated learning. Any later learning or optimization should be based
on representative product usage; do not add machine learning or automatic
classification preemptively.

### E11 — Dashboard

E11 remains a future dashboard evolution with scope yet to be defined. Keep it
separate from the smaller Frontend MVP and do not assume the initial frontend
must include a complete dashboard.

## Primary records

- [E08 intelligent routing specification](../specs/E08-intelligent-routing.md)
- [E09 resource management specification](../specs/E09-resource-management.md)
- [E09.4 usage analytics specification](../specs/E09.4-usage-analytics.md)
- [E09 consolidated audit and closure](../../reports/e09-consolidated-audit-2026-11-03.md)
- [C2 final operational acceptance](../validation/C2.5-live-operational-acceptance-2026-11-02.md)
- [C2 health-state acceptance](../validation/C2.6-health-state-postgres-acceptance-2026-11-03.md)

The C2 PostgreSQL acceptance decision is in AI Memory at
default/rafaeljfec2-connexto-aihub, path
decisions/c2-postgresql-validation-2026-11-03.md, page_id
01a19f9b-00f6-7de2-92e4-53bcc1b84d5e.
