# E07 Phase 8B Baseline Reconciliation

Status: **PREPARATION COMPLETE — E07 remains INCOMPLETE.**

Scope note (2026-11-02): this is a historical baseline-reconciliation record.
Its AC-14 assessments and scanner/credential decisions are preserved as
evidence from that phase; the current E07 SPEC classifies AC-14 and FR-12 as
`OUT OF SCOPE` for the AI Hub product. E07 remains incomplete because AC-5
failed in the latest real E2E evidence.

Assessment date: 2026-11-01.

## 1. Scope and Evidence

This report reconciles the E07 working tree, records the scanner and credential decisions still required, and prepares a clean-baseline procedure and a recommendation for a future E2E plan. It does not approve a scanner or E2E plan, establish a clean baseline, manipulate credentials, start AI Hub or Codex, run tests, execute a scanner, or send provider requests.

The approved E07 SPEC remains the source of acceptance criteria. The current evidence was checked against the E07 readiness, blocker, failure-analysis, validation, design, SPEC, and proposed E2E documents. The SDD evidence claim for this report is manual and command-based; this documentation-only change has no automated-test requirement. No E2E or live validation is authorized here.

ai-memory was consulted. Its relevant search results were historical session records; they were not opened or used to resume prior work. Current checkout and repository documents are the evidence for this assessment.

## 2. Current Git State

| Check | Verified state |
| --- | --- |
| Branch and HEAD | `main`, `a8ac19af7cf6457e30f3eb284965c4966ecf033b` |
| Local upstream ref | `origin/main` equals HEAD; `git status` reports `0` ahead and `0` behind |
| Remote `main` | `git ls-remote origin refs/heads/main` returned the same revision |
| Remediation commit | `40196001536b6d07084b8bbd671d646bdd674d32` is an ancestor of HEAD |
| Staged files | None |
| Pre-existing changes at phase start | 11 modified tracked files and 17 untracked files: 28 E07 artifacts |
| This report | Added by Phase 8B; it is a 29th E07 artifact and is untracked |
| Clean baseline | **NO — the tree remains dirty** |

The 28-item inventory below is the complete pre-existing set. The final row records this report separately so the inventory remains auditable without treating a document as present before its creation. No ignored `.env` contents were inspected. Credential checks inspected only the current tool process and emitted presence states.

## 3. Artifact Reconciliation Matrix

Classification uses the artifact's observed function. Git paths and names do not establish authorship. Every row still needs a human provenance and disposition decision before staging or committing.

