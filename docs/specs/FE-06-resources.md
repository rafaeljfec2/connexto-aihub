# SPEC FE-06 — Resources operational panel (Setor 7 / FE-06)

**Status:** Approved — project lead authorized FE-06 delivery brief (2026-11-03)

**Date:** 2026-11-03

**Scope:** MVP roadmap Setor 7 part 2 — `features/resources` read-only governance visibility

**Design reference:** Doop canvas `Qactw8snfC`, frame `Q9sTNaOnH5`, style guide `ai-hub-visual-system`

**Related:** `docs/frontend/MVP-ROADMAP.md` §4 and Setor 7; `decisions/frontend-feature-first-architecture-2026-11-03.md`; `docs/specs/E09-resource-management.md`; `docs/specs/E09.4-usage-analytics.md`; E09.5 metering contracts; E09 consolidated audit (null vs zero semantics)

## 1. Intent

Expose E09 resource admission, concurrency, and provider-reported usage as an operator panel. The UI must show configured policies, current utilization, near-limit signals, concurrency occupancy, and provider usage aggregates with explicit data provenance and refresh timestamps. This is not billing, not generic analytics, and not invented KPIs from Doop sample numbers.

## 2. Backend contracts (verified)

All admin routes use **Admin API key** via Vite `/api` proxy (`Authorization: Bearer ${AIHUB_ADMIN_API_KEY}`). Responses set `Cache-Control: no-store`.

### 2.1 Resource policies (Source A — configuration)

| Route | Method | Response |
| --- | --- | --- |
| `/api/resource-policies` | GET | `{ policies: ResourcePolicy[] }` |

**ResourcePolicy:** `id`, `scope` (`global` \| `provider` \| `model`), optional `provider_id`, optional `model_id`, `daily_limit`, `enabled`, `updated_at` (RFC3339).

Source: `internal/gateway/resource_admin_handler.go`.

### 2.2 Quota utilization snapshot (Source A — utilization)

| Route | Method | Query | Response |
| --- | --- | --- | --- |
| `/api/resource-analytics/quota-utilization` | GET | optional `provider_id`, `model_id`, `period_start` (RFC3339) | `{ quota_utilization: QuotaUtilization[] }` |

**QuotaUtilization:** `policy_id`, `scope`, optional `provider_id`, `model_id`, `daily_limit`, `enabled`, `period_start`, `reset_at`, `admitted_count`, `remaining_requests`, `utilization_percent`.

When `enabled` is false or `daily_limit <= 0`, `utilization_percent` is `0` in JSON but must **not** be shown as “0% utilized” — display as not applicable.

Source: `internal/gateway/analytics_admin_handler.go`.

### 2.3 Admissions series (Source A — optional overview chart)

| Route | Method | Query | Response |
| --- | --- | --- | --- |
| `/api/resource-analytics/admissions` | GET | required `from`, `to` (RFC3339 or YYYY-MM-DD); optional `provider_id`, `model_id` | `{ from, to, total_admitted, daily: [{ period_start, admitted_count }] }` |

Gap-filled UTC daily series (allowed admissions only).

### 2.4 Concurrency (Source B)

| Route | Method | Response |
| --- | --- | --- |
| `/api/concurrency-policies` | GET | `{ policies: ConcurrencyPolicy[] }` |
| `/api/resource-analytics/concurrency` | GET | optional `provider_id`, `model_id` → `{ concurrency: ConcurrencySnapshot[] }` |

**ConcurrencyPolicy:** `id`, `scope`, optional `provider_id`, `model_id`, `concurrency_limit`, `enabled`, `updated_at`.

**ConcurrencySnapshot:** `policy_id`, `scope`, optional ids, `concurrency_limit`, `enabled`, `active_count`, `available`.

Alternative live meter: `GET /api/concurrency-usage` (same snapshot fields). FE-06 uses analytics concurrency endpoint for consistency with quota analytics.

Sources: `internal/gateway/concurrency_admin_handler.go`, `analytics_admin_handler.go`.

### 2.5 Provider usage (Source C — separate from admission)

| Route | Method | Query | Response |
| --- | --- | --- | --- |
| `/api/resource-analytics/provider-usage` | GET | required `from`, `to`; optional `provider_id`, `model_id` | See §2.6 |
| `/api/resource-analytics/provider-usage/distribution` | GET | same | `{ from, to, by_provider[], by_model[] }` |

Token fields are `number` or JSON `null` (unknown). Explicit zero remains zero. Execution status buckets: `reported_executions`, `partial_executions`, `unavailable_executions`, `invalid_executions` (counts, not token sums).

**Not in current contract:** `cache_tokens` (no backend field at this HEAD).

Sources: `internal/gateway/analytics_provider_usage_handler.go`, `internal/gateway/analytics_admin_handler_test.go` (null vs zero).

### 2.6 Provider usage summary body

