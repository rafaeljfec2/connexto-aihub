# LLM Homologation — Gate 0 JSON Schema Retest (Post-Catalog Sync)

- **Date:** 2026-11-10 (UTC execution window: `2026-11-11T00:10:28Z`–`00:11:36Z`)
- **Repository:** `/home/rafael/dev-rafael/connexto-aihub`
- **Branch / HEAD:** `main` @ `5eaad85b019939fa85b0b61bd4682330553b726e`
- **Scope:** Expressly authorized `POST /v1/catalog/sync`, post-sync validation, and sequential `json_schema` (`strict: true`) retest of the four previously blocked conditional models. `aihub/auto` production traffic remains **NOT authorized**.

---

## 1. Environment State (Before vs. After)

| Check | Before Sync | After Sync |
|---|---|---|
| Gateway (`127.0.0.1:8080`) | UP (`PID 97405`), `/healthz` 200, `/readyz` 200 (`database: ok`) | UP, `/healthz` 200, `/readyz` 200 (`database: ok`) |
| Catalog `freshness` | `stale` across all 4 providers (`last_success_at`: `2026-11-09T00:14:14-03:00`) | **`fresh`** across all 4 providers (`2026-11-11T00:10:28Z`–`00:10:30Z`) |
| Target models in `/v1/catalog/models` | `presence: present`, `freshness: stale` | `presence: present`, `freshness: fresh` |
| Working tree | Clean on tracked files; prior evidence preserved | Prior evidence untouched; new artifacts in `retest-2026-11-10/` |

---

## 2. Authorized Synchronization Result (`POST /v1/catalog/sync`)

- **Executed once:** `2026-11-11T00:10:28Z` (`Content-Type: application/json`, body `{}` with Bearer auth)
- **HTTP Status / Duration:** `200 OK` in `1.66s`
- **Partial Failures / Warnings:** None (`outcome: success`, `error_class: none`, `upstream_status: 200` for every provider)

| Provider | Outcome | Discovered | Added | Updated | Unchanged | Missing | Duration | New `last_success_at` | Freshness |
|---|---|---|---|---|---|---|---|---|---|
| `antigravity` | `success` | 4 | 0 | 0 | 4 | 0 | 181 ms | `2026-11-11T00:10:28.971Z` | `fresh` |
| `cursor` | `success` | 6 | 0 | 0 | 6 | 0 | 94 ms | `2026-11-11T00:10:29.152Z` | `fresh` |
| `opencode-go` | `success` | 30 | 1 | 1 | 28 | 1 | 483 ms | `2026-11-11T00:10:29.575Z` | `fresh` |
| `openrouter` | `success` | 433 | 12 | 19 | 402 | 4 | 900 ms | `2026-11-11T00:10:30.501Z` | `fresh` |

All four target models (`deepseek-v4.1-flash`, `gpt-6-luna`, `composer-2.5`, `antigravity:gemini-3.8-flash`) were verified `presence: present` and `freshness: fresh` before testing.

---

## 3. Test Matrix & Individual Classification

Contract: `POST /v1/responses`, `stream: false`, `max_output_tokens: 256`, `text.format: {type: "json_schema", name: "connexto_meta", strict: true, schema: {name, language, type, additionalProperties: false}}`, sequential, no retries, no fallback.

| Model | Expected Provider | HTTP | Status | Latency | Output Contract Check | Classification |
|---|---|---|---|---|---|---|
| `deepseek-v4.1-flash` | `opencode-go` | 429 | error (`server_error`) | 0.290 s | Not executed at inference (upstream overload) | **BLOCKED** |
| `gpt-6-luna` | `opencode-go` | 200 | `completed` | 1.792 s | Valid JSON, no fences, exact keys & values | **PASS** |
| `composer-2.5` | `cursor` | 200 | `completed` | 9.001 s | Wrapped in ```` ```json ... ``` ```` fences | **FAIL** |
| `antigravity:gemini-3.8-flash` | `antigravity` | 200 | `completed` | 8.058 s | Valid JSON, no fences, exact keys & values | **PASS** |

---

## 4. Sanitized Response Evidence & Schema Validation

1. **`deepseek-v4.1-flash`** (`reports/llm-homologation/json-contract/retest-2026-11-10/deepseek-v4.1-flash-text_format_json_schema.json`)
   - Body: `{"error":{"type":"server_error","message":"The inference service is overloaded. Please try again later."}}`
   - Validation: N/A — upstream returned HTTP 429 capacity overload. Classified `BLOCKED`, not a `json_schema` capability failure.

2. **`gpt-6-luna`** (`gpt-6-luna-text_format_json_schema.json`)
   - Output text: `{"name":"Connexto AI-Hub","language":"Go","type":"API"}`
   - Validation: E08 admitted -> reached `opencode-go` -> `completed` -> valid JSON -> zero markdown fences -> zero extra text -> all required fields (`name`, `language`, `type`) present as strings -> `additionalProperties: false` honored -> exact prompt values.

