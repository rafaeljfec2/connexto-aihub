# LLM Homologation — Remediation and Re-test Report

- Date: 2026-11-09
- Scope: remediation of the first smoke battery, JSON contract classification, Antigravity quota handling, homologation matrix
- Out of scope (not implemented): Auto AI-Hub, automatic model selection, provider fallback, cost-driven routing, Intelligent Routing itself
- Status of changes: **uncommitted working tree**. No commit, push or deploy was made.

## 1. Mathematical audit of the original battery (Phase 0)

| Item | Originally published | Recomputed | Status |
|---|---|---|---|
| Reachable-model pass rate | 29/35 (82.9%) | **32/35 (91.4%)** | Corrected in `reports/llm-smoke-adjustments.md` with a visible correction note |
| Real failures | 3 (T04 markdown fences) | 3 | Confirmed |
| Quota-blocked calls | 15 | 15 | Confirmed (3 Antigravity models × 5) |
| Gemini usage 61,252 in / 1,351 out | Presented as battery total | Covers only T01–T04 (4 calls); T05 reported no usage | Clarified in `reports/llm-smoke-test-results.md` |
| Battery execution integrity | 50 calls | 50 calls, 5 per model, 18:20:11–18:22:34 UTC, zero provider mismatches | Re-verified from gateway records |

Evidence limits: the raw battery files (`results.json`, 50 request/response pairs, runner, audit snapshot) were lost from the working directory after the session. The primary execution records were recovered from the gateway data directory and saved to `reports/llm-homologation/jsonl-reconstructed/battery-executions.json`. Per-call token usage existed only in the lost response bodies. The gateway records store `usage: null` for every execution, which is a telemetry gap (see §9).

## 2. Antigravity quota notice classification (Phase 1)

**Problem.** The upstream returned HTTP 200 with the notice `⚠ Individual quota reached… Resets in Xh Ym Zs.` as the model text. Callers received it as generated content.

**Fix.** Detection happens inside the Connexto Proxy client, for the `antigravity` logical provider only, before any byte is committed downstream.

- `internal/providers/antigravity_quota_notice.go` (new): strict anchored matcher for the complete notice with an `hXmYsZ` reset window. Responses JSON is classified on the whole output text. SSE is classified on the accumulated text of the complete events in a bounded prefix (64 KiB raw, 256 runes). Divergence from the canonical lead-in opts the response out immediately. Unknown media types are never interpreted.
- `internal/providers/connexto_proxy.go`: hook in `OpenTargetResponses`, restricted to `LogicalProviderID == antigravity`.
- `internal/execution`: new cause `upstream_quota`, `Result.QuotaResetInNS`, classification via an unexported interface (same seam as `ResponseHeaderTimeoutError`, no package cycle).
- `internal/gateway`: `writeExecutionFailureResponse` maps the opening failure to **HTTP 429 `upstream_quota_exceeded`** with `reset_at`. The upstream notice text is never echoed. Historical 502/504 mappings are unchanged.
- `internal/records/record.go`: `upstream_quota` added to the cause vocabulary.

**Tests added.** Providers: JSON notice blocks (plain and with leading newline plus emoji variation selector); legitimate mention of quota passes through byte-identical; empty output passes through; unknown media type untouched; error status untouched; SSE notice blocks; SSE notice split across deltas blocks; SSE partial notice passes through; SSE legitimate mention passes through; notice delivered only in `response.output_item.done` blocks; near-match negatives; client-level antigravity classified and cursor not classified. Execution: quota classification with reset window. Gateway: 429 mapping with `reset_at` (~54h ahead), no notice leakage, 502/504 preserved.

**Quality gates.**
- `go vet ./internal/...`: clean.
- `go test -race -count=1` on `providers`, `execution`, `gateway`, `records`: pass.
- `go test -count=1 ./...`: 37 packages pass, 0 failures.
- Frontend: not changed, not run.

**Live validation (controlled, no retries).**

| Call | HTTP | Content-Type | Body | Record |
|---|---|---|---|---|
| `antigravity:claude-sonnet-4-6`, non-stream | 429 | application/json | `upstream_quota_exceeded`, `reset_at=2026-11-12T00:46:33Z` | failed / opening / upstream_quota / committed=false |
| `antigravity:claude-sonnet-4-6`, stream | 429 | application/json | same shape, `reset_at=2026-11-12T00:46:32Z` | failed / opening / upstream_quota / committed=false |

Stream headers are never sent for a quota-blocked response, so no SSE error event was needed. Evidence: `reports/llm-homologation/quota-revalidation/`.

## 3. Gemini context injection (Phase 2)

**Finding.** The AI-Hub adds no context on the Antigravity path. `normalizeAntigravityResponsesModel` only prefixes the model id. The Connexto Proxy's Antigravity provider (`packages/providers/antigravity/src/provider.ts`) opens an ACP session per request, with `cwd` (the workspace directory) and `mcpServers`. It builds an MCP bridge only when the request carries tools. The smoke battery sent no tools.

