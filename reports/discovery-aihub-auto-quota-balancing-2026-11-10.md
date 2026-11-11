# Discovery — aihub/auto: deterministic selection and adaptive quota balancing

**Date:** 2026-11-10
**Status:** discovery only. No implementation is authorized by this document.
**Repositories inspected (read-only):** `connexto-aihub`, `connexto-proxy-cli-llms`
**Language:** Brazilian Portuguese, per the reports language policy.

This document separates **proven facts**, **hypotheses**, and **recommendations**. It does not change runtime behavior.

---

## A. Current diagnosis

### Proven

`aihub/auto` is a virtual model id (`internal/routing/virtual.go`). The gateway lists it on `GET /v1/models` and intercepts it in `SelectDeterministic` (`internal/routing/selector.go`). Resolution is `selectVirtualAuto` (`internal/routing/virtual_selector.go`):

1. Zero eligible candidates → existing rejection (`routing_unavailable` or `no_eligible_model`).
2. One eligible candidate → that candidate, criteria `rule:virtual_auto_unique_candidate` and `profile:balanced`.
3. Several eligible candidates → first hit in a **hard-coded administrative list** of four targets:
   - `opencode-go` / `gpt-6-luna`
   - `openrouter` / `anthropic/claude-haiku-4.5`
   - `opencode-go` / `glm-5.3-flash`
   - `opencode-go` / `claude-haiku-5-5`
4. If none of those four is eligible → first candidate in eligibility order (`rule:virtual_auto_snapshot_fallback`).

The live request path is `routeResponses` (`internal/gateway/responses_routing.go`). It calls `ValidateStructure`, `ProjectRequirements`, `CaptureSnapshot`, `EvaluateEligibility`, and `SelectDeterministic`. It does **not** call `UnderstandRequest`. The parsed `optimization_profile` is therefore unused on the hot path. Every virtual decision stamps `profile:balanced` regardless of the JSON field.

`EvaluateEligibility` filters provider binding, credentials, Responses support, catalog presence, catalog freshness, required capabilities, and `max_output_tokens` versus a known output limit. It does **not** read quota snapshots, homologation status, context window, latency, or cost. A search of `internal/routing` finds no quota symbol.

`AUTO-SPEC.md` (approved 2026-11-09) describes a different system: Gate 0 over all 10 models, quota pruning, context-window check, weighted score `fit + latency + quota`, and four transparency headers. That selector is **not** what the code executes. E08.6–E08.8 remain unauthorized as an intelligent ranker (`docs/decisions/intelligent-routing-status-2026-11-09.md`). The implemented slice is administrative priority on top of E08 eligibility.

The E08 contract (`docs/specs/E08-routing-contract.md` §1) forbids reading message text to infer task type, quality, or intent. Structural fields only.

Execution records schema v1 **require** `usage` to be JSON `null` (`internal/records/codec.go`). Per-call usage that providers return lives in the response body and, when captured, in the separate metering normalizer (`internal/metering/normalize.go`). It is not available to the router.

`QuotaService` caches official readings with a default stale TTL of **2 minutes** (`internal/providers/quota_service.go`). The router does not consume that cache.

### Limitations that follow from the facts

- Four OpenCode Go models share one administrative ladder, so `gpt-6-luna` absorbs Auto traffic whenever it is eligible.
- `deepseek-v4.1-flash`, `composer-2.5`, and all four Antigravity models are absent from the priority list. They are chosen only as a unique eligible candidate or as snapshot fallback.
- There is no allowlist. Any fresh, capability-compatible catalog model can win the fallback.
- Unknown capability excludes the candidate (`ReasonCapabilityUnknown`). Declared-but-unverified catalog claims still pass, because verification state is not an eligibility gate.
- Production Auto remains blocked by the Gate 0 reports of 2026-11-10 until the three Antigravity models and the execution-record telemetry gap are closed. That block is a report recommendation, not a runtime flag found in `selectVirtualAuto`.

---

## B. Matrix of the ten models

### Canonical identifiers

The user table uses dotted Antigravity ids (`claude-opus-4.6-thinking`, `claude-sonnet-4.6`). The catalog ids proven in the 2026-11-09 preflight (`reports/llm-smoke/preflight/summary.md`) use hyphens:

| Provider (runtime id) | Canonical model id | Integration |
|---|---|---|
| `openrouter` | `anthropic/claude-haiku-4.5` | Hub OpenRouter adapter, direct HTTPS |
| `opencode-go` | `glm-5.3-flash` | Hub OpenCode Go adapter, direct HTTPS |
| `opencode-go` | `deepseek-v4.1-flash` | Hub OpenCode Go adapter |
| `opencode-go` | `claude-haiku-5-5` | Hub OpenCode Go adapter |
| `opencode-go` | `gpt-6-luna` | Hub OpenCode Go adapter |
| `cursor` | `composer-2.5` | Hub → Connexto Proxy → Cursor provider |
| `antigravity` | `antigravity:claude-opus-4-6-thinking` | Hub → Proxy → `agy` ACP runtime |
| `antigravity` | `antigravity:claude-sonnet-4-6` | Hub → Proxy → `agy` |
| `antigravity` | `antigravity:gemini-3.8-flash` | Hub → Proxy → `agy` |
| `antigravity` | `antigravity:gpt-oss-120b-medium` | Hub → Proxy → `agy` |

Dotted ids are **not** the runtime identifiers. An allowlist must use the hyphenated catalog ids.

### Limits proven at preflight (2026-11-09)

Context / max output, catalog freshness `fresh`, listing state `stale`, no catalog sync executed:

