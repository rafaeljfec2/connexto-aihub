# LLM Multi-Model Smoke Test — Preliminary Plan (Phases 0 and 1)

- Date: 2026-11-09
- Executor agent: OpenCode (GLM 5.3 Flash Free), acting as the smoke-test executor on behalf of the plan's "Codex" actuation model
- Status: plan only. **No model has been called yet.** All test results below are `PENDING`.
- Sanitization: no token, key, cookie or credential is quoted in this document. Only environment-variable *names* and file presence are referenced.

## 1. Objective and scope

Standardized smoke-test battery, one model at a time, to determine which LLMs available in the current development setup are individually reachable and usable. Execution will happen through the OpenAI-compatible endpoint configured for the Codex CLI (`model_provider = "aihub"` → Connexto AI Hub), so every call goes directly to the explicitly selected model, never through `Auto AI-Hub` and never via automatic fallback.

Out of scope, reported as `NOT IMPLEMENTED / OUT OF SCOPE`: Auto AI-Hub routing, AI-Hub automatic routing, unimplemented AI-Hub integrations, automatic cross-provider fallback.

## 2. Phase 0 inventory — findings

### 2.1 Codex configuration (`~/.codex/config.toml`)

| Item | Value |
|---|---|
| Default model | `composer-2.5` (explicit override supported by CLI and executor) |
| Model provider | `aihub` — "Connexto AI Hub" |
| Endpoint / base URL | `http://127.0.0.1:8080/v1` (loopback, this repo's gateway) |
| Wire API | `responses` (OpenAI Responses API, `POST /v1/responses`) |
| Auth mechanism | `env_key = AIHUB_GATEWAY_API_KEY`, sent as `Authorization: Bearer` |
| Model catalog source | `~/.codex/opencode-go-models.json` (local snapshot of the AI-Hub catalog) |
| Reasoning effort | `medium` (configurable per call; rendering levels low → max) |
| Truncation policy (all models) | tokens, limit 10000 |

### 2.2 AI-Hub gateway (this repository)

- README confirms `POST /v1/responses` as a pass-through to the bound provider, `GET /v1/models` (aggregated, or scoped with `X-AIHub-Provider`), plus `/diagnostics/providers`, `/v1/catalog/*`, `/v1/records/executions*`.
- Every route except `/healthz` and `/readyz` requires `Authorization: Bearer $AIHUB_GATEWAY_API_KEY` and an allowed `Host`.
- Streaming: supported. The provider pass-through emits `text/event-stream` (SSE) bodies, and the execution telemetry detects SSE responses.
- Per-call provider pinning is available via `X-AIHub-Provider` on model-list routes; for generation calls the provider is resolved from the requested model ID; explicit model IDs guarantee no auto routing.
- `.env` is present in the repo and defines (names verified, values never read by the executor): `AIHUB_SERVER_*` (port 8080), `AIHUB_GATEWAY_API_KEY` (non-empty), `AIHUB_ADMIN_API_KEY`, `AIHUB_PROVIDER_OPENCODEGO_BASE_URL` + `AIHUB_PROVIDER_OPENCODEGO_API_KEY`, `AIHUB_PROVIDER_CONNEXTO_PROXY_BASE_URL=http://127.0.0.1:9090/v1` + `AIHUB_CONNEXTO_PROXY_GATEWAY_KEY`.
- Process inventory:
  - Gateway on `127.0.0.1:8080`: **not listening** at inventory time (`ss -ltn` has no `:8080` entry) → must be started before execution (needs authorization).
  - Connexto CLI proxy on `:9090`: **listening**; `healthz` answers HTTP 401 (auth-gated, operational).
  - Cursor-antigravity helper hubs (`agy --hub`) running on loopback ports 43817 and 35629.
  - Codex: extension binary `codex 0.164.0-alpha.11.1` running as VS Code app-server; CLI binary available at the extension path (`codex exec` usable from the shell).

### 2.3 Executor tooling

- Codex CLI: `codex-cli 0.164.0-alpha.11.1` (binary under the `openai.chatgpt-26.1021.22027` extension directory).
- For standardized evidence capture (HTTP status, latency, raw body, SSE chunk logging), the battery uses direct `curl` calls to `POST /v1/responses`, driven by the executor agent. `codex exec -m <slug>` is kept as a cross-check command per model (optional).

## 3. Model inventory — confirmed identifiers

All identifiers below were read from `~/.codex/opencode-go-models.json` (`models[].slug` / `models[].display_name`), the catalog snapshot consumed by the Codex configuration. No identifier was invented. Provider attribution comes from each catalog entry's own description.

| IDE display name | Confirmed slug | Provider (per catalog description) | Catalog state |
|---|---|---|---|
| Claude Haiku 4.5 | `anthropic/claude-haiku-4.5` | Anthropic via OpenRouter routing (AI Hub catalog) | listed (`visibility: list`), `supported_in_api: true` |
| GLM 5.3 Flash | `glm-5.3-flash` | Z.ai GLM, served via AI Hub (OpenCode Go) | listed, `supported_in_api: true` |
| DeepSeek V4.1 Flash | `deepseek-v4.1-flash` | DeepSeek, served via AI Hub (OpenCode Go) | listed, `supported_in_api: true` |
| Claude Haiku 5.5 | `claude-haiku-5-5` | Anthropic Claude Haiku 5.5, served via AI Hub (OpenCode Go) | listed, `supported_in_api: true` |
| 6 Luna | `gpt-6-luna` | GPT-6-Luna (Codex/OpenCode Go catalog entry) | listed, `supported_in_api: true` |
| Composer 2.5 | `composer-2.5` | Cursor Composer 2.5 via AI Hub (Connexto proxy) | listed, `supported_in_api: true` |
| Claude Opus 4.6 (Thinking) | `antigravity:claude-opus-4-6-thinking` | Antigravity upstream | listed, `supported_in_api: true` |
| Claude Sonnet 4.6 | `antigravity:claude-sonnet-4-6` | Antigravity upstream | listed, `supported_in_api: true` |
| Gemini 3.8 Flash | `antigravity:gemini-3.8-flash` | Antigravity upstream | listed, `supported_in_api: true` |
| OSS 120B (Medium) | `antigravity:gpt-oss-120b-medium` | Antigravity upstream | listed, `supported_in_api: true` |

Additional catalog entry observed but **out of scope**: `openrouter/auto` ("Auto AI-Hub" router) — never exercised in this battery.

Catalog metadata common to the in-scope models (relevant to limits):

- Context window: 272k tokens nominal, 872k max, 95% effective (exception: `anthropic/claude-haiku-4.5` — 128k nominal / 200k max).
- Default reasoning level: `medium`; supported levels: low, medium, high, xhigh, max.
- Input modalities (catalog): text, image. Verbosity supported (default `low`).
- Tool mode: `code_mode_only` — irrelevant here, external tools are disabled by the battery design.

## 4. Endpoint and call method

- Endpoint: `POST http://127.0.0.1:8080/v1/responses` (Connexto AI Hub gateway, loopback).
- Headers: `Authorization: Bearer $AIHUB_GATEWAY_API_KEY` (expanded at execution time from the repository `.env`; never echoed), `Content-Type: application/json`.
- Body (non-streaming): `{"model": "<slug>", "input": "<prompt>", "stream": false, "max_output_tokens": 128}`.
- Body (streaming, TEST-05): same body with `"stream": true`, consumed via `curl -N`, capturing `Content-Type: text/event-stream` and all SSE events to a raw log file.
- Independent per call: fresh request, no conversation history, no tools, no attachments, no retries ("--max-time" and `-sS` only; a failed call is recorded, not retried).
- Evidence of which model actually answered: the gateway returns the requested model id in the response, and the gateway's execution records (`GET /v1/records/executions`, `provider_id`, `model_id`) are checked afterwards to detect unauthorized provider substitution or hidden fallback.

## 5. Execution configuration (Phase 2 standardization)

Applied identically to all 10 models:

- Same infrastructure: local gateway, local `.env`, same shell, same working directory.
- Same endpoint and same auth mechanism.
- Same prompt set (verbatim, below).
- Small output cap: `max_output_tokens: 128` for TEST-01/02/04, 256 for TEST-03, streaming TEST-05 128.
- Reasoning effort: provider default (`medium`); no custom effort overrides.
- No external tools, no code execution, no attachments, no prior context.
- No automatic retries; no fallback to another model or provider.
- One model at a time, sequential battery.
- Timeouts: `--max-time 45` (non-stream), `--max-time 60` (stream).
- A temperature/top-p are not sent (API defaults) to keep the contract minimal.

## 6. Planned tests (per model, identical set)

### TEST-01 — Connectivity
- Prompt: `Responda somente com: CONNECTION_OK`
- Pass when: HTTP 2xx; non-empty text; no auth error (401/403); no routing error; requested model id echoed in the response; exactly the requested string (small numeric noise tolerated by grapheme; strictness defined as: contains `CONNECTION_OK`).

### TEST-02 — Basic generation
- Prompt: `Explique em até três frases o que é uma goroutine em Go.`
- Pass when: non-empty coherent text; ≤ 3 sentences (relative leniency only if truncation was quantified); no serialization error; output cap respected.

### TEST-03 — Reasoning and instruction following
- Prompt: `Considere que uma aplicação possui três workers. Cada worker processa quatro tarefas. Quantas tarefas são processadas no total? Responda com o cálculo e o resultado.`
- Expected: 12, with the calculation shown.
- Pass when: correct result; requested format; no irrelevant extra content.
- Limitation: a single spurious result triggers `FAIL` for the specific test, but is evaluated again in the results summary as one sample (never used to declare a model incapable of reasoning).

### TEST-04 — Structured output
- Prompt: `Gere exclusivamente um JSON válido com os campos name, language e type. Use os valores Connexto AI-Hub, Go e API.`
- Pass when: response parses as JSON; fields present; values exactly `Connexto AI-Hub`, `Go`, `API`; no Markdown fences or prose around the JSON (leading/trailing whitespace tolerated; anything else → `FAIL`).

### TEST-05 — Streaming
- Method: same battery prompt as TEST-01, `stream: true`.
- Pass when: first SSE chunk arrives before generation completes; chunks interleave correctly; final text is reconstructible from deltas; stream termination is clean (proper `[DONE]`/event close or HTTP close without error frames).
- If the endpoint or model does not support streaming: `UNSUPPORTED`, never `FAIL`.
- Catalog evidence pre-classifies streaming as supported (SSE pass-through in the repo), but runtime `UNSUPPORTED` remains possible per model.

## 7. Commands the executor will use

Setup only (no secrets printed; the key lives in an in-memory shell variable sourced from the repo `.env`):

```bash
cd /home/rafael/dev-rafael/connexto-aihub
set -a; source .env; set +a   # provides AIHUB_GATEWAY_API_KEY; never echoed
mkdir -p reports/llm-smoke/raw
```

Pre-flight (read-only, no model calls): `curl -fsS http://127.0.0.1:8080/healthz` and `GET /v1/models` to confirm the 10 slugs are served.

Per model `<MODEL>` (one shell function loops over the battery; each call is independent):

```bash
# TEST-01 / TEST-02 / TEST-03 / TEST-04 (stream=false)
curl -sS --max-time 45 \
  -o "reports/llm-smoke/raw/<MODEL>-<TEST>.json" \
  -w "HTTP=%{http_code} TIME=%{time_total}\n" \
  -X POST http://127.0.0.1:8080/v1/responses \
  -H "Authorization: Bearer ${AIHUB_GATEWAY_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"model":"<MODEL>","input":"<PROMPT>","stream":false,"max_output_tokens":<128|256>}'

# TEST-05 (streaming)
curl -N -sS --max-time 60 \
  -o "reports/llm-smoke/raw/<MODEL>-test05.sse" \
  -w "HTTP=%{http_code} TIME=%{time_total} CT=%{content_type}\n" \
  -X POST http://127.0.0.1:8080/v1/responses \
  -H "Authorization: Bearer ${AIHUB_GATEWAY_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"model":"<MODEL>","input":"Responda somente com: CONNECTION_OK","stream":true,"max_output_tokens":128}'
```

Optional cross-check (one per model, cheap): `codex exec -m <MODEL> "Responda somente com: CONNECTION_OK"` using the already-configured `aihub` provider.

Post-run anti-fallback audit (read-only):

```bash
curl -sS -H "Authorization: Bearer ${AIHUB_GATEWAY_API_KEY}" \
  "http://127.0.0.1:8080/v1/records/executions?limit=100"
```

## 8. Known limitations

- The gateway is **not currently running**; execution requires starting it (`pnpm dev` / `go run ./cmd/ai-hub` in this repo), which is a runtime action requiring user authorization (it is not a config change; no file gets edited).
- Provider attribution (section 3) is evidence from catalog descriptions, not a runtime proof; real provider ids will be confirmed on `GET /v1/models` and `X-AIHub-Provider`-scoped listings during Phase 4 preamble.
- JSON strictness (TEST-04) depends on each model's instruction following; scoring is deterministic (parse + exact fields/values).
- Single reasoning sample (TEST-03) is a smoke signal, not a reasoning benchmark.
- `max_output_tokens` is honored by Responses-API semantics; where a model ignores the cap, TEST-02/04 record the overflow as a `FAIL` detail rather than silent acceptance.
- Latency includes gateway + upstream network (proxy `:9090` adds a local hop for Cursor-routed models; Antigravity slugs depend on the running `agy --hub` helpers).
- Codex CLI default model would send `composer-2.5` — irrelevant because every battery call sets `-m`/body `model` explicitly.

## 9. Estimated consumption per test

- Input: ≤ 120 tokens per call (prompt + system default).
- Output: capped at 128–256 tokens (`max_output_tokens`).
- Per test ≈ 0.25–0.4k tokens; full battery (10 models × 5 tests) ≈ 13–20k tokens total. Cost per model cannot be converted to currency without provider pricing; token counts are available from responses/usage fields.
- A second run of a failed test (re-test after investigation) may add up to ~4k tokens, still far below any heavy-usage threshold. No load testing.

## 10. Approval criteria (per model)

- `PASS` for connectivity, generation and structured output is mandatory.
- Reasoning: `PASS` expected from 9 of 10 samples (single-sample basis, noted above).
- Streaming: `PASS` or `UNSUPPORTED` (registry of catalog intent says all should stream).
- Overall model verdict:
  - `USABLE`: all five tests `PASS` (or streaming `UNSUPPORTED` with documented reason).
  - `PARTIALLY USABLE`: connectivity `PASS`, one or more of TEST-02/03/04 `FAIL`.
  - `NOT USABLE`: connectivity `FAIL` (unserved slug, auth or route error).
  - `BLOCKED`: cannot run for external dependency (gateway down, upstream helper missing).
- Latencies are recorded per test; no latency-based disqualification (informational).
- Unauthorized fallback = any response whose id/provider deviates from the requested slug → recorded and repeated once noun-fallback (verification against `/v1/records/executions`; if the deviation repeats without fallback evidence, the record is judged `FAIL`).

## 11. Models unavailable or with unconfirmed identifiers

- None. All 10 selector names matched catalog entries 1:1 (`display_name` ↔ IDE label), including `6 Luna` → `gpt-6-luna` and `OSS 120B (Medium)` → `antigravity:gpt-oss-120b-medium`.
- Runtime availability remains unverified — that is exactly what the battery measures.

## 12. Explicitly out of scope

- `openrouter/auto` ("Auto AI-Hub") — never called, never scored.
- AI-Hub automatic routing / fallback between providers.
- Any AI-Hub integration not yet implemented.
- Load/performance testing; large-context probing; tool/function calling; multimodal inputs (even though catalog advertises image input); retries-in-passing ("PASS by retry" is forbidden and logged as violation if detected).

## 13. Consolidated table (Phase 3 pre-execution state)

| Model | Confirmed ID | Connectivity | Generation | Reasoning | JSON | Streaming |
|---|---|---|---|---|---|---|
| Claude Haiku 4.5 | `anthropic/claude-haiku-4.5` | PENDING | PENDING | PENDING | PENDING | PENDING |
| GLM 5.3 Flash | `glm-5.3-flash` | PENDING | PENDING | PENDING | PENDING | PENDING |
| DeepSeek V4.1 Flash | `deepseek-v4.1-flash` | PENDING | PENDING | PENDING | PENDING | PENDING |
| Claude Haiku 5.5 | `claude-haiku-5-5` | PENDING | PENDING | PENDING | PENDING | PENDING |
| 6 Luna | `gpt-6-luna` | PENDING | PENDING | PENDING | PENDING | PENDING |
| Composer 2.5 | `composer-2.5` | PENDING | PENDING | PENDING | PENDING | PENDING |
| Claude Opus 4.6 (Thinking) | `antigravity:claude-opus-4-6-thinking` | PENDING | PENDING | PENDING | PENDING | PENDING |
| Claude Sonnet 4.6 | `antigravity:claude-sonnet-4-6` | PENDING | PENDING | PENDING | PENDING | PENDING |
| Gemini 3.8 Flash | `antigravity:gemini-3.8-flash` | PENDING | PENDING | PENDING | PENDING | PENDING |
| OSS 120B (Medium) | `antigravity:gpt-oss-120b-medium` | PENDING | PENDING | PENDING | PENDING | PENDING |
| Auto AI-Hub | `openrouter/auto` | OUT OF SCOPE | OUT OF SCOPE | OUT OF SCOPE | OUT OF SCOPE | OUT OF SCOPE |

## 14. Pending configuration items before Phase 4 execution

1. **Gateway not running** — start the AI-Hub gateway (`127.0.0.1:8080`) with an explicit user authorization (`pnpm dev` or `go run ./cmd/ai-hub`; `.env` already satisfies the mandatory gateway key, verified non-empty without reading the value).
2. **Go toolchain on PATH** — `/usr/local/go/bin/go` exists but is not on this shell's `PATH`; the launcher will call it by absolute path or via `pnpm` scripts (no env files will be modified).
3. **Post-catalog verification** — `GET /v1/models` and, if needed, `POST /v1/catalog/sync` may be required to bind provider catalogs if slugs are stale (catalog sync is a state change inside the gateway and will also be requested from the user before it runs).
4. **Antigravity helpers** — `agy --hub` processes are running on this host; verified present, so no action is expected unless execution reports `BLOCKED`.

## 15. Confirmation

- Phases executed so far: 0 (read-only inventory) and 1 (test design) producing this plan.
- **No real model call was made.** No inference request was sent to any provider, upstream, or gateway (gateway also not listening, so it is technically impossible within this session's shell).
- No configuration files (Codex, OpenCode, gateway `.env`) were modified. No code changes. No commits, pushes, or deploys.
- Await authorization to start the gateway (and, if required, catalog sync) and then run the Phase 4 battery.