| Path | Git state | Classification | Responsibility and dependencies | Review outcome, group, and decision |
| --- | --- | --- | --- | --- |
| `cmd/ai-hub/e07_integrity_remediation_test.go` | M | `VALIDATION_TEST` | Root discovery and environment-file fixture tests; exercises integrity helpers. | E07-focused; group `CODE-INT`; human provenance review required. |
| `cmd/ai-hub/e07_integrity_test.go` | M | `VALIDATION_HELPER` | Snapshot types, repository-root/environment discovery, fingerprints, comparison; used by integrity and live setup checks. | E07-focused; group `CODE-INT`; human provenance review required. |
| `cmd/ai-hub/e07_integrity_manifest_support_test.go` | ?? | `VALIDATION_HELPER` | Private manifest location, permissions, serialization, and load helpers; depends on integrity snapshots. | E07-focused; group `CODE-INT`; human provenance review required. |
| `cmd/ai-hub/e07_integrity_manifest_test.go` | ?? | `VALIDATION_TEST` | Manifest privacy, round-trip, redaction, and production-baseline tests; depends on integrity helpers. | E07-focused; group `CODE-INT`; human provenance review required. |
| `cmd/ai-hub/e07_integrity_tree_test.go` | ?? | `VALIDATION_HELPER` | Tree/symlink hashing, Git state, tracked-file hashing, and E06 production-baseline comparison; tested with integrity fixtures. | E07-focused; group `CODE-INT`; human provenance review required. |
| `cmd/ai-hub/e07_live_auth_support_test.go` | M | `VALIDATION_HELPER` | Synthetic provider and isolated auth-probe Hub fixture; depends on the validation Hub and provider observer. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_live_auth_test.go` | M | `VALIDATION_TEST` | Unauthenticated `401` and zero-provider-delta assertion; depends on the auth fixture and shared budget observer. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_live_http_test.go` | M | `VALIDATION_HELPER` | E06 execution-record types and authenticated list/by-ID reader; used by record assertions. | E07-focused; group `CODE-RECORDS`; human provenance review required. |
| `cmd/ai-hub/e07_live_record_assertions_test.go` | ?? | `VALIDATION_HELPER` | Live list, by-ID, and restart assertion helpers moved out of the overlength HTTP test file; contains no `TestE07` functions. | E07-focused; review with the extraction in `CODE-RECORDS`; human provenance review required. |
| `cmd/ai-hub/e07_live_hub_test.go` | M | `VALIDATION_HELPER` | Validation-only Hub composition, lifecycle, request capture, and provider-client timeout wiring; depends on observer and records helpers. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_live_scenario_support_test.go` | M | `VALIDATION_HELPER` | Replay diagnostics, auth probe, and same-response upstream-error observation; depends on Codex, provider-observer, and fidelity helpers. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_live_scenarios_test.go` | M | `VALIDATION_TEST` | Opt-in E2E orchestration; now wires the bounded provider transport and observer. | Test-only; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_observer_support_test.go` | M | `VALIDATION_HELPER` | Replay/terminal observation and exact Codex child-environment allowlist; provider-observer implementation was extracted to its own file. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_provider_budget_support_test.go` | ?? | `VALIDATION_HELPER` | Nine-call ceiling, mutex-protected reservations/counters, and non-retrying transport constructor. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_provider_budget_test.go` | ?? | `VALIDATION_TEST` | Sequential, concurrent, redirect, startup/restart, unauthenticated, shared-budget, and retry tests. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_provider_observer_test.go` | ?? | `VALIDATION_HELPER` | Shared provider observer, budget enforcement, replay observation, and same-response digest capture. | E07-focused; group `CODE-LIVE`; must remain with observer/fidelity dependencies; human provenance review required. |
| `cmd/ai-hub/e07_provider_timeout_test.go` | ?? | `VALIDATION_TEST` | Fake-transport cancellation test for the provider HTTP timeout helper. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_response_fidelity_test.go` | ?? | `VALIDATION_TEST` | In-memory digest observation and complete-EOF/read-error fidelity tests. | E07-focused; group `CODE-LIVE`; depends on provider observer; human provenance review required. |
| `cmd/ai-hub/e07_upstream_response_test.go` | ?? | `VALIDATION_TEST` | Generic error-envelope and same-response probe assertions; depends on live scenario support. | E07-focused; group `CODE-LIVE`; human provenance review required. |
| `cmd/ai-hub/e07_records_test.go` | M | `VALIDATION_TEST` | E05 terminal-observation record expectation test and deterministic record fixture; large test extraction is split into record files below. | E07-focused; group `CODE-RECORDS`; human provenance review required. |
| `cmd/ai-hub/e07_record_assertions_test.go` | ?? | `VALIDATION_TEST` | Independent list, correlation, outcome, schema, and by-ID assertion tests; depends on record reader/helpers. | E07-focused; group `CODE-RECORDS`; human provenance review required. |
| `cmd/ai-hub/e07_record_context_test.go` | ?? | `VALIDATION_TEST` | Poll-deadline and fresh post-restart context tests; depends on record assertion helpers. | E07-focused; group `CODE-RECORDS`; human provenance review required. |
| `cmd/ai-hub/e07_record_outcomes_test.go` | ?? | `VALIDATION_TEST` | E05 completed-response, transport-failure, and body-read-failure outcome tests. | E07-focused; group `CODE-RECORDS`; human provenance review required. |
| `docs/validation/E07-codex-first-harness-integration.md` | M | `EVIDENCE_DOCUMENT` | Historical E2E outcomes and deterministic remediation/budget evidence; references the E07 tests and proposed plan. | Keep as timestamped validation evidence; group `DOC-EVIDENCE`; human provenance review required. |
| `docs/discovery/E07-final-acceptance-readiness.md` | ?? | `EVIDENCE_DOCUMENT` | Acceptance-criteria matrix and current gaps; depends on SPEC, failure analysis, and validation report. | Point-in-time evidence; group `DOC-EVIDENCE`; human provenance review required. |
| `docs/validation/E07-environment-readiness.md` | ?? | `EVIDENCE_DOCUMENT` | Prior environment preflight, credentials, scanner, budget, and readiness status. | Overlaps later blocker report on prerequisites but records a distinct assessment; group `DOC-EVIDENCE`; human provenance review required. |
| `docs/validation/E07-environment-blocker-resolution.md` | ?? | `EVIDENCE_DOCUMENT` | Latest prior blocker assessment and initial 27-file inventory; references readiness, acceptance, validation, and E2E plan documents. | Reconcile its initial inventory with the current 28-item set; group `DOC-EVIDENCE`; human provenance review required. |
| `docs/validation/E07-final-e2e-execution-plan.md` | ?? | `SPEC_OR_DESIGN` | Proposed ordered E2E, budget, preconditions, evidence, and stop rules. | Remains `PROPOSED`; group `DOC-PLAN`, never treat inclusion or commit as approval; human decision required. |
| `docs/validation/E07-phase-8b-baseline-reconciliation.md` | ?? (created in this phase) | `EVIDENCE_DOCUMENT` | Current inventory, scanner proposal, credential procedure, baseline steps, and future-plan recommendation. | This report; group `DOC-EVIDENCE`; independent review pending. |

No artifact is classified as unrelated production code. No authorship or shared origin is inferred from these paths. Potential documentation overlap is recorded above; the readiness and blocker reports have different timestamps and scope, so this review does not recommend deleting or overwriting either one.

## 4. Forensic Review Conclusions

### Validation integrity

The split integrity files now cover Git-root discovery, root/nested environment files, already-dirty tracked content, untracked production files, symlink targets/cycles, private digest-only manifests, and category-only summaries. The production-baseline helper compares the E06 production paths and rejects untracked production files. These are validation-only helpers. The existing validation report records the focused checks as passing; this phase did not rerun them. The reported entry-count bound and unbounded streamed file size remain limitations, and no new live integrity result exists.

### Records and response fidelity

The record helpers derive expected outcomes from E05 terminal observations rather than HTTP status. List, by-ID, schema, correlation, and restart assertions are separate; per-query contexts are bounded, and restart checks use a fresh context. The response-fidelity helper compares digests only for one fully read response, discards body bytes after comparison, and treats incomplete/read-error bodies as unverified. Deterministic evidence does not establish live provider fidelity, record completeness, or restart recovery.

### Provider-call budget

The current test harness defines a maximum of nine calls. `e07ProviderObserver.RoundTrip` reserves under a mutex and rejects excess calls before forwarding. The same budget can be shared across observers and Hub restarts. Deterministic test cases cover ten sequential attempts, fifty concurrent attempts, redirects, readiness accounting, no unauthenticated-request consumption, no automatic connection retry, and a fake request timeout. The live validation Hub supplies a two-minute HTTP timeout, and the isolated Codex config sets request and stream retries to zero. The existing report records these deterministic tests as passing; this phase verified the code and test declarations only, made no provider call, and consumed no E2E allowance.

### Scenario and documentation consistency

The current reports consistently keep E07 `INCOMPLETE`; the E2E plan is `PROPOSED`, and the two previously authorized passes are consumed. The remediation report records their original outcomes and states that helper fixes do not replace real-provider evidence. Six acceptance criteria remain unresolved in the acceptance assessment: AC-1, AC-5, AC-7, AC-9, AC-10, and AC-14. AC-13 may remain `NOT VERIFIED` only under the SPEC condition where gateway streaming passes. The reports describe Snyk Code authentication as a separate historical limitation; it is not evidence of an approved secret scan.

### Scope and source preservation

All 23 pre-existing Go changes are `_test.go` validation code. The other five pre-existing artifacts are E07 documents. No production source, dependency, API, ADR, or E01–E06 contract is changed by the current 28-item set. The current diff from the E06 production baseline contains E07 tests and documentation plus the earlier README documentation change; no production source path differs. Do not promote these helper tests into a production-support claim.

## 5. Proposed Review Clusters and Commit Boundary

The code clusters below are useful review slices, not proven independent commit units. They share package-level helpers across boundaries: `CODE-LIVE` uses integrity summary and record-query/assertion helpers from `CODE-INT` and `CODE-RECORDS`, while record summaries also use `e07IntegrityResult` from `CODE-INT`. Until an independently buildable and testable closed boundary is demonstrated, recommend all 23 Go files as one `CODE-ALL` commit unit. Human provenance may still require a different disposition, which needs an owner decision.

| Review cluster | Files | Responsibility and dependencies | Focused review |
| --- | --- | --- | --- |
| `CODE-INT` | Five integrity rows in §3: `e07_integrity_remediation_test.go`, `e07_integrity_test.go`, `e07_integrity_manifest_support_test.go`, `e07_integrity_manifest_test.go`, `e07_integrity_tree_test.go`. | Snapshot, manifest, tree, and root-discovery helpers. Defines package-level integrity summary helpers used by live-scenario and record-summary code. | Path coverage, redaction, symlink behavior, and no production-baseline false pass. |
| `CODE-RECORDS` | Six record rows in §3: `e07_live_http_test.go`, `e07_live_record_assertions_test.go`, `e07_records_test.go`, `e07_record_assertions_test.go`, `e07_record_context_test.go`, `e07_record_outcomes_test.go`. | Record readers, fixtures, E05 outcome expectations, assertions, and restart contexts. The live scenarios call these readers/assertions; summaries use `e07IntegrityResult`. | E05 terminal semantics, E06 list/by-ID behavior, correlation, and independent assertions. |
| `CODE-LIVE` | Twelve live/provider rows in §3: `e07_live_auth_support_test.go`, `e07_live_auth_test.go`, `e07_live_hub_test.go`, `e07_live_scenario_support_test.go`, `e07_live_scenarios_test.go`, `e07_observer_support_test.go`, `e07_provider_budget_support_test.go`, `e07_provider_budget_test.go`, `e07_provider_observer_test.go`, `e07_provider_timeout_test.go`, `e07_response_fidelity_test.go`, `e07_upstream_response_test.go`. | Budget, observer, Hub wiring, same-response capture, timeout, and live assertions. Depends on integrity summaries and record readers/assertions from the other clusters. | Concurrency, pre-forward rejection, redirects/retries, request timeout, redaction, and auth isolation. |

| Proposed commit unit | Files | Required validation and review | Suggested commit message | Production behavior |
| --- | --- | --- | --- | --- |
| `CODE-ALL` | All 23 Go files in `CODE-INT`, `CODE-RECORDS`, and `CODE-LIVE`. | After an explicit owner decision: focused E07 tests, `go test ./... -count=1 -race`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run ./...`, `go build ./...`, formatting, and `git diff --check`; independent review of the complete Go change. Do not run the opt-in live scenario in local validation. | `test(e07): complete deterministic validation harness` | No. |
| `DOC-EVIDENCE` | `docs/validation/E07-codex-first-harness-integration.md`, `docs/discovery/E07-final-acceptance-readiness.md`, `docs/validation/E07-environment-readiness.md`, `docs/validation/E07-environment-blocker-resolution.md`, and this report. | Manual link, status, acceptance-criteria, privacy, English-language, and diff review; independent evidence review. No automated tests. | `docs(e07): reconcile validation readiness evidence` | No. |
| `DOC-PLAN` | `docs/validation/E07-final-e2e-execution-plan.md`. | Manual comparison to current blockers, nine-call counter, two-minute request cap, stop/cleanup rules, and SPEC acceptance criteria. Keep it `PROPOSED`; do not execute it. | `docs(e07): propose bounded final validation plan` | No. |