```json
{
  "from": "RFC3339",
  "to": "RFC3339",
  "input_tokens": null | number,
  "output_tokens": null | number,
  "total_tokens": null | number,
  "reported_executions": number,
  "partial_executions": number,
  "unavailable_executions": number,
  "invalid_executions": number,
  "daily": [
    {
      "period_start": "RFC3339",
      "input_tokens": null | number,
      "output_tokens": null | number,
      "total_tokens": null | number,
      "execution_count": number
    }
  ]
}
```

Distribution rows: `period_start`, `provider_id`, optional `model_id`, token fields nullable, `executions` count.

### 2.7 Errors

400 `invalid_request`, 401/403 admin auth, 500 `internal_error` (analytics admin generic message). UI surfaces `ApiError.message`.

## 3. Data integrity rules (E09 audit)

| Situation | UI |
| --- | --- |
| JSON `null` token field | Label **Unknown** (not zero) |
| Explicit `0` | **0** |
| `unavailable_executions` | Show count; do not treat as zero tokens |
| `invalid_executions` | Show count; exclude from token narrative |
| `partial_executions` | Show count; tokens may be partial at row level |
| Disabled policy or zero daily limit | Utilization **N/A**, not 0% |
| No policies / empty arrays | Empty state, not fabricated metrics |

Do not change backend contracts in FE-06.

## 4. Frontend behavior

### 4.1 Non-goals

- No billing or cost estimation.
- No polling (`refetchInterval: false`).
- No policy PUT/enable/disable in this slice (read-only panel; editors deferred).
- Do not use Doop illustrative metrics as live data.

### 4.2 Query keys (TanStack Query)

- `['resources', 'resourcePolicies']`
- `['resources', 'quotaUtilization', filter]`
- `['resources', 'concurrencyPolicies']`
- `['resources', 'concurrencySnapshot', filter]`
- `['resources', 'admissions', range, filter]`
- `['resources', 'providerUsage', range, filter]`
- `['resources', 'providerUsageDistribution', range, filter]`

Manual **Refresh** refetches all active queries. Optional provider filter applies to analytics queries only.

Default analytics range: last 7 UTC days (`YYYY-MM-DD`).

### 4.3 UI composition (Doop-aligned)

- Route `/resources`, sidebar enabled.
- Page header eyebrow `GOVERNANCE / RESOURCES`, title **Policies & concurrency**, lead explaining cross-gateway limits.
- Summary metric row derived from **live** policy/utilization/concurrency data (counts, peak utilization, concurrency occupancy, near-limit alerts).
- Tabs: **Overview**, **Resource policies**, **Concurrency**, **Provider usage** (fourth tab adds E09.5 surface absent from Doop mock).
- Overview: admissions daily bars (real series), quota highlight table, concurrency summary.
- Resource policies: merged policy list + quota utilization columns.
- Concurrency: snapshot cards + policies table.
- Provider usage: range controls, token summary with Unknown semantics, execution measurement status row, distribution tables, source footnote (`/api/resource-analytics/provider-usage*` + fetched-at).
- Provenance footers on sections: endpoint path + `dataUpdatedAt` from query cache.

### 4.4 Filters

- Date range: `from` / `to` date inputs (Apply).
- Optional `provider_id` select (from `/api/providers` ids) for analytics endpoints.

## 5. Acceptance criteria

- **AC-FE06-1:** Resource and concurrency policies plus quota/concurrency snapshots load from live admin APIs (smoke on `:8080` when PostgreSQL mode available).
- **AC-FE06-2:** Provider usage summary and distribution load from `/api/resource-analytics/provider-usage*` with required range.
- **AC-FE06-3:** Null token metrics render as Unknown; explicit zero as zero; unavailable/partial/invalid execution counts displayed separately.
- **AC-FE06-4:** Disabled or unlimited quota policies do not show misleading 0% utilization.
- **AC-FE06-5:** Provider usage visually and logically separated from admission/quota metrics.
- **AC-FE06-6:** Manual refresh only; no polling timers.
- **AC-FE06-7:** Route `/resources` registered; `features/resources` feature-first layout; no cross-feature internal imports.
- **AC-FE06-8:** Loading, empty, error, and degraded (503/unavailable admin) states are explicit.

## 6. Test matrix

| Area | Tests |
| --- | --- |
| Token formatting | null → Unknown, 0 → 0, positive formatting |
| Quota utilization display | disabled / zero limit → N/A; enabled → percent |
| Near-limit detection | threshold ≥ 90% |
| API path builder | query params for analytics range and filters |
| Provider usage status labels | reported/partial/unavailable/invalid copy |
| Components | table headers, a11y roles, empty/error banners |
| HTTP errors | ApiError message propagation helper |

Integration smoke (manual): `GET /api/resource-policies`, `GET /api/resource-analytics/quota-utilization`, `GET /api/resource-analytics/concurrency`, `GET /api/resource-analytics/provider-usage?from=&to=` → 200 when admin + DB available.

## 7. Verification commands

- `pnpm --filter @connexto/frontend check`
- `pnpm --filter @connexto/frontend build`
- `go test ./... -count=1`
- `git diff --check`

## 8. Out of scope

- Policy mutation UI, billing, E10/E11, backend changes, commits without lead authorization.
- `cache_tokens` until backend exposes them.