| Model | Context | Max output |
|---|---:|---:|
| `anthropic/claude-haiku-4.5` | 200000 | 64000 |
| `glm-5.3-flash` | 1000000 | 131072 |
| `deepseek-v4.1-flash` | 1000000 | 384000 |
| `claude-haiku-5-5` | 1000000 | 128000 |
| `gpt-6-luna` | 1050000 | 128000 |
| `composer-2.5` | 200000 | 16384 |
| `antigravity:claude-opus-4-6-thinking` | 200000 | 64000 |
| `antigravity:claude-sonnet-4-6` | 200000 | 64000 |
| `antigravity:gemini-3.8-flash` | 1048576 | 65536 |
| `antigravity:gpt-oss-120b-medium` | 128000 | 16384 |

These numbers are a point-in-time catalog reading. The raw `catalog-models.json` cited by the preflight is **not** in the tree now. Per-model capability claims (tools, image, reasoning, structured output) were not recovered from that file. The normalizer (`internal/catalog/capability.go`) sets `streaming` and `custom_tools` to **unknown** always. Missing modality lists stay **unknown**, not unsupported. Unknown required capabilities exclude the model.

### What was actually tested

Three layers must stay distinct.

**1. Manual smoke (2026-11-09), five short calls, no tools, no images, one arithmetic reasoning sample.** Source: `reports/llm-homologation-remediation.md` §6–§7. Token totals below are reconstructed from lost response bodies, not from execution records (`usage: null` by schema).

| Model | Smoke class on 2026-11-09 | Latency range | Notes |
|---|---|---|---|
| `anthropic/claude-haiku-4.5` | HOMOLOGATED for the battery; JSON clean only via `json_schema` | 0.87–1.72 s | OpenRouter monetary cost reported on the body |
| `glm-5.3-flash` | HOMOLOGATED | 1.12–2.33 s | `go-plus` allowance on the body, not a dollar cost |
| `deepseek-v4.1-flash` | CONDITIONAL (schema not tested then) | 0.93–3.12 s | Same allowance family |
| `claude-haiku-5-5` | HOMOLOGATED | 0.92–2.10 s | Same allowance family |
| `gpt-6-luna` | CONDITIONAL (schema not tested then) | 0.97–1.86 s | Same allowance family |
| `composer-2.5` | CONDITIONAL (latency, schema not tested) | 2.98–11.13 s | Usage present on the body; no monetary cost field |
| `antigravity:gemini-3.8-flash` | CONDITIONAL | 5.55–8.35 s | ~15.3k input tokens per short call from the closed `agy` runtime |
| three other Antigravity models | BLOCKED | n/a | HTTP 200 notice rewritten to HTTP 429 `upstream_quota_exceeded`, `reset_at` ≈ `2026-11-12T00:46Z` |

**2. Gate 0 `json_schema` strict (2026-11-10), after a Proxy unwrap/validate change.** Source: `reports/llm-homologation/gate0-json-schema-post-fix-2026-11-10.md`.

Passed on the gateway body: `composer-2.5`, `deepseek-v4.1-flash`, `antigravity:gemini-3.8-flash`. Earlier the same day, `gpt-6-luna` and `gemini-3.8-flash` had already passed; `composer-2.5` had failed by emitting Markdown fences; `deepseek-v4.1-flash` had returned HTTP 429 overload. The post-fix pass for Composer is **Proxy enforcement** (`enforceStructuredOutputFormat` unwraps one fence and rejects invalid JSON with HTTP 502). It is not evidence that the model emits clean JSON by itself. Streaming structured output was not part of that retest (`runOutput.toolCalls.length === 0` and non-streaming path only).

**3. Production release.** Not equivalent to either layer. The post-fix report still says keep `aihub/auto` out of production until the three blocked Antigravity models are retested and the execution-record usage gap is closed. AUTO-SPEC Gate 0 requires all 10 models formally approved. That gate is **not** satisfied.

### Capabilities by evidence class

| Dimension | Proven | Not proven |
|---|---|---|
| Text generation, short instruction following | Smoke PASS on the seven reachable models | The three blocked Antigravity models |
| Streaming of a short text | Smoke PASS on the seven | Blocked three; not a tool or schema stream |
| Reasoning quality | One arithmetic item per reachable model. Not a benchmark | Relative ranking among models |
| Coding quality | Not in the battery | Any ordering by name (`opus`, `sonnet`, `composer`) |
| Tool / function calling | Not exercised. Catalog claim, if present, is unverified | Routing must treat unverified tools as insufficient when tools are required |
| Image / audio | Not exercised. Smoke plan explicitly excluded multimodal | |
| Structured output | Strict `json_schema` PASS for 7 models, with the Composer caveat above | The three blocked models; streaming strict schema; native Composer compliance |
| Context window versus a large prompt | Catalog numbers only | No large-context probe was run |
| `agy` overhead | ~15.3k input tokens per Gemini call, measured | Whether Opus/Sonnet/gpt-oss share that overhead. Hypothesis: the ACP session is shared by the Antigravity provider, so the overhead likely applies to all four. Unproven for the three blocked models |

No quality scores are assigned in this discovery.

---

## C. Quota map

Provider, account, and quota pool are different objects. The Hub can store N configured accounts. Each quota adapter reads one account. The Auto MVP allowlist does not by itself create separate pools.

### OpenCode Go — one subscription pool for four models

**Proven in code and in the 2026-11-04 live validation** (`docs/validation/etapa-2-opencode-go-quota-2026-11-04.md`):