The ~15,300 input tokens per Gemini call therefore come from the `agy` runtime, a closed binary, when it creates the ACP session. The repository does not contain the injected content. Its exact composition cannot be measured from the outside.

**Action.** No code change. The overhead is documented, not removed. Removing it would require changing the closed `agy` runtime, which is out of scope. The before/after measurement from the plan's acceptance criterion cannot be produced because nothing changed.

**Risk for routing.** Routing cost estimates for `antigravity:gemini-3.8-flash` must include ~15k input tokens per call, even for trivial prompts.

## 4. Structured JSON contract (Phase 3)

Nine controlled calls, one per model and variant, non-stream, `max_output_tokens=256`. Each variant was requested through the gateway's Responses endpoint. Raw responses and the script are in `reports/llm-homologation/json-contract/`.

| Model | prompt only | `text.format` json_object | `text.format` json_schema (strict) | Classification |
|---|---|---|---|---|
| `anthropic/claude-haiku-4.5` | PASS_FENCED | PASS_FENCED | **PASS** | JSON via json_schema only |
| `glm-5.3-flash` | PASS | PASS | **PASS** | JSON via prompt, json_object or json_schema |
| `claude-haiku-5-5` | PASS_FENCED | PASS | **PASS** | JSON via json_object or json_schema |

Observations:
- Prompt-only output is **not deterministic**. `glm-5.3-flash` was fenced in the first battery and plain here. Sample size is one per variant.
- `json_schema` was the only variant that produced clean output on every model tested. It is the recommended contract for machine consumers.
- Models not run in this phase (deepseek, luna, composer, gemini) have JSON marked `NOT_TESTED` for structured variants.

## 5. Catalog freshness (Phase 5)

**Not executed.** `POST /v1/catalog/sync` is a state change and requires your express authorization.

Current state (from the preflight): all 10 slugs `presence: present`, `freshness: fresh`, `listing_state: stale`. Serving is unaffected.

Expected impact of a sync: refresh listing state for providers `openrouter`, `opencode-go`, `cursor`, `antigravity`. Capabilities and limits would be re-read from upstream. Antigravity models would still be blocked by quota.

## 6. Performance and consumption profile (Phase 6)

Latency values are from the recovered gateway record durations and the original battery. Averages are over the 5 battery calls per model, end-to-end through the gateway.

| Model | Avg latency | Range | Tokens (5 calls) | Explicit monetary cost | Franchise / billing |
|---|---|---|---|---|---|
| `anthropic/claude-haiku-4.5` | 1.35 s | 0.87–1.72 s | 399 | ≈ $0.00146 (openrouter) | Paid per call |
| `glm-5.3-flash` | 1.61 s | 1.12–2.33 s | 413 | none reported | opencode-go go-plus allowance |
| `deepseek-v4.1-flash` | 1.92 s | 0.93–3.12 s | 543 | none reported | opencode-go go-plus allowance |
| `claude-haiku-5-5` | 1.27 s | 0.92–2.10 s | 431 | none reported | opencode-go go-plus allowance |
| `gpt-6-luna` | 1.33 s | 0.97–1.86 s | 258 | none reported | opencode-go go-plus allowance |
| `composer-2.5` | 5.72 s | 2.98–11.13 s | 631 | not in usage | cursor |
| `antigravity:gemini-3.8-flash` | 6.65 s | 5.55–8.35 s | ~62,600 (T05 not counted) | not in usage | agy runtime, ~15.3k input per call |

Notes:
- The go-plus allowance is an included subscription quota, not a free service. Its consumption is recorded per call in the upstream response (for example, 2.33e-5 for `glm-5.3-flash` T04), but it is not a monetary cost.
- Composer 2.5 is slower than peers in every test. Five samples are not enough to disqualify it. A short repeat with more samples is recommended before using it in latency-sensitive routes. No load test was run.
- Token usage is missing from all gateway execution records (§9).

## 7. Homologation matrix (Phase 7)

Classifications: `HOMOLOGATED`, `CONDITIONAL`, `BLOCKED`, `NOT_SUPPORTED`, `NOT_TESTED`.

