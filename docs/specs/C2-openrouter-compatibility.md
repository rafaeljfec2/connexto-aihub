# Phase C2 OpenRouter compatibility

## Supported contract

The adapter calls `GET /api/v1/models` and `POST /api/v1/responses` relative to
the configured provider base URL. It sends the protected provider API key as a
Bearer credential. Responses request bytes and the supplied content type pass
through unchanged, including the upstream model identifier. The AI Hub execution
lifecycle owns timeout, cancellation, and progressive SSE forwarding.

The provider endpoint and credential come from the existing PostgreSQL-backed
provider configuration. Credentials remain sealed at rest and are never added
to logs. The adapter does not retry or fall back to another model or provider.

## Compatibility limits

- Model features vary. A model listed by OpenRouter is not proof that it
  supports every Responses input, tool, modality, or output option. The adapter
  does not infer or normalize unsupported features. Responses execution errors
  preserve the upstream HTTP status while replacing the upstream error body
  with a redacted JSON error that surfaces the upstream message (and bounded
  code/provider_code metadata) without credentials or full raw bodies. The
  native model-list route passes successful provider responses through and
  applies the same error-body sanitization.
- The current execution record stores bounded metadata only. It does not retain
  prompts, responses, or OpenRouter response state. Multi-turn continuity must
  be supplied by the caller in each request; the AI Hub does not manage
  `previous_response_id` state.
- Responses request bytes pass through unchanged except for bounded OpenRouter
  input shaping: non-replayable thinking blocks (for example
  `redacted_thinking`) and `include: reasoning.encrypted_content` are removed
  before upstream execution so multi-turn Cursor/Codex histories can route
  through OpenRouter. OpenRouter's Responses schema specifies `store` as
  `false`; callers should omit it or send `false`. The adapter does not rewrite
  a conflicting `store` value.
- OpenRouter documents `HTTP-Referer` and `X-OpenRouter-Title` for attribution.
  They are not sent because this phase has no application URL/title settings.
- Provider-level health is based on the configured endpoint and credential.
  It does not guarantee that a particular model or feature is available.

## References

