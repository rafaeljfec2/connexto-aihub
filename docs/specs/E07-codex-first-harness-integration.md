# SPEC E07 — Codex First Harness Integration on the E06 Baseline

Status: approved by the project lead on 2026-10-31.

Scope disposition (2026-11-02): AC-14 and FR-12 are out of scope for the AI Hub
product and are excluded from E07 acceptance. The AI Hub centralizes access to
LLM providers; it does not execute agents or administer their local host
environments. Its product boundary covers provider integrations, context-aware
routing and model selection, cost/token policies, and centralized
observability. This disposition does not waive provider-credential protection,
gateway authentication, agent/provider credential isolation, request/response
security, or consumption governance. Repository and Git integrity checks are
development hygiene, not product behavior or an E07 acceptance criterion.

Workflow stage: approved SPEC; Phases 0–3 completed. Design, test, implementation,
validation, and independent review are authorized under the project SDD workflow.

Date: 2026-10-31.

Baseline: `main` at `0d33ca26c15acf54fac418cb33380b5871b4d899` (E06), equal to
`origin/main`. The working tree also contains the untracked
`docs/discovery/E07-discovery.md`; that Discovery is input and must not be
modified by E07.

## 1. Intent

Formally support Codex CLI as AI Hub's first harness by publishing one isolated
and repeatable setup path, then validating the real
`Codex → AI Hub → OpenCode Go` route on the committed E05/E06 baseline. Codex
has already reached OpenCode Go in E02/E04 validation and one E05 smoke. E07
closes the remaining baseline gap: no available end-to-end artifact covers the
current E06 execution-record lifecycle and the broader accepted scenario set.

E07 productizes and verifies the existing direct Responses path. It does not
add a harness adapter, change request or response behavior, or make Codex
conversation state durable in AI Hub.

## 2. Verified baseline and boundaries

The current request route is:

```text
Codex → E04 authenticated gateway → E05 execution manager → OpenCode Go
                                              └──────────→ E06 records observer
```

The gateway creates an execution command with `provider_id = opencode-go` and
no `model_id`; it forwards the admitted request bytes without parsing or
rewriting them. The provider client is a concrete OpenCode Go adapter. It does
not select or route among providers. Sources: `internal/gateway/responses_handler.go`
(`responsesHandler.ServeHTTP`, line 71), `internal/execution/types.go`
(`Target`, line 10; `Command`, line 16), and `internal/providers/client.go`
(`OpenTargetResponses`, line 108).

E04 owns static bearer authentication, Host/Origin checks, request admission,
and request correlation. Every protected route, including `/v1/responses` and
both records routes, uses the same policy. Sources: `internal/server/router.go`
(`NewRouter`, line 22), `internal/access/authorize.go` (`Policy.Authorize`,
line 6), and `cmd/ai-hub/main.go` (record routes, lines 148 and 153).

E06 records terminal operational metadata after E05 cleanup. Schema v1 has
`usage: null`; the current gateway leaves model identity absent, so records
have `model_id: null`. Records are at-most-once and best-effort. Queue overflow,
storage failure, crash, retention, or flush expiry can lose history. Sources:
`internal/records/record.go` (`Record`, line 52),
`internal/records/service.go` (`OnExecutionFinished`, line 154),
`internal/gateway/records_handler.go` (query handlers, lines 49 and 77),
ADR-017, and ADR-018.

Shutdown closes execution admission, drains or cancels registered E05 work,
then stops E06 records before guarded provider release. The E06 record writer
has a fixed two-second drain window after E05 confirms execution cleanup.
Sources: `cmd/ai-hub/lifecycle/lifecycle.go` (`Stop`, line 32),
`cmd/ai-hub/main.go` (module order, line 164), ADR-015, and
`docs/design/E06-execution-records-design.md` §3.2.

Existing Codex reports provide useful but older evidence:

- E02 at `ad9ce7c` reports basic generation, multi-turn replay, tools, upstream
  errors, and gateway-hop streaming, with Codex-hop progressive rendering not
  verified (`docs/validation/E02-codex-opencode-go-e2e.md`).
- E04 at the E03 base plus an uncommitted E04 tree reports authenticated
  Codex access and similar scenarios, but predates E05/E06
  (`docs/validation/E04-gateway-hardening-access-control-e2e.md`).