| Question | Class | Evidence |
|---|---|---|
| Query endpoint | Available and proven | `GET https://opencode.ai/zen/go/v1/usage` with the stored bearer key. Console `GET /console/api/policies/current-budgets`, `/console/api/go/status`, `/console/api/usage/summary`, `/console/api/usage/cost-by-day`. The experimental `/api/experimental/usage/limits` route is **not** used (cloud 404 in the spike). |
| Total limit in tokens or requests | Not available on the Go meters | Meters expose `percent`, `status`, `resetsAt` only (`mapOpenCodeGoMeters`). |
| Remaining absolute quota | Not available | Only consumed percent. |
| Utilization percent | Available and proven | `rolling`, `weekly`, `monthly`. |
| Separate per-model quota | Not available | The snapshot is `ScopeSubscription`. `glm-5.3-flash`, `deepseek-v4.1-flash`, `claude-haiku-5-5`, and `gpt-6-luna` draw the same Go meters. |
| Per-request consumption | Available on the inference body, partially | `opencode.allowance` (go-plus units) plus token counts. Metering can normalize it. Execution records drop it. It is not an account snapshot. |
| Reset | Available when the provider sends `resetsAt` | Preserved, not computed by the Hub. |
| HTTP 429 on the quota API | Available | Mapped to `rate_limited`. Distinct from an inference 429. |
| Freshness | Cached | `QuotaService` stale TTL default 2 minutes. A failed refresh can mark the previous snapshot stale. |
| Multiple accounts | Possible in the provider store | One snapshot per configured OpenCode account. The MVP runtime evidence is a single `go-plus` subscription. |

Console budgets may add a USD `limit`/`spent` with its own reset. The probed account on 2026-11-04 returned an empty budget array. Do not assume a budget pool exists.

### Antigravity — account session, group buckets, and per-model fractions

**Proven in Proxy** `packages/providers/antigravity/src/usage/usage-service.ts` and `docs/antigravity-usage.md`. The Hub reads `GET /v1/providers/antigravity/usage` (`internal/providers/antigravity_quota.go`).

| Question | Class | Evidence |
|---|---|---|
| Query endpoint | Available and proven | Proxy usage route, local OAuth under the `agy` home, Cloud Code PA `fetchAvailableModels` plus `retrieveUserQuotaSummary`. |
| Absolute used / limit / remaining | Not available | Fields stay null. |
| Remaining fraction | Available and proven in the contract | `remaining_fraction` 0.0–1.0 per model and per group bucket. Explicit 0 is preserved. The Hub derives consumed/remaining percent **only** from that fraction (`antigravity_quota_test.go`). |
| Per-model pool | Partially available | Each model id is its own metric with `scope: "model"`. |
| Shared pool | Partially available | Summary `groups[].buckets[]` are separate metrics with `scope: "group"`. The code does not document which of the four MVP models belong to which group. |
| Reset | Available when upstream sends `resetTime` | `reset_at`. |
| Cooldown | Proven for the quota **notice**, not as a numeric remaining field | Individual quota text on HTTP 200 is rewritten to HTTP 429 `upstream_quota_exceeded` with `reset_at`. Live probes of Sonnet, and the report's probes of Opus and gpt-oss, shared `reset_at` ≈ `2026-11-12T00:46Z`. |
| Same pool for all four models | Hypothesis | Identical reset timestamps are compatible with one window or with aligned individual windows. The notice says "individual". Do not treat the four models as independent balances, and do not treat them as one balance, until a usage snapshot maps model id → group. |
| Per-request tokens | Partial | Gemini returned usage on the body. Cost was null. ~15.3k input overhead. |
| Real time | Pull on demand, then Hub cache (2 min) | Not pushed. Proxy refresh of the Google token is a side effect of a read; it does not spend inference quota. |
| Accounts | One local Antigravity login per Proxy host | Not an API key in the Hub. Multiple Hub accounts would still point at that host session unless the Proxy grows account routing. Unproven beyond one session. |

### Cursor — one browser-collected account for `composer-2.5`

**Proven in** `internal/providers/cursor_usage_quota.go` and Proxy `POST /v1/providers/cursor/usage/collect`.

| Question | Class | Evidence |
|---|---|---|
| Query | Available, slow, session-based | Headless Chrome against the logged-in Cursor profile. Adapter timeout 120s. Not suitable for a decision on the request path. |
| Weekly included budget | Available when the dashboard payload has the fields | `total_credits` (USD limit), `total_usage` (USD used), `weekly_used_percent`, `weekly_resets_at`. |
| Prepaid credits | Available when present | `prepaid_available_credits` as remaining USD. A memory note records a case of ~100% weekly use with prepaid still remaining. Weekly exhaustion and prepaid availability are different pools. |
| Per-model quota | Not available | Metrics are account-level. The allowlist has one Cursor model, so the pool and the model coincide for the MVP. |
| Percentage invented by the Hub | Partial violation risk | If `weekly_used_percent` is absent and both `total_usage` and `total_credits` exist, the adapter computes `(used/limit)*100`. That is Hub arithmetic. Routing should prefer the provider percent and treat the derived value as a labeled fallback. |
| HTTP 429 causes | Not classified in this collector | Collect returns auth-required or upstream HTTP status. Inference 429 is a different path. |
| Freshness | Stale by design | Background collect, coalesced, cache. A routing decision must not trigger a browser collect. |

### OpenRouter — one API-key money pool for `anthropic/claude-haiku-4.5`

**Proven in** `internal/providers/openrouter_quota_key.go` (`GET {endpoint}/key`).

