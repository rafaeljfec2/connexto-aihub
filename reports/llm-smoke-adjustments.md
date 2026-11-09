# LLM Smoke Test — What Worked / What Did Not Work (Adjustments Report)

- Date: 2026-11-09
- Basis: 50 executed requests (10 models × 5 tests, 1 call each, sequential, no retries, no fallback), audited via `GET /v1/records/executions` (provider/model integrity confirmed, 0 mismatches)
- Evidence: `reports/llm-smoke/results.json`, `reports/llm-smoke/raw/` (50 raw pairs), `reports/llm-smoke/audit-executions.json`
- Companion reports: `reports/llm-smoke-test-results.md` (measurement detail), `reports/llm-smoke-test-plan.md` (plan)

## 1. What worked (validated OK)

| Area | Evidence | Status |
|---|---|---|
| Gateway lifecycle (start, healthz, readyz) | `preflight/summary.md`, battery: 0 transport errors in 50 calls | OK |
| Loopback-only exposure | `ss` shows only `127.0.0.1:8080`; external interface refused | OK |
| Auth (`AIHUB_GATEWAY_API_KEY`) | 50/50 calls authorized; no auth failures | OK |
| ConnectivityManager: every reachable model answered explicitly-selected slug | `model_echo == requested slug` on all non-stream calls | OK |
| Routing integrity: no Auto AI-Hub, no cross-provider fallback | audit: provider matched expected map for all 10 models | OK |
| Basic generation + instruction following (T01, T02) | 7/7 reachable models PASS | OK |
| Reasoning (T03: 3×4=12 with calculation) | 7/7 reachable models PASS, result correct | OK |
| Streaming (T05, SSE) | all reachable models PASS; SSE reconstructed (`CONNECTION_OK`), clean termination; e.g. glm 79 events, deepseek 26 | OK |
| Usage/cost reporting from upstream | `usage` populated (openrouter gives explicit cost; opencode-go reports go-plus allowance) | OK |

Model verdicts (functional):
- Fully usable: `deepseek-v4.1-flash`, `gpt-6-luna`, `composer-2.5`, `antigravity:gemini-3.8-flash` (5/5).
- Usable with one strictness deviation: `anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5` (4/5, only T04).

## 2. What did not work (with cause and suggested adjustment)

### 2.1 Strict-JSON output wrapped in Markdown fences (T04)

