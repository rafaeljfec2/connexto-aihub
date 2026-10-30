# OpenCode Go Protocol Spike

**Date of verification:** 2026-10-30
**Status:** completed (research only; no production code)
**Scope:** validate whether Codex → AI Hub → OpenCode Go → DeepSeek can preserve
OpenAI Responses semantics before E02 is designed.

## 1. Method

- Official documentation consulted at `opencode.ai/v2/docs` (the legacy
  `/docs/zen` path now redirects to the Console documentation).
- Live requests executed against `https://opencode.ai/inference/go` with an
  existing Console API key found in the environment.
- Codex client behavior inspected in the public `openai/codex` repository
  (`main` branch, fetched 2026-10-30). Branch state is not pinned to a commit,
  so line numbers may drift.
- No credential, token fragment or full request header was written to this
  document, to logs or to the repository.

### Evidence classification

| Label | Meaning |
| --- | --- |
| **Documented** | Stated in current official OpenCode documentation |
| **Verified** | Reproduced with a real request/response pair |
| **Inferred** | Derived from source code or documented semantics, not directly executed |
| **Unknown** | Not documented and not safely reproducible |

## 2. Findings

### 2.1 Endpoint and base URLs — Documented + Verified

| Plan | Base URL | Status |
| --- | --- | --- |
| Pay-as-you-go | `https://opencode.ai/inference` | Documented |
| Go | `https://opencode.ai/inference/go` | Documented + Verified (`POST .../go/openai/v1/responses` → 200) |

API paths appended to the base:

| API | Path |
| --- | --- |
| OpenAI Responses | `/openai/v1/responses` |
| OpenAI Chat Completions | `/openai/v1/chat/completions` |
| Anthropic Messages | `/anthropic/v1/messages` |
| Gemini | `/google/v1beta/models/<model>:generateContent` |

Model discovery does **not** follow the same prefix: the OpenAI-compatible path
`/go/openai/v1/models` returns `404`, while `/go/v1/models` returns `200`.
The global `/inference/v1/models` also returns `200`. Both lists are readable
without a credential.

### 2.2 Authentication — Documented + Verified

- `Authorization: Bearer <key>`, where keys are Console personal or service
  account keys with the `oc_sk_` prefix.
- Invalid credential → `401` with `{"error":{"type":"server_error","message":"Upstream request failed: Invalid credential"}}`.
- The key may carry scoped permissions (`Models: Read` versus `Models: Request`);
  a read-only key can list models but not call them (Documented, not tested).

### 2.3 Required session header — Documented + Verified

Go requires a stable conversation identifier:

```
x-opencode-session: <stable-id>
```

Omitting it returns `400`:

```json
{"type":"error","error":{"type":"MissingSessionID","message":"Request is missing x-opencode-session and cannot be routed efficiently. ..."}}
```

An arbitrary stable string worked; no `ses_` prefix validation was observed.
Console also recognizes native session headers from Codex, Claude Code, Hermes
and Pi. The header is required for Go but not for pay-as-you-go.

### 2.4 Protocols and translation — Documented + Verified

Documented rule: *"Every API serves every paid and Go text model: Console sends
a request to the model's own endpoint when it speaks that API, and translates it
otherwise."*

Verified with `deepseek-v4.1-flash`, whose native API is Chat Completions
(`api.chat-completions` in the model metadata):

- `POST /go/openai/v1/responses` → `200`, full Responses object.
- `POST /go/openai/v1/chat/completions` → `200`, standard Chat Completions
  object.

So the Responses translation is performed server-side by Console/Go, not by the
client. AI Hub does not need to implement a Responses↔Chat translation for this
provider.

### 2.5 Model discovery and identifiers — Documented + Verified

- Pay-as-you-go catalog: 37 models. Go catalog: 28 models.
- Go entries include `id`, `canonical_id`, `name`, `type`, `family`,
  `modalities`, `capabilities`, `limit`, `cost` and `api`.
- DeepSeek availability in Go:

| Model ID | Native API | Context | Input modalities | Notes |
| --- | --- | --- | --- | --- |
| `deepseek-v4.1-flash` | Chat Completions | 1,000,000 | text + images | cheapest viable first model |
| `deepseek-v4-pro` | Chat Completions | 1,000,000 | text | reasoning-heavy |
| `deepseek-v4.1-flash-free` | Chat Completions | — | — | free tier reserved for OpenCode clients |

`deepseek-v4.1-flash` metadata declares: tools supported, reasoning supported
(effort `low`/`high`/`max`), structured output, implicit cache, temperature,
top_p and top_k.

### 2.6 Responses API support — Verified

A real `POST /go/openai/v1/responses` returned a complete Responses object with
`output` items of type `reasoning` and `message`, `usage` and `status:
"completed"`. The response `id` is a `resp_...` identifier.

### 2.7 Chat Completions support — Verified