### Traceability

| Review cluster | SPEC and Design obligations | Validation evidence | Test/helper evidence |
| --- | --- | --- | --- |
| `CODE-INT` | `docs/specs/E07-codex-first-harness-integration.md` AC-1, AC-12, AC-14 and FR-12 private-manifest requirement; `docs/design/E07-codex-first-harness-integration-design.md` in-memory before/after integrity fingerprints. | `docs/discovery/E07-final-acceptance-readiness.md` AC-1/12/14 and integrity/remediation evidence in `docs/validation/E07-codex-first-harness-integration.md`. | `e07_integrity_remediation_test.go`, `e07_integrity_manifest_test.go`, and `e07_integrity_tree_test.go`; `e07IntegrityResult` is shared by summaries. |
| `CODE-RECORDS` | `docs/specs/E07-codex-first-harness-integration.md` AC-9 and AC-10; `docs/design/E07-codex-first-harness-integration-design.md` record correlation, list/by-ID lookup, and restart recovery. | `docs/discovery/E07-final-acceptance-readiness.md` AC-9/10 and record remediation evidence in `docs/validation/E07-codex-first-harness-integration.md`. | `e07_record_assertions_test.go`, `e07_record_context_test.go`, `e07_record_outcomes_test.go`, and `e07_live_record_assertions_test.go` (helpers used by the scenarios). |
| `CODE-LIVE` | `docs/specs/E07-codex-first-harness-integration.md` AC-2 through AC-8 and AC-13; `docs/design/E07-codex-first-harness-integration-design.md` isolated live harness, bounded provider transport, and response observation. | `docs/discovery/E07-final-acceptance-readiness.md` AC-2 through AC-8/13 and budget, fidelity, and live-run evidence in `docs/validation/E07-codex-first-harness-integration.md`. | `e07_provider_budget_test.go`, `e07_provider_timeout_test.go`, `e07_response_fidelity_test.go`, `e07_upstream_response_test.go`, and `e07_live_scenarios_test.go`. |

