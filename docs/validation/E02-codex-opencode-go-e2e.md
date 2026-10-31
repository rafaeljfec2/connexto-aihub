# E02 — Codex + AI Hub + OpenCode Go — E2E Functional Validation

Date: 2026-10-30 | Hub under validation: `main @ ad9ce7c` | Verdict: **E02 PARTIALLY VALIDATED** (details per scenario below).

This is a validation activity, not implementation work. No E01/E02 production
code was modified during this mission. Everything was executed against the
**real OpenCode Go upstream**, not against mock HTTP fixtures.

## 1. Environment

| Item | Value |
| --- | --- |
| Branch / commit | `main` / `ad9ce7c` (clean working tree) |
| Go | 1.23.0 linux/amd64 |
| AI Hub endpoints | `http://127.0.0.1:8080` (loopback, E02-documented paths) |
| Codex CLI | `codex-cli 0.166.0` (Rust binary; **not** previously installed on this host) |
| Model | `deepseek-v4.1-flash` (native API: chat-completions; translated server-side by OpenCode Go) |
| Credential source | `OPENCODE_API_KEY` present in `.env` (quoted value); bridged in-memory to `AIHUB_PROVIDER_OPENCODEGO_API_KEY` when launching the AI Hub process. The value was never printed, copied or written anywhere. |
| `.env` / shell profiles / global Codex config | **Not modified.** Codex ran with an isolated `CODEX_HOME=/tmp/opencode/codex-home`; the user's `~/.codex` was untouched. |

Note: `codex` was absent from this machine, so a pinned upstream release
(`rust-v0.166.0`) was downloaded to `/tmp/opencode/codex-bin` (isolated,
non-global, no Node dependency). This is the "temporary, isolated
configuration" approach; no global software was installed and no artifact was
left outside `/tmp/opencode`.

## 2. AI Hub startup observations

- First start with the bridged credential **failed the connectivity check**:
  `provider connectivity check failed ... status 401`, diagnostics:
  `{"id":"opencode-go","enabled":true,"state":"unhealthy"}`.
  - Root cause: my probe script extracted `.env` values with raw `sed`, keeping
    the surrounding double quotes of `OPENCODE_API_KEY="..."`. The AI Hub dotenv
    loader itself handles quoting correctly; this was a probe-script mistake,
    not an E02 defect.
  - Alt: startup succeeded with provider unavailable (`/healthz` up, `/readyz`
    up, 503 on `/v1/*`) — D2/R-D behavior observed positively by accident.
- Second start (correctly stripped value): `provider connectivity verified`,
  diagnostics `state":"healthy"`, `/healthz` `{"status":"up"}`, `/readyz`
  `{"status":"up"}`.
- Log scan: no `oc_sk_…` fragment and no `Authorization` material in any AI
  Hub log line.

## 3. Direct gateway validation (real upstream, before Codex)