- Affected: `anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5`.
- Behavior: fields/values were exactly correct (`name=Connexto AI-Hub`, `language=Go`, `type=API`) but wrapped in ``` fences — plan's strict test forbids any Markdown around exclusive JSON → FAIL by spec, not by capability.
- Adjustment options (pick per consumer contract):
  1. Call sites needing machine-parseable output: send `response_format` / JSON-schema mode if the provider contract supports it, or add a fence-stripping sanitizer on the consumer side;
  2. Relax the strictness rule (accept fenced JSON when values match) in the runner spec;
  3. Update model guidance (system preamble) to forbid fences.
- Non-blocking for development use; three of ten models affected consistently.

### 2.2 Antigravity quota exhaustion surfaced as HTTP 200 "model text"

- Affected: `antigravity:claude-opus-4-6-thinking`, `antigravity:claude-sonnet-4-6`, `antigravity:gpt-oss-120b-medium` (15/15 calls returned the substitution).
- Behavior: upstream served `⚠ Individual quota reached. Please upgrade your subscription... Resets in ~54h` in place of generated text, HTTP 200, normal SSE — indistinguishable from a real completion at transport level.
- Why it is not a model verdict: SSE transport worked (9 events, clean termination); the Antigravity `cursor`-provider upstream itself is the limiter. Classification: `BLOCKED`, re-test after quota reset (~54h from 2026-11-09, i.e. around 2026-11-11 18:00 UTC).
- Adjustments:
  1. Upstream integration: classify subscription/quota notices and surface as HTTP 429 / structured error instead of in-band text;
  2. Add gateway-level detector for known quota-notice patterns on outputs from `provider=antigravity` (defensive telemetry change only);
  3. Operational: keep the three models pinned out of automatic routing until reset, so consumers aren't surprised.

### 2.3 Hidden large context injection on `antigravity:gemini-3.8-flash`

- Behavior: usage reports ~15.3k input tokens per call (T01 15303, T02 15303, T03 15321, T04 15325) for ≤120-token prompts — a fixed upstream-prefixed context multiply.
- Impact: real input-token cost ~100× the logical request size; distorts all cost-per-call math for callers.
- Adjustments:
  1. Identify the injected context source (Antigravity helper prompt template) and shorten or opt-out;
  2. If injection cannot be removed, at minimum surface `usage.input_tokens` breakdown to callers and document the overhead in routing metadata;
  3. Watch T05 for the same overhead (usage absent in some SSE summary events).

### 2.4 Latency profile outliers

- `composer-2.5`: 2.98–11.13 s (3–6× slower than peers; T04 11.13 s). Usable, but plan capacity for interactive flows; possible causes: cursor proxy hop + upstream generation speed.
- `antigravity:gemini-3.8-flash`: 5.55–8.35 s even for trivial outputs — consistent with large injected prompt size (§2.3). Fix likely reduces latency too.
- Others: 0.9–3.1 s — healthy.

### 2.5 Catalog freshness hygiene (observation)

- All ten models: `presence: present`, `freshness: fresh`, but catalog `listing_state: stale` — snapshot is older than provider polls. Serving was not affected.
- Adjustment: schedule an authorized `POST /v1/catalog/sync` (state change; requires its own go-ahead), then re-check listing freshness.

### 2.6 Runner/infra operational note

- The gateway background process bound to this session's shell was reaped when the session shell was discarded → port-free afterward.
- Adjustment: run long-lived gateway processes under a supervisor (systemd unit / `nohup` with log file / Docker) rather than as a child of an agent session; per-run startup takes ~1 s, so ad-hoc start is also fine for testers.

## 3. Not executed / pending

| Item | State | Action needed |
|---|---|---|
| Re-test blocked antigravity models (functional verdict) | Pending quota reset (~54h) | Re-run battery with runner; same plan |
| `POST /v1/catalog/sync` | Not executed (state change) | Prior authorization |
| Auto AI-Hub routing validation | Out of scope of this SPEC | Separate initiative |

## 4. Quick failure math

- 50 planned → 50 executed → 35 PASS-equivalent calls from 7 reachable models; 15 calls returned upstream quota text (BLOCKED, not FAIL).
- 3 real FAILs total, all in T04 (strict JSON with fences) — 0 capability failures detected among reachable models.
- Absolute pass rate over reachable models: **32/35 correct (91.4%)**; pure capability defects: 0 (the 3 misses are spec strictness only). Correction note (Phase 0 audit of the remediation plan): an earlier version of this section stated 29/35 (82.9%) and "35 PASS-equivalent calls", which double-counted the verdict table; recomputation from execution records and the per-test matrix gives 4+4+5+4+5+5+5 = 32/35. 15 additional calls were quota-blocked, not capability failures.
- Token totals achieved within cap (128/256): no truncation detected.
- Worst-case latency: 11.13 s (composer-2.5 T04).

## 5. Adjustments checklist (proposed, by owner)

| # | Adjustment | Area | Priority |
|---|---|---|---|
| A1 | Surface upstream quota/subscription notices as structured errors (429) in gateway telemetry | ai-hub upstream integration | High |
| A2 | Re-run battery on `antigravity:*` after quota reset to obtain functional verdicts | Testing (automation) | High |
| A3 | Reduce/opt out the ~15.3k-token injected context on `antigravity:gemini-3.8-flash` calls | antigravity helper/provider | High |
| A4 | Decide strictness for T04: JSON-schema mode vs. fence-stripping vs. spec relaxation | Consumer contract | Medium |
| A5 | Authorize + run catalog sync to clear stale listing states | ai-hub catalog | Medium |
| A6 | Document composer-2.5 latency profile / plan capacity | Ops/planning | Low |
| A7 | Long-lived gateway supervision (systemd unit/`nohup` instead of agent-shell child) | Ops/planning | Low |

No code/config was modified during this battery; all adjustments above are proposals pending approval.