These are recommendations only. Artifact disposition, review boundaries, staging, and commits each require explicit approval from the project owner when that action is proposed. Reviewer approval or inclusion in a clean baseline does not authorize staging, commit, push, or E2E execution. No file was staged or committed in this phase.

## 6. Secret-Scanner Governance Proposal

Verdict remains **`APPROVED_SCANNER_NOT_ESTABLISHED`**. No repository governance file, E07 SPEC, or current report formally approves a scanner. Gitleaks is a candidate only.

The official release page and tagged README checked on 2026-11-01 identify `v8.30.1` as the latest release and document `gitleaks dir`, the CLI `--redact` option, and exit code `1` for either a finding or an error. The release page dates it to 2026-03-21. Approval remains blocked pending review of material issues reported against this release: issue #2086 says the tag points outside the `master` history, and the maintainer acknowledged a release error. This does not by itself prove the binary is defective, but release provenance must be established before approval. Issues #2293 and #2296 also report allowlist coverage and output-redaction risks; the proposal below treats them as unresolved until verified against the exact candidate version. No scanner was run.

| Policy field | Proposed decision for governance review |
| --- | --- |
| Scanner and pin | Gitleaks `v8.30.1`, pinned exactly; version changes require another review. |
| Official source and binary verification | First establish the `v8.30.1` tag/release provenance with the maintainers because of issue #2086. If approved, use only the official `gitleaks/gitleaks` release asset for the target OS/architecture. Verify the exact asset against that release's checksum file, independently compute and record its SHA-256, confirm the binary reports the pinned version, and stop on any mismatch. Do not substitute package-manager binaries or an unpinned container tag. |
| Coverage | Use filesystem `dir` mode over the current checkout, not Git-history mode. Include tracked and untracked working-tree files, hidden files, and Markdown. Exclude only `.git` internals. Verify traversal and detection with synthetic positive and negative canaries, including a Markdown positive canary, and reconcile the scanner's input file set. Incomplete coverage or a missed canary is not a pass. |
| Exclusions and allowlists | Start with no path exclusions, no commit baseline, and no allowlist. Issue #2293 reports that the default allowlist may suppress custom-rule matches for some values, including values containing `false`, starting with `true`, or ending with `null`. Before approval, reproduce or rule out those cases using synthetic custom-rule canaries against the exact version/configuration. A false positive stops the scan for human triage. Any later exception must be narrow to one rule/finding, documented with rationale, and separately approved; never allowlist a directory or credential pattern broadly. |
| Execution mode | Pre-provision and verify the pinned binary through the approved channel. Run locally with outbound network disabled. No download, install, build, or scanner execution is part of Phase 8B. |
| External secret verification | None. Do not submit candidate findings to providers or third-party verification services. |
| Redacted reporting | `--redact` is a CLI option, not a TOML setting. Issue #2296 reports that a `Finding` output line may still disclose the match, so `--redact=100` alone is insufficient evidence of safe output. Before scanning real repository content, exercise every stdout/stderr/report channel with synthetic canaries and confirm matches cannot be displayed or retained in clear text. If that cannot be guaranteed, do not approve this scanner workflow. Retain only version, verified asset digest, input revision, configuration, completion status, and aggregate redacted result; never publish secret values, matching lines, or unnecessary paths. |
| Exit interpretation | `0` is eligible for “no findings” only when traversal completed and all coverage checks passed. `1` means finding **or operational error** and is a stop requiring secure human triage. Unknown flags and every other nonzero result are scan failures. Never convert an incomplete run into a pass. |
| Reproducibility | Record exact version, official asset URL, verified binary digest, command/config, source revision, and redacted aggregate result. Repeat after any file or configuration change that affects the candidate baseline. |