- E05 records one successful Codex smoke but not the broader E06 records path
  (`docs/specs/E05-execution.md` §14).

These reports are historical input only. E07's validation must record the
actual E06 revision it exercises.

## 3. Goals

- Make Codex CLI the first explicitly supported harness for the existing
  OpenCode Go Responses route.
- Give a developer one documented setup path using isolated Codex and AI Hub
  state, without editing global Codex configuration or tracking secrets.
- Establish current-baseline evidence for basic generation, multi-turn replay,
  tool use, gateway authentication, upstream error passthrough, observable
  gateway-hop streaming, record capture/query, and orderly shutdown/drain.
- Publish scenario-level results with exact revision and environment
  information and with limitations stated plainly.
- Preserve E01–E06 public behavior, privacy rules, and best-effort record
  guarantees.

## 4. Scope

### 4.1 In scope

1. A Codex CLI setup and reproduction path maintained from the README.
2. An isolated `CODEX_HOME` with a dedicated `config.toml`; no changes to the
   user's `~/.codex` or repository-global Codex settings.
3. A loopback AI Hub listener and a dedicated temporary `AIHUB_DATA_DIR` for
   the documented validation run.
4. An environment-backed gateway key supplied to Codex through its configured
   `env_key`, distinct from the OpenCode Go provider key. Secret values must
   not appear in tracked files, Codex TOML, command-line arguments, or
   published evidence.
5. Real Codex CLI and real OpenCode Go validation against the committed E06
   baseline, plus direct authenticated gateway probes where a scenario cannot
   be observed from Codex CLI output alone.
6. A scenario-level validation report under `docs/validation/`.
7. Existing automated test and quality-gate regression checks. A small opt-in
   validation helper may be added only if needed to make an accepted scenario
   reproducible; it must not modify production source, add runtime behavior,
   or introduce a production dependency.

### 4.2 Explicit non-goals

- A second harness, provider, provider framework, or protocol translation.
- Changes to E01–E06 behavior, HTTP/API contracts, ADRs, or historical
  documents.
- Model selection, ranking, routing, payload rewriting, fallback, or retries.
- Usage parsing, cost calculation, quotas, billing claims, or changes to
  schema-v1 `usage: null`.
- Model attribution changes; current `model_id: null` is preserved.
- Durable AI Hub sessions, conversation persistence, prompt/response storage,
  quality evaluation, learning, dashboards, metrics, or OpenTelemetry.
- Non-loopback exposure, TLS termination, load testing, or stress testing.

If completing E07 appears to require changing an E01–E06 contract, stop and
raise that as a separate decision; do not expand this scope.

## 5. Functional requirements

### FR-01 — Supported harness boundary

The README must identify Codex CLI using the Responses wire API as the first
supported harness and OpenCode Go as the provider used by this integration.
The support claim is limited to the Codex CLI version and environment class
that the E07 validation report actually tests. It must not imply support for
other harnesses or all Codex releases.

### FR-02 — Isolated and reproducible setup

The documented path must:

- use a temporary or otherwise dedicated `CODEX_HOME` and place Codex
  configuration only beneath it; create the directory before launching Codex,
  because `CODEX_HOME` must name an existing directory;
- set Codex's provider base URL to the local AI Hub Responses endpoint and
  configure `wire_api = "responses"`;
- configure `env_key` to name an environment variable carrying the gateway
  key, without putting the key value in TOML; Codex must be launched with that
  variable exported in the Codex process environment, because `env_key` is
  resolved from that process environment;
- provide a reproducible, non-echoing secret-entry step that sets the gateway
  key in the AI Hub process environment as `AIHUB_GATEWAY_API_KEY` and in the
  Codex process environment as the variable named by `env_key` (for example,
  `AIHUB_CLIENT_KEY`); the provider and Codex processes must be started from
  separate environment sessions so the provider key is available only to AI
  Hub and the gateway key is available to Codex;
- verify credential presence in each process environment using presence-only
  checks; never print or record environment values;
- run AI Hub with a dedicated data directory so validation records are
  separated from operator history;
- state prerequisites, startup order, exact Codex CLI version, model ID,
  invocation, and cleanup steps; and
