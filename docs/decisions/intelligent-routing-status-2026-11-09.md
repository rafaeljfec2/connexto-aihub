# Intelligent Routing (E08) — status audit

**Date:** 2026-11-09

**Repository:** `connexto-aihub` @ `60d0d963e0950f7c63894c7e6d8857ab6085766d` (`main`)

**Related:** [E08 SPEC](../specs/E08-intelligent-routing.md), [MVP closure](e08-mvp-closure-2026-11-03.md), [E08.6 assessment](e08.6-scope-assessment-2026-11-03.md), [ADR-022](../adr/ADR-022-deterministic-intelligent-routing.md)

## Phase verdict

| Layer | Status |
| --- | --- |
| E08.1–E08.5 (deterministic Intelligent Routing MVP) | **Accepted** (2026-11-03); **implemented on `main`**; wired on `POST /v1/responses` |
| E08.6–E08.8 (classification, optimization profiles, ranked selection) | **Unauthorized** — vision only |
| Next roadmap development front | **Frontend MVP** — not E08.6 |

## Implementation snapshot

- **Packages:** `internal/routing/*` (eligibility, selector, snapshot, validate, observability) and `internal/gateway/responses_routing.go`.
- **Wiring:** `cmd/ai-hub/gateway_routes.go` passes `ResponsesRouting` into the Responses handler.
- **Verification (2026-11-09):** `go test ./... -count=1` — 1100 tests PASS; `go test ./internal/routing/...` — 59 tests PASS.

## Post-closure code changes (still MVP contract)

- Cursor/Codex capability projection fix (`820eb74`).
- OpenCode Responses input normalization and `image_input` catalog enrichment.
- Antigravity model prefix handling in `SelectDeterministic`.

No ranking among multiple eligible candidates was added.

## Documentation and memory

- Authoritative MVP closure remains `e08-mvp-closure-2026-11-03.md`.
- Pinned AI Memory closure page still references unpublished git at `ea198436`; **outdated** — E08 is on `main` as of this audit.
- Operator report (PT-BR): `reports/levantamento-intelligent-routing-2026-11-09.md`.
- AI Memory audit page: `decisions/intelligent-routing-status-audit-2026-11-09.md`.

## Common misreadings

1. **`reports/llm-homologation-remediation.md` §11** — gate for Auto/cost-driven product routing and homologation; **not** absence of E08.2–E08.5.
2. **Unified Quota Management Stage 2** — future quota-signal routing; separate spec from E08.6+.
3. **`AI-Hub-Product-Architecture-Kickoff.md`** — historical; superseded for E08 delivery by architecture and roadmap.

## Recommendations (planning only)

- Do not start E08.6 without an approved mini-SPEC; E08.7 likely required to consume task metadata.
- Revisit Auto/quota routing only through explicit product authorization.
- Use this document and the PT-BR report for operational status checks until the next increment is approved.
