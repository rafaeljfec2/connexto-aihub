# ETAPA 2 — OpenCode Go Quota Validation

**Date:** 2026-11-04  
**SPEC:** [`../specs/ETAPA-2-opencode-go-consolidation.md`](../specs/ETAPA-2-opencode-go-consolidation.md)  
**Spike:** [`../discovery/opencode-go-usage-limits-spike-2026-11-04.md`](../discovery/opencode-go-usage-limits-spike-2026-11-04.md)

## Claims verified

| Claim | Result | Evidence |
| --- | --- | --- |
| Official Go meters authenticate with stored `oc_sk_` | PASS | Live `GET /zen/go/v1/usage` HTTP 200; gated `TestLiveOpenCodeGoQuotaSmoke` |
| Rolling/weekly/monthly percent + resetsAt preserved | PASS | Adapter tests + live smoke; fixture `opencode_go_usage_sanitized.json` |
| Budgets mapped when present; empty array OK | PASS | Fixture budgets + empty live `[]` |
| Experimental `/api/experimental/usage/limits` not used on cloud | PASS | Spike 404; adapter uses zen + console budgets |
| QuotaService single subscription snapshot for OpenCode | PASS | `TestQuotaService_OpenCodeReturnsSingleSubscriptionSnapshot` |
| Overview renders Assinatura meters with provider % | PASS | Vitest fidelity test |
| Providers page manual refresh, no polling | PASS | `refetchInterval: false` on quotas query |
| E09.5 not mixed into franchise panel | PASS | Providers footer copy + separate Executions metering |
| Secrets not in DTO | PASS | Normalized snapshot fields only; no raw upstream body |

## Commands

```bash
go test ./internal/providers/ -run 'OpenCodeQuota|QuotaService_OpenCode|LiveOpenCodeGoQuota' -count=1
AIHUB_LIVE_SMOKE=1 AIHUB_PROVIDER_OPENCODEGO_API_KEY=… go test ./internal/providers/ -run LiveOpenCodeGoQuotaSmoke -count=1
go test ./internal/providers/... ./internal/gateway/... -count=1
pnpm exec vitest run src/features/overview/components/OverviewProvidersPanel.quota-fidelity.test.tsx src/features/providers
```

Observed: providers/gateway packages **197** tests passed; live OpenCode quota smoke **passed**; Vitest overview/providers quota suites **13** passed.

## Sanitized live reading (operator key)

- Go rolling / weekly / monthly percentages returned as provider numbers (example fixture: 12 / 36 / 24).
- Product label from go/status: `go-plus` (no payment/PII stored).
- Console budgets: empty array for the probed account.

## Residual limits

- Cloud API key auth does not expose `workspace.id`; Hub shows subscription scope + product label.
- Browser E2E against a running Hub was not executed in this session; UI contract covered by Vitest.
- PR not opened (LIM-03 / operator did not request publish).
