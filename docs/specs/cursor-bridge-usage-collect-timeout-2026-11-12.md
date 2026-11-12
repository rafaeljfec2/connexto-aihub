# SPEC — Cursor bridge usage collect per-fetch timeout + partial success

**Status:** approved (project lead request: implement SDD fix for Cursor quota 502, 2026-11-12)  
**Classification:** bug fix  
**Workspace:** `apps/proxy/packages/providers/cursor-usage`  
**DEEPWORK:** no

## 1. Intent

Cursor quota monitoring shows “Proxy retornou HTTP 502” / `collection_failed` even when the Host Browser Bridge session is authenticated and `usage-status` returns valid JSON. Root cause: the in-page collect script awaits `get-user-prepaid-billing` (and analytics) **without per-request timeout**, so a hung prepaid call blocks the entire CDP `evaluate` until gateway timeout (~120s).

## 2. Requirements

1. Each in-page `fetch` used by Host Browser Bridge usage collect MUST abort after a bounded timeout (default **8000 ms** for primary `usage-status`; default **2500 ms** for optional prepaid/analytics).
2. After a useful `usageStatus`, optional prepaid/analytics MUST run in **parallel** and be hard-capped by a **single wall-clock budget** (default **2500 ms**) via `raceWithTimeout`, so the full CDP evaluate stays under `HostBrowserBridgeClient.defaultTimeoutMs` (**15000 ms**) even if multiple prepaid URLs hang.
3. If **any** useful source succeeds (`usageStatus`, `prepaid`, or `analytics`), collect MUST return `ok: true` with available fields; missing optional sources MUST NOT fail the collect.
4. Collect MUST return `ok: false` only when **no** useful source succeeded.
5. Existing Hub HTTP mapping remains: successful bridge collect → proxy `available` → HTTP 200; total failure → `collection_failed` → HTTP 502.
6. No dependency changes. No `.env` changes. No Hub/frontend changes in this SPEC.

## 3. Out of scope

- Improving UI error copy for `collection_failed`
- Concurrent collect locking redesign
- Local Playwright (non-bridge) collect path changes
- Restart/re-login of Windows Chrome profiles

## 4. Affected modules

| Path | Change |
| --- | --- |
| `apps/proxy/packages/providers/cursor-usage/src/bridge-usage-collect.ts` | **New** — timeout helpers + script builder + Node-testable collect orchestration |
| `apps/proxy/packages/providers/cursor-usage/src/cursor-usage-service.ts` | Use script builder in `collectUsageViaBridge` |
| `apps/proxy/packages/providers/cursor-usage/tests/bridge-usage-collect.test.ts` | **New** — Red/Green tests for timeout + partial success |
| `apps/proxy/packages/providers/cursor-usage/src/index.ts` | Export timeout constant / builder only if needed by tests (prefer package-local imports) |

## 5. Acceptance criteria

| ID | Criterion | Observable |
| --- | --- | --- |
| AC-1 | Hung prepaid fetch aborts within timeout and does not block completion | Unit test with never-resolving prepaid + fast usage-status → `ok: true` with usageStatus |
| AC-2 | All sources failing → `ok: false` | Unit test |
| AC-3 | Script builder embeds AbortController / timeout constant | Unit test asserts generated script contains timeout and `AbortController` |
| AC-4 | Service bridge path still returns `available` when evaluate returns partial envelope | Existing or focused service test with mocked bridge |
| AC-5 | `pnpm --dir apps/proxy test` passes for cursor-usage related suites; typecheck clean for package |

## 6. Edge cases

- Prepaid returns HTTP error / non-JSON → treat as null, continue.
- AbortError from timeout → treat as null, continue.
- Empty object `{}` from an endpoint → not useful; continue searching endpoints.
- Prepaid body with `{ error: ... }` → skip (preserve current “no error key” check).

## 7. Evidence

| Field | Content |
| --- | --- |
| **Claim** | Bridge usage collect succeeds when `usage-status` is available even if prepaid/analytics hang; primary fetch ≤8s and optional enrichment ≤2.5s wall-clock so the evaluate stays under the bridge client 15s ceiling |
| **Success type** | `test` |
| **Evidence path** | Vitest: `apps/proxy/packages/providers/cursor-usage/tests/bridge-usage-collect.test.ts` (+ existing bridge service test still green) |
| **Budget** | AC-1, AC-2, AC-3 minimum; AC-4 if service wiring changes; run focused vitest file then `pnpm --dir apps/proxy test` filter if available |
| **Affordance?** | n/a — pure unit tests with injectable `fetch` |

## 8. Approval

Approved for full SDD execution by project lead via chat: “implemente a correção com /sdd-workflow” after agreeing the solution (per-fetch timeout + partial success).
