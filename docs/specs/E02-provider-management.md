# SPEC E02 — Provider Management (OpenCode Go)

Status: approved by the developer (2026-10-30, with adjustments D2/D3, streaming
timeout policy and readiness/diagnostics requirements).
Workflow: SDD. Evidence base: `docs/research/opencode-go-protocol-spike.md`.

## 1. Objective

First real provider integration: AI Hub → OpenCode Go through the OpenAI
Responses API with `deepseek-v4.1-flash`. AI Hub stays a passthrough: no
protocol translation, no `previous_response_id`, no persistence.

## 2. Scope

| In scope | Out of scope |
| --- | --- |
| Provider configuration (env-only), credential from environment | Intelligent routing, learning engine |
| Provider registration and lifecycle (`app.Module`) | Quotas, retries/fallback, conversation persistence |
| OpenCode Go adapter: connectivity, catalog, Responses execution | Dashboard, multi-provider orchestration |
| Exposed endpoints: Responses execution + model catalog + provider diagnostics | Generic protocol translation layer |
| Streaming pass-through with a streaming-specific timeout policy | `previous_response_id` support |
| Error pass-through with safe handling and credential redaction | Catalog persistence (E03+) |
| Automated tests against a fake upstream (`httptest`) | |
| Opt-in live smoke test guarded by environment variables | |

## 3. Packages and contracts

```text
internal/config          + provider settings (OpenCodeGo section)
internal/providers       + concrete OpenCode Go client, provider state machine
internal/gateway         + POST /v1/responses, GET /v1/models, GET /diagnostics/providers
internal/server          NewRouter gains injected routes (variadic; E01 tests unchanged)
internal/health          unchanged: registry stays process-level only (see D-R)
cmd/ai-hub               wires modules: providers -> http-server
docs/                    SPEC, architecture and README updates; ADR-008
```

Deliberate design stance: **no provider abstraction interface is introduced in
E02.** The adapter is a concrete client testable via base-URL/HTTP-client
injection against a fake upstream. The spike supports exactly one adapter shape
(Responses in, Responses out). Abstractions require a second provider.

### 3.1 `internal/providers`

```go
const OpenCodeGoID = "opencode-go"          // provider id from the mission
// Paths verified in spike §2.1; they deliberately differ.
client.responsesPath = "/openai/v1/responses"
client.modelsPath    = "/v1/models"

// Provider state machine — never one bit more than four states.
type State string // StateDisabled | StateUnknown | StateHealthy | StateUnhealthy
type Status struct{ ID string `json:"id"`; Enabled bool `json:"enabled"`; State State `json:"state"` }

NewClient(cfg config.ProviderOpenCodeGo, httpClient *http.Client) (*Client, error)
(c *Client) Enabled() bool
(c *Client) ListModels(ctx context.Context) (*http.Response, error)   // GET modelsPath
(c *Client) OpenResponses(ctx context.Context, body io.Reader,
    contentType, sessionID string) (*http.Response, error)            // POST responsesPath
```

- The request body is forwarded byte-for-byte; the adapter never parses,
  filters or injects fields into the upstream payload (D4).
- `Authorization: Bearer <key>` and `x-opencode-session: <id>` are the only
  headers the adapter adds.
- Transport: dedicated `http.Client` (no overall deadline), dial/TLS
  timeouts of 10s, `ResponseHeaderTimeout` of 120s for time-to-first-byte.

### 3.2 `internal/providers` module (lifecycle + state)

`NewManager(logger *slog.Logger, client *Client) *Manager` implements
`app.Module` (`Name() == "providers"`):

- `Start`: disabled provider → `StateDisabled`, warning log, startup proceeds
  (D2). Enabled provider → verify connectivity against the models endpoint →
  `StateHealthy`/`StateUnhealthy`. **Startup never fails because of the
  provider** — unavailability is a diagnostic state, not a boot condition.
- `Stop`: closes idle connections.
- `Statuses() []Status`: snapshot for the diagnostics endpoint.

