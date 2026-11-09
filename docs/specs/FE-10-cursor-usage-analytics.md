# FE-10 SPEC — Cursor Usage Analytics in Provider Drawer

**Status:** approved for execution by the project lead (2026-11-08, based on the approved discovery plan `docs/discovery/FE-10-cursor-usage-analytics-discovery.md`)
**Evidence path:** `docs/validation/FE-10-cursor-usage-analytics.md`
**Success type:** typed green loop (failing tests written first under Phase 5, then production code)

## 1. Intent

Extend the existing Cursor provider detail drawer shown from the Overview quota sidebar panel with agent activity reported by the grok.com console `get-user-analytics` endpoint. Keep the existing drawer and its provider selection; do not add a route or sidebar navigation item. The drawer shows compact summary metrics, a recent daily-activity chart, and model usage ranking. Heatmap, PostgreSQL history, and a separate page are out of scope.

## 2. Established facts (evidence from code inspection)

1. The Connexto Proxy cursor collect flow (`packages/providers/cursor-usage/src/usage-normalize.ts`) already captures the full `get-user-analytics` payload and embeds it as `usageMap["raw_analytics"]` in the collect response. The AI-HUB adapter (`internal/providers/cursor_usage_quota.go`) currently discards it, deriving only `analytics_agent_requests_total`. **No proxy change is required.**
2. The existing collect route is `POST {endpoint}/providers/cursor/usage/collect` with headers `Content-Type`, `X-Gateway-API-Key` (management credential), `X-Hub-Account-Id`. Non-2xx and `status == ["authentication_required"|"session_expired"|...]` map to snapshot statuses (`QuotaStatusUnauthorized`, etc.).
3. Backend admin routes are registered in `cmd/ai-hub/provider_admin.go` (`providerAdminRoutes`); handlers live as methods on `gateway.ProviderAdmin`; UI routes under `/api/ui/**` are protected by the operator-session access policy (`internal/access/authorize.go`).
4. Quota cache semantics to mirror: `defaultQuotaStaleTTL = 2 * time.Minute`, serve cached immediately, background refresh when stale or `force`, `Refreshing` flag surfaced, concurrent refresh per account coalesced (`internal/providers/quota_service.go`).
5. Frontend forced-refresh contract shipped on 2026-11-08 in `features/providers/pages/ProvidersPage.tsx` (force call → poll every 1.5 s → 120 s cap → sr-only status + `motion-safe` spinner → warning on failure/timeout).
6. Feature-first architecture: per-feature query keys, no `shared` → `features` imports, routes in `app/router.tsx`, nav in `app/layouts/AppSidebarNavigation.tsx`, page metadata in `app/layouts/pageMeta.ts`.

## 3. Backend (Go) — changes

### New `internal/providers/cursor_usage_analytics.go`

- `CursorUsageAnalyticsEnvelope` (with `readonly` JSON contract):
  - `provider_id`, `provider_name`
  - `analytics json.RawMessage` — raw `get-user-analytics` payload, untouched (schema owned by the frontend types)
  - `fetched_at` (RFC3339 at response build time)
  - `source_collected_at` (RFC3339 of the last successful proxy collection)
  - `status`: `available | authentication_required | session_expired | unconfigured_credential | unconfigured_gateway_key | analytics_unavailable | collecting`
  - `error *ProviderQuotaError` (existing reusable type)
  - `refreshing bool`
- `CursorUsageAnalyticsService` (package `providers`, reuses `Service.managementCredentialOf`, `allowedHosts` target guard from `CursorUsageQuotaAdapter.target` pattern, and repo access):
  - Per-provider in-memory cache `{envelope, lastAttemptAt, refreshing}` with mutex; `defaultUsageAnalyticsStaleTTL = 2 * time.Minute` (`WithUsageAnalyticsTTL` option).
  - `Get(ctx, providerID, force bool)`: serve cache immediately; trigger background collection when cold, stale, or forced (single-flight per provider via `refreshing` flag). Cold cache returns `status: "collecting"` with `analytics: null` rather than blocking up to 120 s.
  - Collection goroutine: POST proxy collect (120 s timeout), extract `usage["raw_analytics"]`, map proxy `status` strings to envelope status; on HTTP 401/`authentication_required`/`session_expired` → `authentication_required` status; missing `raw_analytics` → `analytics_unavailable`.
- Provider resolution helper: default to the **first enabled provider with `Type == TypeCursor`**; explicit `?providerId=` overrides (must be a cursor provider).

### `internal/gateway/providers_admin_handler.go`

- `ProviderAdmin.BindCursorUsageAnalytics(svc)` + `CursorUsageAnalytics() http.Handler` with method guard, `?refresh=true` passthrough and `?providerId=` passthrough.

### `cmd/ai-hub/provider_admin.go` + `persistence_runtime.go`

