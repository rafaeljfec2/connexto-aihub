# E07 Codex First Harness Validation Report

Status: **INCOMPLETE — AC-14 is OUT OF SCOPE; AC-5 remains failed by the latest real E2E evidence, so Codex CLI support is not established.**

Report date: 2026-10-31.

## Validation harness remediation (2026-10-31)

Status: helper and evidence remediation only. No E2E pass was executed in this
stage, the original and supplemental passes remain consumed, and this record
does not establish Codex CLI support or any real-provider acceptance criterion.
E07 remains INCOMPLETE.

The approved failure analysis identified confirmed validation defects plus
undetermined runtime causes. The following corrections are limited to the
opt-in E07 helpers and their tests; no production file was changed.

| Defect | Correction |
| --- | --- |
| Repository root was the Go package working directory | Root now resolves from Git metadata with a module-marker fallback, so the complete repository is fingerprinted. The live preflight compares the production paths against the recorded E06 baseline commit and rejects untracked or ignored production files instead of requiring an exact current `HEAD`, so committing the test-only E07 change set does not block a future authorized pass. |
| Integrity coverage missed root/nested `.env`, already-dirty tracked files, and untracked files | The snapshot now combines a full tree digest, environment-file discovery, tracked-file content, and a Git status digest. A content change to an already-dirty tracked file is detected even when `git status` output is unchanged. `.env.example` remains covered by the tree and tracked-file categories but is not treated as a secrets file. |
| Symlink targets below a directory were not hashed | In the repository snapshot, symbolic links now record the target and digest resolvable target content, with directory-link cycle protection and an entry bound. The separate Codex metadata snapshot instead records a nested link's target without following it. |
| Before-state fingerprints were memory-only | A private manifest with digests only is written with owner-only permissions under the isolated temporary root outside the repository, loaded for comparison, and removed with that root. A dedicated helper rejects a manifest directory inside the repository. Reports remain category-only. |
| Record outcome expectations were inferred from HTTP status | Expected outcomes now come from the validation-only E05 terminal observer (the same `FinishedExecution` value the E06 observer receives). A fully read upstream 2xx or 4xx is `completed` with its upstream status; transport and body-read failures keep their E05 failure outcomes. The pre-E05 authentication probe is excluded by captured request identifier, never by status, so an upstream 401 remains verifiable. |
| List failure suppressed the by-ID and restart checks | List, correlation, outcome, schema-v1 null fields, and by-ID lookup are evaluated independently. The by-ID query addresses the execution identifier directly, every query derives its own context bounded by the remaining poll window, and requests captured without an E05 terminal observation are counted separately. |
| Restart checks reused the pre-stop context | Post-restart verification derives a fresh context without the pre-stop deadline or cancellation. |
| Upstream error check compared bodies from independent provider calls | The real-provider probe now reports direct and Hub status separately, requires status equality, and classifies only the generic error envelope. Body equality is never asserted across independent calls. Exact status and byte preservation are proven by a deterministic fixed-upstream test, which also proves the content type and forwarded request bytes are unchanged. |
| Multi-turn replay collapsed several conditions into one result | Replay now reports separate outcomes for first turn, second turn, observed request count, marker recall, prior-turn context presence, and `previous_response_id` absence. The summary contains only structural booleans and aggregate counts, and the observer restores the original request bytes before forwarding. |

The live record assertion validates E06 projection, storage, correlation, and
recovery against the E05 value the service actually observed. E05 terminal
semantics themselves are proven separately by the deterministic cases that run
the real execution manager; the live assertion does not independently re-derive
E05 classification.

Focused deterministic tests added or extended:

- `TestE07RepositoryRoot*` — root discovery from a nested package directory,
  module-marker fallback, and rejection of an unrelated directory.
- `TestE07ProductionBaselineComparisonIgnoresTestOnlyChanges` — production
  paths are compared to the baseline, untracked and ignored production files are
  rejected, and test-only changes are ignored.
- `TestE07EnvFilesCoverRootAndNestedLocations` — root and nested environment
  file discovery with non-environment files and the template excluded.
- `TestE07RepositorySnapshotDetectsEveryRelevantChange` — unchanged baseline,
  pre-dirty tracked content, untracked file, root `.env`, and a nested symbolic
  link whose target is outside the repository.
- `TestE07IntegrityManifestIsPrivateAndRoundTrips`,
  `TestE07PrivateManifestDirectoryRejectsRepositoryLocations`, and
  `TestE07IntegritySummaryIsAggregatedAndRedacted` — private manifest placement
  and permissions, rejection of an in-repository location, and category-only
  output with no path, content, or digest.
- `TestE07FingerprintTreeTerminatesOnSymbolicLinkCycle` — deterministic
  symbolic-link cycle handling.
- `TestE07ExpectedRecordsDeriveOutcomeFromE05TerminalObservation` and
  `TestE07RecordVerificationFollowsE05TerminalSemantics` — completed upstream
  2xx, 4xx, and 401, transport failure, and body-read failure against the real
  E05 manager, E06 service, and E06 query handlers.
- `TestE07RecordVerificationKeepsAssertionsIndependent`,
  `TestE07RecordAssertionsSurviveParentCancellation`,
  `TestE07RecordQueryContextClampsToPollWindow`, and
  `TestE07RestartRecordVerificationUsesFreshContext` — independent assertions,
  per-query contexts clamped to the remaining poll window, and a fresh
  post-restart context, including `model_id` and `usage` null.
- `TestE07GatewayPreservesFixedUpstreamResponseDeterministically` — fixed
  upstream 400 and 429 responses preserve status, content type, and exact body
  bytes, and the forwarded request bytes are unchanged.
- `TestE07ReplayDiagnostics*` — every replay condition and failure branch,
  including the environment-limited branch.
- `TestE07UpstreamErrorEnvelopeClassification` and
  `TestE07UpstreamProbeRequiresSameResponseFidelity` — envelope classification
  and same-response status/content-type/digest observation. The supporting
  fidelity tests are `TestE07ResponseObserverComparesOneCompleteResponseAtBothBoundaries`
  and `TestE07ResponseObserverDoesNotAcceptIncompleteOrFailedReads`.

Local verification for this remediation:

| Check | Result |
| --- | --- |
| `go test ./cmd/ai-hub/ -count=1 -run TestE07` | PASS |
| `go test ./... -count=1 -race` | PASS |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `golangci-lint run ./...` | PASS — 0 issues |
| `go build ./...` | PASS |
| `gofmt -l cmd internal` | PASS — empty |
| `git diff --check` | PASS |

Repository secret scan: `NOT RUN — no approved scanner tooling is available`.
`gitleaks`, `trivy`, `detect-secrets`, `trufflehog`, and `gosec` are absent
from the environment and the repository configures no alternative scanner. No
tool was installed, because installation requires separate authorization. This
record does not claim that the repository is free of secrets.

Remaining undetermined items are unchanged: the source of the global Codex-tree
change, real-provider multi-turn replay behavior, real-provider same-response
body fidelity, real execution-record correlation and recovery, and AC-14. No
global Codex file was read beyond fingerprinting, modified, or restored.

Known limitations of the instrumentation remain explicit. The integrity entry
bound limits the number of traversed entries; file content is streamed without a
byte cap. The live scenario composes the same modules and lifecycle order as the
command entry point, but it is a separate validation composition, and its
injected transport does not inherit the production client's transport timeouts.
Record recovery is exercised only for normal operation and orderly restart, not
for overflow, storage failure, crash, retention, or flush expiry.

## Final test readiness remediation (2026-10-31)

Status: test-only preparation. No AI Hub process configured with real
credentials, Codex scenario, OpenCode Go request, or E2E pass was started. The
original and supplemental passes remain consumed. E07 remains **INCOMPLETE**,
and the six pending acceptance criteria remain unchanged: AC-1, AC-5, AC-7,
AC-9, AC-10, and AC-14.

### Test organization

The four E07 test files that exceeded 300 lines were split by responsibility.
All modified and new Go files remain at or below 300 lines.

| Area | Resulting files and line counts |
| --- | --- |
| Repository snapshot and manifest | `e07_integrity_remediation_test.go` (239), `e07_integrity_manifest_test.go` (189), `e07_integrity_test.go` (209), `e07_integrity_manifest_support_test.go` (82), `e07_integrity_tree_test.go` (249) |
| E06 HTTP record lookup and assertions | `e07_live_http_test.go` (153), `e07_live_record_assertions_test.go` (198) |
| E05/E06 record behavior and contexts | `e07_records_test.go` (153), `e07_record_outcomes_test.go` (83), `e07_record_assertions_test.go` (182), `e07_record_context_test.go` (74) |
| Provider-call budget and transport | `e07_provider_budget_support_test.go` (66), `e07_provider_observer_test.go` (138), `e07_provider_budget_test.go` (216) |
| Upstream response and same-response fidelity | `e07_response_fidelity_test.go` (154), `e07_upstream_response_test.go` (85) |
| Observer and scenario orchestration | `e07_observer_support_test.go` (83), `e07_live_hub_test.go` (188), `e07_live_scenario_support_test.go` (214), `e07_live_scenarios_test.go` (204), `e07_live_auth_support_test.go` (43), `e07_live_auth_test.go` (29), `e07_provider_timeout_test.go` (38) |

Of the 32 existing E07 test functions, 31 remain unchanged by name through the
splits. The former cross-call error-probe test was replaced by a stricter
same-response fidelity test, and ten deterministic budget, transport, fidelity,
and timeout tests were added. No test was removed to reduce file size or to make
the suite pass.

### Hard provider-call budget

The injected E07 transport now reserves one unit under a mutex before calling
the underlying transport. The shared budget accepts configured limits from one
through nine and has no reset operation. Its snapshot distinguishes attempted,
admitted, rejected, and forwarded calls. The same observer instance is retained
across the planned Hub restart, so readiness requests consume the same budget.
The validation transport disables keep-alive reuse to prevent `http.Transport`
from retrying an idempotent request on a previously used connection; redirect
attempts still pass through the budget. The Hub error probe also refuses
redirects so a provider response cannot cause a client-side request outside the
observed boundary. The test Hub's provider `http.Client` has an explicit
two-minute timeout, including while reading a streaming response body, matching
the per-request limit in the final E2E plan. A deterministic cancellation test
uses a 20 ms timeout and confirms the fake transport receives context
cancellation after exactly one forwarded attempt.

Red evidence from the pre-implementation tests showed 10 sequential calls, 50
concurrent calls, and 10 redirect attempts reaching the fake underlying
transport. Green tests observe exactly nine underlying calls in each case.
Additional deterministic tests verify limit validation, the four counters,
startup and restart readiness accounting, zero budget change for an
unauthenticated `401`, shared state across observer instances, no automatic
transport retry, and same-response status/content-type/digest matching only
after complete reads through EOF. Incomplete reads and read errors remain
unverified. Digests and response bytes are discarded after comparison; the
scenario summary contains only booleans and the aggregate provider-call count.

The former two-call upstream error probe now makes one authenticated Hub
request and observes that same upstream response at the provider transport and
Hub client boundaries. Its provider-call count must equal one for a pass. This
change aligns the helper with the approved final execution plan and does not
change the historical E2E results above.

### Local verification

The gates below ran with `AIHUB_E07_LIVE` and `AIHUB_LIVE_SMOKE` unset. They
used only local fakes and deterministic tests; no live provider credential or
OpenCode Go request was used.

| Check | Result |
| --- | --- |
| `go test ./cmd/ai-hub -count=1 -run TestE07` | PASS |
| `go test ./cmd/ai-hub -run '^TestE07ProviderHTTPClientCancelsRequestAtTimeout$' -count=1` | PASS — fake transport observed cancellation; one attempt was forwarded |
| `go test ./... -count=1 -race` | PASS |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `golangci-lint run ./...` | PASS — 0 issues |
| `go build ./...` | PASS |
| `gofmt -l cmd internal` | PASS — empty |
| `git diff --check` | PASS; untracked E07 Go files also passed a separate whitespace check |

No production source, API, dependency, ADR, or E01–E06 behavior was changed.
The E06 baseline remains the production comparison target.

### Environment limitations and remaining acceptance work

- Approved secret scanner: **BLOCKED — APPROVED SECRET SCANNER UNAVAILABLE**.
  `gitleaks`, `trufflehog`, and `detect-secrets` were unavailable; no tool was
  installed. This does not establish that the repository is secret-free.
- Gateway credential: **BLOCKED — GATEWAY CREDENTIAL NOT AVAILABLE**. The
  presence-only check found it absent. No value was requested or read.
- Provider credential: present by presence-only check; its value was not read
  or used.
- Snyk Code scan: not completed because the Snyk MCP reported that the account
  is unauthenticated. This is separate from the approved secret-scanner check.
- Independent `senior-code-review-judge` review: Round 1 and Round 2 were
  **APPROVED** with no critical or important findings. Round 2 confirmed the
  two-minute provider HTTP timeout, the shared nine-call limit, the test split,
  and the evidence updates. The review did not execute tests or live scenarios.

Before a separately authorized E2E pass, provide the distinct gateway key to
the Hub and Codex through the approved non-echoing secret facility, and make an
approved secret scanner available. Do not send either credential through chat.
These test-only results do not establish any real-provider acceptance
criterion, and no support claim is permitted.

## Supplemental E2E pass — executed once

On 2026-10-31, the project lead authorized exactly one additional real E2E
pass beyond the original one-pass budget. The authorization remains limited to
validation and evidence documentation. An earlier preflight found `bwrap`
unavailable and did not start a scenario. After the environment was prepared,
the preflight below passed and the single authorized pass was executed. It is
now consumed; no retry was made.

| Preflight item | Result |
| --- | --- |
| AI Hub revision | `0d33ca26c15acf54fac418cb33380b5871b4d899` (E06 baseline) |
| Go version and environment | `go1.23.0 linux/amd64` |
| Provider credential presence | `PRESENT` (presence-only check; value not read into output) |
| `bwrap` availability | `PRESENT`, `bubblewrap 0.4.0` |
| Linux sandbox initialization | `PASS`; the live tool scenario also completed under Codex `workspace-write` sandboxing |
| Codex CLI | `codex-cli 0.164.0-alpha.11.1` |
| Isolated `HOME` and `CODEX_HOME` | `PASS`; both used dedicated temporary state for the version probe and scenario child processes |
| Selected model | `deepseek-v4.1-flash` |
| Initial integrity state | Recorded before the isolated Codex version probe |
| Additional pass consumed | `YES — exactly one invocation` |

