# LLM Multi-Model Smoke Test — Results (Phase 4)

- Date: 2026-11-09 (battery window: single sequential run, ~1 minute of wall time)
- Environment: local AI-Hub gateway (`127.0.0.1:8080`, loopback-only, started once via `go run ./cmd/ai-hub`, no config changes)
- Executor: agent-driven canonical runner `reports/llm-smoke/run-battery.py` (curl engine, Responses API). Codex CLI reference version: `codex-cli 0.164.0-alpha.11.1` (same gateway/provider config: `aihub`, `wire_api=responses`)
- Execution contract honored: 50 requests planned, **50 executed, 1 call per test, no retries, no automatic fallback, sequential per model**, output capped at 128–256 tokens
- Audit: `GET /v1/records/executions` cross-checked → provider of every executed record matches the expected mapping (no unauthorized provider substitution, no fallback; `outcome: completed` 100 %)
- Sanitization: no tokens/keys/secrets recorded in this report; gateway key only ever expanded in-runtime from `.env`

## 1. Models tested (real identifiers confirmed at runtime)

| Model (IDE) | Requested slug | Provider (echoed by records) | Result summary |
|---|---|---|---|
| Claude Haiku 4.5 | `anthropic/claude-haiku-4.5` | openrouter | 4/5 — PARTIALLY USABLE |
| GLM 5.3 Flash | `glm-5.3-flash` | opencode-go | 4/5 — PARTIALLY USABLE |
| DeepSeek V4.1 Flash | `deepseek-v4.1-flash` | opencode-go | 5/5 — USABLE |
| Claude Haiku 5.5 | `claude-haiku-5-5` | opencode-go | 4/5 — PARTIALLY USABLE |
| 6 Luna | `gpt-6-luna` | opencode-go | 5/5 — USABLE |
| Composer 2.5 | `composer-2.5` | cursor | 5/5 — USABLE |
| Claude Opus 4.6 (Thinking) | `antigravity:claude-opus-4-6-thinking` | antigravity | BLOCKED (upstream quota) |
| Claude Sonnet 4.6 | `antigravity:claude-sonnet-4-6` | antigravity | BLOCKED (upstream quota) |
| Gemini 3.8 Flash | `antigravity:gemini-3.8-flash` | antigravity | 5/5 — USABLE |
| OSS 120B (Medium) | `antigravity:gpt-oss-120b-medium` | antigravity | BLOCKED (upstream quota) |

Out of scope (never called, never scored): `openrouter/auto` (`openrouter/auto`, `openrouter/auto-beta`, `openrouter/auto-code` family) and all AI-Hub automatic routing.

## 2. Comparative table

| Model | Connectivity | Generation | Reasoning | JSON | Streaming | Verdict |
|---|---|---|---|---|---|---|
| Claude Haiku 4.5 | PASS | PASS | PASS | FAIL (fences) | PASS | PARTIALLY USABLE |
| GLM 5.3 Flash | PASS | PASS | PASS | FAIL (fences) | PASS | PARTIALLY USABLE |
| DeepSeek V4.1 Flash | PASS | PASS | PASS | PASS | PASS | USABLE |
| Claude Haiku 5.5 | PASS | PASS | PASS | FAIL (fences) | PASS | PARTIALLY USABLE |
| 6 Luna | PASS | PASS | PASS | PASS | PASS | USABLE |
| Composer 2.5 | PASS | PASS | PASS | PASS | PASS | USABLE |
| Claude Opus 4.6 (Thinking) | BLOCKED | BLOCKED | BLOCKED | BLOCKED | BLOCKED | BLOCKED (quota) |
| Claude Sonnet 4.6 | BLOCKED | BLOCKED | BLOCKED | BLOCKED | BLOCKED | BLOCKED (quota) |
| Gemini 3.8 Flash | PASS | PASS | PASS | PASS | PASS | USABLE |
| OSS 120B (Medium) | BLOCKED | BLOCKED | BLOCKED | BLOCKED | BLOCKED | BLOCKED (quota) |

## 3. Detailed per-test results and latency

Latency = total wall time per call (gateway + upstream), seconds.

### Claude Haiku 4.5 (`anthropic/claude-haiku-4.5`, openrouter) — 4/5
| Test | Result | HTTP | Latency |
|---|---|---|---|
| T01, T02, T03, T05 | PASS (T05 passed streaming, 10 SSE events) | 200 | 0.92–1.72 |
| T04 | FAIL — correct fields/values but wrapped in markdown fences (plan strictly forbids) | 200 | 0.87 |

### GLM 5.3 Flash (`glm-5.3-flash`, opencode-go) — 4/5
| Test | Result | HTTP | Latency |
|---|---|---|---|
| T01, T02, T03, T05 (79 SSE events) | PASS | 200 | 1.12–2.33 |
| T04 | FAIL — correct values but wrapped in markdown fences | 200 | 1.12 |

### DeepSeek V4.1 Flash (`deepseek-v4.1-flash`, opencode-go) — 5/5
All PASS; latency 0.93–3.12; streaming 26 SSE events; T04 strict JSON.

### Claude Haiku 5.5 (`claude-haiku-5-5`, opencode-go) — 4/5
T01–T03, T05 PASS (latency 0.92–2.10); T04 FAIL — correct values but wrapped in markdown fences.

### 6 Luna (`gpt-6-luna`, opencode-go) — 5/5
All PASS; latency 0.97–1.86; streaming 11 SSE events; strict JSON.

### Composer 2.5 (`composer-2.5`, cursor) — 5/5
All PASS; latency 2.98–11.13 (slowest model of the battery); streaming 10 SSE events.