The exact governance decision is whether to approve Gitleaks `v8.30.1` after release provenance is established, the platform asset is independently verified, positive/negative and allowlist canaries pass, current-tree coverage is demonstrated, and every output channel is shown to protect real matches. The decision must also approve the exclusions, offline execution, and reproducible redacted report policy. Until then, retain `APPROVED_SCANNER_NOT_ESTABLISHED`; do not install, run, or approve Gitleaks.

## 7. Credential Rotation and Provisioning Procedure

Current tool-process presence checks returned `AIHUB_GATEWAY_API_KEY=ABSENT` and `OPENCODE_API_KEY=PRESENT`. No value was read, printed, copied, or used. The Hub and Codex E2E environments do not exist in this phase and remain `NOT_VERIFIABLE`.

The task brief requires treating potentially exposed credentials cautiously. The operator, not this phase, must:

1. Revoke or rotate the potentially exposed gateway and provider credentials through their issuers; do not paste values into chat.
2. Provision distinct replacement credentials through the approved secret manager or process-credential broker. Do not reuse one key for both roles.
3. Launch AI Hub with an explicit process environment containing only its required nonsecret runtime variables plus `AIHUB_GATEWAY_API_KEY` and `OPENCODE_API_KEY`.
4. Use the existing E07 Codex `env_key` mechanism in its isolated, pre-created `CODEX_HOME`: the current helper writes `env_key = "AIHUB_CLIENT_KEY"` and supplies the gateway credential under that process variable. Do not modify or reuse shared global Codex state for the run.
5. Keep secret values out of arguments, shell history, TOML, `.env` files, repository files, logs, fixtures, and evidence. TOML may contain the `env_key` variable name only.
6. Immediately before each authorized launch, inspect the exact process environment and report only `PRESENT`, `ABSENT`, or `NOT_VERIFIABLE`. If any result is unexpected, stop before Hub startup or provider traffic.

