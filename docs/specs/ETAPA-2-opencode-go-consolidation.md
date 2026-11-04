# ETAPA 2 — OpenCode Go Consolidation

**Date:** 2026-11-04  
**Status:** Approved for execution (operator attached SDD plan + implement request)  
**DEEPWORK:** yes  
**Related discovery:** [`../discovery/opencode-go-usage-limits-spike-2026-11-04.md`](../discovery/opencode-go-usage-limits-spike-2026-11-04.md)  
**Related decisions:** [`../decisions/provider-reported-quota-only.md`](../decisions/provider-reported-quota-only.md)

## Intent

Make OpenCode Go a first-class quota integration in Connexto AI Hub so the
operator can see official Go rolling/weekly/monthly franchise meters and
Console budgets in one place, without mixing E09.5 per-execution metering.

## Requirements

1. Query official OpenCode cloud usage meters with the configured account
   credential (`oc_sk_`, permission All or Usage: Read).
2. Query Console current budgets with the same credential.
3. Preserve provider-returned percentages, status strings, monetary fields, and
   reset timestamps; never invent remaining balance or percentages.
4. Identify the reading scope (`subscription`) and optional product label from
   go/status when available without payment/PII fields.
5. Keep E09.5 execution usage in its existing surfaces only.
6. Manual refresh only; no periodic polling.
7. Credentials stay backend-only; frontend receives normalized DTO only.
8. Present readings on Overview and Providers administration.

## Constraints

- Do **not** call `GET /api/experimental/usage/limits` against opencode.ai
  (spike: HTTP 404). That route is local HttpApi only.
- Official cloud routes (spike-verified):
  - `GET https://opencode.ai/zen/go/v1/usage`
  - `GET https://opencode.ai/console/api/policies/current-budgets`
  - Optional product metadata: `GET https://opencode.ai/console/api/go/status`
    (strip `subscriberUserId`, `paymentMethodId`, and related PII).
- Host allowlist: existing `opencode.ai`.
- Same credential field as inference; no second OpenCode management field
  required for this stage.
- Out of scope: Intelligence Engine, automatic routing, account balancing,
  fallback, scraping Console sessions, inferring franchise from executions.

## Design boundaries (DEEPWORK)

| Boundary | Ownership | Gate before next |
| --- | --- | --- |
| A Backend | `OpenCodeQuotaAdapter`, QuotaService workspace/subscription branch, wiring | `go test` claim-scoped green |
| B Frontend | Overview + Providers consume snapshots | Vitest fidelity green |
| C Validation | validation doc + AI Memory | live/manual evidence |

Sequence (deepest first): DTO → adapter → QuotaService branch →
`persistence_runtime` register → TS types → Overview row → Providers table.

### Normalization contract

| Provider field | Hub metric | Notes |
| --- | --- | --- |
| `usage.rolling.percent` | `go_rolling.consumed_percent` | period `rolling` |
| `usage.rolling.resetsAt` | `go_rolling.resets_at` | keep ISO string |
| `usage.rolling.status` | `go_rolling.scope` or label suffix | preserve string `ok`/other as `scope` value `status:<value>` only if needed; prefer metric label includes status when not `ok` |
| same for weekly/monthly | `go_weekly`, `go_monthly` | |
| budget `spent`/`limit` | `budget_<id>` used/limit | unit USD; do not compute remaining |
| budget `resetsAt` epoch ms | `resets_at` RFC3339 | convert timestamp only |
| go/status `product` | snapshot `workspace_name` optional OR source metadata | no workspace id available |

Snapshot: `Scope = subscription`, `source.name = opencode_go_usage_api`.

QuotaService: for `TypeOpenCodeGo`, one subscription snapshot via
`FetchQuota` (not OpenRouter dual api_key/account path).

## Acceptance criteria

1. Authenticated official Go usage read is proven (spike + gated live test).
2. Rolling, weekly, and monthly percent and resetsAt appear in Hub DTO/UI.
3. Budgets array mapped when present; empty array yields no fabricated budgets.
4. No arithmetic for remaining franchise; no E09.5 aggregation in quota UI.
5. Manual refresh only (`refetchInterval: false`).
6. Unit, integration, and UI tests cover mapping and isolation.
7. Validation evidence recorded; AI Memory updated with sanitized facts.

## Evidence

| Claim | Success type | Evidence path | Budget |
| --- | --- | --- | --- |
| Adapter maps zen usage + budgets without arithmetic | test | `go test ./internal/providers/ -run OpenCodeQuota -count=1` | happy path, empty budgets, 401, host deny, ISO resets |
| QuotaService returns one OpenCode subscription snapshot | test | `go test ./internal/providers/ -run QuotaService -count=1` | OpenCode + OpenRouter isolation |
| UI renders Go meters when `consumed_percent` present | test | Vitest Overview/Providers quota fidelity | OpenCode fixture; OpenRouter regression |
| Live key can read zen usage when authorized | command | gated live test or spike doc | status 200, three meters |
| Secrets not in API responses | test | gateway quota handler tests | no credential/raw body |
| E09.5 remains separate | manual | Overview quota vs executions metering copy | existing separation preserved |

**Affordance:** Reuse `GET /api/ui/providers/quotas` and `QuotaAdapter`; no new
public route required.