| Question | Class | Evidence |
|---|---|---|
| Limit, remaining, usage | Available when the key payload includes them | `limit`, `limit_remaining`, `usage`, daily/weekly/monthly usage, optional BYOK usage. Unit USD. |
| Percent | Not derived | The adapter stores the provider numbers and does not compute a percent. |
| Reset | Partial | `limit_reset` is a period string, not always an instant. Budgets may include `resets_at`. |
| Per-model quota | Not available | Key scope. One allowlist model, so the pool and the model coincide unless other non-Auto traffic uses the same key. **Hypothesis:** any other OpenRouter model outside Auto still spends this key. The balancer must count the key, not the Auto model alone. |
| Free-model daily requests | Available only if the key returns `free_model_daily_requests` | Not relevant to Haiku 4.5, which is paid. |
| Account credits | Separate adapter | `FetchAccountQuota` uses a management credential. It is not the inference-key reading. |
| Per-request cost | Proven on smoke bodies | OpenRouter `usage.cost` reached the client body and was dropped by execution records. |
| Real time | Pull + 2 min cache | Same `QuotaService` TTL. |

### Pool map for the MVP allowlist

```text
pool:opencode-go:<account>     rolling + weekly + monthly percent
  glm-5.3-flash
  deepseek-v4.1-flash
  claude-haiku-5-5
  gpt-6-luna

pool:openrouter:<api-key>      USD limit_remaining (and optional budgets)
  anthropic/claude-haiku-4.5
  (plus any non-Auto OpenRouter traffic on that key)

pool:cursor:<browser-account>  weekly included USD + optional prepaid USD
  composer-2.5

pool:antigravity:<local-session>
  group buckets (shared, membership unproven)
  model fractions:
    antigravity:claude-opus-4-6-thinking
    antigravity:claude-sonnet-4-6
    antigravity:gemini-3.8-flash
    antigravity:gpt-oss-120b-medium
```

Routing today ignores every pool.

---

## D. Telemetry diagnosis

| Signal | Where it exists | Where it does not | Routing can use it today |
|---|---|---|---|
| Input / output / total tokens | OpenRouter, OpenCode, Cursor, and Antigravity Gemini **response bodies** (smoke and Gate 0). Metering normalizer knows OpenCode, OpenRouter, Cursor, and Antigravity shapes. | Execution record `usage`, forced null by schema v1. | No |
| OpenCode allowance (go-plus) | Inference body | Records; not a remaining-balance | No |
| OpenRouter monetary cost | Inference body | Records | No |
| Antigravity cost | Often null | — | No |
| Cursor cost | "not in usage" on the smoke | — | No |
| Duration | Execution record `duration_ns` | TTFT is not a record field | Not read by the selector |
| Time to first token | Not found as a stored metric | — | No |
| Call count | Records, one line per execution | — | Not aggregated for routing |
| HTTP 429 | `upstream_status` on the record. Antigravity quota notice becomes `cause=upstream_quota` and HTTP 429 **before** commit. OpenCode overload 429 is forwarded as a completed forwarding outcome (`response_committed: true`). | The record does not store a remaining quota | Not read by the selector |
| HTTP 503 | Routing rejection `routing_unavailable` (catalog stale / unbound) is also 503, distinct from an upstream 503 | — | The selector never sees the upstream 503; AUTO-SPEC's single pre-commit retry is not implemented in `selectVirtualAuto` |
| Quota snapshot | `QuotaService`, UI | Router snapshot | No |

### Estimated versus confirmed consumption

- **Confirmed:** provider `usage` object on the response, when present.
- **Estimated:** `UnderstandRequest` uses `len(input JSON)/4`. It is unused by eligibility. It counts JSON syntax, not tokens, and ignores the Antigravity ACP prefix.
- **Structural gap:** Gemini's ~15.3k input tokens are absent from the client prompt. Any estimate from the request body undercounts that pool by that overhead.
- **Record gap:** schema v1 cannot store usage without a schema revision. Metering is the place that already normalizes usage. A balancer that reads records will see null forever.

### Gaps that should close before a balancer trusts consumption

1. Decide the store: extend execution-record schema **or** read metering. Do not silently overload `usage: null`.
2. Persist provider usage for all four runtimes, including OpenCode allowance and OpenRouter cost, with an explicit `unavailable` status when the provider omits it.
3. Label Antigravity input as `confirmed_including_runtime_overhead`, not as user prompt size.
4. Keep quota snapshots out of the per-request critical path. Publish the last successful snapshot plus age into the routing snapshot.
5. Do not treat a forwarded HTTP 429 as proof of franchise exhaustion. The DeepSeek 429 on 2026-11-10 was overload, and the call still completed as forwarding.

These gaps do not block writing the selector against **injected fixtures**. They block treating production records as a remaining-quota source.

---

## E. Deterministic selection proposal

### Single source of request requirements

`ProjectRequirements` is already the source of capability requirements: `text_generation`, `function_tools`, `custom_tools`, `reasoning` (only when the JSON `reasoning` field is present and not null/empty), `structured_output` (`text.format.type` of `json_schema` or `json_object`), `image_input`, `audio_input`, `audio_output`, and `Unmapped` for `input_file` or an unknown format.

`UnderstandRequest` calls `ProjectRequirements` and then adds streaming, a character heuristic, virtual-model detection, and `optimization_profile`. The hot path duplicates the requirements call and drops the rest.

**Recommendation:** `routeResponses` should build one `RequestUnderstanding` and pass `Requirements` into eligibility. Do not keep a second capability parser. `UnderstandRequest` must not grow prompt-text rules.