- leave the user's global Codex settings, tracked files, and project `.env`
  unchanged.

The setup must use explicit environment configuration for the Hub's gateway
and provider credentials. The provider key is consumed only by AI Hub; the
gateway key is consumed by Codex. The setup must not require either credential
to be copied into a repository file.
The gateway key must continue to satisfy E04's 32–512 character `b64token`
grammar and remain distinct from the provider key.

The README setup must not place secret values in shell command text, process
arguments, TOML, tracked or untracked repository files, or documentation. A
non-echoing interactive prompt or an existing secure environment-secret
facility may supply each value to its intended process. Do not introduce a new
Codex configuration mechanism.

### FR-03 — Gateway authentication and admission

The real Codex request must authenticate with the gateway key through the
configured environment variable. A separate request without a bearer key must
receive the existing `401 unauthorized` response and must not reach the
provider. The unauthenticated probe must be bracketed by a validation-only
provider-boundary call counter around the AI Hub provider HTTP transport. The
counter may use the existing provider-client HTTP-client injection point and
must observe only the aggregate call count; it must not inspect, emit, or
persist URLs, headers, credentials, bodies, or generated identifiers. The
observer is temporary validation instrumentation and must not enter production
composition or change production behavior. Host/Origin validation, JSON
admission, body bounds, and session header rules remain E04 behavior and must
not be bypassed by the setup.

### FR-04 — Basic generation

At least one Codex prompt must receive a non-empty successful response through
the AI Hub loopback endpoint and the real OpenCode Go provider. Evidence must
show that the request reached the Hub and that Codex was not configured to
call the provider directly.

### FR-05 — Multi-turn full replay

The validation must exercise a Codex conversation across at least two turns:
turn one establishes a synthetic, non-sensitive marker, and a later turn
recalls it correctly through the same Hub route. The setup must not rely on an
AI Hub durable session or on `previous_response_id`; E04 provider-session
headers remain per-request context. The validation must establish that the
second provider request includes prior-turn context and does not rely on
`previous_response_id`. If this requires a temporary request observer, it may
inspect only synthetic test content in memory and must emit only the needed
structural result (prior-context present, `previous_response_id` absent); it
must not persist or publish request payloads. This observer is validation-only
and must not be added to the AI Hub runtime.

### FR-06 — Tool round trip

Codex must complete a bounded local tool task in a dedicated temporary
workspace, such as writing a synthetic value to a file and reading it back.
The task must stay outside the repository. Do not disable Codex approvals or
bypass its sandbox to make this scenario pass. If the environment cannot run
the tool safely, classify the scenario as `NOT VERIFIED` with the environment
limitation; do not claim tool support from a direct HTTP function-call test
alone.

### FR-07 — Upstream error passthrough

A request that causes OpenCode Go to return a known client error (for example,
an intentionally unknown model in a direct authenticated Responses probe) must
retain the upstream status and generic response body through AI Hub. AI Hub
must not replace it with an authentication or local gateway error. The probe
must not be made through a Codex command if the CLI rejects the model before
sending a request.

### FR-08 — Gateway-hop streaming

Streaming must be measured at a client connection to the AI Hub gateway using
the real provider. The evidence must establish that SSE data arrives
progressively before the response completes and that AI Hub does not buffer
the complete response. Codex CLI stdout rendering is a separate observation:
report it as `NOT VERIFIED` if that CLI version does not expose progressive
delivery. Do not infer Codex-side rendering from gateway-hop timing.

### FR-09 — Execution record capture and query

Under normal operation, completed E07 requests must produce records queryable
through the authenticated E06 routes. The report must verify the record's
provider and terminal outcome, `usage: null`, current `model_id: null`, and
correlation to the request without publishing generated identifiers. For
requests rejected before E05 execution admission, no execution record is
expected. These observations do not imply complete history under queue
overflow, storage failure, crash, retention, or flush expiry.

### FR-10 — Orderly shutdown and record drain

After the scenario run, the documented procedure must stop AI Hub gracefully,
restart it with the same dedicated data directory, and query the retained
records. Successful shutdown/restart evidence must show that the records
accepted and written under normal operation remain available. The report must
state E06's two-second writer drain and at-most-once semantics; it must not
claim forced process termination, storage failure, or queue-overflow records
are preserved. E05 cleanup and the guarded provider-release order remain
unchanged.