3. **`composer-2.5`** (`composer-2.5-text_format_json_schema.json`)
   - Output text: ```` ```json\n{"name":"Connexto AI-Hub","language":"Go","type":"API"}\n``` ````
   - Validation: E08 admitted -> reached `cursor` -> `completed` -> **FAIL** on strict raw JSON criterion: output was enclosed in Markdown code fences (` ```json ... ``` `), violating criterion 8 (no markdown fences or text outside JSON object).

4. **`antigravity:gemini-3.8-flash`** (`antigravity-gemini-3.8-flash-text_format_json_schema.json`)
   - Output text: `{\n  "name": "Connexto AI-Hub",\n  "language": "Go",\n  "type": "API"\n}`
   - Validation: E08 admitted -> reached `antigravity` -> `completed` -> valid JSON -> zero markdown fences -> zero extra text -> all required fields present as strings -> `additionalProperties: false` honored -> exact prompt values.

---

## 5. Execution Records Audit (`GET /v1/records/executions?limit=20`)

Window `>= 2026-11-11T00:11:10Z` returned **exactly 4 records** (1:1 match with the 4 requests):

| Model | `execution_id` | `provider_id` | `outcome` | `phase` | `upstream_status` | `response_committed` | Record `duration_ns` | Record `usage` |
|---|---|---|---|---|---|---|---|---|
| `deepseek-v4.1-flash` | `b3b15c67f390b7ab9188c16e7615191c` | `opencode-go` | `completed` | `forwarding` | 429 | `true` | 247,992,224 (248 ms) | `null` |
| `gpt-6-luna` | `aba027104c2fe587ecd2beadde02b7f3` | `opencode-go` | `completed` | `forwarding` | 200 | `true` | 1,777,233,700 (1.78 s) | `null` |
| `composer-2.5` | `15a51912162751a4822b6ed1c4a0eeae` | `cursor` | `completed` | `forwarding` | 200 | `true` | 8,972,092,116 (8.97 s) | `null` |
| `antigravity:gemini-3.8-flash` | `5010bc2a6629b60be104c83f0941347f` | `antigravity` | `completed` | `forwarding` | 200 | `true` | 8,042,101,813 (8.04 s) | `null` |

### Retries, Fallbacks & Divergence Check
- **Retries / Fallbacks / Provider Swaps:** **0** across all 4 calls. Every `provider_id` and `model_id` matched the expected route.
- **HTTP 429 vs. `outcome: "completed"` check (`deepseek-v4.1-flash`):** Confirmed consistent with gateway semantics (`internal/execution/manager_test.go:43`): transparent forwarding of an upstream HTTP 429 (`response_committed: true`, `upstream_status: 429`) completes the forwarding phase with `outcome: completed`. No divergence.

---

## 6. Latency & Reported Usage

| Model | Gateway Latency | Record Duration | Response Body `usage` | Record `usage` |
|---|---|---|---|---|
| `deepseek-v4.1-flash` | 0.290 s | 248 ms | Not present (HTTP 429 error) | `null` (absent) |
| `gpt-6-luna` | 1.792 s | 1,777 ms | 73 in / 43 out (13 reasoning), 116 total; `cost: 0`, `go-plus` allowance `0.0001152` | `null` (telemetry gap) |
| `composer-2.5` | 9.001 s | 8,972 ms | 171 in / 17 out, 188 total; estimated cost `$0.000128` | `null` (telemetry gap) |
| `antigravity:gemini-3.8-flash` | 8.058 s | 8,042 ms | 15,375 in / 290 out (258 reasoning), 15,665 total; `cost: null` | `null` (telemetry gap) |

`usage: null` in execution records remains an open gateway telemetry gap (absent data, not zero consumption). `antigravity:gemini-3.8-flash` continues to incur ~15.3k input token overhead from the external `agy` runtime ACP session.

---

## 7. Gate 0 Progression & Remaining Pending Items

- **Gate 0 Status:** **Not globally approved.**
- **Structured Outputs (`json_schema`, `strict: true`) Homologated (5/10 models):**
  - Previously approved (2026-11-09): `anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5`
  - Newly approved (2026-11-10): `gpt-6-luna`, `antigravity:gemini-3.8-flash` (with documented ~15.3k input token overhead)
- **Non-Approved / Pending (5/10 models):**
  1. `composer-2.5`: **FAIL** on strict `json_schema` (emitted markdown fences); must be excluded from strict structured-output routes unless adapter/upstream compliance is remediated.
  2. `deepseek-v4.1-flash`: **BLOCKED** (transient HTTP 429 upstream overload); requires single retest when `opencode-go` capacity clears.
  3. `antigravity:claude-opus-4-6-thinking`, `antigravity:claude-sonnet-4-6`, `antigravity:gpt-oss-120b-medium`: Still **BLOCKED** by Antigravity quota (full 5-test smoke + `json_schema` pending after `2026-11-12T00:46Z`).
- **Systemic Pending Items:**
  - Populate `usage` in `/v1/records/executions` (currently `null` across all records).
  - Keep `aihub/auto` disabled in production until Gate 0 criteria and telemetry gaps are resolved.

