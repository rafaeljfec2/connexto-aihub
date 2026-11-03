# E08 MVP closure — Intelligent Routing (E08.1–E08.5)

**Date:** 2026-11-03

**Status:** MVP complete (local implementation and formal acceptance); not published to `main` remote at HEAD `ea198436` without local diff.

## Closure verdict

The deterministic Intelligent Routing MVP defined for E08.2–E08.5 is **functionally complete** and **formally accepted** in project records (repo decisions + AI Memory). E08.1 / E08.1.1 contract artifacts remain the specification baseline.

## Increment acceptance matrix

| Increment | Deliverable | Repo evidence | AI Memory | Closure note |
| --- | --- | --- | --- | --- |
| E08.1 + E08.1.1 | Contract, SPEC, ADR-022 | `docs/specs/E08-routing-contract.md`, `docs/specs/E08-intelligent-routing.md`, `docs/adr/ADR-022-*` | `decisions/e08-formal-acceptance-2026-11-03.md` (`01a19f14…`), session E08.1.1 | Accepted; SPEC §7 table still says E08.1 “pending” — **doc drift** |
| E08.2 | Eligibility engine | `internal/routing/eligibility*.go`, tests | Session `47bd9dba…`, E08.1 acceptance page | Accepted in memory; **no dedicated `e08.2-formal-acceptance` repo file** |
| E08.3 | Deterministic selector | `internal/routing/selector*.go` | `decisions/e08.3-formal-acceptance-audit-2026-11-03.md` | Accepted |
| E08.4 | Gateway integration | `internal/gateway/responses_routing.go`, `cmd/ai-hub/gateway_routes.go` | `decisions/e08.4-formal-acceptance-independent-review-2026-11-03.md` (`01a19f2a-bed0…`) | Accepted (review APPROVED); repo `e08.4-gateway-routing-integration.md` status lags formal acceptance |
| E08.5 | Observability + E2E | `internal/routing/observability.go`, gateway observability tests | `decisions/e08.5-independent-review-formal-acceptance-2026-11-03.md` (`01a19f92…`) | Accepted |

## Code vs roadmap alignment

| Area | Expected (SPEC/ADR) | Observed in repo |
| --- | --- | --- |
| Production routing path | E08 pipeline on Responses when configured | `ResponsesRouting` wired in `gateway_routes.go` |
| Deterministic selection | No LLM classifier in MVP | No classifier code in `internal/routing` |
| Gateway HTTP mapping | Contract §6.3 + E08.4 table | `mapRoutingRejection` + operational override |
| Content-free evidence | §8 bounded logs | `routing.LogDecision` |
| Legacy tests | Nil routing for codex fixtures | Handlers pass `nil` routing in selected tests |
| E06 schema | Unchanged | No E08 changes under `internal/records` |

## Acceptance criteria gaps (non-blocking for MVP closure)

| Gap | Severity | Notes |
| --- | --- | --- |
| SPEC §7 increment table outdated (E08.1 pending, E08.2 in progress, E08.4 “implemented” only) | Documentation | Does not invalidate code; should be reconciled on next doc pass |
| Missing repo files for E08.2 and E08.4 **formal** acceptance (wiki/memory hold truth) | Traceability | Optional repo mirrors of memory decisions |
| Legacy path without `routing decision` log | Known limitation | Documented in E08.5 acceptance |
| C2 PostgreSQL integration tests | Validation debt | Independent of E08 MVP |
| OpenRouter SSE test flake | Pre-existing | Intermittent catalog sync race |

## Validation snapshot (2026-11-03 closure audit)

- `go test ./... -count=1 -race`: **866** tests, 25 packages — PASS
- Local diff: E08.4–E08.5 + observability not committed at HEAD `ea198436`

## Explicit non-goals closed with MVP

- E08.6–E08.8 intelligent classification and optimization profiles (vision only)
- E09–E11
- Durable routing decision API / E06 schema v2

## Recommended next actions (planning only)

1. **Publish** local E08 work (commit/PR) when authorized — MVP is not on remote at current HEAD alone.
2. **Reconcile** SPEC §7 table with accepted increments.
3. **Defer E08.6** unless product requires auto task routing beyond deterministic MVP (see `e08.6-scope-assessment-2026-11-03.md`).
4. Address **C2 PostgreSQL** validation debt on its own track.