A real request returned `object: "chat.completion"`, standard `choices` with
`finish_reason`, and `usage` including `prompt_tokens`, `completion_tokens` and
`reasoning_tokens`.

### 2.8 Streaming — Verified

`stream: true` returned SSE with the Responses event family:

```
response.created, response.in_progress,
response.output_item.added/done,
response.reasoning_summary_part.added/done,
response.reasoning_summary_text.delta/done,
response.content_part.added/done,
response.output_text.delta (x4), response.output_text.done,
response.completed
```

`response.completed` carries the final `usage` block, including `cost` and the
`opencode` billing metadata.

### 2.9 Tool calling — Verified

- Function tool: returned an `output` item of type `function_call` with
  `name`, `call_id` and JSON-string `arguments`.
- Round trip: sending `function_call` plus `function_call_output` produced a
  final assistant message. The agent loop works.
- Freeform/custom tool: `{"type":"custom","name":"apply_patch","format":{...}}`
  was accepted and produced a `custom_tool_call` item with a string `input`.
  This matters because Codex uses `type: "custom"` for freeform tools such as
  `apply_patch`.

Malformed JSON produced a `400` with an empty body; that is a client-side
failure, not a capability gap (retested with valid JSON → `200`).

### 2.10 Reasoning and usage metadata — Verified

- Reasoning items appear with `summary` and, when
  `include: ["reasoning.encrypted_content"]` is sent, with a non-null
  `encrypted_content`.
- Replaying a reasoning item with `encrypted_content` in the next request's
  `input` was accepted (`200`), so stateless multi-turn replay works.
- Usage includes OpenAI-standard counters plus vendor metadata:

```json
"usage": {
  "input_tokens": 36,
  "output_tokens": 19,
  "total_tokens": 55,
  "cost": 0,
  "opencode": {
    "billing_source": "go",
    "list_cost": 0.0000168,
    "allowance": 0.0000126,
    "tokens": {"input": 36, "output": 19, "reasoning": 15, "cache_read": 0}
  }
}
```

`cost` was `0` because the request was covered by the Go plan allowance;
`allowance` reports what was consumed from the monthly limit.

### 2.11 Errors and rate limits — Partially Verified

| Condition | Status | Envelope |
| --- | --- | --- |
| Missing session header | 400 | `{"type":"error","error":{"type":"MissingSessionID",...}}` |
| Unknown model | 400 | `{"error":{"type":"server_error","message":"Upstream request failed: Model is unavailable."}}` |
| Invalid credential | 401 | same shape, `"Invalid credential"` |
| Free model from non-OpenCode client | 400 | `"Model is unavailable."` |

Observed discrepancy: the documentation states other clients receive `403` for
free-tier models; the live response was `400`. Error mapping is therefore
generic and the documented status code is not authoritative.

Rate limits: documented only as monthly dollar allowances with 5-hour, weekly
and monthly windows. No `429` body, `Retry-After` behavior or rate-limit headers
were documented or safely reproducible. **Unknown.**

### 2.12 `previous_response_id` — Verified limitation

| Model | `previous_response_id` | Result |
| --- | --- | --- |
| `gpt-6-luna` (native Responses) | supported | `200` |
| `deepseek-v4.1-flash` (translated) | rejected | `400` |

Error:

```json
{"error":{"type":"invalid_request_error","message":"previous_response_id: requires a native Responses target; portable routing requires the full conversation in input"}}
```

Consequence: for translated models, the full conversation must be resent in
`input`. Any client relying on server-side response chaining cannot use
DeepSeek through Go.

## 3. Codex compatibility analysis

Codex speaks the Responses API and is configured by OpenCode as follows
(Documented):

```toml
model = "kimi-k3"              # or any Go model ID
model_provider = "opencode-go"

[model_providers.opencode-go]
name = "OpenCode Go"
base_url = "https://opencode.ai/inference/go/openai/v1"
env_key = "OPENCODE_API_KEY"
wire_api = "responses"
```

Codex behavior inspected in `openai/codex` (`codex-rs/core/src/client.rs`):

- The HTTP Responses request is built with `store: false` and `stream: true`,
  sending the full `input` each turn (`client.rs:1157-1166`).
- `previous_response_id` appears only in the WebSocket continuation path
  (`client.rs:2269-2290`), not in the HTTP request struct, whose fields are
  `model`, `input`, `tools`, `tool_choice`, `parallel_tool_calls`, `reasoning`,
  `store`, `stream`, `stream_options`, `include`, `service_tier`,
  `prompt_cache_key`, `text`, `client_metadata` (`client.rs:420-450`).
- Codex always includes `"reasoning.encrypted_content"` (`client.rs:1117`).
- Tools are serialized as `type: "function"` or `type: "custom"` (freeform)
  (`codex-rs/tools/src/tool_spec.rs:23,54`).