### What the gateway can recognize without another model

| Signal | Structural source | Usable as a hard filter |
|---|---|---|
| Tools | `tools[].type` | Yes, against a **verified** `function_tools` / `custom_tools` claim |
| Structured output | `text.format` | Yes, against verified `structured_output`. Today verification is not a gate; catalog "supported" is enough, and several claims may be unknown |
| Reasoning requested | non-empty `reasoning` object | Yes, against `reasoning`. This is "the client asked for reasoning controls", not "the task is hard" |
| Image / audio | input item type or output modality prefix | Yes |
| Streaming | `stream` | Weak. Catalog streaming is always unknown, so a hard streaming filter would reject every model. Do not hard-filter on streaming until the catalog stops marking it unknown |
| Output size | `max_output_tokens` vs `limits.output` | Already implemented |
| Input size | not implemented | Compare an explicit estimate with `limits.context` only after the estimator is defined. The current `/4` heuristic is not that estimator |
| Programming vs general vs agentic | not present | **Forbidden** by the E08 contract if inferred from prompt text |

### Task fit without scores

Use bands, not grades.

**Band A — hard requirements.** Drop any model that lacks a verified capability, exceeds a known limit, is absent, stale, unbound, or has a **known** exhausted pool (remaining fraction 0, provider percent at the cap with `limited`/`status` not ok, or an unexpired `upstream_quota` reset). Unknown is not exhausted and is not healthy.

**Band B — proven contract fit.** For `structured_output`, prefer models whose strict `json_schema` passed on the gateway without relying on a fence unwrap. Today that proven set is: `anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5`, `gpt-6-luna`, `deepseek-v4.1-flash`, `antigravity:gemini-3.8-flash`. `composer-2.5` is proven only **with** the Proxy enforcer on the non-streaming path. The three blocked Antigravity models are unproven.

**Band C — profile, inside the band, using measured facts only.**

- `balanced` (default): among Band B, apply quota balancing (section F). Latency from the smoke is a tie hint, not a score. Sample size is five short calls.
- `efficiency`: prefer the lower confirmed overhead and the healthier pool. Gemini's 15.3k input overhead is a proven reason to deprioritize it for short calls. Do not invent a cost for Cursor or Antigravity.
- `intelligence`: the spec reserves this for "top benchmark reasoning". **No such benchmark exists here.** Until a comparative test is approved, `intelligence` must not reorder models. It should behave as `balanced` and emit reason `intelligence_profile_no_evidence`.

**Do not add a programming profile.** Programming is not a structural field. A client that wants a coding-oriented policy can send `optimization_profile` only after a profile has evidence. Keyword lists over the prompt would violate E08 and would duplicate `ProjectRequirements`.

### Tie break

Keep an explicit total order **inside the surviving band**, published as configuration, covering all 10 allowlist ids. The current four-entry list is not that order. Lexical and random breaks stay forbidden. Snapshot order stays the last resort and must be logged as `snapshot_fallback`.

---

## F. Adaptive balancing proposal

### What the data can support

The pools do not share a unit. OpenCode is a percent. Antigravity is a remaining fraction. Cursor is weekly USD percent plus prepaid USD. OpenRouter is remaining USD. A weighted round-robin that pretends these are the same token budget will lie.

**Recommended algorithm: scarcity-weighted selection inside the eligible band (a bounded form of weighted fair sharing).**

For each surviving model, compute a pool pressure `p` in `[0, 1]`:

- OpenCode: `p = max(rolling, weekly, monthly) / 100`, using only provider percents. The tightest window dominates.
- Antigravity model: `p = 1 - remaining_fraction`. If a group bucket is known to contain that model, `p = max(model pressure, group pressure)`. Until membership is known, use the model fraction and treat group buckets as an account-level ceiling only when **every** group is high. That is conservative and may under-use Antigravity. It will not over-commit an unknown shared bucket.
- Cursor: `p = weekly_used_percent / 100` while prepaid remaining is 0 or absent. If prepaid remaining is present and positive, weekly cap is not exhaustion; pressure stays on the weekly meter but the model stays eligible. Document this as two sub-pools.
- OpenRouter: if `limit` and `limit_remaining` are both present and `limit > 0`, `p = 1 - (remaining/limit)`. If `limit` is null, pressure is **unknown**, not zero.

Weight `w = clamp(1 - p, floor, 1)` with a small floor (recommendation: `0.05`) so a nearly empty pool can still serve a request that no other model can serve. Selection is deterministic: the largest weight wins; ties use the configured order.

This is weighted fair sharing on **relative remaining capacity**, updated from the last snapshot. It is not Deficit Round Robin and not classic WRR.

| Algorithm | Fit |
|---|---|
| Weighted Round Robin with static weights | Ignores live remaining quota. Reject. |
| Deficit Round Robin | Needs a byte or token quantum and a per-request cost **before** the call. We do not have a reliable pre-call cost for Cursor or Antigravity. Reject for v1. |
| Weighted Fair Queuing | Same missing per-request cost. The scarcity weight above is the part we can compute. |
| Equal split across providers | Fights the shared OpenCode pool (four models, one meter) and the unequal pool sizes. Reject. |
| Scarcity weight inside the eligible band | Uses only proven fields, keeps hard filters first, and is explainable. **Adopt for v1.** |

### Distribution rules