- Construct `CursorUsageAnalyticsService`, bind to `ProviderAdmin`, route `{GET, "/api/ui/providers/cursor/usage-analytics"}` appended to `providerAdminRoutes` (operator-session policy applies automatically through the `/api/ui` namespace).

## 4. Frontend (React) — changes

### New feature `apps/frontend/src/features/cursor/`

- `types.ts` — typed mirror of the `get-user-analytics` payload: `CursorDailyMetric` (all fields optional, arrays optional), `CursorActivityUsageEntry { name?: string; count: number }`, `CursorUsageAnalyticsPeriod`, `CursorUserAnalytics`, plus `CursorUsageAnalyticsEnvelope` for the BFF response.
- `api/queries.ts` — `cursorQueryKeys.usageAnalytics(providerId)`; `getCursorUsageAnalytics(providerId?)` and `getCursorUsageAnalyticsForce(providerId?)` reading the BFF envelope (typed; no `any`).
- `lib/cursorAnalytics.ts` — pure aggregations over `dailyMetrics[]`:
  - `summarizeCursorDailyMetrics(metrics)` → totals: agentRequests, chatRequests, subscriptionIncludedReqs, usageBasedReqs, linesAdded, linesDeleted, acceptedLinesAdded, acceptedLinesDeleted, totalApplies, totalAccepts, totalRejects, activeDays, firstActivityAt, lastActivityAt. Missing fields aggregate as absent (never rendered as zero when the entire source is absent).
  - `rankCursorModelUsage(metrics, limit)` → aggregated `{name, count}` sorted desc with an "Others" bucket beyond `limit`.
  - `buildCursorDailySeries(metrics, metric)` → `{date, value, hasData}[]` for metric agents/chat/lines-added toggling (absent fields → `hasData: false`, rendered as gap, not zero).
- `hooks/useCursorUsageQueries.ts` — TanStack Query mirroring the ProvidersPage contract: normal GET on mount; `refresh()` forces (`?refresh=true`), keeps cached content, polls 1.5 s until `refreshing` is false and final state present (120 s cap → warning), reuses cache for navigation.
- `components/CursorUsageDrawerSection.tsx` — compact summary metrics, recent daily agent-request chart, model ranking, physical refresh action, and loading/collecting/error statuses.
- `index.ts` publicly exports `CursorUsageDrawerSection` as the cross-feature integration point.

### Existing files edited

- `features/overview/components/ProviderQuotaDetailDrawer.tsx` — render the Cursor usage section in the existing drawer when `provider_id === 'cursor'`.
- `features/cursor/index.ts` — expose the drawer section as the feature's public integration point.

## 5. Acceptance criteria

1. **AC-1** Opening the existing Cursor provider drawer loads the analytics section without replacing the drawer or navigating to a new screen; loading and collecting states are explicit.
2. **AC-2** With a successful collection, summary cards and panels render exactly the endpoint-reported values; fields absent in the source never render as `0`.
3. **AC-3** Clicking "Atualizar" triggers `GET /api/ui/providers/cursor/usage-analytics?refresh=true`; cached content stays visible; the button is disabled and the refresh icon spins (`motion-safe`) until polling settles; a failure or 120 s timeout surfaces the warning "Não foi possível atualizar; exibindo os dados carregados anteriormente."
4. **AC-4** Backend TTL semantics: within 2 min, non-forced GETs serve the cache; forced GETs always trigger background collection and return the existing cached analytics with `refreshing: true`. Unit tests cover cold/stale/forced/concurrent coalescing.
5. **AC-5** Envelope status mapping is covered: `authentication_required`, `analytics_unavailable`, `collecting` render distinct UI states (no fake zeros/numbers).
6. **AC-6** Drawer provider selection remains unchanged; when the selected provider is Cursor, show its usage; for other providers, preserve existing drawer behavior.
7. **AC-7** Aggregation pure functions reconcile with manually summed fixtures (unit tests), tolerating optional/absent fields and `count`-only entries.
8. **AC-8** Full gates green: `go test ./...`, `go vet`, `tsc --noEmit && vitest run`.
9. **AC-9** No new route/sidebar item/page metadata; existing overview drawer and other pages retain their current navigation and behavior.

## 6. Out of scope (this slice)

- Heatmap with streaks, extension mix, client versions tables.
- PostgreSQL persistence of historical analytics.
- `/providers` detail-panel widget for Cursor analytics.
- Changes to Connexto Proxy (its payload is consumed as-is).

## 7. Edge cases

- `raw_analytics` absent or non-object → `analytics_unavailable`; UI shows explicit state.
- Provider without management credential → `unconfigured_gateway_key` → UI guidance to configure the gateway key in `/providers`.
- Multiple cursor providers → default first enabled; `?providerId=` selects another; unknown/invalid type → 404/400 (`ErrNotFound`/`ErrInvalid`).
- Mock/opt: `dailyMetrics` with entries lacking any totals (idle days) → `activeDays` counts only days with any reported activity; charts render gaps.