The replacement values, revocation, and secure injection remain operator actions. This phase did not access an external credential panel or create credentials.

## 8. Process-Isolation Requirements and Current Gap

| Process | Allowed secret material |
| --- | --- |
| AI Hub | `AIHUB_GATEWAY_API_KEY`, `OPENCODE_API_KEY` |
| Codex CLI | Gateway credential only, supplied through the isolated provider config’s `env_key` (`AIHUB_CLIENT_KEY` in the current E07 helper) |
| Codex-spawned shell/tools and other child processes | No inherited secrets unless a reviewed task explicitly requires one |

The existing E07 helper builds a restricted environment for the Codex process, but its generated Codex configuration does not set a `shell_environment_policy`. Official Codex documentation states that this policy controls variables passed to spawned commands and supports `inherit = "none"` plus explicit nonsecret values/filters. Therefore the current helper does **not** establish the required no-secret inheritance for Codex tool subprocesses. This is a readiness gap, not evidence that a particular secret leaked.

Before any separately authorized E2E pass, the isolated Codex configuration must use an explicit shell-environment policy that excludes the gateway key from spawned tools, retains only required nonsecret runtime variables, and preserves the configured `env_key` in the Codex parent process. Verify the exact configured CLI release with a synthetic environment canary: Codex has the gateway key, spawned shell/tool children do not, and the provider key is absent from both. Preserve sandbox and approval controls. If the configured CLI cannot demonstrate this boundary, stop before the live scenario and request a revised design decision. Do not change global `~/.codex/config.toml` as part of that work.