### FR-11 — Scenario report

The E07 report must include the exact AI Hub commit, Go version, Codex CLI
version, operating-system/environment class, selected model ID, scenario
result, and limitations. Each scenario uses one of `PASS`, `FAIL`,
`NOT VERIFIED`, or `NOT RUN — environment/provider limitation`. A required
scenario marked `FAIL`, `NOT VERIFIED`, or `NOT RUN` means E07 is not fully
validated; it must not be summarized as supported or passed. The Codex-hop
progressive-rendering observation is non-blocking only under FR-08 and AC-13;
gateway-hop streaming remains required.

The report must not contain request/response content, provider or gateway
credentials, provider-session IDs, request IDs, execution IDs, or other
generated identifiers. It may report aggregate counts and status codes.

### FR-12 — OUT OF SCOPE: Host environment integrity

Auditing the global `~/.codex` tree, external symbolic-link destinations,
project `.env` files, or the host filesystem is not an AI Hub product
requirement. The AI Hub does not execute Codex or administer the environment in
which Codex runs. No host-tree fingerprint, host `.env` fingerprint, or secret
scanner result is an E07 acceptance condition. Repository and Git scope checks
remain development hygiene and must not be presented as product functionality.

## 6. Non-functional requirements and invariants

- No production behavior, public endpoint, response format, security boundary,
  or provider contract changes.
- No new runtime dependency. Validation helpers, if any, are opt-in and
  separate from the AI Hub runtime.
- Validation uses loopback only and dedicated temporary state. Temporary
  secrets and Codex state are removed after the run; cleanup must not target
  global user configuration.
- Secrets must not be passed as command-line arguments or printed in logs.
  Evidence and diagnostics are reviewed and redacted before being saved.
- No prompts, generated content, tool arguments/results, provider session IDs,
  or headers are added to Hub logs or records.
- E06 remains best-effort and at-most-once; `404` continues to describe only
  the current retained/queryable set, and `degraded` remains process-local.
- Real provider calls are limited to the minimum scenario set. E07 does not
  perform load testing, automatic retries, or repeated runs to improve a
  result.

## 7. Acceptance criteria

1. A developer can follow the README setup from a dedicated, pre-created
   `CODEX_HOME` and `AIHUB_DATA_DIR`; Codex uses the configured loopback Hub
   URL and Responses wire API. The README shows `env_key` containing only the
   gateway environment variable's name and demonstrates exporting that
   variable before launching Codex. A presence-only check confirms the gateway
   key reaches the Codex process and the provider key does not; the provider
   key reaches only the AI Hub process.
2. The setup identifies the exact tested Codex CLI release and model. Every
   Codex Responses request in the scenario run reaches the AI Hub endpoint;
   no direct Codex-to-provider route is used.
3. The gateway accepts the correctly authenticated Codex request and rejects
   the unauthenticated probe with the existing `401` contract. The
   provider-boundary call counter is unchanged across that probe, proving no
   provider call occurred; the status alone is not sufficient evidence.
4. Basic generation succeeds through the real OpenCode Go provider.
5. A temporary or equivalent wire observation confirms that the second
   provider request includes prior-turn context and omits
   `previous_response_id`; the second Codex turn correctly recalls the
   synthetic marker; the report makes no claim of persistent AI Hub
   conversation state or retained payload inspection.
6. A safe tool round trip succeeds in the temporary workspace without
   bypassing Codex sandbox or approval controls.
7. The upstream error probe preserves the upstream client status and generic
   body. Local authentication behavior remains distinct from upstream errors.
8. Gateway-hop streaming evidence shows multiple SSE arrivals before
   completion. Codex-side progressive rendering is reported independently and
   may remain `NOT VERIFIED` when the CLI offers no measurable signal.
9. E06 query responses contain records for the observed successful executions
   under normal operation, with the expected provider/outcome and current
   null semantics; no request/response content or generated identifier is
   published.
10. Orderly shutdown followed by restart with the same temporary data
    directory preserves the records accepted and written during the run; the
    report accurately describes the bounded two-second drain and best-effort
    loss windows.