A single request reproducing that payload — `instructions`, message-array
`input`, function tool, `tool_choice: "auto"`, `parallel_tool_calls`,
`reasoning.effort`, `store: false`, `stream: true`,
`include: ["reasoning.encrypted_content"]`, `prompt_cache_key`,
`text.verbosity`, `service_tier` and `client_metadata` — returned `200` and the
full SSE event family against `deepseek-v4.1-flash`.

**Conclusion (Inferred, high confidence):** for a custom HTTP provider, Codex
sends full conversation state and does not require `previous_response_id`.
The translated DeepSeek path therefore satisfies Codex's HTTP contract. This
must still be confirmed end-to-end when the gateway exists (E07).

## 4. Capability matrix

| Capability | Codex expects | DeepSeek via Go | Assessment |
| --- | --- | --- | --- |
| Responses request/response | yes | yes | Preserved |
| SSE streaming | yes | yes | Preserved, same event names |
| Function tools | yes | yes | Preserved |
| Freeform/custom tools | yes (`apply_patch`) | yes | Preserved |
| Tool result round trip | yes | yes | Verified |
| Reasoning encrypted items | yes | yes | Preserved and replayable |
| `store: false` + full replay | yes | yes | Preserved |
| `previous_response_id` | only WebSocket path | rejected | Not used on HTTP; must not be introduced |
| Usage/cost metadata | tolerated | extended | Preserved + vendor extension |
| `text.verbosity`, `service_tier`, `client_metadata` | sent | accepted | Ignored or translated upstream |
| Prompt caching | yes | implicit | Preserved as provider-side optimization |
| Model listing | vendor-specific | `/go/v1/models` | Non-OpenAI path; adapter must handle |

## 5. Architecture recommendation

1. **Expose Responses to Codex.** Codex only speaks Responses; exposing Chat
   Completions would require translation and lose semantics.
2. **Call OpenCode Go with Responses.** Console translates to Chat Completions
   for DeepSeek server-side; AI Hub stays a passthrough.
3. **Direct adapter is viable.** No protocol translation is required in AI Hub
   for this first integration. Keep the adapter thin: headers, base URL, model
   validation, session propagation, error pass-through.
4. **Do not build a generic protocol abstraction yet.** The evidence supports
   one adapter shape: Responses in, Responses out.
5. **DeepSeek is viable as the first model**, with two hard requirements:
   always send full conversation state, and never emit `previous_response_id`
   toward a translated model.
6. **Generate a stable `x-opencode-session`** in AI Hub per conversation and
   preserve it across turns, including when Codex does not send one.

## 6. Proposed E02 scope

In scope:

- Provider record for `opencode-go`: base URL, credential reference, enabled
  flag, protocol `responses`.
- Credential via environment (`OPENCODE_API_KEY`), never persisted in the
  repository or logged.
- Model catalog via `GET {base}/v1/models`, mapping `id`, `name`, `canonical_id`,
  `limit`, `modalities`, `capabilities` and `cost`.
- Thin adapter: `POST {base}/openai/v1/responses` with streaming pass-through.
- Session header injection and propagation.
- Connectivity test against the models endpoint plus one minimal generation.
- Error pass-through with the upstream status preserved.
- Contract tests against a fake upstream (`httptest`) and an optional live smoke
  test gated by environment variable.

Out of scope for E02:

- `previous_response_id` support and conversation storage.
- Generic multi-protocol abstraction (Anthropic/Gemini).
- Quota/budget enforcement (E09), retries/fallback (E05), catalog persistence
  beyond the provider's live list (E03).

## 7. Limitations and open questions

- Rate-limit (`429`) semantics, `Retry-After` and concurrency limits are
  undocumented and untested.
- Free-tier behavior differs from the documented status code (400 observed,
  403 documented).
- `service_tier`, `text.verbosity` and `client_metadata` were accepted but their
  effect upstream is unverified.
- The Codex source review used the `main` branch without a pinned commit.
- The end-to-end Codex → AI Hub hop was not exercised because the gateway does
  not expose Responses yet (E04/E07).
- Go privacy settings may block DeepSeek paid endpoints by default; the
  workspace used here already had access.

## 8. Verdict

**GO** for Codex → AI Hub → OpenCode Go → DeepSeek, constrained to stateless
HTTP Responses with full conversation replay and a stable
`x-opencode-session`.

**NO-GO** for any design that depends on `previous_response_id` with DeepSeek,
and for exposing Chat Completions to Codex instead of Responses.

## 9. Sources

- https://opencode.ai/v2/docs/console/go/ — plans, model list, session requirement
- https://opencode.ai/v2/docs/console/api/ — endpoint and API key model
- https://opencode.ai/v2/docs/console/api/inference/ — API families, cost metadata
- https://opencode.ai/v2/docs/console/models/ — model table and native APIs
- https://opencode.ai/v2/docs/console/harnesses/ — base URLs and session header
- https://opencode.ai/v2/docs/console/harnesses/codex/ — Codex `config.toml`
- https://github.com/openai/codex — `codex-rs/core/src/client.rs`, `codex-rs/tools/src/tool_spec.rs`