- [OpenRouter Create a response API](https://openrouter.ai/docs/api/api-reference/responses/create-responses)
- [OpenRouter Models API](https://openrouter.ai/docs/api/api-reference/models/get-models)
- [OpenRouter API key and attribution headers](https://openrouter.ai/docs/app-attribution)

## Operational validation history through C2.2

The [C2.1 operational validation report](../validation/C2.1-operational-validation-2026-11-02.md)
records the evidence collected on 2026-11-02. Phase C2 remains incomplete:

| Criterion | Result |
| --- | --- |
| OpenCode Go real Responses request | PASS |
| Progressive streaming through OpenCode Go | PASS |
| PostgreSQL server restart and AI Hub recovery | PASS |
| Provider configuration persistence across restart | PARTIAL — the OpenRouter credential was added after the restart test |
| OpenRouter authenticated model listing | PASS — HTTP 200, 415 models |
| Provider state after runtime reconfiguration | PASS — stale probe state is invalidated to `unknown` until another probe |
| OpenRouter real Responses request | BLOCKED — two C2.1 requests returned HTTP 429; the first authorized C2.2 attempt failed before the gateway request, and the next single request returned HTTP 429 |
| OpenRouter progressive streaming | NOT VERIFIED — no successful Responses stream |
| OpenRouter model ID preservation | PASS — automated request-forwarding coverage; the later real request included the requested model but returned HTTP 429, so successful model execution remains unverified |
| Provider isolation and default selection | PARTIAL — automated coverage and explicit live OpenCode Go/OpenRouter routing; live provider toggles and a real request without the provider header remain unverified |
| Credential secrecy during validation | PASS |

The OpenRouter credential was subsequently configured through the protected
provider administration API. Its presence is confirmed without exposing the
value. The runtime diagnostic still reported `enabled=true` with state
`disabled` after the update, while the authenticated model-list request returned
HTTP 200. Do not mark Phase C2 complete until a successful OpenRouter Responses
request and the remaining approved isolation checks have evidence.

## C2.2 runtime diagnosis (2026-11-02)

The `enabled=true` / `state=disabled` mismatch was caused by stale in-memory
provider state. Startup records `disabled` when a configured provider has no
credential. A later administrative update reconfigures the matching adapter,
but the provider Manager previously retained that startup state. Runtime
reconfiguration now invalidates the prior probe result to `unknown`; a new
successful probe is required before it reports `healthy`. The current Manager
runs probes during process startup only, so the state remains `unknown` after a
PATCH until a future startup. Name-only updates do not alter runtime settings
or invalidate the probe result.

The HTTP 429 source is narrowed to the OpenRouter request path, but its cause is
not established. OpenRouter documents two possible sources: its own free-model
per-minute or daily quota and abuse limits, or rate limiting/capacity at an
upstream inference provider. The upstream error can carry `provider_code`;
platform limits can carry `X-RateLimit-*` headers, and provider retry hints can
carry `Retry-After`. The authenticated `GET /api/v1/key` response exposes
account-level free-model counters. The adapter preserves the status but
replaces upstream error bodies with a generic safe response, while the gateway
forwards only status, content type, and body. Execution logs record status and
bounded request metadata, not the upstream error body or rate-limit headers.
Consequently, the available 429 evidence cannot distinguish platform/account
quota from upstream provider capacity. The earlier successful model listing
shows that the credential was accepted for that route; it does not establish
generation quota. A later read-only `GET /api/v1/key` using the current literal
key assignment in `.bashrc` returned HTTP 200 with 131 of 1,000 daily free-model
requests used and 869 remaining. The C2.1 record says this assignment was used
to configure OpenRouter, but the current `.bashrc` key was not compared with
the sealed database credential. This rules out the daily cap for the queried
key at check time, conditionally for the AI Hub credential; OpenRouter does
not expose the per-minute limit in this response. See
[`C2.2 operational validation`](../validation/C2.2-operational-validation-2026-11-02.md)
for the source findings and remaining blockers.

The current public model page for `liquid/lfm-2.5-2.6b:free` lists zero input
and output token prices and exposes the Responses API for the model. At the
time of this check, OpenRouter reported 100% uptime and 98.28% inference
availability over the previous three days. These public platform metrics and
the zero token price make it the next validation candidate; they do not prove
that the configured account has quota remaining or that a request will succeed.
OpenRouter states that free endpoints are rate limited, and failed attempts may
count toward the daily free-model limit. Liquid also states that prompts and
outputs may be retained and used to train its models. A new generation request
still requires explicit authorization. See the C2.2 report for the dated source
details and remaining uncertainty.

## C2.3 final operational acceptance (2026-11-02)

The [C2.3 final operational acceptance report](../validation/C2.3-final-operational-acceptance-2026-11-02.md)
is the latest consolidated evidence. Earlier tables and reports preserve their
dated results. C2 remains incomplete because OpenRouter credential recovery
after an AI Hub restart and successful OpenRouter Responses execution have not
been verified.

| Criterion | C2.3 result | Evidence or limitation |
| --- | --- | --- |
| OpenCode Go default without provider header | PASS — automated | Header-absent gateway tests select OpenCode Go; no new live request without the header was made |
| Explicit provider selection | PASS — automated | Responses tests route both provider IDs to separate local test adapters |
| Provider configuration and state isolation | PASS — automated | Registry tests keep endpoint/credential settings separate; Manager tests show disabling OpenRouter leaves OpenCode Go healthy |
| No silent fallback after adapter failure | PASS — automated regression guard | OpenRouter HTTP 401 remains the response and the OpenCode Go test adapter receives zero requests |
| OpenRouter credential ciphertext present in current database | PASS | The protected API reports the OpenRouter row enabled with a configured credential; the store derives this flag from `credential_ciphertext IS NOT NULL`. The ciphertext was not read |
| Credential recovery after AI Hub restart | NOT VERIFIED | No post-credential restart was performed. AI Hub startup probes enabled providers through their model-list endpoints; restarting would call OpenRouter, prohibited by the C2.3 operating constraint |
| Credential confidentiality | PASS | No credential or ciphertext was printed, recorded, or returned in validation output |
| Go tests | PASS | `go test ./... -count=1` |
| Race detector | PASS | `go test ./... -count=1 -race` |
| Go vet and build | PASS | `go vet ./...`; `go build ./...` |
| Workspace check and build | PASS | `pnpm check`; `pnpm build` |
| PostgreSQL integration suite | NOT RUN | `AIHUB_TEST_DATABASE_URL` was unset; no additional database or container was created |
| OpenRouter real Responses request | BLOCKED — historical HTTP 429 | No new OpenRouter API request or generation was made in C2.3 |
| OpenRouter progressive streaming | NOT VERIFIED | No successful OpenRouter Responses stream exists |
| OpenRouter model ID execution | NOT VERIFIED | Automated request forwarding preserves the requested model; prior real requests were rejected with HTTP 429 |
| OpenCode Go real streamed request | PASS — prior explicit selection | C2.2 recorded HTTP 200, completed SSE stream, and `OK.` for `deepseek-v4.1-flash`; C2.3 made no generation request |

Final decision: **Phase C2 remains incomplete.** Completion still requires
permitted evidence for OpenRouter credential recovery after restart and a
successful OpenRouter Responses stream, plus any remaining live checks after
their explicit authorization.

## C2.4 startup probe isolation (2026-11-02)

C2.4 removes provider network requests from process startup. The Provider
Manager now initializes enabled providers as `unknown` and disabled or
credentialless providers as `disabled`; it does not call the model-list
endpoint. Persisted provider configuration and credentials are still loaded
and applied before this initialization. Since catalog startup sync runs only
for providers already marked `healthy`, it skips `unknown` providers and also
makes no startup request.

The existing `POST /api/providers/{id}/validate` route is the explicit health
check. It reads the persisted endpoint and credential, uses only the selected
provider type's model-list probe, enforces a ten-second timeout and request
cancellation, and updates runtime Manager state. The probe does not run
inference. Success records `healthy`, failure or timeout records `unhealthy`,
and caller cancellation leaves the state `unknown`. Responses and logs are
sanitized. Endpoint and credential changes invalidate the previous result;
name-only changes retain it. A probe result is ignored if that provider was
reconfigured while the probe was in flight.

| C2.4 criterion | Result | Evidence |
| --- | --- | --- |
| Startup makes no provider requests | PASS — automated | Fake model-list adapters receive zero calls from Manager startup; the catalog skips `unknown` providers |
| Persisted credentials are restored before health initialization | PASS — automated | Multi-provider bootstrap/restart test verifies both persisted credentials and initial `unknown` state |
| Explicit validation updates one provider only | PASS — automated | Service and admin-route tests verify per-provider probe calls and Manager state |
| Timeout, cancellation and stale-result handling | PASS — automated | Deadline, cancellation and credential-rotation-during-probe regression tests |
| Failure responses and logs are sanitized | PASS — automated | Admin-route test injects a secret-shaped error and checks response and log output |
| Provider selection and no-fallback behavior | PASS — existing automated coverage | C2.3 default/explicit routing and no-fallback regression tests remain in scope for full-suite validation |
| Real provider requests during C2.4 | NOT PERFORMED | No OpenRouter or OpenCode Go request was made |
| Live AI Hub restart against PostgreSQL | NOT PERFORMED | An existing AI Hub process was active on `127.0.0.1:8080` and was left running; the automated recovery test uses a fake repository |

See the [C2.4 operational report](../validation/C2.4-startup-probe-isolation-2026-11-02.md)
for command results, operational checks, independent review and remaining C2
limitations. C2 remains incomplete until a successful OpenRouter Responses
execution and other outstanding live acceptance evidence are obtained under
their respective authorization constraints.

## C2.6 health-state semantics (2026-11-03)

The provider diagnostic exposes independent health and availability fields:

| Field or value | Meaning |
| --- | --- |
| Validation response `reachable` | The configured authenticated model-list probe succeeded. This is the validation route's response label, not the diagnostic health state. |
| Diagnostic `state=healthy` | The latest current provider probe succeeded. For OpenRouter, the defined functional check is an authenticated model-list request that returns HTTP 200. This does not establish inference, model, or feature availability. |
| Diagnostic `state=unhealthy` | The latest current provider probe failed or timed out. |
| Diagnostic `state=unknown` | No current probe result exists, the result was invalidated by endpoint or credential reconfiguration, or the caller canceled the probe. |
| Diagnostic `state=disabled` | The bound runtime is not enabled or lacks its required credential. |
| Diagnostic `enabled=true` | The bound runtime is enabled and configured to receive execution requests. Health remains independent; execution does not gain an automatic fallback. |

The C2.5 operational report recorded an OpenRouter validation response of
`reachable` while the later diagnostic still showed `unknown`. C2.6 reproduced
that result in the then-running Hub process. After restarting only the Hub, one
explicit validation returned `reachable`, and the authenticated diagnostics
route reported `enabled=true`, `state=healthy`. A local service/admin regression
test now exercises the actual OpenRouter probe against a controlled HTTP server
and verifies the same state transition. The stuck state was cleared by the Hub
restart, but its exact in-process cause was not isolated; no runtime code change
is claimed. No inference was performed.

The C2.6 regression coverage verifies that diagnostics begin at `unknown`, a
successful explicit validation returns `reachable` and updates the same
provider to `healthy`, a failed validation updates only its target provider,
and credential or endpoint changes invalidate prior health while a name change
preserves it. Existing tests continue to cover cancellation, startup without
probes, disabled execution, and sanitized failures.

C2 remains incomplete until the PostgreSQL integration suite passes against an
explicitly approved isolated test instance. The integration helper creates and
drops throwaway databases from `AIHUB_TEST_DATABASE_URL`; an unverified or
shared application database must not be used.