### Claude Opus 4.6 (Thinking) / Claude Sonnet 4.6 / GPT-OSS 120B (Medium) (`antigravity:*`)
- 0/5 executed successfully: every response returned HTTP 200 with body text `⚠ Individual quota reached. Please upgrade your subscription to increase your limits. Resets in ~54 h 24 m` instead of model output.
- SSE transport was actually functioning (9 stream events, clean termination), so this is **not** a streaming/connectivity defect — it is an upstream subscription-quota exhaustion of the antigravity helper.
- Classification: **BLOCKED** (external dependency/quota), per the rule that temporary unavailability must not be attributed to functional lack. Re-test after quota reset (~54 h from 2026-11-09) is recommended.

### Gemini 3.8 Flash (`antigravity:gemini-3.8-flash`) — 5/5
All PASS; slowest effective end-to-end generation (5.55–8.35 s), streaming 9 SSE events, strict JSON.
**Consumption anomaly:** upstream usage reports ~**15.3k input tokens per call** (T01 15303, T02 15303, T03 15321, T04 15325), i.e. the antigravity gateway injects a large fixed upstream context per request (~61k input tokens billed over the 4 non-stream tests alone). Low-value traffic to this model is disproportionately expensive in input tokens.

## 4. Errors observed

1. `antigravity:claude-opus-4-6-thinking`, `antigravity:claude-sonnet-4-6`, `antigravity:gpt-oss-120b-medium`: upstream quota-exhaustion notice routed as normal 200/SSE output (5× each). No HTTP error code surfaced — accessibility degradation is expressed in-band (text), which can silently damage callers expecting model output; flagged as a robustness observation for upstream integration.
2. T04 strict-JSON violations (exact values correct, but wrapped in ``` fences): `anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5`.
3. No timeouts (curl exit 0 in all 50 calls), no HTTP 4xx/5xx, no empty responses, no auth failures, no model-id mismatch (`model_echo` equals the requested slug on all non-stream calls).

## 5. Consumption

| Model | total input tokens | total output tokens | total tokens (5 calls) | explicit cost reported |
|---|---|---|---|---|
| Claude Haiku 4.5 | 134 | 265 | 399 | ≈ $0.00146 (OpenRouter pricing) |
| GLM 5.3 Flash | 142 | 271 | 413 | $0 (go-plus allowance) |
| DeepSeek V4.1 Flash | 214 | 329 | 543 | $0 (go-plus allowance) |
| Claude Haiku 5.5 | 176 | 255 | 431 | $0 (go-plus allowance) |
| 6 Luna | 111 | 147 | 258 | $0 (go-plus allowance) |
| Composer 2.5 | 444 | 187 | 631 | n/a in usage |
| antigravity (quota-blocked ×3) | 444 each | 108 each | 552 each | n/a (blocked) |
| Gemini 3.8 Flash | 61.252 | 1.351 | 62.603 | n/a |

> Phase 0 audit correction (2026-11-09): the Gemini totals above compose **only the 4 non-streaming calls** (T01 input 15.303, T02 15.303, T03 15.321, T04 15.325 = 61.252 input; 1.351 output across the same 4 calls). The T05 streaming call did not report usage in its SSE completion event, so battery-wide true totals are **at least** 61.252 input / 1.351 output. Also note: the gateway execution records store `usage: null` — token usage was only available from the response bodies of the original battery.

Battery-wide: total token use is dominated by Gemini 3.8 Flash's upstream context injection; every other model consumed under 1k tokens across its 5 calls (per the per-call usage captured in the response bodies of the original battery).

## 6. Evidence chain

- Raw request/response pairs (bodies, HTTP codes, latencies, content types): `reports/llm-smoke/raw/<model-slug>-T0X.(json|sse)` (50 files)
- Structured evaluation of all 50 calls: `reports/llm-smoke/results.json`
- Anti-fallback audit snapshot: `reports/llm-smoke/audit-executions.json`
- Battery runner (exact commands actually used): `reports/llm-smoke/run-battery.py`
- Preflight evidence: `reports/llm-smoke/preflight/summary.md` and sibling JSONs
- Plan (Phases 0–1): `reports/llm-smoke-test-plan.md`

## 7. Limitations

- Single sample per test; adequate for smoke verdicts, not for capability benchmarking (reasoning in particular).
- Strict-JSON scoring rejects markdown fences by spec; the three fenced models produced every correct field/value — the capability exists, the strictness does not (relaxation is a SC/standards decision, not a model defect).
- Latency measured end-to-end via gateway; upstream-only latency not isolated.
- Antigravity quota reset (~54 h) prevents functional verdict on its three models for now.
- `max_output_tokens` honored everywhere; no truncation was detected in evaluated outputs.

## 8. Recommendations

1. Re-run the battery for the three `antigravity:*` models after quota reset (~54 h from 2026-11-09); until then they are `BLOCKED`, not `FAIL`.
2. Add an upstream-output pre-classifier (or explicit error mapping) so quota/subscription notices do not masquerade as successful model text with HTTP 200 — at least in gateway telemetry.
3. For strict machine-protocol consumers, prefer DeepSeek V4.1 Flash / 6 Luna / Composer 2.5 / Gemini 3.8 Flash today; for cheap conversational filler, `gpt-6-luna` had the lowest token use and best latency score.
4. Investigate/limit the ~15.3k injected input tokens on `antigravity:gemini-3.8-flash` upstream calls (context injection), which multiplies input-token billing ~100× for short prompts.
5. Composer 2.5 is functional but 3–6× slower than peers — acceptable, plan capacity accordingly.

## 9. Status of governed restrictions

- No code/config/`.env`/catalog modified; no catalog sync executed (stale-listing observation remains documented, harmless for serving).
- No commits, pushes, deploys. No load tests. Battery stayed within the small-output contract.
- All five test types ran per model where execution was possible (STREAM test included); no test skipped.