11. The validation report records revision, environment, result, and
    limitations for every required scenario and contains no secret, prompt,
    response, session ID, request ID, or execution ID.
12. No production source file differs from the E06 baseline.
    `go test ./... -count=1 -race`, `go vet ./...`, `staticcheck ./...`,
    `golangci-lint run ./...`, `go build ./...`, formatting checks, and
    `git diff --check` pass on the E07 change set. The existing test suite
    remains entirely green, with all existing tests passing; no existing test
    is removed, weakened, or made less stringent. Any new validation helper's
    dedicated tests pass. No E01–E06 production behavior is changed. If a
    production-source change appears necessary, stop and request a separate
    scope decision.
13. The README support claim is added only after every required scenario is
    `PASS`; a Codex-hop progressive-rendering limitation may be `NOT VERIFIED`
    if gateway-hop streaming is `PASS` and that limitation is explicit.
## AC-14 — OUT OF SCOPE

The former host-environment-integrity criterion is excluded from E07 product
acceptance. It covered the global `~/.codex` tree, external symlink targets,
project `.env` files, host filesystem fingerprints, and repository secret
scanning. These are outside the AI Hub architectural boundary. This status is
not a PASS and does not change the historical outcomes recorded in the E07
validation report. Repository/Git scope checks remain development hygiene.

## 8. Requirement traceability

| Requirement | Acceptance criteria | Evidence |
| --- | --- | --- |
| FR-01–FR-02: explicit support and isolated setup | AC-1–2, AC-13 | README walkthrough; inspect temporary Codex config for variable names only; verify process-environment presence without values |
| FR-03: authentication and admission | AC-3 | Real authenticated Codex request plus an unauthenticated gateway probe returning 401 and an unchanged provider-boundary call count |
| FR-04: basic generation | AC-4 | Real Codex output and matching Hub request status |
| FR-05: multi-turn full replay | AC-5 | Two-turn Codex scenario plus ephemeral structural observation; no content retained in report |
| FR-06: tool round trip | AC-6 | Temporary workspace file result and Codex run status; sandbox bypass is prohibited |
| FR-07: upstream error passthrough | AC-7 | Direct authenticated request with a known-invalid model; compare upstream and Hub status/body class |
| FR-08: streaming | AC-8 | Gateway-boundary SSE arrival timing; Codex output instrumentation separately recorded |
| FR-09: records | AC-9 | Authenticated E06 list/get queries correlated in-session to scenario requests, then identifiers redacted |
| FR-10: shutdown and drain | AC-10 | Graceful stop, restart on same temporary data directory, authenticated record query |
| FR-11: report and privacy | AC-11, AC-13 | Manual artifact review and secret/identifier redaction check |
| Existing runtime compatibility | AC-12 | Full Go quality gates and diff/scope review |
| FR-12: host environment integrity | OUT OF SCOPE | No AI Hub product acceptance evidence; repository/Git integrity remains development hygiene |
| Former AC-14: global Codex/host integrity and repository secret scan | OUT OF SCOPE | Architectural boundary disposition recorded above; historical E07 evidence remains unchanged |

## 9. Validation plan and TDD/no-test justification

E07 does not specify any change to production behavior. The primary claim is
about real-world `Codex → AI Hub → OpenCode Go` scenarios through the existing
E05/E06 path; those scenarios are the primary evidence and cannot be
established by mocked tests. Validate them with the documented scenario
matrix, authenticated E06 queries, an orderly shutdown/restart, and the
existing automated test suite, which must remain entirely green with all
existing tests passing.

No new production behavior or production-source tests are required. Design may
add a small opt-in validation helper only when needed to make an accepted
scenario reproducible; any helper must have focused tests for its repeatable
behavior and must not replace the real-world scenarios. Mocked tests are not a
substitute for those scenarios. No existing test may be removed, weakened, or
made less stringent. This SPEC is approved and its E07 development cycle is
authorized by the project lead; helper tests and real-world evidence remain
subject to the normal SDD gates below.

Evidence sequence:

1. Record baseline HEAD, Codex CLI version, Go version, OS/environment class,
   and selected OpenCode Go model ID.