| Model | Conn. | Gen. | Instr. | Reasoning | JSON | JSON Schema | Streaming | Usage | Errors | Routing integrity | Latency | Quota | Classification |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `anthropic/claude-haiku-4.5` | PASS | PASS | PASS | PASS | PASS (json_schema) | PASS | PASS | cost reported | none | OK | 0.9–1.7 s | n/a | **HOMOLOGATED** (JSON via json_schema) |
| `glm-5.3-flash` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | allowance | none | OK | 1.1–2.3 s | allowance | **HOMOLOGATED** |
| `deepseek-v4.1-flash` | PASS | PASS | PASS | PASS | PASS (prompt) | NOT_TESTED | PASS | allowance | none | OK | 0.9–3.1 s | allowance | **CONDITIONAL** (schema not tested) |
| `claude-haiku-5-5` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | allowance | none | OK | 0.9–2.1 s | allowance | **HOMOLOGATED** |
| `gpt-6-luna` | PASS | PASS | PASS | PASS | PASS (prompt) | NOT_TESTED | PASS | allowance | none | OK | 1.0–1.9 s | allowance | **CONDITIONAL** (schema not tested) |
| `composer-2.5` | PASS | PASS | PASS | PASS | PASS (prompt) | NOT_TESTED | PASS | not in usage | none | OK | 3.0–11.1 s | n/a | **CONDITIONAL** (latency, schema not tested) |
| `antigravity:gemini-3.8-flash` | PASS | PASS | PASS | PASS | NOT_TESTED | NOT_TESTED | PASS | not in usage; ~15.3k input/call | none (battery) | OK | 5.6–8.3 s | quota (live) | **CONDITIONAL** (overhead, schema not tested) |
| `antigravity:claude-opus-4-6-thinking` | BLOCKED | BLOCKED | BLOCKED | BLOCKED | NOT_TESTED | NOT_TESTED | BLOCKED | n/a | quota 429 | n/a | n/a | blocked until 2026-11-12 00:46 UTC | **BLOCKED** |
| `antigravity:claude-sonnet-4-6` | BLOCKED | BLOCKED | BLOCKED | BLOCKED | NOT_TESTED | NOT_TESTED | BLOCKED | n/a | quota 429 | n/a | n/a | blocked until 2026-11-12 00:46 UTC | **BLOCKED** |
| `antigravity:gpt-oss-120b-medium` | BLOCKED | BLOCKED | BLOCKED | BLOCKED | NOT_TESTED | NOT_TESTED | BLOCKED | n/a | quota 429 | n/a | n/a | blocked until 2026-11-12 00:46 UTC | **BLOCKED** |
| `openrouter/auto` (Auto AI-Hub) | — | — | — | — | — | — | — | — | — | — | — | — | Out of scope |

Reading the matrix:
- **HOMOLOGATED** requires every necessary dimension proven. The three models marked so have JSON via `json_schema` or `json_object` proven.
- **CONDITIONAL** models are usable, with documented limits, but must not be selected for structured-output tasks until the schema variant is tested.
- **BLOCKED** models are excluded from automatic selection until their quota resets and they are re-tested.
- Nothing was homologated merely because it appears in `/v1/models`.

## 8. Phase 4 — Antigravity re-test

**Not executed in full.** Each of the three models was probed once. All three returned `429 upstream_quota_exceeded` with `reset_at=2026-11-12T00:46:3x Z`. The full five-test battery must run after that time. No quota workaround was attempted.

## 9. Pending items and risks

| # | Item | Owner / gate |
|---|---|---|
| 1 | `POST /v1/catalog/sync` (Phase 5) | Requires your express authorization |
| 2 | Re-run the full battery for the three Antigravity models after the quota reset | After 2026-11-12 00:46 UTC |
| 3 | Populate token usage in execution records (currently `usage: null`) | Gateway telemetry change, not in this remediation |
| 4 | Test `json_schema` on deepseek, luna, composer, gemini | Optional for routing, required before using them for structured output |
| 5 | Commit and push the working tree | Requires your express authorization (not granted) |
| 6 | Gateway runs under `nohup` with a log file; no supervisor | Operational, recommend a systemd unit or equivalent |
| 7 | The Antigravity overhead (~15k input tokens) is in the closed `agy` runtime | Cannot be changed from this repository |
| 8 | Prompt-only JSON is non-deterministic (sample size one) | Use `json_schema` for machine consumers |

## 10. AI Memory evidence references

- `procedures/llm-smoke-test-results-2026-11-09.md`: original battery results.
- `procedures/llm-smoke-test-preflight-2026-11-09.md`: preflight evidence.
- `procedures/llm-homologation-remediation-phase0-1-2026-11-09.md`: Phase 0 audit and Phase 1 implementation and live validation.
- This report records Phases 2–7. Its memory page is written as the final step (see §11).

## 11. Recommendation on Intelligent Routing

**Not yet. Conditional go after two items.**

The gate in the plan requires that models are classified individually and limitations documented. That is now true for every model except the three Antigravity models, which are documented as `BLOCKED` and must be excluded from automatic selection.

Before the router is built, two items need to be closed:
1. The catalog sync (Phase 5), so listing state is current. This needs your authorization.
2. Token usage recorded in execution records (§9, item 3). The router's cost decisions depend on it, and the current records have no usage.

Until the Antigravity re-test succeeds, the router must treat `antigravity:*` as unavailable. Structured-output routes must use only models with `json_schema` proven (`anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5`).