This check must be added to the future plan and helper verification before execution. Presence in the current tool shell is not proof of isolation.

## 9. Hard Provider-Call Budget Evidence

Static inspection confirms the nine-call guard remains in the checkout:

- The budget accepts limits from one through nine and reserves under a mutex before forwarding to the underlying transport.
- An over-budget attempt returns a local error before the base transport is called; counters distinguish attempted, admitted, rejected, and forwarded calls.
- The validation Hub keeps the same observer/budget across restart. Its provider HTTP client has a two-minute timeout.
- The synthetic unauthenticated `401` path is asserted to leave the budget unchanged. Separate tests cover concurrent attempts, redirects, disabled connection retries, timeout cancellation, and same-response fidelity.
- The isolated Codex config sets `request_max_retries = 0` and `stream_max_retries = 0`.

The existing E07 validation report records focused deterministic tests and the Go quality gates as passing. They were not rerun during Phase 8B. No real provider call was made, no live test ran, and neither consumed pass allowance was reset or reinterpreted.

## 10. Clean-Baseline Procedure

The current dirty tree is not a valid final E2E baseline. After the owner makes the pending decisions, use this sequence:

1. Review every inventory row for provenance, scope, and disposition. Make no deletion, restore, move, rename, or overwrite without an explicit owner decision.
2. Approve the logical boundaries and retain the Go package extraction atomically unless a smaller group is independently shown to build and test.
3. Review the complete diff for the chosen files. Confirm only E07 validation tests and documentation changed; compare production paths to E06 and require no production difference.
4. Obtain explicit staging/commit authorization. Stage only the reviewed file list, run the required local quality gates, obtain independent review, and commit the approved bundles. Do not push without separate authorization.
5. After all approved documentation is final, obtain scanner governance approval, make the pinned scanner available through its approved channel, scan the final checkout including tracked/untracked Markdown, and retain only redacted aggregate evidence. Any edit after the scan requires a new scan.
6. Verify the resulting commit ID, branch/upstream relation, empty `git status --porcelain`, E06 production-path equivalence, and the exact approved E07 file scope. Record the commit as the immutable proposed baseline; do not call it clean until the empty status check passes.
7. Re-run the approved non-provider preflight against that exact revision immediately before any later E2E authorization. If a clean approved revision cannot be produced in this branch, use a separate reviewed worktree at the exact approved revision.

These are proposed future actions only. Each action requires explicit approval from the project owner when it is presented for execution. Phase 8B did not stage, commit, push, or clean the tree.

## 11. Recommendation for a Revised E2E Plan

The current `E07-final-e2e-execution-plan.md` remains **`PROPOSED`**. Do not edit or approve it in this phase. A later revision should carry forward both consumed passes, the helper-remediation evidence, and the six unresolved criteria while requiring all of the following as preconditions:

- A reviewed immutable clean baseline, approved scanner decision, completed redacted scan, credential rotation/provisioning, and verified Hub/Codex/child-process isolation.
- One fresh pass only, up to four Codex scenario invocations and at most nine provider-boundary calls: four Codex requests, one same-response error probe, one stream, and up to three identified readiness calls.
- At most two minutes per provider request, 30 minutes total including setup/cleanup/integrity checks, and zero automatic retries. Keep the existing five-minute per-Codex-invocation cap unless a separately reviewed plan changes it.
- No unbudgeted provider/model probe. Identify every readiness call before execution. Reject call ten before transport and stop on unexpected accounting.
- Stop before provider traffic for any missing prerequisite. After traffic begins, stop on credential exposure/isolation failure, integrity delta, sandbox or approval bypass, unexpected call count, budget/time exhaustion, or Hub lifecycle failure. Do not retry a failed scenario.
- Full private integrity checkpoints before execution, after each Codex/provider-bearing scenario before further traffic, and after cleanup. Cover the complete Codex state tree, project `.env` files without opening their values, repository content (tracked and untracked), and Git scope. Compare privately, report categories only, and remove temporary manifests/state.
- Graceful shutdown/restart, post-run verification, redacted scenario evidence, cleanup, and independent final review. Any required AC not established keeps E07 `INCOMPLETE` and forbids a support claim.