| Required scenario | Result | Evidence and limitation |
| --- | --- | --- |
| Basic generation through Codex | `PASS` | Codex received a successful non-empty response through the Hub. |
| Multi-turn full replay | `FAIL` | The combined synthetic-marker recall and structural replay check did not pass. No payload was retained. |
| Local tool round trip under the real sandbox | `PASS` | The bounded temporary-workspace write/read task completed with `workspace-write`; no sandbox or approval bypass was used. |
| Authenticated Codex route | `PASS` | Four Codex Responses requests reached the Hub; the unauthenticated probe returned `401` without increasing the provider-boundary counter. |
| Upstream error passthrough | `FAIL` | The probe returned status `400`, but the generic error-envelope comparison did not pass. Response content is not published. |
| Gateway-hop streaming | `PASS` | The Hub client observed 142 SSE data events. The first arrived at about 1.271 s; the stream completed at about 3.720 s, with multiple events before completion. |
| Codex stdout progressive rendering | `NOT VERIFIED` | The CLI output did not provide a measurable progressive-rendering signal. This remains non-blocking under AC-13 because gateway-hop streaming passed. |
| Execution record capture and query | `FAIL` | Six records were observed in aggregate, but the expected correlation and schema/outcome checks did not pass. Generated identifiers and record contents are omitted. |
| Graceful shutdown, restart, and record drain | `FAIL` | The post-restart query did not confirm the expected retained records. |
| Credential isolation | `PASS` | Presence-only checks confirmed the Codex child received the gateway key and excluded the provider key and Hub-only variables. |

The run reported nine aggregate provider-boundary calls and four Codex
Responses requests. It exited unsuccessfully because required scenarios failed.
Execution records remain best-effort and at-most-once; the writer drain window
is two seconds, and queue overflow, storage failure, crash, retention, or flush
expiry can lose records.

| Additional-pass integrity comparison | Result |
| --- | --- |
| Complete global Codex tree | `FAIL` — the tree changed during the pass; the source of the change was not established. |
| Existing project `.env` files | `PASS` |
| Repository worktree during the pass | `PASS` |
| Git state during the pass | `PASS` |
| Repository secret scan | `NOT RUN` — `gitleaks` was unavailable. |

The independent before/after comparison from the preflight baseline likewise
reported a changed global Codex tree and unchanged project environment files,
repository worktree, and Git state. The integrity result does not establish
AC-14. The report contains no credential values, prompts, responses, paths,
hashes, session IDs, request IDs, execution IDs, or other generated identifiers.

## Original real-provider pass environment

| Field | Observed value |
| --- | --- |
| AI Hub revision | `0d33ca26c15acf54fac418cb33380b5871b4d899` |
| Go | `go1.23.0` |
| Codex CLI | `codex-cli 0.164.0-alpha.11.1` |
| Environment class | Linux `amd64` |
| Selected model | `deepseek-v4.1-flash` |
| Linux sandbox helper | `bwrap` unavailable |

Credentials were checked by presence only. The provider credential came from
the process environment as `OPENCODE_API_KEY`; a temporary gateway key was
generated in memory. The Codex child environment contained the gateway key
under `AIHUB_CLIENT_KEY` and excluded the provider credential and Hub-only
variables. The isolated TOML contained the environment-variable name, not a
credential value. No prompt, response, tool content, credential, request ID,
provider-session ID, execution ID, or integrity fingerprint was saved here.

## Original pass scenario results

| Scenario | Result | Evidence and limitation |
| --- | --- | --- |
| Basic generation through Codex | `FAIL` | Codex returned no successful non-empty result. No Codex Responses request reached the Hub. |
| Multi-turn full replay | `FAIL` | No Codex Responses request reached the Hub; the replay observer could not establish prior-turn context. |
| Local tool round trip | `NOT VERIFIED` | The tool task did not complete and no Codex Responses request reached the Hub. The environment lacked `bwrap`; sandbox bypass was not used. Tool support was not established. |
| Authenticated Codex route | `FAIL` | The Codex scenario generated zero Responses requests at the Hub. |
| Unauthenticated gateway rejection | `PASS` | The live loopback gateway returned `401`; the provider transport counter did not increase. A separate local test with a synthetic provider transport also passed. |
| Upstream error passthrough | `FAIL` | The gateway returned status `400`, but the direct-upstream and gateway error-envelope comparison did not pass. Body passthrough is unverified. |
| Gateway-hop streaming | `PASS` | The Hub client observed 147 SSE data events. The first arrived at 1.221 s, the last at 3.851202 s, and the response completed at 3.851202 s. Multiple events arrived before completion. |
| Codex stdout progressive rendering | `NOT VERIFIED` | The CLI run produced no measurable progressive-rendering signal. |
| Execution record capture and query | `FAIL` | Two records were returned, but the expected request correlation and schema-v1 outcome checks did not pass. No record-content evidence is published. |
| Graceful shutdown, restart, and record drain | `FAIL` | The post-restart query did not confirm the expected retained records. |

The provider-boundary observer counted five transport calls in aggregate. This
count includes provider startup checks, the direct and gateway error probes,
and the streaming request; it does not include a Codex Responses request.

## Original pass environment integrity

| Category | Result |
| --- | --- |
| Complete global Codex tree | `FAIL` |
| Existing project environment files | `PASS` |
| Repository worktree | `PASS` |
| Git scope | `PASS` |

The before/after check detected a change in the global Codex tree. It emitted
no path, content, or fingerprint. The original pass did not establish AC-14.
After it, the validation helper was changed so both `HOME` and `CODEX_HOME` use
temporary state and the version probe runs inside that isolated environment.
The later supplemental pass exercised the updated helper; its separate
integrity results appear above and also report a changed global Codex tree.

## Conclusion

E07 remains incomplete, and Codex CLI support is not established. The
supplemental pass passed basic generation, the authenticated Codex route,
credential isolation, the sandboxed tool round trip, unauthenticated rejection,
and gateway-hop streaming. It failed multi-turn replay, upstream error
passthrough, execution-record capture/query, and shutdown/restart/drain.
Codex-side progressive rendering remains `NOT VERIFIED`. The global Codex tree
changed during the pass, and no repository secret scanner was available, so
AC-14 is not established. The README continues to label the setup experimental
and makes no Codex support claim.

The validation harness remediation above corrects the confirmed helper defects
and adds deterministic evidence for them. It does not re-establish any
real-provider acceptance criterion: AC-1, AC-5, AC-7, AC-9, AC-10, and AC-14
still require a newly authorized real validation pass. The global Codex-tree
change source remains `UNDETERMINED`.

Execution-record guarantees remain E06 best-effort and at-most-once. The
writer's drain window is two seconds; queue overflow, storage failure, crash,
retention, or flush expiry can lose records. This run does not establish
preservation outside normal orderly shutdown.

## Final implementation and acceptance checkpoint — 2026-11-01

Status: **INCOMPLETE — final real-provider acceptance was not started.** The
two historical E2E passes remain consumed. The user authorized one further
pass conditionally, but no provider-boundary request was made in this
checkpoint; the new-pass count is `0/1` and its provider-call count is `0/9`.
The two consumed passes account for 14 historical provider-boundary calls
(five in the original pass and nine in the supplemental pass); no provider
call was made in this checkpoint. The loopback fake Responses server used by
the synthetic canary is not part of that provider-boundary count.

### Git state and baseline

| Check | Result |
| --- | --- |
| Branch | `main` |
| HEAD | `56cbd0b509494d452e2a1e9472ef3e01830792d7` |
| `origin/main` | `a8ac19af7cf6457e30f3eb284965c4966ecf033b` |
| Remote `main` | Matches `origin/main` at `a8ac19af7cf6457e30f3eb284965c4966ecf033b` |
| Local divergence | One commit ahead, zero behind |
| Remediation commit | `40196001536b6d07084b8bbd671d646bdd674d32` is an ancestor of HEAD |
| Staged changes | None |
| Worktree | Dirty; seven tracked files modified and seven new E07 test files untracked |
| `.env` worktree state | No modified or untracked `.env` paths reported; ignored values were not read |
| Approved immutable baseline | **NOT ESTABLISHED** |

Checkpoint runtime: Go `go1.23.0 linux/amd64`; environment `Linux amd64`;
Codex CLI `codex-cli 0.164.0-alpha.11.1` as observed by the isolated synthetic
canary; `bwrap` is present. The prior supplemental real-provider pass used
`deepseek-v4.1-flash`. No model-availability probe or provider request was made
in this checkpoint.

The Phase 8B inventory contained 29 E07 artifacts. Four existing test helpers
and the README are modified; seven new test files are untracked; the two
validation reports are updated. These 14 pending files were observed at that
checkpoint; the execution-plan authorization update recorded later in this
report added `docs/validation/E07-final-e2e-execution-plan.md`, for 15 in
total:

- `README.md`
- `cmd/ai-hub/e07_live_codex_test.go`
- `cmd/ai-hub/e07_live_scenarios_test.go`
- `cmd/ai-hub/e07_live_setup_test.go`
- `cmd/ai-hub/e07_observer_support_test.go`
- `cmd/ai-hub/e07_codex_env_policy_support_test.go`
- `cmd/ai-hub/e07_codex_env_policy_test.go`
- `cmd/ai-hub/e07_codex_environment_support_test.go`
- `cmd/ai-hub/e07_codex_policy_provider_test.go`
- `cmd/ai-hub/e07_codex_process_linux_test.go`
- `cmd/ai-hub/e07_codex_process_other_test.go`
- `cmd/ai-hub/e07_codex_tree_manifest_test.go`
- `docs/validation/E07-codex-first-harness-integration.md`
- `docs/validation/E07-environment-blocker-resolution.md`

This report update does not stage, commit, or discard any artifact. The full
E07 worktree therefore cannot be treated as a clean approved baseline for
provider traffic.

### Codex subprocess policy and local canary

