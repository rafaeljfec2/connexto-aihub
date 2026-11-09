# FE-10 Discovery — Cursor Usage Analytics Screen (Grok Console dailyMetrics)

**Status:** proposed — awaiting lead approval before SPEC refinement
**Date:** 2026-11-08
**Related:** `docs/specs/FE-PROVIDER-QUOTA-CONSUMPTION.md`, `docs/decisions/cursor-quota-percentage-presentation.md`, forced-refresh flow in `ProvidersPage` (2026-11-08)

## 1. Problem

The Cursor provider card in `/providers` correctly reports weekly included-usage percent and prepaid credits via the Cursor Usage API, but the **Grok (xAI) console is where the subscription actually meters the Cursor agent activity**. The provider detail panel currently cannot show any official reading of this activity ("Sem leitura oficial de franquia Go" for the Go quota family), and there is no screen that exposes the rich daily analytics the Grok console already provides.

## 2. Data source

`GET https://grok.com/dashboard/api/dashboard/get-user-analytics`

- **Auth:** grok.com browser session (cookie-based). Credentials are not shared with the Cursor Usage API flow; a dedicated auth chain is required (see §6 research items).
- **Response shape** (observed from a real session):

```
dailyMetrics[]: {
  date: string            // epoch ms as string, UTC day bucket
  linesAdded?: number
  linesDeleted?: number
  acceptedLinesAdded?: number
  acceptedLinesDeleted?: number
  totalApplies?: number
  totalAccepts?: number
  totalRejects?: number
  totalTabsShown?: number
  totalTabsAccepted?: number
  chatRequests?: number
  agentRequests?: number
  subscriptionIncludedReqs?: number
  usageBasedReqs?: number
  cmdkUsages?: number
  modelUsage?: { name: string; count: number }[]     // e.g. default, grok-4.7-high, composer-2.5, gpt-5.1-codex-max
  extensionUsage?: { name?: string; count: number }[]  // file-language mix; name may be absent
  tabExtensionUsage?: { name?: string; count: number }[]
  clientVersionUsage?: { name: string; count: number }[] // Cursor client versions
}
period: { startDate: string; endDate: string }   // epoch ms bounds
applyLinesRank, tabsAcceptedRank, totalTeamMembers, totalApplyLines, teamAverageApplyLines, totalTabsAccepted, teamAverageTabsAccepted, totalMembersInTeam
```

- Several optional fields are absent on idle days; counts of `1` appear as single-element arrays where `count` carries the value and `name` may be missing. All aggregation code must treat missing fields as absent (never zero), consistent with the provider-reported-quota ADR.
- The endpoint returns the **full observed period** (startDate 2025-11-09 through endDate now in the observed payload); no query parameters were required for the covered range.

## 3. Product goal

A dedicated sidebar screen **"Cursor"** in AI-HUB that exposes the subscription's Cursor activity ingested from the Grok console, consistent with the control-plane look and the provider-reported-only philosophy (no estimated values; only what the console reports).

### Proposed layout (mirrors existing control-plane patterns)

1. **Page head** — eyebrow `CURSOR ACTIVITY / GROK CONSOLE`, title, lead; period selector (day / 7d / 30d / all-period, clamped to `period`).
2. **Summary cards** (same slot geometry as `providers-metrics`):
   - Agent requests (period total; Chat requests shown as secondary)
   - Lines added vs. accepted (and deleted vs. accepted-deleted)
   - Applies / accepts (+ rejects) and accept rate
   - Subscription-included vs. usage-based request split
3. **Daily activity panel**: bar/line chart `agentRequests` (+ optionally `linesAdded`) per UTC day; toggle All / Agent / Chat like the console heatmap. Reuse the `usage_chart` presentation types (`ProviderUsageChart`) already defined for quotas.
4. **Model usage panel**: ranked table of `modelUsage` aggregated across days (top N + "Others"), with per-day availability.
5. **Activity heatmap (possible second slice)**: GitHub-style calendar grid of agent requests with streak stats (most active day/month, current streak — all derivable from `dailyMetrics`).
6. Extension mix and client versions as secondary tables only if space allows (candidate for deferral).

## 4. Architecture (per repository conventions)

### Backend (Go gateway)