These numbers and steps are recommendations only. A new plan, the exact pass/call/time budget, and execution each require separate explicit approval. This report grants none of them.

## 12. Human Decisions and Remaining Blockers

- **Scanner governance:** approve or reject Gitleaks `v8.30.1` and the exact policy in §6.
- **Credential owner:** revoke/rotate potentially exposed keys and provision distinct replacements through a secure, noninteractive process.
- **Process isolation:** review and verify the Codex spawned-command environment policy; the existing helper does not yet prove child-process secret exclusion.
- **Project owner:** decide artifact provenance/disposition and approve the commit boundaries; no author is inferred here.
- **Commit gate:** separately authorize any staging/commit action; no commit or push is authorized by this report.
- **E2E gate:** independently review and approve a revised plan and fresh budget, then provide a separate execution authorization.
- **Acceptance:** the previous real-provider failures and global Codex-tree integrity result remain unresolved. Deterministic helpers do not replace live evidence.

## 13. Next Phase

The proposed next phase is **E07 Phase 8C — Operator Decisions and Baseline Establishment**. It begins only after the human decisions above. It should resolve scanner governance, credential rotation/provisioning, child-process environment filtering, artifact disposition, local validation, independent review, and the clean approved baseline. A separate plan-review/approval gate must follow before any E2E execution. E07 remains `INCOMPLETE` until every required acceptance criterion is established.

## 14. SDD Evidence and External References

- **Claim:** This report records the current E07 artifacts and blockers, recommends technically coherent baseline groups, and preserves all explicit approval gates without disclosing credential values or authorizing E2E.
- **Success type:** `manual` and `command`.
- **Evidence path:** Git status/HEAD/upstream/remote/ancestry; presence-only process checks; direct review of the 28-item set and relevant E07 sources; cross-check against the approved SPEC, Design, readiness, failure-analysis, validation, and proposed-plan documents; independent review of this report.
- **No-test rationale:** The only new artifact is documentation. No executable behavior changed; no live test, provider request, scanner run, or E2E is needed or permitted in this phase.

Official scanner references checked on 2026-11-01:

- [Gitleaks v8.30.1 release](https://github.com/gitleaks/gitleaks/releases/tag/v8.30.1)
- [Gitleaks README at v8.30.1](https://github.com/gitleaks/gitleaks/blob/v8.30.1/README.md)
- [Gitleaks issue #2086 — release tag provenance](https://github.com/gitleaks/gitleaks/issues/2086), open at the time of review; the maintainer acknowledged a release error.
- [Gitleaks issue #2293 — default allowlist and custom-rule matches](https://github.com/gitleaks/gitleaks/issues/2293), open at the time of review.
- [Gitleaks issue #2296 — redaction and finding output](https://github.com/gitleaks/gitleaks/issues/2296), open at the time of review.
- [Gitleaks checksum report issue #2164](https://github.com/gitleaks/gitleaks/issues/2164), closed after the reporter confirmed its Windows asset was modified in transit; the exact selected asset must still be verified independently.

Official Codex configuration references checked on 2026-11-01:

- [Codex Advanced Configuration — Shell environment policy](https://learn.chatgpt.com/docs/config-file/config-advanced)
- [Codex Configuration Reference](https://learn.chatgpt.com/docs/config-file/config-reference)
- [Codex Environment Variables — provider `env_key`](https://learn.chatgpt.com/docs/config-file/environment-variables)

Independent review completed in three rounds: Round 1 and Round 2 returned `APPROVED_WITH_NOTES`; Round 3 returned `APPROVED`. The findings from the first two rounds were corrected within scope. The reviewer did not execute tests, a scanner, AI Hub, Codex, or provider calls. This report does not establish a clean baseline, approve a scanner, rotate credentials, approve a plan, or authorize E2E.
