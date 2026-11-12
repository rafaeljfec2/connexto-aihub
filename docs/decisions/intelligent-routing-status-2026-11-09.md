# Intelligent Routing (E08) — status audit

**Date:** 2026-11-12 (supersedes 2026-11-09 audit)

**Related:** [E08 SPEC](../specs/E08-intelligent-routing.md), [AUTO-SPEC v1.1.0](../specs/AUTO-SPEC.md), [E08.6–E08.8 acceptance](e08.6-8-formal-acceptance-2026-11-12.md)

## Phase verdict

| Layer | Status |
| --- | --- |
| E08.1–E08.5 (deterministic Intelligent Routing MVP) | **Accepted** (2026-11-03); implemented on `main` |
| E08.6–E08.8 (`aihub/auto` structural understanding, profiles, pool selector) | **Accepted** (2026-11-12) |
| E08 epic (product scope AUTO-SPEC v1.1.0) | **Closed** — Gate 0 green for seven-model allowlist |
| Next roadmap development front | **Frontend MVP** |

## Implementation snapshot

- **Auto path:** `internal/routing/auto_*.go`, Gateway headers in `responses_handler.go`.
- **Gate 0:** `AutoGate0Satisfied()` true; evidence `reports/llm-homologation/gate0-battery-2026-11-12.md`.
- **Verification:** `go test ./internal/routing/... -count=1`; gateway VirtualAuto integration test.

## Allowlist note

Three Antigravity models were removed from `AutoMVPAllowlist` after sustained HTTP 429 on retest. They remain catalog models but are not Auto candidates until re-homologated.

## Documentation

- Formal acceptance: `e08.6-8-formal-acceptance-2026-11-12.md`
- Allowlist reduction: `e08-auto-gate0-allowlist-reduction-2026-11-12.md`
- Operator report (PT-BR): `reports/llm-homologation/gate0-battery-2026-11-12.md`
- Roadmap: `docs/roadmap/AI-HUB-ROADMAP.md` (E08 row updated)