- New endpoint: `GET /api/ui/providers/cursor/usage-analytics` (BFF proxy contract unchanged; `vite.config.ts` untouched).
- New service consistent with `QuotaService` patterns:
  - In-memory cache by Cursor account, **staleness TTL (default 2 min, same constant)**.
  - Returns latest snapshots immediately; triggers **background collection** when stale, or immediately when `?refresh=true` is passed, and flags `refreshing: true` while collecting — mirroring the forced-refresh flow shipped for `/providers` on 2026-11-08 so the sidebar screen can reuse the identical UX contract (cache visible + spinner until collection settles).
- Explicit mapping exceptions vs. quotas: this is **usage analytics**, not a franchise reading. It must not surface in `quotaPresentation` grid arithmetic, the "Atingido/Próximo do limite" summaries, or any place where quotas are interpreted as limits.

### Frontend (feature-first)

- Feature module `apps/frontend/src/features/cursor/`: `api/queries.ts` (`cursorQueryKeys.usageAnalytics`), `types.ts` mirroring the payload, `hooks/useCursorUsageQueries.ts`, `pages/CursorUsagePage.tsx`, `components/` (summary cards, daily chart, model table), page-shaped skeleton (`CursorUsagePageSkeleton` in the same slots), tests.
- `/cursor` route registered in `app/router.tsx`; sidebar item under **Operação** right after "Provedores" (`AppSidebarNavigation`); `pageMetaByPath['/cursor']` entry with PT-BR strings consistent with the rest of the UI.
- Manual refresh button reuses the forced-refresh + polling contract (force endpoint, keep content, `motion-safe` spinner, sr-only status).
- No `shared` → `features` imports; analytics types stay inside the feature unless the chart types are reused from `features/overview/types/quota` via the feature's public exports per the architecture decision.

## 5. Acceptance criteria (draft, pre-SPEC)

1. First load renders a page-shaped skeleton occupying the same grid slots; error state offers retry.
2. Summary cards and charts render only values reported by the endpoint (missing fields absent, never rendered as zero).
3. Clicking "Atualizar" physically re-collects via `?refresh=true` and keeps cached content visible until `refreshing` flags settle (≤120 s cap).
4. Screen navigation keeps TanStack cache; returning reuses it without forcing a collection.
5. Aggregations (28-day period, streaks, model ranking) reconcile exactly with the summed `dailyMetrics` — covered by unit tests.
6. `tsc --noEmit && vitest run` remain green; feature tests cover aggregation, missing-field tolerance, and loading/error/refresh states.

## 6. Research items (Phase R1 — spike before SPEC)

1. **Auth chain**: confirm the endpoint works with the Connexto Proxy host-browser-bridge session capture used for Cursor quota; evaluate storing/refreshing the grok.com session cookie in provider credentials. Determine cookie TTL and refresh strategy.
2. **Endpoint stability & params**: whether `get-user-analytics` accepts period/query params (observed call used none); rate limits; response variability for team accounts (`totalMembersInTeam > 1`).
3. **Scope decision**: does the screen read analytics for the Cursor provider account linked to the grok.com session, or must it be tied to a configured provider record (`isConfigured` gate)?
4. **History retention**: whether AI-HUB should persist daily metrics snapshots in PostgreSQL (E-series storage) for longer-than-console history, or stay stateless like the quota screen.

## 7. Open questions for the lead

1. Sidebar label and placement: "Cursor" under Operação (proposed) — confirm.
2. Should the summary also be surfaced inside the Cursor detail panel on `/providers` (link or widget), or dedicated screen only?
3. Is the heatmap part of the first slice or deferred?
4. Session-cookie storage preference: Connexto Proxy bridge vs. credential storage on AI-HUB (affects secrets governance on the proxy repo).

## 8. Phasing proposal (RPI → SDD)

- **R1** — spike: auth chain + endpoint contract validation on a real session (evidence in this docs tree).
- **D1** — SPEC refinement (`docs/specs/FE-10-cursor-usage-analytics.md`) + backend contract.
- **R2** — Red tests (aggregation pure functions, service TTL/force semantics, page states).
- **G** — Green implementation: backend service + endpoint, then frontend feature + route + sidebar + skeleton wiring (reuse the forced-refresh pattern).
- **V** — validation report + `senior-code-review-judge` independent review before close.
