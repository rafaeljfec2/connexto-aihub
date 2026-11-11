# LLM Homologation — Gate 0 JSON Schema Post-Fix Retest

- **Date:** 2026-11-10 (UTC window: `2026-11-11T01:31:18Z`–`01:31:39Z`)
- **Repositories:**
  - `/home/rafael/dev-rafael/connexto-proxy-cli-llms` (proxy remediation)
  - `/home/rafael/dev-rafael/connexto-aihub` (`main` @ `5eaad85b019939fa85b0b61bd4682330553b726e`, unmodified)
- **Scope:** Execute Option A from the approved remediation plan (`appendJsonConstraint` hardening + non-streaming strict structured-output validation/unwrapping in `connexto-proxy-cli-llms`), then re-verify `composer-2.5`, `deepseek-v4.1-flash`, and `antigravity:gemini-3.8-flash` (regression) via `127.0.0.1:8080/v1/responses`.

---

## 1. Proxy Changes Applied (`connexto-proxy-cli-llms`)

1. **`packages/compatibility/openai/src/protocol/transcripts/helpers.ts` (`appendJsonConstraint`)**
   - Updated `json_object` and `json_schema` prompt constraints to explicitly forbid Markdown code fences (` ```json ` / ` ``` `) and surrounding prose, and reinforced strict field/property compliance when `strict: true`.
2. **`packages/compatibility/openai/src/protocol/structured-output.ts` (new, 166 lines)**
   - Added `unwrapMarkdownJsonFence` (unwraps only a single outer Markdown fenced block) and `enforceStructuredOutputFormat` (parses JSON and validates `strict: true` schemas recursively, throwing `HttpError(502, "upstream_invalid_structured_output")` on invalid JSON or schema violations).
3. **`apps/api/src/routes/responses/create.ts`**
   - Applied `enforceStructuredOutputFormat` on the non-streaming path when `runOutput.toolCalls.length === 0`.
4. **`packages/compatibility/openai/tests/protocol/structured-output.test.ts` (new, 108 lines)**
   - Added 5 unit tests verifying fence prohibition in prompts, fence unwrapping, strict schema validation, and error rejection on invalid/incomplete outputs.

### Automated Verification (`connexto-proxy-cli-llms`)
- `pnpm -r typecheck`: **PASS** (all 7 workspace packages clean; pre-existing unrelated test-only TS error in `packages/providers/antigravity/tests/provider.test.ts:405` untouched).
- `pnpm vitest run packages/compatibility/openai/tests/ tests/api.integration.test.ts`: **PASS** (9 test files, **110/110 tests passed**, including live Cursor API integration).
- `pnpm -r run build`: **PASS**.

---

## 2. Live Gateway Retest Results (`POST /v1/responses`)

Contract: `text.format = {type: "json_schema", name: "connexto_meta", strict: true, schema: {name, language, type, additionalProperties: false}}`, `stream: false`, `max_output_tokens: 256`, sequential, no retries, no fallback.

| Model | Provider | HTTP | Latency | Output | Body `usage` | Classification |
|---|---|---|---|---|---|---|
| `composer-2.5` | `cursor` | 200 | 4.231 s | `{"name":"Connexto AI-Hub","language":"Go","type":"API"}` | 171 in / 14 out (185 total), est. `$0.0001205` | **PASS** |
| `deepseek-v4.1-flash` | `opencode-go` | 200 | 1.984 s | `{"name":"Connexto AI-Hub","language":"Go","type":"API"}` | 61 in / 85 out (65 reasoning, 146 total), `go-plus` `0.00014436` | **PASS** |
| `antigravity:gemini-3.8-flash` | `antigravity` | 200 | 9.140 s | `{\n  "name": "Connexto AI-Hub",\n  "language": "Go",\n  "type": "API"\n}` | 15,373 in / 466 out (434 reasoning, 15,839 total), `cost: null` | **PASS** |

---

## 3. Execution Records Audit (`GET /v1/records/executions?limit=20`)

Window `>= 2026-11-11T01:31:18Z` returned **3 records** (1:1 match):

| Model | `execution_id` | `provider_id` | `outcome` | `upstream_status` | `duration_ns` | Record `usage` |
|---|---|---|---|---|---|---|
| `composer-2.5` | `518b04aa5daa384137c5dcc2e281a9d8` | `cursor` | `completed` | 200 | 4,186,613,555 (4.19 s) | `null` |
| `deepseek-v4.1-flash` | `6bfc41380898060dfedf9e50d66fc38e` | `opencode-go` | `completed` | 200 | 1,956,793,112 (1.96 s) | `null` |
| `antigravity:gemini-3.8-flash` | `8d22896e4955bd927da0c003c3728d45` | `antigravity` | `completed` | 200 | 9,124,986,820 (9.12 s) | `null` |

Zero retries, zero fallbacks, zero provider mismatches.

---

## 4. Updated Gate 0 Consolidation (7/10 Homologated)

- **Homologated with `json_schema` (`strict: true`) proven (7/10):**
  `anthropic/claude-haiku-4.5`, `glm-5.3-flash`, `claude-haiku-5-5`, `gpt-6-luna`, `antigravity:gemini-3.8-flash`, `composer-2.5` (non-stream guaranteed via proxy validation), `deepseek-v4.1-flash`.
- **Blocked by Antigravity Quota (3/10):**
  `antigravity:claude-opus-4-6-thinking`, `antigravity:claude-sonnet-4-6`, `antigravity:gpt-oss-120b-medium` (pending retest after `2026-11-12T00:46Z`).
- **Remaining Pending Items:**
  - Fix `usage: null` telemetry gap in `/v1/records/executions`.
  - Keep `aihub/auto` disabled in production until the 3 Antigravity models and telemetry gap are closed.

