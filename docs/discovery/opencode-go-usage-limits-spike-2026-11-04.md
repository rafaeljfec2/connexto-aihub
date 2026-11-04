# OpenCode Go Usage Limits Spike

**Date:** 2026-11-04  
**Status:** completed (read-only; go for Hub integration with proven cloud routes)  
**Related plan:** ETAPA 2 — OpenCode Go Consolidation

## Method

- Reviewed local OpenCode HttpApi docs for `GET /api/experimental/usage/limits`.
- Audited Hub runtime: inference base `https://opencode.ai/inference/go`, bearer `oc_sk_` ([`internal/providers/client.go`](../../internal/providers/client.go)).
- Executed authenticated read-only probes against OpenCode cloud hosts with the
  existing Console API key from the operator environment.
- No credential, token fragment, or raw PII was written to the repository.

### Evidence classification

| Label | Meaning |
| --- | --- |
| **Documented** | Stated in official OpenCode documentation |
| **Verified** | Reproduced with a real request/response pair |
| **Unavailable** | Documented elsewhere but not reachable on cloud with API key |

## Findings

### 1. Experimental HttpApi route is not the cloud quota API — Verified

| URL | HTTP | Result |
| --- | --- | --- |
| `https://opencode.ai/api/experimental/usage/limits` | 404 | HTML app shell |
| `https://dev.opencode.ai/api/experimental/usage/limits` | 404 | HTML app shell |
| `https://opencode.ai/console/api/experimental/usage/limits` | 404 | empty |

The experimental route belongs to a **local OpenCode server** HttpApi (location /
signed-in Console session). It is **not** an authenticated cloud Console route
for Hub backends.

### 2. Official cloud Go meters with the same inference key — Verified

| URL | HTTP | Auth |
| --- | --- | --- |
| `https://opencode.ai/zen/go/v1/usage` | 200 | `Authorization: Bearer oc_sk_…` |
| `https://opencode.ai/inference/go/v1/usage` | 200 | same body as zen |

Sanitized response shape (live values replaced with fixture values in
[`internal/providers/testdata/opencode_go_usage_sanitized.json`](../../internal/providers/testdata/opencode_go_usage_sanitized.json)):

```json
{
  "usage": {
    "rolling": { "status": "ok", "percent": 12, "resetsAt": "2026-11-04T18:31:46.000Z" },
    "weekly":  { "status": "ok", "percent": 36, "resetsAt": "2026-11-09T00:00:00.000Z" },
    "monthly": { "status": "ok", "percent": 24, "resetsAt": "2026-12-01T19:00:43.000Z" }
  }
}
```

Notes versus the experimental `UsageLimits.Meter` schema:

- Cloud meters expose `status` + `percent` + ISO `resetsAt`.
- Cloud meters do **not** expose a boolean `limited` field.
- No arithmetic was applied; percentages and reset timestamps are provider-returned.

### 3. Console budgets with the same key — Verified

| URL | HTTP | Body |
| --- | --- | --- |
| `https://opencode.ai/console/api/policies/current-budgets` | 200 | `[]` on the probed workspace |

Empty array is a valid reading (no Console budgets configured). A sanitized
non-empty budget fixture is kept for adapter tests:
[`opencode_current_budgets_sanitized.json`](../../internal/providers/testdata/opencode_current_budgets_sanitized.json).

### 4. Subscription / access metadata — Verified (restricted use)

`GET https://opencode.ai/console/api/go/status` returns product and access meter
windows. The live payload also includes subscriber and payment identifiers.
**Hub must not persist or forward payment/PII fields.** A sanitized product-only
fixture is
[`opencode_go_status_sanitized.json`](../../internal/providers/testdata/opencode_go_status_sanitized.json).

### 5. Workspace id — Unavailable via key-auth cloud probes

Probes to `/console/api/workspace`, `/workspaces`, `/me`, `/account`, `/org`
returned 404. The Console docs state each API key is bound to one workspace at
creation time, but no key-authenticated cloud endpoint returned `workspace.id`
during this spike.

Hub implication: report scope as `subscription` (Go plan meters) and surface
`product` from go/status when present; do not invent a workspace id.

### 6. Credential model — Verified

The same stored inference `oc_sk_` (operator permission **All**) authorized:

- Go usage meters (`/zen/go/v1/usage`)
- Current budgets (`/console/api/policies/current-budgets`)
- Go status (`/console/api/go/status`)

Inference base URL (`…/inference/go`) remains for Responses/models only. Quota
reads use `https://opencode.ai` host paths above. No session cookie scraping.

## Go / no-go

| Decision | Rationale |
| --- | --- |
| **GO** for Hub quota adapter | Official cloud endpoints authenticate with the existing API key and return Go rolling/weekly/monthly percentages and reset times. |
| **DO NOT** call `/api/experimental/usage/limits` from Hub cloud integration | 404 on opencode.ai; local-server semantics only. |
| **DO** call `/zen/go/v1/usage` + `/console/api/policies/current-budgets` | Proven, bearer-auth, read-only. |
| **Optional** `/console/api/go/status` | Product label only; strip payment/PII. |
| **DO NOT** mix E09.5 execution usage into franchise panels | Unchanged product decision. |

## Sanitized fixtures committed

- `internal/providers/testdata/opencode_go_usage_sanitized.json`
- `internal/providers/testdata/opencode_current_budgets_sanitized.json`
- `internal/providers/testdata/opencode_go_status_sanitized.json`

## Sources

- OpenCode HttpApi (experimental usage limits schema): https://dev.opencode.ai/v2/docs/api
- OpenCode Console API keys: https://opencode.ai/v2/docs/console/api
- Live authenticated probes executed 2026-11-04 (operator environment)
