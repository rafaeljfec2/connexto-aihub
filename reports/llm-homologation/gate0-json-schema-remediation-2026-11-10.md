# LLM Homologation — Gate 0 / JSON Schema Remediation

- Date: 2026-11-10
- Execution: Gate 0 Structured Outputs (`json_schema`, `strict: true`) Remediation
- Target Models:
  1. `deepseek-v4.1-flash`
  2. `gpt-6-luna`
  3. `composer-2.5`
  4. `antigravity:gemini-3.8-flash`
- Constraints Honored: No `POST /v1/catalog/sync`, no code changes, no configuration changes, no `.env` changes, no commit/push, zero secrets exposed.

---

## 1. Preflight Verification

| Check | Expected | Observed | Status |
|---|---|---|---|
| Git working tree & branch | `main`, clean | Clean, `5eaad85b019939fa85b0b61bd4682330553b726e` | PASS |
| Gateway Listener | `127.0.0.1:8080` | `127.0.0.1:8080` (PID 97405, `ai-hub`) | PASS |
| Gateway `/healthz` | HTTP 200 `status: up` | HTTP 200 `status: up` | PASS |
| Gateway `/readyz` | HTTP 200 `database: ok` | HTTP 200 `database: ok` | PASS |
| Authentication | Bearer API Key valid | Validated without exposing secret | PASS |
| Catalog Model Mapping | All 4 targets mapped | All 4 mapped to respective providers | PASS |

### Confirmed Provider Mappings

- `deepseek-v4.1-flash` -> `opencode-go`
- `gpt-6-luna` -> `opencode-go`
- `composer-2.5` -> `cursor`
- `antigravity:gemini-3.8-flash` -> `antigravity`

---

## 2. Controlled Execution Evidence

Exactly one request per target was sent sequentially to `POST /v1/responses` with identical strict contract:
- `text.format.type`: `json_schema`
- `strict`: `true`
- Schema: strict object with `name`, `language`, `type` required and `additionalProperties: false`
- Non-streaming (`stream: false`), `max_output_tokens: 256`
- Identical synthetic prompt: `"Gere exclusivamente um JSON válido com os campos name, language e type. Use os valores Connexto AI-Hub, Go e API."`
- No retries, no automatic routing, no fallback

### Results Matrix

| Target Model | Expected Provider | HTTP Code | Gateway Response Type | Duration | Result Classification | Note |
|---|---|---|---|---|---|---|
| `deepseek-v4.1-flash` | `opencode-go` | 503 | `routing_unavailable` | 0.012s | **BLOCKED** | Operational block: catalog stale |
| `gpt-6-luna` | `opencode-go` | 503 | `routing_unavailable` | 0.005s | **BLOCKED** | Operational block: catalog stale |
| `composer-2.5` | `cursor` | 503 | `routing_unavailable` | 0.006s | **BLOCKED** | Operational block: catalog stale |
| `antigravity:gemini-3.8-flash` | `antigravity` | 503 | `routing_unavailable` | 0.005s | **BLOCKED** | Operational block: catalog stale |

---

## 3. Root Cause Analysis — Operational Blockade

In accordance with Section 1 (`Se o Gateway não estiver operacional, interromper e reportar o bloqueio`) and Section 6 (`Não executar POST /v1/catalog/sync. Essa operação exige autorização separada`):

1. **Catalog Stale Status:**
   Inspection of `GET /v1/catalog/providers` revealed:
   - `last_success_at`: `2026-11-09T00:14:14-03:00` for all providers.
   - Elapsed time: > 44 hours.
   - Configured `staleAfter` threshold (`internal/gateway/responses_routing.go:68`): 24 hours.
   - Provider sync state evaluated as `freshness: stale`.

2. **Deterministic Routing Gate Rejection:**
   - In `internal/routing/eligibility.go:65-68`, candidates are evaluated:
     `if view.Freshness != catalog.FreshnessFresh { reasons = append(reasons, ReasonCatalogStale); class = ExclusionInsufficient }`
   - All models were excluded with `ReasonCatalogStale`.
   - `eligibility.Eligible` was empty; `Coverage.CompleteFreshCoverageForEligible` was `false`.
   - In `internal/routing/selector.go:152` and `internal/gateway/responses_routing.go:198`, the deterministic selector returned `RejectRoutingUnavailable`:
     `{"error":{"type":"routing_unavailable","message":"routing evidence is insufficient to select a model"}}`.

3. **Pre-Execution Boundary:**
   - Because the rejection occurred in the admission routing phase, requests never reached upstream providers (`reached_execution = false`).
   - Zero execution records were created in `/v1/records/executions`.

---

## 4. Execution Records Audit

- Endpoint audited: `GET /v1/records/executions?limit=20`.
- Baseline record prior to test: `1717586f200104ed6fa158cb26000961` (`2026-11-09T20:18:34.430Z`).
- Records created during test window (`>= 2026-11-10T23:57:07Z`): **0 records**.
- Provider switches / fallbacks: **0**.
- Retries: **0**.
- Reported Usage: `null` (no execution occurred).

---

## 5. Summary & Recommendation

1. **Gate 0 Status:**
   - No inconclusive result has been converted into an approval.
   - The four conditional models remain unapproved for `json_schema` strict outputs due to the operational block (`BLOCKED`).
   - Models approved previously (`anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5`) maintain their status from 2026-11-09.

2. **Next Steps Awaiting Explicit Authorization:**
   - Authorization required to execute `POST /v1/catalog/sync` to refresh provider catalogs.
   - Immediately following catalog synchronization, re-run the 4-model structured output battery.