### 3.3 `internal/gateway`

```go
NewResponsesHandler(client *providers.Client, logger *slog.Logger,
    inactivity time.Duration) http.Handler   // POST /v1/responses
NewModelsHandler(client *providers.Client, logger *slog.Logger) http.Handler
                                                                 // GET /v1/models
NewProvidersHandler(report Statuses) http.Handler                // GET /diagnostics/providers
```

Behavior:

- Disabled provider → `503` with
  `{"error":{"type":"provider_unavailable","message":"provider opencode-go is not configured"}}`.
- Upstream connection/setup failure → `502` with a generic safe body; details
  go to logs only.
- Success → upstream status and `Content-Type` preserved; body copied in
  bounded chunks with `Flush` per chunk (the E01 `responseRecorder` supports
  `Flush` and `Unwrap`, `middleware.go:127-140`).
- Response headers are allowlisted: only `Content-Type` is mirrored (D5). No
  upstream header, query string or body content is reflected into logs.

### 3.4 Streaming timeout policy (mandatory adjustment)

ADR-008 applies (see `docs/adr/ADR-008-streaming-inactivity-deadline.md`):

- Ordinary E01 behavior and `server.write_timeout` are untouched.
- The gateway arms a **write deadline of `inactivity`** via
  `http.ResponseController.SetWriteDeadline` before the first chunk and
  re-arms it after every successfully written chunk.
- A per-chunk **reader watchdog** bounds upstream silence: reads happen in a
  dedicated goroutine feeding a bounded channel; the handler loop selects on
  chunks, `request.Context().Done()` and the inactivity timer. On expiry the
  upstream body is closed and the stream is aborted.
- The timeout is **inactivity between chunks**, not a total-duration cap.
- Client disconnection cancels `r.Context()`; the handler closes the upstream
  body and returns, and the reader goroutine terminates through the read error.
- Default `inactivity` = 2 minutes (`gateway.StreamInactivityTimeout`),
  injectable for tests.

Existing-architecture note: the E01 `responseRecorder` already exposes `Unwrap`,
so `http.ResponseController` reaches the real connection writer; no server
change is required beyond the injected route.

## 4. Decisions

| ID | Decision |
| --- | --- |
| D1 | `AIHUB_PROVIDER_OPENCODEGO_BASE_URL` defaults to `https://opencode.ai/inference/go`; the adapter appends `/openai/v1/responses` and `/v1/models` (paths from spike §2.1). |
| D2 | Missing credential ⇒ provider starts **disabled**; AI Hub startup proceeds; `/v1/*` returns 503 ineligible. Configuration state and availability state are separate (disabled / unknown / healthy / unhealthy). |
| D3 | `X-Request-Id` (request correlation) and `x-opencode-session` (provider execution context) are independent. Incoming `x-opencode-session` is preserved when explicitly supplied; otherwise AI Hub generates one (`aihub-ses-` prefix, crypto/rand). Provider session IDs are never derived from correlation IDs. E02 does **not** claim cross-request/cross-turn session continuity; the limitation is documented and the design stays compatible with a future AI Hub context/session layer. |
| D4 | Responses passthrough is byte-for-byte; no `previous_response_id`; no protocol translation. |
| D5 | Credentials never appear in logs, errors or responses. Base URLs with embedded user information are rejected at configuration validation. Response header mirroring is allowlisted to `Content-Type`. |
| D6 | Live smoke is opt-in (`AIHUB_LIVE_SMOKE=1` + credential present). Mock verification vs real upstream validation stays explicitly separated. |
| D-R | Readiness stays process-level: E02 registers no readiness check, so a disabled or flappy provider never flips `/readyz` to 503 (`disabled ≠ application unavailable`). Provider configuration/availability is exposed at `GET /diagnostics/providers`, which must never leak credentials or sensitive upstream detail. |

## 5. Configuration

```env
AIHUB_PROVIDER_OPENCODEGO_BASE_URL=   # default https://opencode.ai/inference/go
AIHUB_PROVIDER_OPENCODEGO_API_KEY=    # oc_sk_...; empty => provider disabled
```