2. Capture secure before-state fingerprints; create isolated temporary Codex,
   AI Hub data, and tool-workspace directories; start AI Hub on loopback with
   its gateway and provider keys supplied only in the AI Hub process
   environment. In a separate environment session, export the gateway key under
   the `env_key` variable name and launch Codex. Verify process-variable
   presence without emitting values.
3. Run the basic, multi-turn, and tool scenarios through Codex; run direct
   authenticated probes for gateway rejection, upstream error passthrough, and
   gateway-hop streaming. For the unauthenticated probe, compare the
   provider-boundary call counter before and after and require a zero delta.
4. Query E06 records under normal load, stop AI Hub gracefully, restart with
   the same temporary data directory, and query again.
5. Save only redacted scenario outcomes and aggregate evidence. Compare the
   final environment and Git state to the secure before-state; remove temporary
   secrets, Codex state, workspace files, integrity manifests, and test data.
6. Run the AC-12 automated and formatting checks; inspect the final diff for
   scope, privacy, and README accuracy. Run dedicated tests for any new
   validation helper; these do not replace the real-world scenarios.

## 10. Dependencies, risks, and limitations

| Item | Treatment |
| --- | --- |
| Real OpenCode Go availability and valid provider credential | Required for a `PASS` on the real-provider scenarios; if unavailable, mark the affected rows `NOT RUN — environment/provider limitation` and do not declare E07 validated. |
| Codex CLI custom Responses provider support | The current official config reference defines `env_key`, `base_url`, and `wire_api`; the CLI implementation resolves `env_key` from the running process environment. Select and record an exact tested release and revalidate the configuration against it in the E07 run. `CODEX_HOME` must exist before launch. |
| Provider/model availability can change | Select one explicit available model for the run and record its ID; do not introduce selection or fallback behavior. |
| Codex CLI does not expose progressive output | Measure streaming at the Hub boundary; mark Codex-hop rendering `NOT VERIFIED` unless directly observable. |
| Codex sandbox or approvals unavailable | Do not disable controls; mark the tool scenario not verified and leave E07 incomplete. |
| Best-effort records can be lost | Test only normal operation and orderly shutdown; document overflow, storage, crash, retention, and flush limitations. |
| Live provider usage may incur provider charges | Bound the run to one pass of the required scenarios; do not load test or retry automatically. |
| Codex/provider/model versions drift | The support statement is scoped to the tested Codex version, environment class, and explicitly recorded model. |

No architectural decision or ADR is required while E07 remains a docs and
validation-only increment. If design requires new runtime behavior or an
existing contract change, pause for separate developer evaluation.

## 11. Evidence block

- **Claim:** A developer can configure Codex CLI in an isolated environment to
  call the authenticated local AI Hub Responses endpoint, complete the required
  real OpenCode Go scenarios on the committed E06 baseline, query the resulting
  normal-operation records, and restart AI Hub without changing E01–E06
  behavior or exposing secrets/content in published evidence.
- **Success type:** `manual`, `command`, and `test`.
- **Evidence path:** README setup; scenario-level real Codex/OpenCode Go report;
  authenticated and unauthenticated Gateway probes with provider-boundary
  call-count observation; gateway-hop SSE timing; E06 record queries
  before/after orderly restart; secure before/after environment-integrity
  comparison; full Go regression and quality checks.
- **Budget:** One bounded pass through each required scenario with one explicit
  model and no automated retries or stress/load traffic, plus the existing
  full Go regression and quality gates.
- **Affordance:** Temporary `CODEX_HOME`, temporary AI Hub data directory,
  temporary tool workspace, environment-only credentials, a gateway-boundary
  streaming probe, and a validation-only provider-boundary call counter. A
  temporary wire observer may be used to prove full replay; it inspects
  synthetic content in memory and emits structural facts only. A private
  temporary integrity manifest may hold before-state fingerprints outside the
  repository and must be deleted after comparison. These are validation-only;
  no durable product affordance is introduced.
- **No-test justification accepted for SPEC refinement only:** E07 specifies no
  production behavior change; real-world scenarios are the primary evidence,
  and the existing suite must remain entirely green. Any new validation helper
  has dedicated tests, but mocked tests cannot substitute for the real-world
  scenarios. This rationale alone did not approve the SPEC or authorize
  implementation; the separate approval and execution authorization are
  recorded in §12.

