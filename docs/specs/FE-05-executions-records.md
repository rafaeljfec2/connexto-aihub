# SPEC FE-05 — Content-free execution records (Setor 7 / FE-05)

**Status:** Approved — project lead authorized FE-05 delivery brief (2026-11-03)

**Date:** 2026-11-03

**Scope:** MVP roadmap Setor 7 part 1 — `features/executions` read-only audit trail

**Superseded UI:** The live `/executions` route implements [Generations Experience Refinement](./FE-EXECUTIONS-GENERATIONS-REFINEMENT.md) (generation projection, overlay drawer, `/v1/generations`). Section 3.2 below describes the original FE-05 master/detail layout and remains historical context for E06-only behavior.

**Design reference:** Doop canvas `Qactw8snfC`, frame `PgFh4Y5Uka`

## 1. Intent

Give operators a content-free view of gateway execution records: paginated list,
single-record detail, API-backed filters, manual refresh, zero polling.

## 2. Backend contracts (verified)

| Route | Auth (dev BFF) | Notes |
| --- | --- | --- |
| `GET /v1/records/executions` | Gateway key via Vite `/v1` proxy | `Cache-Control: no-store` |
| `GET /v1/records/executions/{execution_id}` | Same | 32-hex `execution_id` |

**List query (only these):** `limit` (1–200), `provider_id`, `model_id`, `outcome`,
`started_after`, `started_before`, `cursor` (RFC3339 bounds).

**Outcomes:** `completed`, `failed`, `canceled`, `timed_out`.

**List body:** `records[]`, `count`, `next_cursor`, `degraded`.

**Record body (exact fields):** `schema_version`, `execution_id`, `request_id`,
`provider_id`, `model_id`, `started_at`, `completed_at`, `duration_ns`, `outcome`,
`phase`, `cause`, `upstream_status`, `response_committed`, `usage` (always `null`).

**Errors:** 400 `invalid_request`, 404 `not_found`, 503 `records_unavailable`.

Sources: `internal/gateway/records_handler.go`, `records_list_handler_test.go`,
`internal/records/record.go`.

## 3. Frontend behavior

### 3.1 Non-goals

- No prompt/response bodies or invented KPIs (Doop sample metrics are illustrative only).
- No polling (`refetchInterval: false`).
- No FE-06 resources surfaces.

### 3.2 UI composition (Doop-aligned)

- AppShell + page header eyebrow `OBSERVABILITY / EXECUTIONS`.
- Toolbar: client text filter (loaded rows only), outcome chips, provider select
  (from `/api/providers` ids), model id input, Apply model filter + Refresh.
- Outcome and provider filters apply to the API immediately; model id applies on
  button click or Enter.
- Main grid: table list + fixed detail panel (~285px).
- Row selection loads detail query; highlight selected row.
- Cursor pagination: “Load more” when `next_cursor` present.

### 3.3 Detail panel

Display only schema fields. `usage` shown as “Not reported (null)”. Optional compact
JSON block with the same fields (no extra keys).

## 4. Acceptance criteria

- **AC-FE05-1:** List and detail use live `/v1/records/executions*` (smoke with backend).
- **AC-FE05-2:** No rendering of content outside the record schema.
- **AC-FE05-3:** API filters map 1:1 to supported query params; invalid combos surface API errors.
- **AC-FE05-4:** Loading, empty, error, and degraded states are explicit.
- **AC-FE05-5:** Manual refresh only; navigation + explicit actions refetch.
- **AC-FE05-6:** Route `/executions` registered; sidebar link enabled.

## 5. Verification

- `pnpm --filter @connexto/frontend check`
- `pnpm --filter @connexto/frontend build`
- `go test ./... -count=1`
- Backend smoke: list + detail HTTP 200 when records store available

## 6. Out of scope / future

- UI copy remains English for MVP; **pt-BR + en i18n** is tracked as post-MVP
  (`MVP-ROADMAP.md` §3.3, gap **G9**).
- Date/time filters `started_after` / `started_before` in toolbar (API supports;
  optional follow-up).