Validation: base URL must parse with scheme http/https, non-empty host, no
query/fragment and no embedded user credentials; aggregation in the single
config error matches `Config.Validate`.

## 6. Test strategy (TDD)

Order: config → adapter → module/state → gateway (incl. streaming policy) →
wiring. Fake upstream via `httptest`; real listeners for deadline behavior;

1. Provider config: defaults, env override, unconfigured credential
   (disabled), invalid base URLs (scheme, host, query, user info).
2. Adapter `OpenResponses`: byte-identical body, `Content-Type`, Bearer,
   session header injection vs propagation, models path for the catalog.
3. Streaming policy: chunk flow shorter than the ordinary write timeout
   survives (streams longer than a deliberately small `write_timeout`);
   inactivity expiry aborts; client disconnect aborts without panic.
4. Errors: upstream 401/400/5xx status+body preserved; connection-refused →
   502 with safe body; provider disabled → 503.
5. Redaction: fixtures assert the API key never appears in logs, error bodies
   or mirrored response headers.
6. Diagnostics: no credentials or upstream internals in the payload; disabled
   provider keeps `/readyz` up.
7. Lifecycles: provider module start ordering with app rollback behavior.
8. Wire/log compatibility: `NewRouter` variadic routes keep E01 tests valid.

Live smoke (`internal/providers/live_smoke_test.go`): `t.Skip` unless
`AIHUB_LIVE_SMOKE=1` **and** `AIHUB_PROVIDER_OPENCODEGO_API_KEY` is set;
prompts the real models endpoint and one minimal streaming Responses call.

## 7. Evidence block

| Field | Value |
| --- | --- |
| Claim | E02 exists and can expose the validated Responses passthrough to OpenCode Go while preserving neuralevents, streaming, tool calls, reasoning content, errors, session identity and credential secrecy. |
| Success type | `test` (automated fake-upstream suite) plus optional `command` (live smoke). |
| Evidence path | `go test ./... -race` green on all suites in §6; live smoke executed or explicitly reported as **pending**. |
| Budget | The suites in §6 through §8 cover claim boundaries: passthrough, streaming, timeout policy, errors, redaction, state machine, wiring. |
| Affordance | Fake upstream (`httptest`) makes upstream state controllable/observable; live smoke opt-in via env. |

## 8. Edge cases

- API key present but invalid upstream: reflected upstream status/body, no local
  retry, gateway returns exactly the upstream 401 envelope.
- Base URL points to a closed port: 502 safe body; no credential leaked.
- Provider connection drops mid-stream (client disconnects/timeout): 500-style
  handler returns; upstream context cancelled; inactivity timer stopped; no goroutine leak (`CloseIdle` + body close).
- Unknown provider env typo: strict `AIHUB_*` unknown detection still warns.
- Stream chunk boundaries: buffered copy up to 32 KiB per chunk preserves
  SSE framing verbatim (no parsing).

## 9. Known limitations

- Session continuity across turns is not implemented (D3); injected/generated
  session IDs are per-request.
- No retry, quota, fallback or catalog persistence (E03+/E05+/E09).
- 429 semantics unknown upstream (spike §2.11).
- Provider state is refreshed only at module start in E02 (no background probe).

## 10. Verification checklist (from the mission)

- [ ] Configuration validation with missing/invalid credentials.
- [ ] Model catalog retrieval (path from spike).
- [ ] Connectivity failures (502 safe handling).
- [ ] Responses pass-through byte-identical, no `previous_response_id`.
- [ ] Stable session injection/propagation (D3).
- [ ] Streaming preservation (SSE semantics byte-for-byte) with robust
      inactivity deadline policy.
- [ ] Tool and custom tool call preservation through the passthrough.
- [ ] Reasoning/encrypted content preserved (payload untouched).
- [ ] Credential redaction in logs, errors and mirrored headers.
- [ ] Real smoke pending or executed only with credentials.
