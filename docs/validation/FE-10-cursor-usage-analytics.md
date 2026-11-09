# FE-10 Validation Report — Cursor Usage Analytics in Provider Drawer

**Date:** 2026-11-08 · **SPEC:** `docs/specs/FE-10-cursor-usage-analytics.md`

## Claim

The existing Overview provider drawer renders Cursor activity from the grok.com console `get-user-analytics` payload (collected through the existing Connexto Proxy cursor collect flow, embedded as `usage.raw_analytics`). No route or sidebar item was added. Drawer navigation uses cache; "Atualizar uso do Cursor" physically re-collects.

## Phase 5 (Red) evidence

- Go: `internal/providers/cursor_usage_analytics_test.go` package build failed — `undefined: CursorUsageAnalyticsService/NewCursorUsageAnalyticsService/CursorUsageAnalyticsEnvelope/UsageAnalyticsStatus*` (9 tests first).
- Frontend: vitest transform failure — 2 test files, "no tests" (feature modules absent).

## Typed Green evidence

- `internal/providers/cursor_usage_analytics.go`: per-provider cache, 2 min TTL (`WithUsageAnalyticsTTL`), serve-cache-immediately + background collection (cold/stale/forced) with single-flight coalescing; cold cache ⇒ `collecting` envelope (no 120 s block). Collection reuses management credential + host allowlist + `POST /providers/cursor/usage/collect`; `raw_analytics` passed through as `json.RawMessage`.
- Gateway: `GET /api/ui/providers/cursor/usage-analytics` (+ `?refresh=true`, `?providerId=`), bound in `cmd/ai-hub/persistence_runtime.go`, route appended in `provider_admin.go`; `/api/ui` operator-session policy applies unchanged.
- Frontend feature `features/cursor/`: typed payload mirror, pure aggregations (`summarizeCursorDailyMetrics`, `rankCursorModelUsage`, `buildCursorDailySeries` — absent fields are gaps, never zero), compact drawer summary metrics, recent daily chart, model ranking, and a physical refresh action with collecting/error states.

## Verification (HR-13)

- `go test ./...` — **1076 passed / 39 packages** (incl. 11 service + 3 route tests new).
- `go vet ./...`, `go build ./...` — clean.
- `npm run check` (`tsc --noEmit && vitest run`) — **65 files / 207 tests passed** (9 relevant tests: 7 aggregation tests + 2 drawer integration tests).
- `npm run build` — pass; Vite emitted the existing large-chunk advisory.

## Scope check (HR-19)

Touched: `features/cursor/**`, existing `ProviderQuotaDetailDrawer.tsx`, its drawer integration test, `styles.css` (`cursor-drawer-usage-*` rules), `internal/providers/cursor_usage_analytics*.go`, `internal/gateway/providers_admin_handler.go`, `cmd/ai-hub/provider_admin.go` and `persistence_runtime.go`. No new route, sidebar item, or page metadata; no quota arithmetic or proxy changes.

## Acceptance criteria

AC-1..AC-9 satisfied (see SPEC §5): existing drawer preserved; provider-reported values only; forced refresh loop (1.5 s poll / 120 s cap); TTL + force semantics unit-tested; drawer integration and aggregation fixtures verified; all gates green; no extra navigation surface.

## Scope correction

The project lead clarified that the analytics belongs inside the Cursor drawer shown from Overview, not on a dedicated screen. The `/cursor` route, page components, sidebar navigation item, and page metadata were removed; the drawer now owns the compact analytics presentation. A failed forced collection retains the previous successful payload and surfaces a warning. The independent review approval preceded this UI-scope correction, so re-review is required before close.

Heatmap/streaks, extension & client-version tables, PostgreSQL history retention remain deferred.