- Balance **pools**, then pick a model inside the chosen pool by the same band rules. Choosing among four OpenCode models does not spread franchise; it only spends the same meter.
- A model that fails Band A is never selected to "use up" another provider.
- A model in a lower contract band is never selected to spare quota. Example: a strict `json_schema` request does not fall through to an unproven model because the proven pool is at 70%.
- Short-circuit: if the best weight is below a reserve threshold (recommendation: remaining under 10%, `p > 0.9`) and another eligible model in the same band has `p <= 0.9`, skip the scarce pool. If it is the only model in the band, select it and log `quota_reserve_breached`.
- Do not refill weights with estimated tokens after the call in v1. The snapshot percent already includes provider-side consumption, with lag.

### Concurrency

The snapshot is shared and has no lease. Two in-flight requests can both observe 9% remaining and both choose that pool. Guarantees that **are** possible:

- Single-process advisory debit: when a decision is made, add a configured **provisional cost** to an in-memory counter for that pool, expire it after the request finishes or after a timeout, and include it in `p`.
- Provisional cost must be marked estimated. For OpenCode and OpenRouter a conservative constant per call is enough to avoid a stampede. For Antigravity, the constant must be at least the proven ~15k input tokens **only if** the fraction is later calibrated to tokens. It is not calibrated today, so the provisional debit for Antigravity should be "one request" against a small in-process budget, not a fake token subtraction from `remaining_fraction`.

Guarantees that are **not** possible:

- Exact remaining quota under concurrent callers, because the provider is the source of truth and the cache is minutes old.
- Fairness across multiple Hub instances without a shared counter. v1 should state single-instance advisory fairness.

### Policy when data is missing

Absence is not availability.

| Situation | Policy | Effect |
|---|---|---|
| Provider has no quota API for this pool | Weight unknown. Rank **after** every model with a known healthy weight in the same band. If all are unknown, use the configured order and reason `quota_unknown_all`. | May concentrate traffic on the known pools. That is accepted. |
| Quota fetch failed | Keep the last success only if its age is within the stale TTL and pressure was below the reserve. Otherwise treat as unknown. A last snapshot that was already scarce stays scarce until a newer success says otherwise. | Avoids spending a pool that was about to run out just because the read failed. |
| Snapshot older than TTL but refresh in flight | Use the stale values and reason `quota_stale`. Do not block the request on Cursor's browser collect. | Decisions can be up to minutes late. |
| Usage omitted on the call | Do not update pressure from that call. Wait for the provider snapshot. | No invented burn. |
| Reset unknown | Do not accelerate consumption because a reset might be soon. Reset time is a tie hint only when both models are under the reserve and one reset is sooner. | Prevents a guess from emptying a pool. |
| HTTP 429 without remaining quota | Classify before acting. `upstream_quota` with `reset_at` excludes that Antigravity model until `reset_at`. A forwarded inference 429 (DeepSeek overload) sets a **short cooldown on that model** (recommendation: 60s, single process) and does not mark the whole OpenCode pool exhausted. A quota-API 429 does not change inference eligibility. | Stops one overload signal from disabling four models. |
| All eligible models scarce | Select the least scarce in band, reason `quota_reserve_breached`. If none remain in Band A, return the existing no-eligible rejection. Do not leave the allowlist. | |

AUTO-SPEC already forbids automatic retry on HTTP 429. Keep that. The cooldown above is a **local exclusion**, not a second upstream call.

---

## G. Proposed architecture

```text
POST /v1/responses
  -> ValidateStructure
  -> UnderstandRequest
       -> ProjectRequirements          (only capability source)
       -> stream, profile, input-size estimate
  -> allowlist intersection            (the ten ids, Auto only)
  -> CaptureSnapshot
       catalog + admin + runtime
       + QuotaService last snapshot (age, status, pool metrics)
       + in-process provisional debits and cooldowns
  -> EvaluateEligibility               (Band A, including known exhaustion)
  -> contract band                     (Band B, homologation evidence table)
  -> profile gate                      (intelligence with no evidence => balanced)
  -> scarcity weight by pool           (section F)
  -> configured tie order
  -> substitute model id, dispatch existing pipeline
  -> headers before commit:
       X-AIHub-Resolved-Model
       X-AIHub-Resolved-Provider
       X-AIHub-Routing-Profile
       X-AIHub-Routing-Reason
  -> audit criteria codes only, no prompt text
```

### Responsibilities

| Component | Owns | Must not own |
|---|---|---|
| `ProjectRequirements` | Capability requirements | Profile, quota, scores |
| `UnderstandRequest` | The single structural reading, including profile | A second capability mapping; prompt classification |
| Allowlist | The ten MVP ids for `aihub/auto` only | Global catalog filtering |
| `EvaluateEligibility` | Hard exclusions | Ranking |
| Quota snapshot adapter (read-only view) | Pool id, pressure, age, reset, exhaustion flag | HTTP calls on the request path |
| Scarcity selector | Weight and tie order inside the band | Selecting an ineligible model |
| Execution pipeline | The call, unchanged | A second model choice after the response is committed |
| Metering | Confirmed usage after the fact | The decision itself in v1 |
| Proxy | Antigravity and Cursor quota, Antigravity execution, structured-output enforcement for Cursor | Hub ranking |

Reason codes to add, all content-free: `allowlist_miss`, `capability_unverified`, `quota_exhausted`, `quota_unknown`, `quota_stale`, `quota_reserve_breached`, `pool_shared_opencode`, `contract_band`, `intelligence_profile_no_evidence`, `cooldown_overload`, `tie_order`.

---

## H. Scenario matrix

These rows are **specification fixtures**, not live calls. Eligible sets assume the structural filter already ran. "Decision" is the recommended rule, not current code. Current code would pick `gpt-6-luna` whenever it is eligible.