## 12. Approval record

- **Approved by:** Project lead.
- **Date:** 2026-10-31.
- **Approved scope:** E07 intent, boundaries, requirements, acceptance
  criteria, traceability, validation plan, and Evidence block in this SPEC.
- **Execution authorization:** Complete the E07 cycle under the mandatory
  project SDD workflow, including Design, TDD where applicable, implementation,
  real-provider validation, quality gates, and independent review.
- **Publication boundary:** No commit or push is authorized by this approval.

### Supplemental E2E pass authorization (2026-10-31)

- **Authorized by:** Project lead.
- **Budget relationship:** The original one-pass budget in §11 remains the
  baseline. Exactly one additional real E2E pass is authorized, for a maximum
  of two total passes. No individual scenario retries, repeat runs, or third
  pass are authorized.
- **Scope:** Validation and evidence documentation only. This authorization
  does not change requirements, acceptance criteria, or implementation scope.
- **Preflight gate:** Start the additional pass only when `bwrap` is available
  in `PATH`; keep `HOME` and `CODEX_HOME` isolated. Do not use a fallback or
  bypass if `bwrap` is unavailable.
- **Disposition:** An initial 2026-10-31 preflight found no `bwrap` executable
  and did not start a scenario. After the environment was prepared, a later
  preflight confirmed `bubblewrap 0.4.0`, isolated `HOME` and `CODEX_HOME`,
  and the required Codex CLI release. Exactly one additional pass was then
  executed; it is consumed, with no retry or third pass. Scenario and integrity
  results are recorded in
  [`docs/validation/E07-codex-first-harness-integration.md`](../validation/E07-codex-first-harness-integration.md).

## 13. References

- [`docs/discovery/E07-discovery.md`](../discovery/E07-discovery.md) — E07
  recommendation and current-state evidence.
- [`docs/specs/E05-execution.md`](E05-execution.md) — E05 target, passthrough,
  cancellation, streaming, and shutdown contracts.
- [`docs/specs/E06-execution-records.md`](E06-execution-records.md) — schema-v1
  records, privacy, best-effort semantics, and drain contract.
- [`docs/design/E06-execution-records-design.md`](../design/E06-execution-records-design.md)
  — E06 composition and lifecycle.
- [`docs/adr/ADR-015-coordinated-execution-shutdown.md`](../adr/ADR-015-coordinated-execution-shutdown.md)
  — E05 shutdown ordering and provider guard.
- [`docs/adr/ADR-017-execution-records-storage.md`](../adr/ADR-017-execution-records-storage.md)
  and [`docs/adr/ADR-018-execution-terminal-observation.md`](../adr/ADR-018-execution-terminal-observation.md)
  — E06 persistence, observer, privacy, and loss boundaries.
- [`docs/validation/E02-codex-opencode-go-e2e.md`](../validation/E02-codex-opencode-go-e2e.md)
  and [`docs/validation/E04-gateway-hardening-access-control-e2e.md`](../validation/E04-gateway-hardening-access-control-e2e.md)
  — historical Codex configuration and scenario evidence, not E06 validation.
- [`README.md`](../../README.md) — current E01–E06 setup, configuration,
  endpoints, and known limitations.
- [OpenAI Cookbook: Harness-aware evaluation of plugins](https://developers.openai.com/cookbook/examples/partners/harness_aware_plugin_evals/harness_aware_plugin_evals)
  — current official example of running Codex with an explicit isolated
  `CODEX_HOME`; the exact AI Hub Responses-provider configuration remains
  grounded in this repository's E02/E04 validation evidence.
- [Codex CLI Configuration Reference](https://developers.openai.com/codex/config-reference/)
  — current definitions for custom provider `base_url`, `env_key`, and
  `wire_api = "responses"`.
- [Codex CLI `env_key` implementation](https://github.com/openai/codex/blob/main/codex-rs/model-provider-info/src/lib.rs)
  — resolves the configured variable name from the running process environment.
- [Codex CLI `CODEX_HOME` implementation](https://github.com/openai/codex/blob/main/codex-rs/utils/home-dir/src/lib.rs)
  — resolves Codex configuration and state from the specified existing
  directory.
