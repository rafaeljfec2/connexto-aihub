# E07 Codex First Harness Validation Report

Status: **INCOMPLETE — Codex CLI support is not established.**

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
| Symlink targets below a directory were not hashed | Symbolic links now record the target and digest resolvable target content, with directory-link cycle protection and an entry bound. |
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
  `TestE07UpstreamProbeDoesNotRequireCrossCallBodyEquality` — envelope-only
  classification, status preservation, and no cross-call body equality.

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
real-provider acceptance criterion: AC-5, AC-7, AC-9, AC-10, and AC-14 still
require a newly authorized real validation pass. The global Codex-tree change
source remains `UNDETERMINED`.

Execution-record guarantees remain E06 best-effort and at-most-once. The
writer's drain window is two seconds; queue overflow, storage failure, crash,
retention, or flush expiry can lose records. This run does not establish
preservation outside normal orderly shutdown.