Shared OpenCode pressure is one number for models O1–O4.

| # | Request | Eligible in band | Quota state | Expected decision | Why | If nothing else qualifies |
|---|---|---|---|---|---|---|
| 1 | Short text, profile balanced | All 10 in band (hypothetical; blocked models are not in band today) | All pools `p=0.3` | Configured tie order inside the best equal weight. With equal weights the order decides. Spread is not forced. | Equal health means no scarcity signal. Do not pretend to rotate without a deficit counter. | n/a |
| 2 | Short text, balanced | Haiku, OpenCode four, Composer, Gemini | OpenCode `p=0.95`; others `0.3` | Highest weight among non-OpenCode models, then tie order. OpenCode excluded by the reserve rule. | One shared pool is scarce. The four OpenCode models are not four resources. | If only OpenCode remains in band, pick the tie-order OpenCode model and log `quota_reserve_breached`. |
| 3 | Short text | Haiku and Composer | OpenRouter remaining healthy but `resets_at` in minutes; Cursor weekly `p=0.4` with reset in days | Composer if Haiku is under the 10% reserve; otherwise the higher weight. A near reset does not override a healthy other pool. | Reset is a tie hint, not a reason to drain the pool. | The nearer reset wins only when both are under the reserve. |
| 4 | Short text | The four OpenCode models only | One shared `p=0.4` | Tie order among the four (`gpt-6-luna` first if that order is kept) | Balancing cannot create a second pool. | Same pool. |
| 5 | Short text | Haiku (quota unknown) and Gemini (`p=0.2`) | OpenRouter read failed, no fresh snapshot | Gemini | Unknown ranks after known healthy. | If Gemini is also unknown, tie order and `quota_unknown_all`. |
| 6 | Short text | Composer | Cursor snapshot age greater than TTL, last `p=0.5`, refresh failed | Treat as unknown, not as `p=0.5`, unless a grace rule is explicitly approved. Recommendation: unknown. | Stale success is not current availability. | If Composer is the only model in band, select it with `quota_stale`. |
| 7 | Short text | Sonnet with `reset_at` in the future, plus Haiku healthy | Antigravity cooldown active | Haiku | `upstream_quota` excludes until reset. | If Haiku is out of band, reject. Do not call Sonnet. |
| 8 | Coding-looking prompt, no tools, no profile | Same structural band as a general prompt | Balanced pools | Same decision as a general prompt | The prompt text is not an input. A programming profile does not exist. | n/a |
| 9 | `reasoning` object present | Models whose catalog `reasoning` is supported **and** verified by policy | Pools equal | Tie order among those models | "Heavy reasoning" is not observable. Only the reasoning **control** is. | `no_eligible_model` if every reasoning claim is unknown. Unknown must not pass. |
| 10 | `tools` non-empty | Models with verified `function_tools` | Any | Only that set, then scarcity | Quota never reintroduces a model without tools. | Reject if none verified. Do not use catalog "unknown". |
| 11 | `text.format.json_schema` strict | The six natively proven models. Composer only if the non-streaming Proxy enforcer is in force. | OpenCode scarce | Haiku or Gemini by weight, not an unproven Antigravity model | Contract band outranks balance. | Reject rather than send schema to an unproven model. |
| 12 | Estimated input above 200k and below 1M | Catalog context: the two OpenCode 1M-class models, Luna (1.05M), Gemini (1,048,576). Haiku, Composer, Opus, Sonnet, gpt-oss excluded by context. | OpenCode `p=0.5`, Gemini `p=0.2` but 15.3k overhead still inside the window | Gemini if it remains in the contract band for this request; else the healthier OpenCode model by tie order | Context is a hard filter. Overhead does not remove Gemini from a large-context band; it does deprioritize Gemini on **short** calls. | Reject when the estimate exceeds every allowlist context. |
| 13 | Short text | Antigravity models all in cooldown or unbound | Others healthy | Best non-Antigravity weight | Provider outage is an eligibility failure for that provider only. | Reject when the other pools are also ineligible. |
| 14 | Short text | All eligible, every pool `p>0.9` except Haiku `p=0.2` | One healthy pool | Haiku | The reserve rule concentrates on the only comfortable pool. That is correct. It is not a permanent pin; the weight changes when the snapshot changes. | If Haiku then fails Band A, `quota_reserve_breached` on the least scarce remaining model. |

Scenario 1 needs an explicit product choice: equal health does **not** rotate providers. If rotation under equal health is required, add an in-process deficit counter later. It is not justified by the quota data we have.

Scenario 12 depends on an estimator that does not exist yet. Until it exists, do not exclude models for input size. The scenario shows the intended rule, not current behavior.

---

## I. Decisions pending approval