The [official Codex configuration reference](https://developers.openai.com/codex/config-reference/)
documents
`shell_environment_policy.inherit` (`all`, `core`, or `none`), `set` as a
string map, `ignore_default_excludes`, and `experimental_use_profile`. The
validation helper now writes `inherit = "none"`, an explicit non-secret
allowlist in `set`, `ignore_default_excludes = false`, and
`experimental_use_profile = false`. Codex's parent environment is built from
an allowlist and supplies the gateway credential only under `AIHUB_CLIENT_KEY`;
temporary `HOME`, `CODEX_HOME`, temporary directories, and XDG config/data/cache/
state/runtime roots are asserted to remain under one temporary root. Normal
`workspace-write` sandboxing remains enabled, no approval override is written,
and the tool invocation does not bypass approvals or sandboxing.

`TestE07CodexShellPolicyDisablesUserProfile` first failed because the generated
policy omitted the profile setting, then passed after the setting was added.
`TestE07CodexEnvironmentExcludesProviderCredentials` passed with synthetic
values and checks the parent and shell-environment builders against
`OPENCODE_API_KEY` and `AIHUB_PROVIDER_OPENCODEGO_API_KEY`. These unit tests do
not establish effective CLI behavior by themselves.

| Regression test | Red evidence | Green correction |
| --- | --- | --- |
| `TestE07CodexTreeRootsTrackOnlyCodexManagedLocations` | The manifest included XDG locations outside the Codex-managed roots. | Snapshot roots now follow the installed Codex state locations; the test passes. |
| `TestE07CodexTreeMetadataExcludesDirectoryAccessTimes` | Tree enumeration changed directory access metadata and made a stable tree appear modified. | Access-time fields are excluded from stable metadata comparison; the test passes. |
| `TestE07StableCodexPreflightRejectsActiveProcess` | Preflight proceeded while a Codex process was active. | Preflight now fails closed before taking the baseline; the test passes. |
| `TestE07CodexShellPolicyDisablesUserProfile` | The generated shell policy omitted the setting that prevents user-profile loading. | The policy explicitly sets `experimental_use_profile = false`; the test passes. |
| `TestE07CodexTreeSnapshotRejectsSymlinkStateRoot` and `TestE07CodexTreeSnapshotRecordsNestedSymlink` | Snapshots accepted symlinks without enumerating their targets. | A symbolic-link state root now rejects the snapshot; a nested link is recorded with its target, and retargeting it fails the comparison. Both tests pass. |
| `TestE07CodexProcessInspectionFailsClosedOnPermissionError` | The Linux process probe treated a permission error as proof that no Codex process was active. | Only a vanished process entry is ignored; unreadable process state returns an error. The test passes. |
| `TestE07CodexProcessUnsupportedPlatformFailsClosed` | Unsupported process inspection returned `false, nil`. | Unsupported platforms return an error, and the non-Linux test package cross-compiles. |

An isolated local synthetic canary reached Codex CLI
`codex-cli 0.164.0-alpha.11.1` and a loopback-only fake Responses server. Its
runtime-path, configuration-redaction, gateway authentication, sandbox, and
shell-child assertions passed for the variables included in that run; no real
provider or AI Hub process was involved. One canary run found the global Codex
tree unchanged; a later race-suite run detected a global-tree change during
cleanup. The before/after manifests remained private and were deleted with the
temporary test root. Before that private manifest was established, one
`codex --version` probe ran in the inherited environment; it made no provider
request and its effect on global state cannot be measured retrospectively. No
global Codex file contents were read or copied, and no global file was
restored or edited by this agent. A later race-suite comparison detected a
global-tree change while Codex was active. The source cannot
be attributed, so global-tree integrity is **FAIL** and its cause is
**INCONCLUSIVE**. The final helper retains a fail-closed preflight: when
another Codex process is active or the tree is unstable, validation stops
before launching the CLI. No further Codex probe was run after the global-tree
change was observed. Private manifest paths, hashes, contents, and identifiers
are not included in this report.

The effective canary run predates adding the AI Hub's exact provider-key
variable to the shell-child observation. The new unit test covers that
variable at the environment-builder boundary. **Superseded:** the later
isolated canary section (2026-11-01) observed the exact
`AIHUB_PROVIDER_OPENCODEGO_API_KEY` name as `ABSENT` in the Codex shell child
inside the private namespace, so this limitation no longer applies to the
policy proof.

### Local gates and scanner

All test commands explicitly removed `AIHUB_E07_LIVE` and real credential
variables from the test process. The focused E07 run did not complete: the
stable-tree preflight stopped before another Codex launch while a Codex process
was active. The full race run reached the loopback synthetic canary, then
failed its final global-tree comparison; other packages passed. Subsequent
fail-closed preflight and environment-policy regression tests passed, but the
Codex CLI canary was not repeated. Thus the required focused and full test
gates are not `PASS`. Other final gate results are:

| Check | Result |
| --- | --- |
| `go test ./cmd/ai-hub -count=1 -run TestE07` | `BLOCKED` — no passing completion; the focused run stopped at the fail-closed preflight, and the suite was not rerun because it includes the Codex CLI canary after a global-tree change was observed |
| `go test ./... -count=1 -race` | `FAIL` — local synthetic canary detected a global-tree delta during cleanup; other packages passed |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` |
| Independent readonly review | `APPROVED` — confirmed the restored `.env` assertion, symlink no-read behavior, local resume fix, AC-14 limitation, and file scope; no tests or E2E were run by the reviewer |
| Selected deterministic E07 budget, replay, fidelity, records, environment, auth, timeout, manifest, and process tests | `PASS` — 32 tests; Codex CLI and AI Hub were not started |
| Darwin E07 test package cross-compilation | `PASS` — test binary compiled; temporary binary removed |
| E07 modified/created Go files below 300 lines | `PASS` — the largest E07 Go file is 278 lines |
| Repository-wide Go file limit | `FAIL` — eight pre-existing files exceed 300 lines; none is part of this E07 scope |
| Snyk Code scan | `BLOCKED` — Snyk MCP reports that the account is unauthenticated |
| Approved repository secret scan | `NOT RUN` — no scanner approval or verified scanner release is available |

Snyk Code is not treated as a secret scanner. Gitleaks was not installed,
executed, or approved. No repository secret-free claim is made. No existing
tests were removed or weakened, and no production source file or E01–E06
contract was changed.

The repository-wide 300-line rule is not satisfied by existing files:
`internal/catalog/manager_test.go` (877), `internal/catalog/store_test.go`
(593), `internal/gateway/catalog_handler_test.go` (575),
`internal/gateway/responses_handler_test.go` (454),
`internal/catalog/capability_test.go` (442), `internal/catalog/manager.go`
(397), `internal/gateway/catalog_handler.go` (386), and
`internal/catalog/store.go` (312). They were not changed because they are
outside the authorized E07 scope and include E01–E06 production paths.

### Independent review

The first `senior-code-review-judge` round returned `REJECTED` with two
`IMPORTANT` findings: symlink targets were not represented in the snapshot,
and process inspection could report inactive when inspection was unsupported
or denied. Both findings were corrected in the E07 harness. The second round
returned `APPROVED_WITH_NOTES` (`SDD APPROVED`), with no open code findings.
The reviewer did not run a Codex canary, start AI Hub, or contact a provider.
A separate documentation review found that the README initially passed
credentials through `go run` and later found a non-temporary build home and
incomplete cleanup. The README now builds Hub and test binaries with a
temporary home/cache and no credentials, launches them directly through the
broker, and removes temporary build/data directories. The independent
follow-up found no open findings in the revised README setup section. These
review results do not establish acceptance scenarios or change E07's
`INCOMPLETE` status.

### Current credential states

Presence-only checks of the current tool environment report:

| Variable | State |
| --- | --- |
| `AIHUB_GATEWAY_API_KEY` | `ABSENT` |
| `AIHUB_CLIENT_KEY` | `ABSENT` |
| `AIHUB_PROVIDER_OPENCODEGO_API_KEY` | `ABSENT` |
| `OPENCODE_API_KEY` | `PRESENT` |

The present `OPENCODE_API_KEY` was not read, printed, copied, or used. The
intended Hub and Codex launch environments do not exist in this checkpoint and
remain `NOT_VERIFIABLE`; no Hub was started. The operator must provide new,
distinct credentials through an approved non-interactive secret facility and
verify the exact Hub, Codex, and tool-child environments before any E2E start.

### Cumulative acceptance matrix

Historical scenario outcomes remain intact. A deterministic test does not
replace the real-provider evidence required by the SPEC.

| AC | Result | Evidence and limitation | Failure cause |
| --- | --- | --- | --- |
| AC-1 | `FAIL` | Previous real setup passed several scenarios, but global-tree integrity failed; complete process separation is not approved, and the exact AI Hub provider-key variable was not observed in a real Codex child. | `INCONCLUSIVE` |
| AC-2 | `PASS` | The supplemental pass recorded the tested release/model and routed its Codex requests through the Hub. The local canary reconfirmed the release, but this checkpoint did not repeat the real scenario route. | None recorded |
| AC-3 | `PASS` | The supplemental pass observed authenticated routing and invalid-auth rejection without a provider-call delta. Not repeated in this checkpoint. | None recorded |
| AC-4 | `PASS` | Basic generation succeeded in the supplemental real-provider pass. Not repeated in this checkpoint. | None recorded |
| AC-5 | `FAIL` | The supplemental real replay failed; current structural tests preserve separate predicates but no real replay was run. | `INCONCLUSIVE` |
| AC-6 | `PASS` | The supplemental pass completed the bounded sandboxed tool scenario with normal controls. Not repeated in this checkpoint. | None recorded |
| AC-7 | `FAIL` | The supplemental pass returned HTTP 400 but did not establish same-response envelope fidelity. Deterministic 400/429 response tests pass; no real error probe was run. | `INCONCLUSIVE` |
| AC-8 | `PASS` | The supplemental pass observed progressive gateway-hop streaming. Codex stdout rendering remains `NOT_VERIFIED`, permitted by AC-8. | None recorded |
| AC-9 | `FAIL` | Prior live records were not correlated and verified as required. Deterministic E05/E06 record assertions pass; no new live records were queried. | `INCONCLUSIVE` |
| AC-10 | `FAIL` | Prior post-restart recovery verification failed. Deterministic fresh-context tests pass; no live restart was performed. | `INCONCLUSIVE` |
| AC-11 | `PASS` | This report records versions, cumulative scenario results, gate outcomes, limits, and redacted environment states without payloads or generated IDs. | None recorded |
| AC-12 | `BLOCKED` | Static analysis, lint, build, format, and diff checks pass. The focused E07 run stopped at the active-Codex preflight; the full race run failed the global-tree comparison. The required Go test gates have no passing completion. All changed E07 Go files are below 300 lines, but eight pre-existing repository files exceed the requested global limit. | `ENVIRONMENT_DEFECT` |
| AC-13 | `PASS` | README remains explicit that support is experimental and not established; no support claim was added. | None recorded |
| AC-14 | `FAIL` | A local synthetic run detected a changed global Codex tree; the secret scan and approved baseline are also absent. No paths, contents, hashes, or identifiers are reported. | `INCONCLUSIVE` |

### E2E scenarios and operator actions

| Scenario | Checkpoint result | Historical supplemental result |
| --- | --- | --- |
| Basic Codex generation | `BLOCKED` — no final-pass invocation | `PASS` |
| Multi-turn replay | `BLOCKED` — no final-pass invocation | `FAIL` |
| Sandboxed tool calling | `BLOCKED` — no final-pass invocation | `PASS` |
| Invalid authentication | `BLOCKED` — no final-pass invocation | `PASS` |
| Upstream error passthrough | `BLOCKED` — no final-pass invocation | `FAIL` |
| Streaming | `BLOCKED` — no final-pass invocation | `PASS` |
| Execution-record list and lookup | `BLOCKED` — no final-pass invocation | `FAIL` |
| Orderly shutdown, restart, and recovery | `BLOCKED` — no final-pass invocation | `FAIL` |

Before the one conditionally authorized final pass can start, the operator
must: (1) establish an approved immutable baseline and review the current
uncommitted E07 files; (2) provide governance approval and a verified release
for the secret scanner, then complete a redacted scan; (3) inject new distinct
gateway/provider credentials through a secure non-interactive mechanism and
prove the exact process presence states; and (4) close other Codex sessions and
run the isolated preflight from a stable environment so before/after global
tree integrity can pass. The one-pass, nine-call, 30-minute authorization is
otherwise unchanged and remains unused. Do not retry or start a provider-bound
scenario until every precondition passes.

## Conditional final acceptance preflight — 2026-11-01

The project lead's current final acceptance prompt conditionally authorizes one
final pass with a maximum of nine provider-boundary calls, 30 minutes total,
two minutes per provider request, and zero retries. This supersedes the earlier
authorization statements in the plan and blocker report. The operational
preflight did not pass, so no AI Hub process, Codex scenario, or provider-boundary
request was started. This attempt made zero real provider calls.

| Precondition | Result | Evidence |
| --- | --- | --- |
| Approved immutable baseline | `BLOCKED` | The initial checkout had 14 pending files (seven modified and seven untracked); `HEAD` was `56cbd0b509494d452e2a1e9472ef3e01830792d7`. No clean approved baseline was established. |
| Approved scanner and redacted baseline scan | `BLOCKED` | No scanner approval was found; Gitleaks is not available on `PATH`. No scanner was installed or run, and no secret-free claim is made. |
| New distinct credentials | `BLOCKED` | `AIHUB_GATEWAY_API_KEY`, `AIHUB_CLIENT_KEY`, and `AIHUB_PROVIDER_OPENCODEGO_API_KEY` were absent in the tool process. `OPENCODE_API_KEY` was present but not read or used; its provenance and distinction from a newly provisioned key are not verifiable. |
| Exact Hub, Codex, and tool-child environments | `NOT_VERIFIABLE` | No dedicated Hub or Codex process environments were created. All local gates explicitly removed the credential variables. |
| Codex isolation and stable global tree | `BLOCKED` | `CODEX_HOME` and XDG config/data/state/cache variables were absent. The E07 process guard read process names only to detect an active Codex process, then refused to start the isolated canary. No process details were inspected beyond that boolean and no process was terminated. |
| Sandbox | `PASS`, limited | Codex CLI `0.164.0-alpha.11.1` and bubblewrap `0.4.0` were available. A harmless command completed inside user, PID, and network namespaces. Codex sandbox and approval behavior were not exercised. |
| Local resources | `PASS`, limited | Go `1.23.0`, `staticcheck`, and `golangci-lint` were available; port 8080 was free and temporary storage was writable. Provider/model availability was not verifiable without a provider request. |
| Conditional pass budget | `PASS`, authorization only | The current prompt allows one pass, at most nine real provider calls, 30 minutes, two minutes per request, and zero retries. No persistent live counter was exercised; real calls in this attempt: zero. |

Local gates ran with all AI Hub and Codex credential variables removed:
`AIHUB_E07_LIVE` and `AIHUB_LIVE_SMOKE` were absent, so no live scenario was
enabled.

| Check | Result |
| --- | --- |
| `go test ./cmd/ai-hub -count=1 -run TestE07` | `FAIL` — `TestE07CodexShellEnvironmentPolicyIsEffective` failed closed because an active Codex process prevented global-tree validation; the isolated Codex was not started. |
| `go test ./... -count=1 -race` | `FAIL` — the same environment guard failed in `cmd/ai-hub`; all other listed Go packages passed. |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` — rerun after the documentation updates |

The existing AC-1 through AC-14 matrix remains cumulative historical evidence.
No final-pass scenario was verified in this preflight. AC-1, AC-5, AC-7, AC-9,
AC-10, and AC-14 remain unresolved; AC-12 remains blocked because the required
test gates did not complete successfully. E07 remains `INCOMPLETE`.

Operator actions required before the authorized pass can start:

1. Review the 15 currently pending files (the initial checkout had 14 before
   this authorization update) and establish a clean, immutable approved
   baseline without discarding work.
2. Obtain governance approval for a scanner, make its approved version
   available, and complete the redacted scan of the baseline.
3. Provision distinct, newly rotated credentials through separate
   non-echoing prompts or an approved secret facility and verify the exact Hub,
   Codex, and tool-child environments by presence only.
4. Close other Codex sessions, then repeat the isolation preflight with a
   dedicated `HOME`, `CODEX_HOME`, XDG directories, and temporary workspace;
   establish stable global-tree integrity while preserving normal sandbox and
   approval controls.
5. Confirm provider/model availability and the shared call/time budget without
   an unbudgeted provider request. If that cannot be established, keep the E2E
   blocked.

## Configuration evidence and isolated harness proof — 2026-11-01

Status: preflight evidence only. No AI Hub process was started for a scenario,
no Codex request reached a real provider, and no provider-boundary call was
made. The single conditionally authorized pass remains unused: `0/1` passes and
`0/9` provider calls consumed. This section does not establish Codex CLI
support. E07 remains `INCOMPLETE`.

### Production credential path

The credential question was resolved from production code rather than from
local configuration conventions:

| Step | Evidence |
| --- | --- |
| Environment loading | `cmd/ai-hub/main.go` loads the dotenv file (default `-env-file .env`) through `config.LoadDotEnv` before `config.Load`. Values already present in the process environment win over file values. |
| Provider key variable | `internal/config/types.go` declares `EnvProviderOpenCodeGoAPIKey` (`AIHUB_PROVIDER_OPENCODEGO_API_KEY`) as the only variable read for the OpenCode Go provider key. `OPENCODE_API_KEY` is not referenced by any production file. |
| Startup probe | A temporary build started with `-env-file .env` failed closed with `ai-hub: gateway.api_key is invalid: api key must contain between 32 and 512 ASCII characters`. This proves the dotenv file was read and validated by the real configuration path; it also proves the file does not provide a usable gateway key. No provider request was attempted. |

Presence states were collected without reading values. The `.env` file was
inspected by variable name only, without printing, echoing, hashing, or
measuring any value, and was not modified:

| Variable | In `.env` | In the agent process |
| --- | --- | --- |
| `AIHUB_PROVIDER_OPENCODEGO_API_KEY` | `DECLARED` — one declaration | `ABSENT` |
| `OPENCODE_API_KEY` | `DECLARED` — one declaration, not consumed by production code | `ABSENT` |
| `AIHUB_GATEWAY_API_KEY` | `ABSENT` | `ABSENT` |
| `AIHUB_CLIENT_KEY` | `ABSENT` (injected at launch only) | `ABSENT` |

No duplicate declarations were found. `.env.example` documents
`AIHUB_GATEWAY_API_KEY` as mandatory and does not reference `OPENCODE_API_KEY`
as a provider credential.

### Provider credential source corrected

The approved design allowed the live setup helper to read the provider
credential from `AIHUB_PROVIDER_OPENCODEGO_API_KEY`, the existing
`OPENCODE_API_KEY`, or `.env`. Under the current authorization that fallback is
not acceptable: only the production-consumed variable may be treated as the
provider credential, because equivalence between `OPENCODE_API_KEY` and the
OpenCode Go key is unproven. The old code also copied a legacy-sourced value
into `AIHUB_PROVIDER_OPENCODEGO_API_KEY`, which would have masked a missing
production configuration and made the run appear provisioned.

`e07ProviderCredential` now reads only `AIHUB_PROVIDER_OPENCODEGO_API_KEY` and
returns an empty value otherwise. `TestE07ProviderCredentialUsesOnlyProductionVariable`
is the regression test: the historical name yields an empty value, the
production name resolves, and an absent environment yields an empty value. The
setup also logs `provider_credential_origin=environment|dotenv` without printing
any value, so a future report can distinguish an operator-provisioned key from
one loaded out of `.env`. The report does not treat a `.env`-sourced key as
provisioning evidence.

### Isolated Codex process-policy proof

The previously failing `TestE07CodexShellEnvironmentPolicyIsEffective` guard
refuses to start an isolated Codex CLI while an ambient Codex process could
change global state, because a before/after comparison would then be
unprovable. Four VS Code ChatGPT extension `codex` app-servers were active on
the host. No assertion was relaxed, no guard was bypassed in-process, and no
third-party process was signalled or terminated. The documented procedure is
to run the canary inside a private PID and mount namespace with a fresh `HOME`:

```text
bwrap --unshare-pid --unshare-ipc --unshare-uts --ro-bind / / \
  --dev /dev --proc /proc --tmpfs /tmp --tmpfs /run \
  --bind "$WORK" /run/e07 \
  --setenv HOME /run/e07/home --setenv TMPDIR /run/e07/tmp \
  --setenv GOTOOLCHAIN local --setenv AIHUB_E07_CANARY 1 \
  --setenv PATH "<go and codex bin dirs>" \
  --chdir <repo> go test ./cmd/ai-hub -run TestE07 -count=1
```

`/tmp` must be a fresh writable tmpfs; otherwise the nested Codex sandbox fails
on a read-only `/tmp`.

The canary is opt-in through `AIHUB_E07_CANARY=1`. Without it the default suite
skips the canary, so `go test ./...` no longer depends on the host's Codex
processes; the real E2E helper remains separately gated by `AIHUB_E07_LIVE=1`.
This matches the design's "opt-in validation test helper" boundary. The canary
targets Linux; on its supported platform, enabling it fails closed on every
precondition failure, including a missing CLI, an active Codex process, a
version mismatch, sandbox failure, or an unstable tree.

Result inside the namespace, with the recorded CLI available on `PATH`:

| Observation | Result |
| --- | --- |
| `TestE07CodexShellEnvironmentPolicyIsEffective` | `PASS` (0.97s) — no skip, no relaxation |
| Codex CLI release | `codex-cli 0.164.0-alpha.11.1` — matches the E07 target |
| Gateway authentication | The local synthetic Responses endpoint observed Bearer authentication with the configured gateway key |
| Tool execution | `exec_command` ran under `sandbox_mode = "workspace-write"` with no approval policy written |
| Synthetic credential leakage | Child environment reported `AIHUB_CLIENT_KEY=ABSENT`, `OPENCODE_API_KEY=ABSENT`, `AIHUB_PROVIDER_OPENCODEGO_API_KEY=ABSENT`, `AIHUB_GATEWAY_API_KEY=ABSENT` |
| Temporary Codex configuration | Contained no synthetic credential value |
| Request count at the local endpoint | `2` — the expected bounded count |
| Global Codex tree before/after | Unchanged; the comparison covers the real account `~/.codex` root, which is resolved independently of the isolated `HOME` |

This proves the process and shell environment policy, not a real-provider
scenario. It uses synthetic values against a local endpoint and therefore
cannot satisfy AC-1, AC-2, or AC-14 on its own. The global-tree comparison now
covers the account home even when the process `HOME` is isolated, and it
represents nested symbolic links by target instead of following them; a
symbolic-link state root still fails closed.

Two helper corrections were included while validating the isolation: the
private manifest directory is now checked against the repository before the
first manifest is written (FR-12), and the temporary root is removed only by
the single registered cleanup, so a late failure cannot produce a spurious
integrity error. The Codex child environment also no longer inherits `USER`,
`LOGNAME`, or `TERM`, and its `TMPDIR`, `TEMP`, and `TMP` now come from the
isolated runtime directories instead of the parent; no assertion depends on the
dropped names.

### Local gate results

Gates were executed inside the same isolation, with the recorded Codex CLI on
`PATH` and no AI Hub or provider credentials in the process environment.

| Check | Result |
| --- | --- |
| `AIHUB_E07_CANARY=1 go test ./cmd/ai-hub -run TestE07 -count=1 -v` | `PASS` — 57 passed, 0 failed, 1 skipped |
| `go test ./... -count=1 -race` (no canary flag, Codex CLI on `PATH`) | `PASS` — every Go package `ok`; the canary and the live test skip by design |
| `go test ./cmd/ai-hub -run CodexShellEnvironmentPolicy -count=1 -v` (no canary flag) | `SKIP` with an explicit reason — the default suite is host-independent |
| `AIHUB_E07_CANARY=1` with the CLI absent from `PATH` | `FAIL` as required — the opt-in canary refuses to run instead of silently skipping |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` |

In the opt-in run the only skipped test is `TestE07LiveCodexOpenCodeGo`, which
requires `AIHUB_E07_LIVE=1` and provider credentials. Both skips are expected
and are not counted as scenario evidence. The previous checkpoint's failures
were caused by the active-Codex preflight only; with the documented isolation
the same assertions pass unchanged, and with the canary disabled the default
suite no longer depends on host Codex processes.

No production source file differs from the E06 baseline. The pending change
set is limited to the README E07 setup and test sections, E07 validation tests,
and E07 documentation.

### Residual limitations of the integrity helpers

- The global Codex state comparison uses metadata identity (relative path, mode,
  size, modification time, and stat fields except access time) rather than a
  content digest. Reading and hashing the whole `~/.codex` tree would read
  prompt and session contents, which FR-12 forbids reporting and which this
  validation does not need. On the validated Linux filesystem any write changes
  the modification time, size, or inode metadata, so the comparison still
  detects ordinary edits; a privileged actor that rewrites content while
  restoring timestamps and size could evade it. This trade-off is explicit and
  the isolation proof does not depend on it.
- Nested symbolic links inside a Codex state root are recorded by target and
  never followed, because the real `~/.codex` contains user-managed links. A
  symbolic-link state root still fails closed, but state written *through* a
  nested link is outside this comparison: only the link entry itself is
  observed. The live helper therefore reports `codex_tree_links=<n>` alongside
  the integrity outcome, so an operator can see when linked subtrees are not
  covered and must not treat the result as full-tree coverage.
- When `AIHUB_GATEWAY_API_KEY` is absent, the live helper derives an ephemeral
  in-memory gateway key as permitted by the design. The README instructs the
  operator to supply a real gateway credential and not to rely on harness
  generation; the ephemeral key is not a production credential and must not be
  reported as provisioning evidence.
- `provider_credential_origin` reports only `environment` or `dotenv`. A key
  loaded from `.env` is exactly the credential the operator must rotate, so
  future E2E evidence must require `environment`.

### Updated acceptance matrix

Only the criteria affected by this checkpoint changed value. Criteria that
require real-provider scenarios keep their historical outcome.

| AC | Result | Evidence and limitation | Failure cause |
| --- | --- | --- | --- |
| AC-1 | `FAIL` | The isolated canary proves the process and shell policy with synthetic values, but no real gateway/provider credential reached a real Codex child, and the AC-14 conditions are still unmet. | `INCONCLUSIVE` |
| AC-2 | `PASS` | Unchanged from the supplemental real-provider pass; not repeated. | None recorded |
| AC-3 | `PASS` | Unchanged from the supplemental real-provider pass; not repeated. | None recorded |
| AC-4 | `PASS` | Unchanged from the supplemental real-provider pass; not repeated. | None recorded |
| AC-5 | `FAIL` | No real replay was run; no new evidence. | `INCONCLUSIVE` |
| AC-6 | `PASS` | Unchanged from the supplemental real-provider pass; not repeated. | None recorded |
| AC-7 | `FAIL` | No real upstream error probe was run; no new evidence. | `INCONCLUSIVE` |
| AC-8 | `PASS` | Unchanged; Codex stdout rendering remains `NOT_VERIFIED` as permitted. | None recorded |
| AC-9 | `FAIL` | No live execution records were produced or queried. | `INCONCLUSIVE` |
| AC-10 | `FAIL` | No live shutdown and restart was performed. | `INCONCLUSIVE` |
| AC-11 | `PASS` | This report records versions, gate outcomes, limits, and presence-only states without payloads, credentials, or generated identifiers. | None recorded |
| AC-12 | `PASS` | No production source file differs from the E06 baseline. `go test ./... -count=1 -race`, `go vet`, `staticcheck`, `golangci-lint`, `go build`, formatting, and `git diff --check` all pass on the E07 change set; the existing suite is entirely green; no committed test was removed or weakened; the new helper tests pass. The global Codex-tree comparison uses metadata identity instead of a content digest, and its deterministic tests were updated to that representation; the privacy trade-off and evasion window are recorded under residual limitations. | None recorded |
| AC-13 | `PASS` | The README still states support is experimental and not established. | None recorded |
| AC-14 | `FAIL` | Ambient Codex app-servers keep rewriting the real `~/.codex` tree, so a real before/after comparison is unprovable even though the isolated run cannot write it. No approved baseline and no approved secret scan exist. Individual canary runs inside the namespace are stable. | `ENVIRONMENT_DEPENDENCY` |

### Remaining blockers

| Blocker | Owner action required |
| --- | --- |
| Approved immutable baseline | Review and approve the current 15 pending files (8 modified, 7 untracked). Staging, commit, and push remain separately gated and were not performed. |
| Approved secret scan | Approve a scanner and a specific version, make it available, then complete the redacted scan. No scanner was installed or approved here. |
| Distinct, rotated credentials | Rotate the provider credential that exists in `.env` and provision `AIHUB_GATEWAY_API_KEY`; `AIHUB_GATEWAY_API_KEY` is currently absent, so Hub cannot start from `.env` alone. Both must be injected through an approved non-echoing facility. |
| Real `~/.codex` integrity | Run the authorized pass in an environment where no ambient Codex process writes global state, or accept a scoped decision for a documented isolation boundary. |
| Provider/model availability | Cannot be confirmed without a budgeted provider call. Not attempted. |

The one-pass, nine-provider-call, two-minute-per-request, zero-retry,
30-minute authorization remains unused and unchanged. No retry or second pass
is authorized.

## Local baseline consolidation — 2026-11-01

Status: consolidation and local baseline only. No AI Hub process was started
for a scenario, no Codex request reached a provider, and the conditionally
authorized pass remains unused at `0/1` passes and `0/9` provider-boundary
calls. E07 remains `INCOMPLETE`.

### Final independent review corrections

The final `senior-code-review-judge` round reviewed the pending change set and
returned findings. Every in-scope finding was corrected before the baseline
commit:

| Finding | Correction |
| --- | --- |
| The global-tree gate resolved its root from `$HOME`, so a validation process started with an isolated `HOME` compared no tree and could report `PASS` without observing the real global state. | Snapshot roots now resolve the account home independently of `$HOME` and always include the account `~/.codex`; the process home is an additional root. `TestE07CodexTreeSnapshotKeepsAccountRootWhenProcessHomeIsIsolated` is the regression test. |
| Nested symbolic links in the real tree were rejected outright, which made the comparison unusable on any host whose Codex home contains user-managed links. | A symbolic-link state root still fails closed, while a nested link is recorded with its target and never followed. `TestE07CodexTreeSnapshotRecordsNestedSymlink` proves a retarget changes the comparison. |
| The real-provider helper skipped the run when the installed CLI release differed from the recorded target, reporting an environment limitation instead of a scoping failure. | A CLI release mismatch or an unreadable version now fails before any provider traffic; an absent CLI remains `NOT RUN — environment limitation`. |
| AC-12 claimed no assertion changed while the global-tree comparison moved from a content digest to metadata identity. | The AC-12 row now states the change and refers to the recorded residual limitation. |
| The checkpoint's 14-file list was not labelled as pre-plan-update. | The list now states that the plan-document update adds the fifteenth file. |
| The plan's execution block ran `go test` with credentials in the environment, contradicting the hardened README procedure. | The plan now requires the prebuilt test binary and the sanitized allowlist. |

Two findings were recorded without a change because both sit outside the
authorized 15-file change set: the README's pre-existing `AIHUB_LIVE_SMOKE`
example passes a placeholder credential on a command line (E02 section, not the
E07 procedure), and the approved design still describes the original
provider-credential fallback. The two validation reports record that
supersession explicitly.

### Final gate results

| Check | Result |
| --- | --- |
| `go test ./cmd/ai-hub -count=1 -run TestE07` | `PASS` — 56 passed, 0 failed, 2 skipped (opt-in canary and the `AIHUB_E07_LIVE` scenario) |
| `AIHUB_E07_CANARY=1 go test ./cmd/ai-hub -run TestE07 -count=1 -v` inside the private namespace | `PASS` — 57 passed, 0 failed, 1 skipped |
| `go test ./... -count=1 -race` | `PASS` — every Go package `ok` in the recorded runs; one earlier execution failed on a pre-existing E01–E06 test described below |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` |
| E07 Go files below 300 lines | `PASS` — the largest E07 Go file is 278 lines |
| Production source identical to `0d33ca26c15acf54fac418cb33380b5871b4d899` | `PASS` — no non-test source difference |
| Repository-wide Go source limit | `FAIL` — the eight pre-existing E01–E06 files listed above are unchanged and outside this scope |
| Approved secret scan | `NOT RUN` — no approved scanner is available |

The opt-in canary used the recorded Codex CLI against a loopback synthetic
Responses server with synthetic credentials only. It made no provider-boundary
call, and the global Codex-tree comparison covers the real account root.

### Pre-existing flaky test outside the E07 scope

`internal/catalog/manager_test.go` (`TestManagerTimeoutIsClassified`, E01–E06)
is timing-sensitive. Under `-race` it failed one of four full-suite executions,
and 1 of 8 isolated `-race` repeats classified a 10-millisecond sync deadline as
`persistence` instead of `timeout`. The file is identical to the E06 baseline
`0d33ca26c15acf54fac418cb33380b5871b4d899` and is not part of this change set.
It is recorded here so the `-race` gate result is not overstated; correcting it
would require a separate E01–E06 scope decision. Re-run the suite after any E07
test change so this pre-existing flake is not mistaken for a regression.

### Baseline state

The pending change set is consolidated into three ordered groups — the E07 test
helpers, the validation evidence, and the final E2E execution plan — so each
can be reviewed and committed locally as a single unit under separate
authorization. Nothing is pushed and the remote ref is unchanged. The commit
identifiers and the clean-tree verification belong to the session closeout, not
to this committed document, and this report does not assert them.

`E07-final-e2e-execution-plan.md` remains conditionally authorized and still
blocked by its preconditions: the consolidation must be committed and approved
as an immutable baseline, and approved scanner output, distinct rotated
credentials, and provider/model availability are still outstanding.

## Final real E2E execution preflight — 2026-11-01

Status: **BLOCKED — a mandatory credential preflight failed.** The real E2E
pass was not started, no AI Hub scenario process or Codex scenario was run, and
no provider-boundary request was made. E07 remains **INCOMPLETE**. This is a
new preflight disposition; it does not rewrite or combine any historical pass.

### Baseline and preflight

| Check | Result | Evidence |
| --- | --- | --- |
| Repository baseline at preflight start | `PASS` | Branch `main`, HEAD `6771918954f02c53e517b94ffc152630223daf2d`, clean working tree. The checkout is 0 commits ahead of `origin/main`, while the prompt's auxiliary status said 4; the expected HEAD and clean-tree conditions match. |
| Runtime binaries | `PASS` | Go `1.23.0`, Codex CLI `0.164.0-alpha.11.1`, and bubblewrap `0.4.0` are available. Version queries only; no Codex scenario was run. |
| Provider credential | `MISSING` | `AIHUB_PROVIDER_OPENCODEGO_API_KEY` is missing from the configured repository `.env`. The legacy `OPENCODE_API_KEY` was not used as a fallback. No credential value or derived representation was emitted. |
| Gateway credential in current execution process | `MISSING` | `AIHUB_GATEWAY_API_KEY` is missing from the process performing this preflight. An export declaration exists in `~/.bashrc`, but the value was not read and delivery to an AI Hub process was not verified. |
| Codex client credential | `MISSING` | `AIHUB_CLIENT_KEY` is absent from the preflight process, as expected before Codex launch. The harness configures `env_key = "AIHUB_CLIENT_KEY"`; no main Codex process was started to verify injection. |
| Child-process secret isolation | `NOT VERIFIED` | The harness constructs a restricted Codex shell environment and the prior isolated synthetic canary is documented. No real Codex shell or tool process was launched in this attempt. |
| Codex isolation | `NOT VERIFIED` | Bubblewrap is available and the documented isolated canary had passed previously. The real pass did not begin, so no isolated Codex environment or read-only host mount was created in this attempt. |
| Provider-call budget | `PASS — implementation inspection only` | The harness cap is 9, the shared observer covers initial and restarted Hub readiness, and the budget guard rejects an over-limit call before transport. The observer was not initialized for this attempt. |
| Execution timer | `NOT STARTED` | The 30-minute timer begins with the authorized execution sequence; preflight stopped before that sequence. |
| Harness baseline at preflight start | `PASS` | The finalized E07 harness is present at the expected HEAD with a clean working tree. |

The canonical provider credential check failed, so execution stopped before
starting AI Hub, Codex, the live test binary, or provider traffic. The existing
gateway credential was not substituted, regenerated, or copied into another
credential variable. The existing `~/.bashrc` was not modified.

### Scanner exception and credential handling

- Security scanner: `NOT RUN — operator-approved exception`.
- No scanner was installed or run, and no secret-free repository claim is
  made. No broad secret search was performed.
- Provider credential source checked: repository `.env`; canonical variable
  result: `MISSING`.
- Gateway credential source checked: `~/.bashrc`; current execution process
  result: `MISSING`; actual AI Hub process receipt: `NOT VERIFIED`.
- Codex client credential result in the preflight process: `MISSING`; the
  `env_key` injection path was not exercised.
- No credential values, partial values, lengths, hashes, or credential-bearing
  logs were emitted or recorded.

### Final-pass scenario results

The planned allocation is 8 provider-boundary calls, with an absolute ceiling
of 9 and zero retries. This attempt made 0 provider-boundary calls; 0 calls
were rejected, and no execution timer or live budget observer was started. The
14 historical calls remain consumed with their original outcomes and are not
combined with this attempt.

| Scenario | Result | Timestamps, budget, and execution evidence |
| --- | --- | --- |
| Basic Codex generation | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, Codex result, Hub result, or correlation evidence. |
| Gateway authentication and rejection | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, request status, or provider delta. |
| Multi-turn conversation replay | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, replay observation, or correlation evidence. |
| Real sandboxed Codex tool calling | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, Codex result, or workspace result. |
| Upstream error passthrough | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, or same-response comparison. |
| Progressive gateway streaming | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, SSE observation, or Codex stdout observation. |
| Execution Records persistence, correlation, listing, and lookup | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, record observations, or identifiers. |
| Ordered shutdown, restart, and recovery | `NOT VERIFIED` | Not started; no timestamps, counter snapshots, restart result, or recovery evidence. |
| Credential isolation and Codex filesystem integrity | `NOT VERIFIED` | No real Codex process or before/after integrity comparison was performed. |

### Acceptance criteria disposition

This preflight produced no new acceptance evidence. The cumulative statuses
below remain as recorded in the preceding acceptance matrix; historical results
were not repeated or reinterpreted.

| Criterion | Final status | Current-attempt evidence |
| --- | --- | --- |
| AC-1 | `FAIL` | No real credential-backed Codex run or new integrity evidence. |
| AC-2 | `PASS` | Historical cumulative evidence only; not repeated. |
| AC-3 | `PASS` | Historical cumulative evidence only; not repeated. |
| AC-4 | `PASS` | Historical cumulative evidence only; not repeated. |
| AC-5 | `FAIL` | No real multi-turn replay was run. |
| AC-6 | `PASS` | Historical cumulative evidence only; not repeated. |
| AC-7 | `FAIL` | No real upstream error passthrough scenario was run. |
| AC-8 | `PASS` | Historical gateway-streaming evidence only; not repeated. |
| AC-9 | `FAIL` | No live execution records were created or queried. |
| AC-10 | `FAIL` | No live shutdown, restart, or recovery scenario was run. |
| AC-11 | `PASS` | This report records the blocked disposition and scenario limits without credential values, payloads, or generated identifiers. |
| AC-12 | `PASS` | Historical local gate evidence remains applicable; this attempt made no code or test changes. |
| AC-13 | `PASS` | No README support claim was added; the prior experimental-support wording remains. |
| AC-14 | `FAIL` | No before/after global Codex, `.env`, repository, or Git integrity comparison was performed for a real pass. |

E07 remains **INCOMPLETE**. The authorized real E2E pass remains available but
blocked until the existing canonical provider credential is available in the
repository `.env` and the existing gateway credential can be securely
delivered from its configured source to the actual Hub process. Do not use the
legacy provider variable or generate a replacement credential. This preflight
did not consume the authorized real E2E pass.

After recording this disposition, the only repository change is this E07
validation report. `git diff --check` passed. No `.env` or `~/.bashrc` change,
commit, or push was made.

## Final real E2E preflight — stopped before execution (2026-11-01)

Status: **BLOCKED — the approved live harness hashes credential-bearing `.env`
contents before starting the scenarios.** The execution timer was not started,
no live Hub or Codex scenario was run, and this attempt made 0 provider-boundary
calls. The one authorized real E2E pass remains unconsumed. E07 remains
**INCOMPLETE**.

### Corrected credential status and baseline

The prior preflight above records a false-negative credential result from a
simplified dotenv verifier. The operator has since confirmed that the
production dotenv parser loads both canonical credentials from the repository
`.env`, including entries with spaces around `=`, and that `.env` remains
owner-only and ignored by Git. This operator-confirmed configuration supersedes
the earlier missing-variable diagnosis; no credential value was read, copied,
logged, measured, or hashed during this preflight.

| Check | Result | Evidence |
| --- | --- | --- |
| Git baseline | `PASS` | Branch `main`, HEAD `6771918954f02c53e517b94ffc152630223daf2d`; only this validation report was modified at preflight start. `.env` is ignored by Git. |
| Scanner | `NOT RUN — operator-approved exception` | No scanner was installed or run. No secret-free repository claim is made. |
| Provider and gateway credentials | `OPERATOR-CONFIRMED PRESENT` | The corrected production dotenv result was supplied by the operator. This attempt did not inspect or emit either value. |
| Integrity preflight | `BLOCKED` | `e07PrepareLiveSetup` snapshots the repository before calling `config.LoadDotEnv` (`cmd/ai-hub/e07_live_setup_test.go`). The snapshot includes a complete tree digest and `.env` fingerprint (`cmd/ai-hub/e07_integrity_test.go`); `e07FingerprintTree` and `e07FingerprintEnvFiles` feed regular-file bytes to SHA-256 (`cmd/ai-hub/e07_integrity_tree_test.go`). |
| Provider-call budget | `NOT INITIALIZED` | The harness defines a nine-call cap, but execution stopped before creating its observer. No request was attempted or rejected. |
| Execution timer | `NOT STARTED` | The 30-minute window begins immediately before the live execution sequence, which was not entered. |

Hashing the `.env` file would hash the credential values it contains, contrary
to the explicit credential-handling constraint. Therefore the live harness was
not invoked and no attempt was made to bypass, alter, or substitute its
integrity checks. Resolving this blocker requires an authorized test-harness
integrity method that verifies `.env` stability without hashing its contents;
that harness change is outside this execution-only pass.

### Scenario and acceptance-criteria status

All scenarios in the authorized pass are `NOT VERIFIED` for this attempt. No
provider-call counter was initialized, no timestamps or execution identifiers
were produced, and no new acceptance evidence was collected. Historical E2E
results and the previously recorded cumulative AC-1 through AC-14 statuses
remain unchanged; they are not combined with this stopped preflight. E07 is
still **INCOMPLETE**.

The report is the only repository file changed. No production code, `.env`, or
`.bashrc` was modified. No commit or push was made.

## Authorized integrity correction and final execution checkpoint (2026-11-01)

Status: **INCOMPLETE — the integrity correction is implemented and reviewed, but
the real E2E scenarios did not start because the global Codex state tree was
unstable before launch.** No Hub was started and no provider-boundary request
was attempted in either preflight invocation.

### Integrity correction

The E07-only integrity helpers now classify root and nested `.env` and
secret-bearing `.env.*` entries before opening a file or following a symlink.
Their content is excluded from tree, dotenv, tracked-file, and Git-state
content fingerprints. The metadata-only dotenv fingerprint records relative
names and stable file metadata fields, excluding access time; it does not
record or hash content. It detects ordinary edits that change metadata but
cannot prove byte-for-byte equality if content is rewritten while preserving
those fields. `.env.example` remains included in ordinary file-content
fingerprints. Non-secret tracked and untracked source contents remain covered,
and source changes remain detectable. The secret paths are excluded from the
Git status scope. The file-hashing helper was extracted to keep the tree helper
below the repository's preventive file-length threshold.

The deterministic bubblewrap test mounts a synthetic repository read-only,
loads its `.env` with the production dotenv parser, and verifies that writes to
both `.env` and a source file are rejected. The live command was configured to
mount the host root read-only and use private temporary runtime paths; the
external host remains able to update its own Codex state outside that mount
namespace, which is why the before/after Codex-tree check remains mandatory.

### Regression tests and local gates

The commands below were executed by this implementation session. The
independent reviewer inspected the code and considered the supplied test
evidence; the reviewer did not rerun these commands.

| Evidence | Result |
| --- | --- |
| `go test ./cmd/ai-hub -count=1` | `PASS` |
| E07 protected dotenv and read-only mount tests with `-v` | `PASS`, both tests executed without skips |
| Fingerprint, dotenv non-read, symlink, and source-change tests inside an outer read-only bubblewrap mount with private scratch | `PASS`, six selected tests |
| `go test ./... -count=1 -race` | `PASS` on a sequential rerun. An initial parallel gate run had a timing failure in `internal/catalog/TestManagerTimeoutIsClassified`; the test passed in isolation and the sequential full rerun passed. |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS`, zero issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS`, empty output |
| `git diff --check` | `PASS` |
| Independent `senior-code-review-judge` review of the E07 code correction | `APPROVED`, no critical, important, or minor findings |
| Secret scanner | `NOT RUN — operator-approved exception` |

All changed or new Go files are below 250 lines. The code change is limited to
`cmd/ai-hub/e07_*` helpers and tests; no E01–E06 production file, `.env`, or
`.bashrc` was changed. No commit or push was made. The scanner exception does
not establish that the repository is secret-free.

### Real E2E launch attempts

The authorized test command was invoked twice in preflight mode. Each invocation
started the harness wall-clock timer before setup and stopped before dotenv
loading, Codex version inspection, Hub startup, or provider traffic.

| Preflight | Result | Provider-call counter | Elapsed |
| --- | --- | ---: | ---: |
| Initial read-only bubblewrap launch | `BLOCKED` — a host Codex process was visible to the process guard | `NOT INITIALIZED`; setup exited before Hub startup | `0.002193 s` |
| Read-only bubblewrap launch with a private PID namespace | `BLOCKED` — the global Codex tree changed during the 250 ms stability check | `NOT INITIALIZED`; setup exited before Hub startup | `0.450396 s` |

The PID namespace limited process inspection to the validation process view.
The second invocation still detected a metadata change in the actual global
Codex tree. The root mount was read-only for the validation process, but that
namespace cannot prevent an external host process from changing the underlying
global tree. Continuing would make AC-14 unverifiable.

Both failures occurred before the observer and provider-call budget were
created. The cleanup log's zero fields were default values for a nil observer,
not an initialized counter snapshot. Control flow confirms that no Hub or
provider request was started; report provider calls as `0 confirmed`, with the
counter `NOT INITIALIZED`. Eight calls were expected and nine remained the
maximum. The tenth call remains covered by the deterministic budget test, which
proves rejection before transport. No provider request was retried; none began.

| Required live scenario | Result in this checkpoint |
| --- | --- |
| Basic generation | `NOT RUN — Codex-tree preflight failed` |
| Gateway authentication and rejection | `NOT RUN — Codex-tree preflight failed` |
| Multi-turn replay | `NOT RUN — Codex-tree preflight failed` |
| Sandboxed tool calling | `NOT RUN — Codex-tree preflight failed` |
| Upstream error passthrough | `NOT RUN — Codex-tree preflight failed` |
| Gateway streaming | `NOT RUN — Codex-tree preflight failed` |
| Execution-record persistence and correlation | `NOT RUN — Codex-tree preflight failed` |
| Listing and by-ID retrieval | `NOT RUN — Codex-tree preflight failed` |
| Graceful shutdown, restart, and recovery | `NOT RUN — Codex-tree preflight failed` |
| Credential isolation and repository/Codex integrity | `NOT RUN — Codex-tree preflight failed` |

### Cumulative AC-1 through AC-14 checkpoint

Historical results above remain unchanged. `PASS (historical)` identifies
criteria with prior real-provider evidence that were not repeated in this
checkpoint; it does not combine that evidence with a new E2E run.

| AC | Current status | Checkpoint evidence |
| --- | --- | --- |
| AC-1 | `BLOCKED` | The isolated live setup did not proceed; its required AC-14 integrity dependency is unresolved. |
| AC-2 | `PASS (historical)` | No Codex request was sent in this checkpoint. |
| AC-3 | `PASS (historical)` | No authenticated or unauthenticated live request was sent in this checkpoint. |
| AC-4 | `PASS (historical)` | No basic generation request was sent in this checkpoint. |
| AC-5 | `FAIL` | No real replay was run; the previous cumulative matrix records replay failure. |
| AC-6 | `PASS (historical)` | No tool task was run in this checkpoint. |
| AC-7 | `FAIL` | No real upstream-error probe was run in this checkpoint. |
| AC-8 | `PASS (historical)` | No live stream was run in this checkpoint. |
| AC-9 | `FAIL` | No live execution records were created or queried in this checkpoint. |
| AC-10 | `FAIL` | No live shutdown, restart, or recovery was run in this checkpoint. |
| AC-11 | `BLOCKED` | This entry records the preflight outcomes, durations, and call counters without credentials, payloads, or generated identifiers; current live runtime metadata was not reached. |
| AC-12 | `PASS` | Production remains unchanged and all listed local quality gates passed on the E07 helper change set. |
| AC-13 | `PASS` | No Codex support claim was added; the existing incomplete/experimental wording remains. |
| AC-14 | `BLOCKED` | Read-only fixture and repository tests pass, but the global Codex tree changes during the stability window. The scanner remains `NOT RUN — operator-approved exception`. |

E07 remains **INCOMPLETE**. The authorization and provider-call budget remain
unchanged, and this checkpoint made zero provider calls. Resuming the real pass
requires a quiescent host Codex state tree for the full before/after comparison;
the active writer has not been identified or stopped by this validation. Do not
claim Codex CLI support or repository secret-free status from this checkpoint.

## Isolated final E2E execution — 2026-11-01

Status: **INCOMPLETE — the isolated real-provider pass ran once; multi-turn
replay failed, and AC-14's literal host-global unchanged-tree condition was not
met.** The operator-approved scanner exception remains in effect. No support
claim is made.

### Global-tree diagnosis and isolation

Read-only `/proc` inspection before execution found two ambient `codex
app-server` processes with 65 write-capable open descriptors under the account
Codex tree. This establishes that app-server processes could write there, but
does not associate a particular historical metadata change with a process or
write event. The source of the changes therefore remains **UNDETERMINED**.
Process names, arguments, descriptor paths, and Codex state contents were not
published.

The live test and its descendants ran inside a private mount and PID namespace.
The host root and repository, including `.env`, were read-only. An empty,
read-only bind mount covered the real account `~/.codex`; the dedicated private
runtime bind was read-write. `HOME`, `CODEX_HOME`, `CODEX_SQLITE_HOME`,
`TMPDIR`, `TEMP`, `TMP`, and all configured XDG paths resolved under the private
runtime. The shell policy used `inherit = "none"` and
`experimental_use_profile = false`; Codex retained `workspace-write` and its
normal approval behavior. Network access remained available for OpenCode Go.

The deterministic `TestE07CodexMountBoundaryHidesGlobalState` verified the
empty read-only global mount, writable private state, before/after private
state snapshots, and an actual child process in the same mount namespace. The
live setup guard repeated the mount/path checks before launching Codex. The
live pass recorded seven private state roots before and after, and all nine
per-scenario isolation/repository checkpoints passed.

One launcher assembly attempt ended before the test binary started because it
mounted a private `/tmp` before binding sources located under host `/tmp`. It
started no Hub and made zero provider calls. The corrected launcher bound the
private runtime, empty global mount, and test binary before mounting private
`/tmp`. The scenario pass then ran exactly once; no scenario was retried.

The host-side metadata-only comparison reported the real global Codex tree as
`CHANGED_EXTERNALLY` during both the failed launcher window and the isolated
pass. The E2E process could not see or modify that host tree, and this external
delta was not classified as an E2E isolation failure. No global file was read,
restored, or modified by the validation process.

### Execution environment and budget

| Item | Result |
| --- | --- |
| AI Hub production baseline | `0d33ca26c15acf54fac418cb33380b5871b4d899`; no production source changed |
| E07 test worktree revision | `a4027c24ffdb788136a60956b4f270834f99cce9`, with the reviewed E07 worktree mounted read-only |
| Runtime | Go `go1.23.0`, Codex CLI `0.164.0-alpha.11.1`, bubblewrap `0.4.0`, Linux `amd64` |
| Model | `deepseek-v4.1-flash` |
| Credential source | Existing `.env`, loaded by the production dotenv parser; provider origin reported as `dotenv`; values were not printed, copied, rotated, or regenerated |
| Scanner | `NOT RUN — operator-approved exception`; no secret-free claim is made |
| Scenario passes | `1/1` |
| Provider calls | `8/9` maximum; attempted `8`, forwarded `8`, rejected `0`; no retry |
| Wall-clock duration | `17.019 s`, from corrected namespace launch through cleanup |
| Host `~/.codex` metadata | `CHANGED_EXTERNALLY`; no attribution to a process was established |
| Repository, dotenv metadata, tracked files, Git scope | `PASS` against the reviewed E07 worktree baseline |

The launcher-only failure took 1.002 seconds and had no initialized provider
observer; control flow and the absent test-binary start confirm zero provider
calls. It is not counted as an E2E scenario pass. The successful scenario run
used eight provider-boundary calls, with one call remaining under the hard
ceiling of nine. No model-availability probe or other provider diagnostic was
made outside this pass.

### Scenario results

| Scenario | Result | Evidence |
| --- | --- | --- |
| Basic generation | `PASS` | Non-empty successful response; one Codex Responses request reached the Hub. |
| Authentication and rejection | `PASS` | Authenticated route worked; unauthenticated request returned `401` with zero provider-call delta. |
| Multi-turn replay | `FAIL` | Turn one passed. The second Codex invocation produced no second Hub request: one of the expected two replay requests reached the provider. The first divergence was between the second Codex invocation and Hub ingress. Marker recall and prior-turn-context predicates failed; `previous_response_id` was absent. No retry was made. |
| Sandboxed tool round trip | `PASS` | The bounded write/read task completed in the temporary workspace under `workspace-write` with normal approvals. |
| Upstream error passthrough | `PASS` | Same-response status, content type, complete-body digest, and generic envelope checks all passed for one provider call. |
| Gateway streaming | `PASS` | 118 SSE data events; first at `1.954 s`, last at `3.849 s`, completion at `3.850 s`. Codex stdout progressive rendering remains `NOT VERIFIED`. |
| Execution records, correlation, listing, and by-ID lookup | `PASS` | Pre-stop: 6 expected and 6 matched; list, correlation, outcome, schema, and all 6 by-ID assertions passed. |
| Graceful shutdown, restart, and recovery | `PASS` | Post-restart: 6 expected and 6 matched; all independent record assertions passed. |
| Credential and filesystem isolation | `PASS` | The exact Codex parent environment contained the gateway key and excluded provider/Hub-only keys; config contained no credential values. Private mounts and runtime paths passed at each checkpoint. |
| Codex stdout progressive rendering | `NOT VERIFIED` | The CLI provided no measurable progressive-rendering signal; gateway-hop streaming passed. |

### Focused verification and review

| Check | Result |
| --- | --- |
| `go test ./cmd/ai-hub -count=1 -run '^(TestE07CodexMountBoundaryHidesGlobalState|TestE07ReadOnlyMountLoadsDotEnvAndRejectsWrites)$' -v` | `PASS` — both tests ran without skips |
| `git diff --check` | `PASS` |
| E07 Go file limit | `PASS` — all changed/new Go files are below 250 lines |
| Targeted independent review | Final re-review found no open findings. Initial review findings on empty/read-only mount proof, scenario continuation, and child environment allowlisting were corrected. The reviewer did not run tests or E2E. |
| Full local gate matrix after the isolation delta | `NOT RUN` — only the focused checks were repeated, as requested; prior broad gate results predate this delta |

### AC-1 through AC-14 disposition at the end of the real E2E run

| AC | Current result | Evidence and limitation |
| --- | --- | --- |
| AC-1 | `NOT ESTABLISHED` | Process and credential isolation passed; the required AC-14 global unchanged-tree condition did not. |
| AC-2 | `PASS` | Four Codex Responses requests reached the Hub; none used a direct provider route. |
| AC-3 | `PASS` | Authenticated routing and `401` rejection passed with zero provider-call delta for the rejected request. |
| AC-4 | `PASS` | Basic real-provider generation succeeded. |
| AC-5 | `FAIL` | The second Codex invocation did not reach Hub ingress; replay and marker recall were not established. |
| AC-6 | `PASS` | Real sandboxed tool round trip passed with normal controls. |
| AC-7 | `PASS` | Same-response upstream error fidelity passed. |
| AC-8 | `PASS` | Progressive gateway streaming passed; Codex stdout rendering is `NOT VERIFIED`, permitted by AC-8. |
| AC-9 | `PASS` | Six records passed list, correlation, outcome, schema, and by-ID assertions. |
| AC-10 | `PASS` | Six records remained queryable after orderly restart. |
| AC-11 | `PASS` | This checkpoint records versions, budget, outcomes, and limitations without credentials, payloads, paths, hashes, or generated request/record identifiers. |
| AC-12 | `NOT VERIFIED` | Focused isolation tests and diff checks passed. The full required gate matrix was not rerun after the isolation delta. |
| AC-13 | `PASS` | The README remains experimental and makes no Codex support claim. |
| AC-14 | `FAIL — literal unchanged-tree clause` | The host-side metadata comparison changed, while process isolation passed. The report does not attribute the external change to the E2E. The secret scanner exception also leaves the no-secret-added subcondition unverified. |

The approved SPEC's AC-14 expressly requires before/after evidence that
`~/.codex` is unchanged. The isolated E2E proves it could not read or write the
host tree; that does not satisfy the literal unchanged-tree assertion when the
host-side metadata comparison changes. This distinction is recorded without
changing AC-14 or treating unrelated host activity as an E2E process failure.

At the end of the real E2E run, E07 remained **INCOMPLETE** because AC-5
failed, AC-12 had not been rerun, and AC-14 did not pass. Do not retry the
multi-turn scenario or start another provider pass without a new authorization.
No commit or push was made.

## Local AC-5 correction and AC-12 verification — 2026-11-01

Status: the E07 resume invocation was corrected and verified with the recorded
Codex CLI against a loopback synthetic Responses endpoint. No AI Hub real
provider pass or provider-boundary request was made. The real E2E results above
remain unchanged; AC-5 is not marked PASS without a new authorized real run.

### AC-5 root cause and correction

The real run created the first conversation in a temporary workspace outside a
Git repository with `--skip-git-repo-check`. Its second invocation resumed the
explicit thread ID from the same temporary workspace but omitted that option.
The installed Codex CLI exposes `--skip-git-repo-check` for `exec resume` as
well as `exec`.

`TestE07CodexResumeContinuesFromTemporaryWorkspace` reproduced the failure
before the fix: the first invocation completed two requests against the local
synthetic endpoint, then `exec resume` exited with code 1 before issuing its
request. Adding `--skip-git-repo-check` to the resume invocation made the same
test pass and produced the expected third local request. The live E07 scenario
now passes that option when resuming. Sandbox mode and approval behavior were
not changed.

This A/B reproduction confirms the validation harness cause for the missing
second request. It does not establish real-provider marker recall or full
replay, so AC-5 remains pending a newly authorized E2E pass.

### AC-12 gates after the correction

All commands below ran after the resume fix. The live-provider and canary
environment flags were unset for the package and repository suites.

| Check | Result |
| --- | --- |
| `AIHUB_E07_CANARY=1 go test ./cmd/ai-hub -count=1 -run '^TestE07CodexResumeContinuesFromTemporaryWorkspace$' -v` | `PASS` — actual Codex CLI, temporary state, loopback synthetic endpoint; no provider call |
| `go test ./cmd/ai-hub/ -count=1 -run TestE07` | `PASS` |
| `go test ./... -count=1 -race` | `PASS` — all Go packages |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` |

### Updated disposition after local verification

| AC | Current disposition | Remaining evidence |
| --- | --- | --- |
| AC-1 | `NOT ESTABLISHED` | The real credential and process isolation checks passed, but AC-1 also requires the AC-14 before/after conditions. |
| AC-5 | `REAL E2E FAIL; LOCAL CAUSE CORRECTED` | A new authorized real-provider run must prove that the resumed request reaches the Hub, includes prior-turn context, omits `previous_response_id`, and recalls the marker. |
| AC-12 | `PASS` | All required local gates above passed after the latest code change. |
| AC-14 | `FAIL — literal unchanged-tree clause` | The last real run observed a global Codex metadata delta. Process isolation passed, but the exact writer remains undetermined. A dedicated homologation account or host with a stable account-level `~/.codex` tree and no ambient Codex writer is required for the literal comparison; a temporary `CODEX_HOME` alone is insufficient. The no-secret-added subcondition also remains unverified under the approved scanner exception. |

AC-7, AC-9, and AC-10 remain `PASS` from the latest real E2E run. No new
evidence indicates regression, so they were not reopened. AC-2, AC-3, AC-4,
AC-6, AC-8, AC-11, and AC-13 retain their latest real-run outcomes above.
E07 remains **INCOMPLETE**; no support claim was added.

## Integrity symlink correction and refreshed AC-12 gates — 2026-11-01

No real E2E or provider call was made. The existing real-run results and the
consumed authorization remain unchanged. The live and canary flags were unset
for the local gate matrix.

The integrity regression exposed that a symlink pointing to a secret dotenv
file was classified before its target was inspected. The fingerprint therefore
captured only the symlink metadata, so a target metadata change could be missed.
The traversal now records both link and target metadata for secret dotenv
links. It continues to avoid opening secret dotenv contents, including when a
link itself is named `.env` or `.env.*`. The existing repository snapshot test
still requires a changed `.env` snapshot to compare unequal; it was not relaxed.

Focused checks passed for secret dotenv change detection, protected-file read
guards, secret symlink target changes, the read-only repository mount, and the
repository snapshot regression. The following AC-12 gates passed after this
integrity correction:

| Check | Result |
| --- | --- |
| `go test ./cmd/ai-hub/ -count=1 -run TestE07` | `PASS` |
| `go test ./... -count=1 -race` | `PASS` — all Go packages |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` |

### Remaining AC-1 and AC-14 evidence

AC-1 remains `NOT ESTABLISHED`: the real run proved credential and process
isolation, but its AC-14 prerequisite did not pass. Its remaining evidence is a
validation from a dedicated homologation account or host where the account-level
`~/.codex` tree has no concurrent Codex writer, plus all literal AC-14 checks.
Using a temporary `CODEX_HOME` does not satisfy the SPEC's account-level
before/after condition.

The symlink correction is locally fixable and now covered by regression tests.
The last real run's global `~/.codex` metadata delta is an environment issue:
the exact writer was not identified, so it cannot be attributed to the E2E.
The approved execution plan also explicitly prohibits installing or running a
secret scanner. Consequently the no-secret-added condition remains unverified;
no secret-free claim is made. The metadata-only dotenv check does not inspect or
retain file contents, so its evidence is limited to filesystem metadata and
does not establish byte-for-byte equality when metadata is preserved.

| AC | Current disposition | Remaining evidence |
| --- | --- | --- |
| AC-1 | `NOT ESTABLISHED` | Dedicated homologation environment, literal AC-14 checks, and a new authorized real validation. |
| AC-5 | `REAL E2E FAIL; LOCAL CAUSE CORRECTED` | A new authorized real run must prove Hub ingress, prior-turn context, absent `previous_response_id`, and marker recall. |
| AC-12 | `PASS` | All SPEC-required local gates passed after the latest integrity code change. |
| AC-14 | `FAIL — literal unchanged-tree clause` | Dedicated stable account/host needed for the global Codex tree comparison; scanner execution is prohibited by the approved plan; dotenv metadata cannot prove byte-for-byte equality if metadata is preserved. |

AC-7, AC-9, and AC-10 remain `PASS` from the latest real E2E run; no concrete
regression evidence was found. No historical real-run result was changed.

## Isolated homologation preflight and authorization checkpoint — 2026-11-01

Status: **PREPARED FOR REVIEW — no new real E2E pass was authorized or started.**
This checkpoint made zero provider calls. The previous real pass remains
consumed, and none of its outcomes is retroactively changed by this local
preflight.

### Global Codex writer investigation

The integrity helper resolves the account home with Go's account lookup,
independently of `HOME`. For the current host identity, that home is
`/home/rafael`, so its account-level root is `/home/rafael/.codex`. One
metadata-only `/proc` inspection found three processes named `codex` with 124
write-capable descriptors under that tree. It did not inspect process
arguments, descriptor target names, or Codex state contents. This establishes
write capability only; it does not identify whether a process was a CLI,
app-server, or extension child, and it does not attribute any historical tree
change to a specific process. The historical cause remains **UNDETERMINED**.
No process was signalled or modified.

### Tested isolated Linux environment

The selected preflight uses the installed Bubblewrap 0.4.0 with private mount
and PID namespaces. A temporary `/etc/passwd` view names the executing UID and
GID `1000:1000` as `e07-homologation`, with account home
`/home/e07-homologation`; this entry exists only inside the namespace. No host
account, package, permission, or system configuration was changed. The
namespace-scoped account root monitored by the helper is exactly
`/home/e07-homologation/.codex`. It is an empty read-only mount on a private
tmpfs home. The identity shares the host's numeric UID, so its isolation comes
from the mount and PID namespaces, not from a distinct host UID.

The tested mount layout hides the developer's `/home` behind tmpfs, mounts the
repository read-only at `/tmp/e07/repository`, and keeps the per-run runtime,
`HOME`, `CODEX_HOME`, `CODEX_SQLITE_HOME`, XDG directories, and temporary files
under a private tmpfs runtime. The repository `.env` is readable through the
read-only repository mount and cannot be written through that mount. The
production `config.LoadDotEnv` parser reported only that
`AIHUB_GATEWAY_API_KEY` and `AIHUB_PROVIDER_OPENCODEGO_API_KEY` are present;
their values were not printed, copied, hashed, or measured. The developer's
global Codex tree was not visible in the namespace. A child-process probe
confirmed the same mount and PID namespaces; a same-UID parent could not read
a synthetic file through `/proc/<pid>/root`.

An explicit UID-0 user-namespace variant was rejected: its nested Codex
Bubblewrap sandbox failed with `Operation not permitted`. The tested launcher
therefore retains the current numeric UID and uses the namespace-local account
view. Under that configuration, the normal `workspace-write` Codex sandbox
and approval behavior worked without a bypass.

### Local AC-5 and AC-12 evidence

`TestE07CodexResumeContinuesFromTemporaryWorkspace` passed inside the tested
namespace with Codex CLI `0.164.0-alpha.11.1`. It used a loopback synthetic
Responses endpoint, not AI Hub or the real provider. The initial invocation
made two synthetic requests, and `exec resume` made the expected third request.
Both invocations included `--skip-git-repo-check`; the regression exercises
the second invocation and verifies its request reached the synthetic endpoint.
This confirms the local launcher correction only. AC-5 remains a real E2E
failure until a newly authorized pass proves the replay through AI Hub and
marker recall.

The focused `TestE07` package run passed with `AIHUB_E07_CANARY=1`; the live
flag was unset. The current working tree also passed the complete AC-12 gate
set in the isolated environment:

| Check | Result |
| --- | --- |
| `go test ./... -count=1 -race` | `PASS` |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` |

No real E2E scenario, AI Hub live setup, or provider-boundary request ran in
these checks.

### Current cumulative acceptance disposition

| AC | Current disposition | Evidence and remaining limitation |
| --- | --- | --- |
| AC-1 | `NOT ESTABLISHED` | Depends on the literal AC-14 conditions and a new authorized real run. |
| AC-2 | `PASS` | Latest real E2E routed all tested Codex Responses traffic through AI Hub. |
| AC-3 | `PASS` | Latest real E2E authenticated routing and rejection passed. |
| AC-4 | `PASS` | Latest real E2E basic generation passed. |
| AC-5 | `REAL E2E FAIL; LOCAL CAUSE CORRECTED` | Local resume regression passed; real replay, context forwarding, and marker recall remain unverified. |
| AC-6 | `PASS` | Latest real E2E sandboxed tool round trip passed. |
| AC-7 | `PASS` | Latest real E2E same-response upstream error fidelity passed. |
| AC-8 | `PASS` | Gateway streaming passed; Codex stdout progressive rendering remains `NOT VERIFIED`. |
| AC-9 | `PASS` | Latest real E2E record correlation, listing, and by-ID checks passed. |
| AC-10 | `PASS` | Latest real E2E restart and recovery checks passed. |
| AC-11 | `PASS` | Latest report records the run without payloads, credentials, or generated identifiers. |
| AC-12 | `PASS` | Full required local gates passed after the latest E07 helper changes. |
| AC-13 | `PASS` | No Codex support claim was added. |
| AC-14 | `FAIL — literal unchanged-tree clause` | The last real run observed a host-global Codex metadata delta. The namespace preflight is not a real-run before/after result. The operator-approved scanner exception also leaves the literal no-secret-added subcondition unverified. |

The scanner remains `NOT RUN — operator-approved exception`; no secret-free
claim is made. The namespace preflight does not prove that no secret was added
to the repository, and AC-14 is not marked `PASS`. E07 remains
**INCOMPLETE**. No commit, push, or support claim was made.

### Independent review and enforced execution deadline — 2026-11-01

The first independent review returned SDD `NEEDS_FIX`: the plan stated a
30-minute ceiling but did not enforce it, and the live-test elapsed timer began
after setup. No provider call or real E2E pass occurred during that review.

The execution plan now requires `/usr/bin/timeout --signal=KILL 30m` as the
outermost supervisor before preflight/setup. The deadline covers test-binary
compilation, namespace startup, live setup, scenarios, shutdown, integrity
comparison, and cleanup. All writable run artifacts are specified under
namespace tmpfs; a timeout is `INCONCLUSIVE`, leaves after-state integrity
`NOT VERIFIED`, and permits no retry. The live-test elapsed measurement now
starts before `e07PrepareLiveSetup` and includes its registered cleanup.

Local evidence after this correction:

| Check | Result |
| --- | --- |
| Focused E07 resume, mount-boundary, dotenv-read-only, and secret-symlink tests | `PASS` |
| GNU `timeout` availability | `PASS` — coreutils 8.30 |
| Synthetic watchdog check | `PASS` — a one-second watchdog terminated a local sleep process with exit status 137 |
| `git diff --check` | `PASS` |
| New provider calls / real E2E | `NONE` |

The independent reviewer must recheck the deadline correction before a new
real E2E authorization is requested. The historical provider run, AC-5
disposition, AC-14 literal failure, and scanner exception remain unchanged.

### Final homologation preparation verification — 2026-11-01

No real E2E was launched and no provider-boundary call was made in this
checkpoint. The previously authorized real pass remains consumed.

The local AC-5 launcher regression passed with Codex CLI
`0.164.0-alpha.11.1` and a loopback synthetic Responses endpoint. The first
turn made two synthetic requests and `exec resume` made the expected third.
Both invocations include `--skip-git-repo-check`. This remains local evidence;
the historical real AC-5 result remains `FAIL` pending a newly authorized
route-level replay.

The supervised preflight initially failed because the isolated `/tmp` tmpfs
was remounted read-only, preventing Codex's nested shell sandbox from
constructing its Bubblewrap command. The supervisor now keeps its private
`/tmp` tmpfs writable while retaining the repository bind as read-only. The
preflight also rejects direct `--supervised` invocation unless the parent is
the external `/usr/bin/timeout` with the exact `--signal=KILL 30m` argv. A
60-minute timeout carrying the internal marker was rejected before setup. The
preflight runs an opt-in credential-presence check through the production
dotenv parser. The parser reports only
`AIHUB_GATEWAY_API_KEY=PRESENT` and
`AIHUB_PROVIDER_OPENCODEGO_API_KEY=PRESENT`; no values are emitted. The
operator-approved scanner exception remains unchanged.

After those corrections, `scripts/e07-live-supervisor.sh --preflight` passed,
including the synthetic shell-environment canary, Codex resume regression,
read-only repository and dotenv mount checks, secret-file fingerprint tests,
symlink handling, and child namespace checks. The direct supervised-mode
negative check was rejected before repository discovery. The complete AC-12
local gate set also passed: `go test ./... -count=1 -race`, `go vet ./...`,
`staticcheck ./...`, `golangci-lint run ./...`, `go build ./...`,
`gofmt -l cmd internal`, and `git diff --check`. No live-test flag was set.

#### AC-14 literal disposition — corrected host-side evidence path

Historical real runs observed a global Codex-tree metadata change; the writer
remains `UNDETERMINED`. The Bubblewrap mount checks establish that the child
cannot access the developer home through its namespace view, but those checks
are not a host-tree before/after comparison.

The earlier conclusion that AC-14 requires a distinct host Linux account and a
privileged dotenv descriptor broker was incorrect and is withdrawn. The
supervisor runs the integrity helper outside Bubblewrap, under the current
host account, before and after the sandboxed child. Its 30-minute watchdog
covers baseline capture, the child run, and after-state comparison. The helper
resolves the host home from the current UID and fingerprints the actual
`~/.codex` tree plus existing project `.env` contents and metadata. It uses
read-only `O_NOATIME` access and compares digests in memory. Access time is
checked for stability during helper reads but excluded from cross-run equality
because the production dotenv parser may read the same file during the run.
Codex-tree symlinks are captured through `lstat` metadata without dereferencing
them; referent contents outside that tree are not covered. Any project
symlink prevents safe `.env` discovery and fails closed. The supervisor emits
category outcomes only. The existing `.env` mode and contents are not changed;
a separate host account is not needed for this observation.

This is local implementation and fixture-test evidence only. No new real
provider pass or host before/after result is recorded here, and this correction
does not replace historical results. AC-14 remains `FAIL` until a real pass
establishes the host-tree and `.env` comparisons, repository/Git integrity, and
the required secret-scan condition. The scanner remains a separate unresolved
unit. No secret-free claim is made.

E07 remains `INCOMPLETE`. Any future real pass must retain the authorized
limits of at most 9 provider calls, 30 minutes total, 2 minutes per request,
and zero retries. A distinct host account, permission change, or privileged
dotenv broker is not a precondition for the host-side comparison.

### Separate host-integrity preflight result

This entry records a separate supervised preflight observation and does not
replace or merge with earlier execution results. The raw process output and an
execution timestamp were not retained with this correction; the outcome below
is transcribed from the carried-forward execution record. No preflight or E2E
was rerun for this report update.

| Claim | Classification | Observation |
| --- | --- | --- |
| Codex host-tree integrity | `FACT` | `codex_tree=FAIL` |
| Project environment-file integrity | `FACT` | `project_env_files=PASS` |
| Supervisor exit status | `FACT` | `1` |
| Cause of the Codex-tree change | `UNKNOWN` | The available execution record does not establish a cause. |
| Responsible process | `UNKNOWN` | The available execution record does not identify a writer. |
| Integrity of external symbolic-link destinations | `NOT VERIFIED` | The helper records symbolic-link entries without fingerprinting referents outside the tree. |
| AC-14 disposition for E07 | `FACT` | `NOT PASS`; this observation does not establish all literal AC-14 conditions. |

No Codex process or other writer is attributed to the observed result. The
execution record does not establish whether the Codex-tree failure was caused
by a content, metadata, or traversal difference; that diagnosis remains
`UNKNOWN`. The external symbolic-link destination condition remains
`NOT VERIFIED` and is outside this correction's scope.

### AC-14 external symbolic-link destination coverage attempt

This is a separate local investigation record. It does not replace previous
preflight or E2E results. No host preflight, E2E, provider request, or scanner
was run for this entry.

| Claim | Classification | Observation |
| --- | --- | --- |
| External destination content coverage by the current tree fingerprint | `FACT` | A temporary fixture changed an external target's same-size content while leaving its symlink unchanged; the current `.codex` tree fingerprint remained equal. `rtk proxy python3 -m unittest scripts.test_e07_host_integrity.HostIntegrityTests.test_tree_fingerprint_detects_symlink_destination_change -v` exited `1`, confirming this coverage gap. |
| Initial symlink fixture | `FACT` | An initial fixture replaced the symlink itself and passed; it was discarded because the changed symlink inode did not prove detection of an unchanged link's referent. It is not counted as coverage. |
| Temporary external-target test prototype | `FACT` | `rtk proxy python3 -m unittest scripts.test_e07_host_external_links -v` exited `1`: 11 tests ran, 8 passed, 1 failed, and 2 errored. The prototype was removed and these results are not counted as passing regression coverage. |
| Symlink `atime` after `readlink` | `FACT` | A synthetic filesystem check returned `symlink_atime_preserved=False` (command exit `0`). |
| Symlink `atime` after opening through the link with `O_NOATIME` | `FACT` | A synthetic filesystem check returned `followed_symlink_atime_preserved=False` (command exit `0`). |
| Symlink `atime` after adding an inotify watch through the link path | `FACT` | A synthetic filesystem check returned `inotify_symlink_atime_preserved=False` (command exit `0`). |
| First consolidated host-integrity regression run | `FACT` | The command `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity scripts.test_e07_host_integrity_mutations -v` exited `1` because the directory-replacement assertion failed while the accepted Correction 1 entry revalidation was temporarily absent. |
| Existing host-integrity regression suite | `FACT` | `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity scripts.test_e07_host_integrity_mutations -v` exited `0`; all 16 tests passed after restoring the accepted directory-entry revalidation. |
| External destination integrity during a real E2E | `NOT TESTED` | This execution did not observe the host across an E2E run. |
| AC-14 disposition for this execution | `NOT VERIFIED` | External referent coverage is not implemented, and the atime-preserving observation requirement remains unresolved. |

The failed focused fixture used only temporary synthetic files and emitted no
environment content in this report. Experimental source and test changes
from this investigation were removed; the accepted host-tree directory-entry
revalidation from Correction 1 remains in place. No claim is made that the
tested atime methods cover every possible kernel mechanism. AC-14 remains
unresolved and E07 remains `INCOMPLETE`.

### AC-14 symlink `atime` experiment — descriptor-based `readlinkat`

This is a separate synthetic experiment; it did not inspect the host Codex tree
or any environment file. The fixture captured `lstat` metadata before and
after opening the symlink with `O_PATH | O_NOFOLLOW | O_NOATIME` and calling
`readlinkat` with an empty path. Output contained booleans only.

| Claim | Classification | Observation |
| --- | --- | --- |
| `readlinkat` through the symlink descriptor | `FACT` | The call succeeded on both available fixture mounts. |
| `relatime` mount, atime update eligible | `FACT` | Symlink atime changed; all other compared metadata remained equal. |
| `relatime` mount, atime update suppressed | `FACT` | Symlink atime and all other compared metadata remained equal. |
| `noatime` mount, both timestamp conditions | `FACT` | Symlink atime and all other compared metadata remained equal. |
| Host `~/.codex` mount policy | `FACT` | `relatime`; this was read from mount options only. |
| `openat2` following a synthetic symlink with `O_NOATIME` | `FACT` | Open succeeded, other symlink metadata remained equal, but symlink atime changed. |
| AC-14 external-target verification | `NOT VERIFIED` | The host symlink source is on a `relatime` mount, and the tested descriptor-based resolution changes atime in an eligible condition. No external-target implementation or host integrity run was made. |

Commands exited `0`. The reproductions used only temporary synthetic links and
targets, made no mount or system configuration changes, and printed no target
text, content, metadata values, paths, or hashes. The smallest remaining
requirement decision is whether an atime-neutral read view of the host mount
may be used for symlink resolution; no such view was created in this execution.
AC-14 remains unproven and E07 remains `INCOMPLETE`.

### AC-14 private `noatime` view feasibility attempt

This separate fixture-only attempt evaluated the accepted correction's
prerequisite. It did not inspect the host Codex tree, environment files, or
credentials and did not run a preflight, E2E, provider request, or scanner.

| Claim | Classification | Observation |
| --- | --- | --- |
| Standalone mount namespace availability | `FACT` | `unshare --mount true` exited `1`. |
| Bubblewrap private bind/remount fixture | `FACT` | A synthetic bind mount followed by `remount,bind,ro,noatime` exited `1`. |
| Cause of the private-view setup failure | `UNKNOWN` | The fixture command did not retain diagnostic output sufficient to attribute the failure to Bubblewrap setup or the mount operation. |
| `readlinkat` inside the proposed private view | `NOT TESTED` | The view was not established. |
| Persistent implementation change | `FACT` | None. No experimental source or test file was retained. |
| AC-14 disposition | `NOT VERIFIED` | External target coverage remains absent; the atime-preserving view was not available through these attempts. E07 remains `INCOMPLETE`. |

The attempted commands changed no system mount configuration. The host
`~/.codex` mount remains reported as `relatime`; no host file metadata was
read or changed by this feasibility attempt.

### AC-14 private noatime view and fail-closed gate — 2026-11-02

This entry records a new local investigation after the user authorized one
additional E2E pass under the existing budget. It does not replace historical
results. No host Codex tree, project environment file, or credential was read
for the experiments below. No E2E, preflight, provider request, or scanner was
started.

| Claim | Classification | Observation |
| --- | --- | --- |
| Disposable Ubuntu 24.04 filesystem laboratory | `FACT` | A temporary container used synthetic files only, received `CAP_SYS_ADMIN`, mounted no sensitive host directory, and was removed with `--rm`. |
| Mount namespace and `ro,noatime` bind view in the laboratory | `FACT` | `unshare --mount true` and bind/remount with `ro,noatime` exited `0`; mount options reported read-only and noatime. |
| Write protection in the laboratory | `FACT` | Creating a file through the read-only view failed with `EROFS`. |
| `readlinkat` and metadata in the laboratory `noatime` view | `FACT` | `O_PATH | O_NOFOLLOW | O_NOATIME` plus empty-path `readlinkat` succeeded; symlink atime and the compared other metadata were unchanged. |
| Referent read in the laboratory `noatime` view | `FACT` | Opening with `O_NOATIME` and reading the synthetic target preserved its atime and compared metadata. |
| Laboratory `relatime` condition using the same `O_NOATIME` descriptor | `FACT` | The mount reported `relatime`; the eligible fixture did not show an atime change, and the suppressed fixture preserved atime. This differs from the earlier host-filesystem fixture, which did show a symlink atime update. The container result does not establish safety on the host filesystem. |
| Bubblewrap remount as the current namespace user | `FACT` | After correcting an initial diagnostic redirection that prevented the mount command from running, remount exited `1` with `mount: only root can use "--options" option`. |
| Bubblewrap remount as UID 0 in its unprivileged user namespace | `FACT` | Remount exited `32` with `mount: <synthetic mount point>: permission denied`. The diagnostic did not identify a more specific kernel policy. |
| Host external-symlink target coverage | `NOT VERIFIED` | Bubblewrap could not establish the required private noatime view. The Docker result used synthetic container files and is not evidence about host referents. |
| Fail-closed supervisor behavior | `FACT` | The host snapshot now requests symlink-target coverage and refuses a tree containing a symlink until an observer can verify its target. The generic synthetic tree fingerprint remains available to its existing tests. |
| New E2E authorization and consumption | `FACT` | One additional pass was authorized; `0/1` passes started and `0/9` provider calls were consumed because the host integrity precondition could not be established. |
| AC-14 and E07 | `NOT VERIFIED` | AC-14 remains unproven and E07 remains `INCOMPLETE`. The separate operator-approved scanner exception also remains unresolved; no secret-free claim is made. |

The fail-closed behavior was added test-first. The new regression test first
failed because the prior host snapshot accepted an unobserved symlink target,
then passed after the correction. The existing sixteen host-integrity tests
also passed after the correction.

| Check | Result |
| --- | --- |
| `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity_mutations.HostIntegrityMutationTests.test_host_snapshot_fails_closed_when_codex_tree_has_symbolic_links -v` — before correction | `FAIL`, exit `1`, as expected for RED. |
| Same focused test — after correction | `PASS`, exit `0`. |
| `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity scripts.test_e07_host_integrity_mutations -v` | `PASS`, exit `0`, 17 tests. |
| `rtk git diff --check` | `PASS`, exit `0`. |

The host snapshot cannot currently fingerprint external referents without a
private noatime view. It now fails closed rather than treating link metadata as
complete evidence. Consequently, the live supervisor will stop during its
host baseline before any Codex scenario or provider traffic. The Docker
laboratory established that the mount operation works with `CAP_SYS_ADMIN`
inside the disposable container; it did not authorize mounting host personal
directories there and did not provide the host supervisor with that
capability. The provider budget remains unused. Gitleaks remains `NOT RUN`
under the recorded scanner exception.

### Cross-snapshot access-time coverage correction — 2026-11-02

The independent review found that the earlier integrity signature excluded
`st_atime_ns`. Although the helper's own reads were tested not to update access
time, a change between before and after snapshots could therefore be missed.
The signature now includes access time. The earlier plan text that excluded it
from cross-run equality is superseded; any observed access-time change now
fails the integrity comparison.

| Check | Result |
| --- | --- |
| `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity.HostIntegrityTests.test_integrity_signature_includes_access_time -v` — before correction | `FAIL`, exit `1`, as expected for RED. |
| Same focused test — after correction | `PASS`, exit `0`. |
| `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity scripts.test_e07_host_integrity_mutations -v` | `PASS`, exit `0`, 17 tests. |
| `rtk git diff --check` | `PASS`, exit `0`. |

This correction does not establish host external-symlink referent coverage or
change the E2E gate. The host snapshot still fails closed before any provider
traffic, and AC-14 remains `NOT VERIFIED`.

### Missing-root fingerprint correction — 2026-11-02

The independent review also found that the missing-root branch referenced the
digest accumulator before initialization. A regression test first reproduced
the resulting `UnboundLocalError`; initializing the accumulator before root
inspection restored the intended stable digest for an absent tree.

| Check | Result |
| --- | --- |
| `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity.HostIntegrityTests.test_tree_fingerprint_is_stable_when_root_is_missing -v` — before correction | `ERROR`, exit `1`, with `UnboundLocalError` as expected for RED. |
| Same focused test — after correction | `PASS`, exit `0`. |
| `rtk proxy env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_e07_host_integrity scripts.test_e07_host_integrity_mutations -v` | `PASS`, exit `0`, 18 tests. |
| `rtk git diff --check` | `PASS`, exit `0`. |

The new pass remains authorized but not started. External symlink referent
coverage is still unavailable in the host mount namespace, so no provider
calls were made and AC-14 remains `NOT VERIFIED`.

### Independent review — 2026-11-02

The independent review returned `APPROVED_WITH_NOTES` with no remaining code
or evidence-recording findings. It confirmed that the host snapshot fails
closed on unobserved symlink referents, access time is included in snapshot
equality, the missing-root fingerprint is stable, the 18 synthetic tests pass,
and the current authorization is explicit in the user-provided task. The
review did not execute E2E, preflight, provider calls, or Gitleaks and did not
read `.env` or credentials. AC-14 remains `NOT VERIFIED` for the host mount
capability and scanner limitations already recorded above.

## Architectural scope disposition and current E07 status (2026-11-02)

This dated addendum supersedes earlier statements that AC-14 is a pending E07
acceptance gate. It does not alter or erase any historical run result. The AI
Hub centralizes provider access; it does not execute agents or administer
their local host environments. Its product boundary covers provider
integrations, context-aware routing and model selection, cost/token policies,
and centralized observability. AC-14 and FR-12 are therefore `OUT OF SCOPE`
for AI Hub product acceptance. This is a scope classification, not a PASS.
Historical reports that recorded a changed Codex tree, incomplete host
integrity evidence, or an unrun secret scan remain valid observations for
those executions and are preserved as written.

The product security obligations remain in scope: provider credential
protection, gateway authentication, isolation of provider credentials from
agent processes, request/response security, and consumption governance.
Repository integrity and Git-scope checks remain development hygiene and are
not product features or AC-14 evidence.

### Current acceptance matrix

The matrix below uses the latest recorded real E2E run (2026-11-01), the
subsequent local AC-12/helper verification records, and the scope decision in
this addendum. Historical status tables above remain unchanged and apply to
their named checkpoints.

| AC | Current disposition | Evidence and limitation |
| --- | --- | --- |
| AC-1 | `PASS` | The documented isolated setup and exact Codex child-environment checks showed the gateway credential present and provider/Hub-only credentials absent. AC-14 is no longer a dependency. |
| AC-2 | `PASS` | Latest real E2E routed tested Codex Responses traffic through AI Hub. |
| AC-3 | `PASS` | Latest real E2E authenticated routing and `401` rejection passed with zero provider-call delta for the rejected request. |
| AC-4 | `PASS` | Basic real-provider generation succeeded. |
| AC-5 | `FAIL` | The second Codex invocation did not reach AI Hub ingress; replay context and marker recall were not established. The later synthetic endpoint regression only proves the local launcher correction. |
| AC-6 | `PASS` | Real sandboxed tool round trip passed with normal approvals. |
| AC-7 | `PASS` | Same-response upstream error fidelity passed. |
| AC-8 | `PASS` | Gateway streaming passed. Codex stdout progressive rendering remains `NOT VERIFIED`, which AC-8 permits when stated separately. |
| AC-9 | `PASS` | Live execution records passed correlation, listing, schema/outcome, and by-ID checks. |
| AC-10 | `PASS` | Live records remained queryable after orderly restart. |
| AC-11 | `PASS` | The E2E record contains scenario outcomes and limits without credentials, payloads, or generated identifiers. |
| AC-12 | `PASS` | The recorded local quality gates passed; the later host-integrity helper's 18 focused tests also passed. No production source changed. |
| AC-13 | `PASS` | The README retains experimental wording and makes no Codex support claim. |
| AC-14 | `OUT OF SCOPE` | Excluded from AI Hub product acceptance by the architectural decision above; no historical failure is reclassified as PASS. |

E07 remains **INCOMPLETE** because AC-5 is `FAIL`. AC-14's removal does not
approve E07 or establish Codex support. The separately recorded additional E2E
authorization remains unused; this documentation change neither consumes nor
renews it. Documentary verification for this update: `rtk git diff --check`
passed with exit code `0`. No provider call, E2E, preflight, or secret scan was
performed for this scope decision.

The only currently pending E07 acceptance criterion is AC-5. All other
in-scope criteria are `PASS` in the current matrix above.

### Preserved history and current references

The SPEC, design, current E2E plan, and this report now carry the normative
`OUT OF SCOPE` disposition. These E07 snapshots retain their dated AC-14
assessments as historical evidence and are not current acceptance definitions:

- `docs/discovery/E07-failure-analysis.md`
- `docs/discovery/E07-final-acceptance-readiness.md`
- `docs/validation/E07-environment-blocker-resolution.md`
- `docs/validation/E07-phase-8b-baseline-reconciliation.md`

AC-14 in `docs/specs/E04-gateway-hardening-access-control.md` and
`docs/design/E04-gateway-hardening-access-control-design.md`, and FR-12 in
`docs/specs/E06-execution-records.md`, belong to those epics and are unrelated
to this E07 disposition.