| Check | Result |
| --- | --- |
| `GET /v1/models` | **PASS** — 200; 28 models (matches Go-plan count in the spike); `deepseek-v4.1-flash` present with `canonical_id deepseek/deepseek-v4.1-flash`, `type chat`, 1,000,000 ctx |
| Minimal `POST /v1/responses` (non-streaming) | **PASS** — 200, `status:"completed"`, `output` = `reasoning` + `message("pong")`, full usage incl. `cost` and `opencode` billing (`billing_source:"go"`, `list_cost 0.000015`) |
| Streaming (`stream:true`, `include:["reasoning.encrypted_content"]`) | **PASS** — `Content-Type: text/event-stream`; progressive arrival measured client-side with inter-event gaps ≤ ~0.13 s; full SSE family received (`response.created`, `in_progress`, `output_item.added/done`, `reasoning_summary_part.added/done`, `reasoning_summary_text.delta/done`, `content_part.added/done`, `output_text.delta/done`, `completed`); final `response.completed` with usage preserved |
| Multi-turn context (full replay, no `previous_response_id`) | **PASS** — secret recalled exactly ("teal") on turn 2 when the conversation was replayed in `input`; `previous_response_id` never sent |
| Tool round trip (function tool) | **PASS** — `function_call` (`get_weather`, args `{"city":"Lisbon"}`) came back; replaying `function_call_output` as **string** produced the final assistant message through both full-replay (with `reasoning` items) and call-only variants |
| Error passthrough | **PASS** — unknown model produced the upstream `400` with its generic envelope preserved through the gateway; mirrored response headers limited to `Content-Type` (+AI Hub's `X-Request-Id`, `Date`) |
| Encrypted reasoning | **PASS** — `include:["reasoning.encrypted_content"]` accepted; reasoning items replayed in later turns (200) |

Test-shape note (not an AI Hub defect; documented for harness authors):
`function_call_output.output` must be a **string**. Sending an object-shaped
`output` payload returns the upstream `400` (`"Upstream request failed…"`
envelope). This mirrors OpenAI Responses API semantics.

## 4. Codex integration

### Configuration approach

`CODEX_HOME=/tmp/opencode/codex-home/config.toml` (no secrets):

```toml
model = "deepseek-v4.1-flash"
model_provider = "aihub"

[model_providers.aihub]
name = "AI Hub (opencode-go upstream)"
base_url = "http://127.0.0.1:8080/v1"
wire_api = "responses"
env_key = "AIHUB_CLIENT_KEY"
```

`AIHUB_CLIENT_KEY=aihub-local-dummy` was exported as a dummy because Codex
requires the `env_key` variable to exist; AI Hub ignores the inbound
`Authorization` value (it holds the real credential). Codex was observed
sending every request to `127.0.0.1:8080` — the AI Hub access log recorded
`POST /v1/responses` entries for every prompt (22+ entries over the session),
each with its own correlation ID. No direct-to-OpenAI traffic took part in the
validated path.

### Scenario results

| Scenario | Result | Evidence |
| --- | --- | --- |
| A — Basic conversation | **PASS** | `codex exec "What is 2+2?"` → answer `4` through AI Hub (reasoning summary rendered; 9,293 tokens; access log `POST /v1/responses status=200`) |
| B — Multi-turn | **PASS** | `codex exec` gave codeword `E02-VIOLET`; `codex exec resume --last` recalled it exactly; full-context replay semantics held; no 400 about `previous_response_id` ever occurred (not emitted) |
| C — Tool calling | **PASS** (with caveat) | In `/tmp/opencode/codex-work`, Codex created `note.txt` (`e02 works`) and read it back, replying with exactly the file content. Caveat: required `--dangerously-bypass-approvals-and-sandbox` because `bubblewrap` is absent on this host (Codex sandbox limitation, unrelated to E02); restricted to the isolated temp dir; no project files touched |
| D — Streaming | **PASS** (one sub-item NOT VERIFIED) | Long generation (~12 s reasoning latency) over a single connection: connection held, `turn.completed` received, no timeout. **Progressive SSE rendering at the Codex stdout hop: NOT VERIFIED** — `codex exec` buffers the agent message and `--json` emits only `item.completed`/`turn.completed`; internal trace logging is not exposed by this build. Progressive SSE was proven at the AI Hub gateway hop with sub-100 ms gaps (see §3). This is a Codex CLI rendering limitation, not an AI Hub failure |

## 5. Security and observability

| Check | Result |
| --- | --- |
| Credential exposure | **PASS** — global scan of AI Hub logs, `CODEX_HOME`, evidence files: **no** `oc_sk_…` fragment anywhere; AI Hub access log contains only method/path/status/bytes/duration/correlation/remote |
| `Authorization` forwarded to Codex client | **PASS** — response headers verified with `curl -i` on `/healthz`, `/v1/models`, `/v1/responses`: no `Authorization`; allowlist holds (only `Content-Type` mirrored from upstream) |
| Correlation IDs | **PASS** — every access-log line carries a `correlation_id`; `X-Request-Id` validated/generated and echoed in response headers |
| Provider session vs request ID independence | **PASS at contract level** (unit tests); **NOT VERIFIED live** — E02 does not log session IDs (by design it must not log headers), and live observation requires no code change authority. E02's D3 contract is per-request generation whenever the client does not supply `x-opencode-session`; cross-turn continuity is an explicitly documented non-goal of E02 |
| Internal upstream error exposure | **PASS** — upstream error envelopes reach the caller with generic messages ("Model is unavailable.", "Invalid credential"); no internal AI Hub frames, no URLs with credentials, no header echo; 502 wrap only for transport failures |
| Reasoning/encrypted content logging | **PASS (n/a)** — the gateway copies bytes without parsing; nothing reasoning-related is logged by AI Hub |

## 6. Results matrix

| # | Activity | Classification |
| --- | --- | --- |
| 1 | Environment inspection | PASS |
| 2 | AI Hub startup, health, readiness, diagnostics, provider connectivity | PASS |
| 3 | Direct gateway: models / minimal generation / SSE / multi-turn / tools / errors | PASS (tool payload shape note) |
| 4 | Codex Scenario A (basic) | PASS |
| 4 | Codex Scenario B (multi-turn) | PASS |
| 4 | Codex Scenario C (tool calling) | PASS (bypass sandbox caveat) |
| 4 | Codex Scenario D (streaming) | PASS at gateway hop; **Codex-hop progressive rendering NOT VERIFIED** (CLI buffering) |
| 5 | Security/observability | PASS (session-ID live observation: NOT VERIFIED) |

## 7. Differences: expected vs actual

1. `.env` quoted credential caused an initial `401` during my probe bridging
   (probe-script artifact; AI Hub's own loader is correct).
2. `codex exec` progressive rendering cannot be observed externally; verified
   streaming stops at the gateway hop by direct client measurement.
3. `bubblewrap` absence required the sandbox bypass for Scenario C.
4. `codex` was not installed — a pinned release binary was fetched into
   `/tmp/opencode` (no global install).
5. First observed 401 also confirmed the *positive* D2 behavior: full hub
   boots, health `up`, `/readyz` `up`, `/v1/*` only (provider state unhealthy).

## 8. Limitations

- Provider session continuity across turns is per-request by design (E02
  D3); evaluation of a durable AI Hub session layer is future scope.
- Scenario C executed outside the Codex sandbox (host lacks bubblewrap).
- Codex-hop progressive streaming remains unverifiable through this Codex
  build's CLI surface.

## 9. Final recommendation

**E02 PARTIALLY VALIDATED** — the complete chain Codex → AI Hub → OpenCode Go →
DeepSeek V4.1 Flash works end-to-end over the real provider (basic
conversation, multi-turn replay, tool calling, catalog, streaming completion,
error passthrough, redaction). Two items remain unverified through no fault of
E02: progressive SSE rendering *in the Codex terminal* and live provider
session-ID observation. Both can be closed by the future session layer (E04+)
or by a Codex-side debug build; neither blocks proceeding, but E03 should still
wait for human review of this report.

No commit or push was made; no implementation code was changed during this
validation.