1. **Gate.** Auto stays off until all 10 models are approved, or the allowlist ships with the seven proven models and the three blocked ids hard-excluded. AUTO-SPEC says all 10. The reports say do not enable production Auto.
2. **Canonical ids.** Confirm hyphenated Antigravity ids as the allowlist.
3. **Composer structured output.** Count Proxy enforcement as sufficient for non-streaming strict schema, or require a native pass.
4. **Intelligence profile.** Keep the name and make it a no-op until a comparative test exists, or hide it from clients.
5. **Programming.** Confirm that task type will not be inferred from text, and that no coding profile is added in this increment.
6. **Unknown capability.** Confirm unknown remains an exclusion when the request requires that capability (already E08).
7. **Streaming.** Confirm it is not a hard filter while the catalog marks streaming unknown.
8. **Input estimate.** Approve an estimator and whether Antigravity overhead is added before the context check.
9. **Record schema.** Approve a schema change so `usage` can be non-null, or designate metering as the only consumption store.
10. **Quota on the snapshot.** Approve attaching the last `QuotaService` reading to the routing snapshot, with the 2-minute staleness visible in the reason code.
11. **Reserve threshold.** Confirm 10% remaining (`p > 0.9`) and weight floor `0.05`.
12. **OpenCode windows.** Confirm pressure is the max of rolling, weekly, and monthly.
13. **Antigravity groups.** Require one read-only usage capture (no inference) to map the four model ids onto group buckets before the balancer treats them as independent.
14. **Cursor derived percent.** Forbid Hub-computed percent for routing, or allow it only when labeled.
15. **429 cooldown.** Confirm 60s model-local cooldown for forwarded overload, and pool exclusion until `reset_at` for `upstream_quota`.
16. **Equal-health behavior.** Confirm tie order, not rotation.
17. **Single instance.** Confirm advisory in-process debits are enough for v1.
18. **Headers.** Confirm the four AUTO-SPEC headers and the reason-code list.
19. **Fallback.** Confirm no second upstream call on 429, and at most one pre-commit retry on 503/timeout onto the next band member (AUTO-SPEC) or no retry at all in the first increment.
20. **Non-Auto traffic.** Confirm OpenRouter and OpenCode pools include calls that did not go through `aihub/auto`.

---

## J. Implementation roadmap

Each step is independent enough to review. None of them starts until this discovery is approved. Acceptance is behavioral and fixture-based. No live quota burn in unit tests.

### J1. Allowlist and evidence table

Restrict `selectVirtualAuto` candidates to the ten ids. Add a checked-in evidence table: homologation class, schema proof, blocked-until. Unproven structured output cannot satisfy `structured_output`.

**Accept:** a request for `aihub/auto` never selects a model outside the ten; a strict schema request never selects the three blocked ids or a model whose schema proof is missing; existing explicit model routes are unchanged.

**Depends on:** decisions 1–3.

### J2. One structural reading

Route the hot path through `UnderstandRequest` and delete the duplicate `ProjectRequirements` call. Persist the profile that was actually applied.

**Accept:** unit tests show tools, schema, image, audio, reasoning, and profile come from one function; `routeResponses` does not parse capabilities again.

**Depends on:** decision 5.

### J3. Quota view on the snapshot

Read the cached `QuotaService` snapshot. Map it to pool ids. No network on the request path.

**Accept:** fixtures for each provider produce pressure, age, and exhaustion; a missing snapshot yields unknown; tests do not call providers.

**Depends on:** decisions 10–14. Antigravity group membership can ship as "model fraction only" if decision 13 is deferred, with that limitation in the reason code.

### J4. Bands and scarcity selector

Replace the four-entry ladder with Band A, Band B, weight, and a full tie order.

**Accept:** the scenario fixtures in section H, encoded as table tests, match the expected decisions. A scarce pool cannot override a capability miss. `intelligence` does not reorder without an evidence flag.

**Depends on:** J1–J3 and decisions 6, 11, 15, 16.

### J5. Advisory concurrency and 429 memory

In-process provisional debit and overload cooldown.

**Accept:** two decisions in one process against a 9% pool do not both ignore the debit; an overload 429 cools one OpenCode model and leaves the other three eligible; an Antigravity `upstream_quota` cools that model until `reset_at`.

**Depends on:** J4 and decisions 15, 17.

### J6. Telemetry

Schema or metering path so confirmed usage is queryable. Not required for J4 if tests inject quota fixtures. Required before any rule claims to learn from past calls.

**Accept:** a fixture response with usage is stored as confirmed; a fixture without usage is stored as unavailable; records no longer force null **or** the reader is documented to ignore records and use metering.

**Depends on:** decision 9.

### J7. Transparency and failure matrix

Headers before commit. Single pre-commit retry only if decision 19 allows it.

**Accept:** integration test on `aihub/auto` shows the four headers and a substituted model id; a streamed failure does not switch model; HTTP 429 is not retried.

**Depends on:** J4 and decisions 18–19.

### Explicitly out of this roadmap

- Quality scores, coding benchmarks, and an `intelligence` ordering.
- Prompt-text classification.
- Catalog sync, new homologation calls, and live quota probes.
- Changing Proxy structured-output behavior.
- Multi-instance shared debits.

---

## Evidence index

- `docs/specs/AUTO-SPEC.md`, `docs/decisions/adr-023-auto-aihub-virtual-model.md`
- `docs/specs/E08-routing-contract.md`, `docs/specs/E08-intelligent-routing.md`
- `internal/routing/virtual_selector.go`, `selector.go`, `understanding.go`, `project.go`, `eligibility.go`
- `internal/gateway/responses_routing.go`
- `internal/records/codec.go`, `internal/metering/normalize.go`
- `internal/providers/opencode_quota.go`, `openrouter_quota_key.go`, `antigravity_quota.go`, `cursor_usage_quota.go`, `quota_service.go`
- `internal/catalog/capability.go`
- `reports/llm-smoke/preflight/summary.md`
- `reports/llm-homologation-remediation.md`
- `reports/llm-homologation/gate0-json-schema-post-fix-2026-11-10.md`
- `docs/validation/etapa-2-opencode-go-quota-2026-11-04.md`
- Proxy: `docs/antigravity-usage.md`, `packages/providers/antigravity/src/usage/usage-service.ts`, `docs/providers.md` (Cursor collect)

No code, configuration, credential, catalog sync, container, or git operation was performed for this discovery.
