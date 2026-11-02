# E07 Environment Blocker Resolution

Status: **BLOCKED — environment readiness is incomplete; one final E2E pass is conditionally authorized.**

Scope note (2026-11-02): this report and its addenda preserve historical
environment and AC-14 observations. The current E07 SPEC classifies AC-14 and
FR-12 as `OUT OF SCOPE` for the AI Hub product; those observations are not
current product acceptance gates. E07 remains incomplete because AC-5 failed
in the latest real E2E evidence.

Assessment date: 2026-11-01.

## 1. Scope and Evidence

At the time of the original Phase 8B assessment, this phase reconciled the E07 environment blockers and prepared a proposal for a later readiness review. That assessment did not run tests or E2E, start AI Hub or Codex, contact a provider, consume a provider-call budget, rotate credentials, or change files other than this report. The later conditional final-acceptance preflight and its test results are recorded in the addenda below and in the main validation report. E07 remains **INCOMPLETE**.

The repository rules and SDD references were read. The E07 evidence cross-checked was:

- [`E07-environment-readiness.md`](E07-environment-readiness.md)
- [`E07-final-e2e-execution-plan.md`](E07-final-e2e-execution-plan.md)
- [`E07-final-acceptance-readiness.md`](../discovery/E07-final-acceptance-readiness.md)
- [`E07-codex-first-harness-integration.md`](E07-codex-first-harness-integration.md)
- [`E07-codex-first-harness-integration.md` SPEC](../specs/E07-codex-first-harness-integration.md)

ai-memory was consulted for historical context; repository state and the cited E07 artifacts remain the evidence for this assessment.

## 2. Initial Git State and Inventory

| Check | Initial result |
| --- | --- |
| Branch and HEAD | `main`, `a8ac19af7cf6457e30f3eb284965c4966ecf033b` |
| `origin/main` | Same revision as HEAD |
| Remediation commit | `40196001536b6d07084b8bbd671d646bdd674d32` is an ancestor of HEAD |
| Staged files | None |
| Modified tracked files | 11 |
| Untracked files | 16 |
| Clean baseline | **NO — working tree is dirty** |

The following 27 files were present before this report was created. Classification is based on paths, declarations, and the E07 documents; Git does not establish who authored uncommitted or untracked content.

| File | Initial state | Classification | Purpose and likely origin |
| --- | --- | --- | --- |
| `cmd/ai-hub/e07_integrity_remediation_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | Repository-root and integrity test helpers; E07 validation remediation. |
| `cmd/ai-hub/e07_integrity_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 repository/environment integrity assertions. |
| `cmd/ai-hub/e07_live_auth_support_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 live gateway-auth fixture. |
| `cmd/ai-hub/e07_live_auth_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 unauthenticated gateway probe. |
| `cmd/ai-hub/e07_live_http_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 records HTTP test helpers. |
| `cmd/ai-hub/e07_live_hub_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 validation Hub composition. |
| `cmd/ai-hub/e07_live_scenario_support_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 replay, auth, and upstream-probe helpers. |
| `cmd/ai-hub/e07_live_scenarios_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 live scenario orchestration. |
| `cmd/ai-hub/e07_observer_support_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 observer and process-environment helpers. |
| `cmd/ai-hub/e07_records_test.go` | Modified | `EXPECTED_E07_ARTIFACT` | E07 deterministic record behavior tests. |
| `docs/validation/E07-codex-first-harness-integration.md` | Modified | `EXPECTED_E07_ARTIFACT` | E07 remediation and consumed-pass evidence. |
| `cmd/ai-hub/e07_integrity_manifest_support_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 private integrity-manifest helpers. |
| `cmd/ai-hub/e07_integrity_manifest_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 manifest privacy and baseline tests. |
| `cmd/ai-hub/e07_integrity_tree_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 tree, Git-state, and production-baseline helpers. |
| `cmd/ai-hub/e07_live_record_assertions_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 list, by-ID, and restart record assertions. |
| `cmd/ai-hub/e07_provider_budget_support_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 provider-call budget and transport helpers. |
| `cmd/ai-hub/e07_provider_budget_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 provider-call cap tests. |
| `cmd/ai-hub/e07_provider_observer_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 provider-boundary observer. |
| `cmd/ai-hub/e07_provider_timeout_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 provider HTTP timeout test. |
| `cmd/ai-hub/e07_record_assertions_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 independent record assertion tests. |
| `cmd/ai-hub/e07_record_context_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 query deadline and restart-context tests. |
| `cmd/ai-hub/e07_record_outcomes_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 E05/E06 terminal-outcome tests. |
| `cmd/ai-hub/e07_response_fidelity_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 same-response fidelity helpers and tests. |
| `cmd/ai-hub/e07_upstream_response_test.go` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 upstream error-envelope tests. |
| `docs/discovery/E07-final-acceptance-readiness.md` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 acceptance evidence matrix and gaps. |
| `docs/validation/E07-environment-readiness.md` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 environment gate result. |
| `docs/validation/E07-final-e2e-execution-plan.md` | Untracked at initial inventory; now tracked and modified | `EXPECTED_E07_ARTIFACT` | Initially proposed future-pass plan; conditionally authorized by the later 2026-11-01 prompt, subject to every precondition. |

No staged change, production-source path, credential file, or unrelated path appeared in the initial Git inventory. The test files are within the repository's 300-line source limit. Their E07 purpose is supported by their declarations and references, but this assessment does not certify their behavior or authorship. No file was staged, restored, moved, or removed. This report adds one further untracked E07 document; the final tree therefore remains dirty.

All inventory items are classified as `EXPECTED_E07_ARTIFACT`. No item is identified as an unrelated change or a credential/configuration file. This is not a secret-clearance result: without an approved scanner, file contents across tracked and untracked paths are not established as secret-free. Treat any suspected secret finding as `POTENTIAL_SECRET_EXPOSURE` until an approved, redacted scan and human triage resolve it.

## 3. Scanner Governance

Verdict: **`APPROVED_SCANNER_NOT_ESTABLISHED`**. `AGENTS.md`, the E07 SPEC/plan/reports, repository security/governance references, and available CI/configuration references do not explicitly approve a named secret scanner. Historical mentions of Gitleaks only say it was unavailable. Gitleaks, TruffleHog, detect-secrets, Trivy, and gosec are absent from `PATH`; no scan was run. Installation, download, build, and substitution are outside this phase.

| Candidate | Purpose and local scope | Relevant limitation for this gate |
| --- | --- | --- |
| [Gitleaks](https://github.com/gitleaks/gitleaks) | Purpose-built secret detection using rules and entropy checks; supports local repository/filesystem workflows. | A filesystem invocation must include untracked files; rule coverage is not proof that no secret exists. Findings and reports need redaction. |
| [TruffleHog](https://trufflesecurity.com/docs/filesystem) | Detector-based secret search across filesystem and repository sources; local filesystem scanning is documented. | Optional verification can contact external services. A repository-only scan would need an approved no-verification mode and a reviewed output policy. |
| [detect-secrets](https://github.com/Yelp/detect-secrets) | Code-oriented detector plugins, entropy checks, baselines, and hooks. It supports scanning all files as well as tracked-only workflows. | Baselines and exclusions can suppress findings; approve the plugin set and ensure all tracked and untracked files are included. |
| [Trivy secret scanner](https://trivy.dev/docs/latest/guide/scanner/secret/) | Broader security scanner with filesystem and repository targets and configurable secret rules. | Current documentation says Markdown is excluded by default; E07 documents need explicit coverage. Its broader scope adds configuration and output to review. |

`gosec` is a Go source security analyzer, not a general repository secret scanner for this requirement. A governance owner must choose and approve the tool, pinned version/source, exact current-tree scope (tracked plus untracked), exclusions, network/verification behavior, and a redacted report format. Until then, do not designate any candidate as approved or run it.

## 4. Credential Presence and Handling

Presence checks below report only the permitted status. They inspected the current tool shell process; they did not create or inspect future Hub/Codex process environments.

| Variable | Current tool process | Intended Hub process | Intended Codex process |
| --- | --- | --- | --- |
| `AIHUB_GATEWAY_API_KEY` | `ABSENT` | `NOT_VERIFIABLE` | `NOT_VERIFIABLE` |
| `OPENCODE_API_KEY` | `PRESENT` | `NOT_VERIFIABLE` | `NOT_VERIFIABLE` |
| Configured Codex `env_key` for the gateway credential | `NOT_VERIFIABLE` | Not applicable | `NOT_VERIFIABLE` |

No value was read, printed, copied, or used. The task brief reports that a configuration containing active credentials was shared and instructs treating those credentials as potentially exposed. The repository's harness report does not independently establish external disclosure; it records process-level credential presence and isolated child environments. Following the task brief's precaution, the operator must revoke or rotate the potentially exposed provider and gateway credentials through their issuers and provision distinct replacements. No rotation/revocation was performed, and no external credential panel was accessed.

For later provisioning, use an already approved local secret manager/process-credential broker or the separate non-echoing prompts permitted by the E07 plan. Do not put values in shell history, command text, arguments, environment files, TOML, logs, fixtures, or reports. Build explicit child-environment allowlists: AI Hub receives `AIHUB_GATEWAY_API_KEY` and `AIHUB_PROVIDER_OPENCODEGO_API_KEY`; the Codex child receives only its configured gateway `env_key` and required non-secret runtime variables. Do not inherit the provider-bearing parent environment into Codex. Check presence in each exact child environment immediately before spawn and record only `PRESENT`, `ABSENT`, or `NOT_VERIFIABLE`.

## 5. Process Isolation Requirements

Before any separately approved run, prove against the exact launch environments that the Hub has both intended keys and Codex has the gateway key only. Keep the Hub and Codex in separate processes with explicit environment allowlists; use isolated, pre-created `HOME`/`CODEX_HOME`, Hub data, and tool-workspace directories outside the repository. Keep Codex loopback-only and preserve its normal sandbox/approval controls. Presence in this agent's shell is not evidence of isolation. Any mismatch stops the preflight before Hub startup or provider traffic.

## 6. Working-Tree Reconciliation and Clean-Baseline Proposal — prior checkpoint

This section records the Phase 8B checkpoint before the 2026-11-01 conditional acceptance authorization. Preserve all 27 initial E07 artifacts and this report for review; no deletion or restoration is proposed. Every test and document needs human provenance/scope review because uncommitted Git state cannot establish its author. The ten modified and thirteen new Go test files appear to organize integrity, provider-budget, fidelity, and record-validation helpers. The three initial untracked reports and the modified harness report document E07 evidence and readiness. At that checkpoint, the final execution plan was only a proposal; the current authorization is recorded in §7 and `E07-final-e2e-execution-plan.md`.

To reach a clean baseline, the project lead should review the changes by origin, decide which E07 artifacts are retained, and authorize any separate local commits. Do not combine changes from distinct origins. If the retained artifacts are approved and committed, verify `git status` is empty and the intended commit is the documented E07 baseline; otherwise use a separately reviewed clean worktree at an approved revision. Do not stage or commit as part of this phase. The current tree, including this report, is **not clean**.

Potential later commit boundaries are the Go validation-helper/test changes and the E07 evidence documents, but only if review confirms each group has a common origin. Split either group further when provenance differs; do not infer a shared origin from directory or filename.

## 7. E2E Plan and Budget State

The original-plus-supplemental allowance is consumed: prior reports record five provider-boundary calls in the original pass and nine in the supplemental pass. The project lead's 2026-11-01 final acceptance prompt now conditionally authorizes one additional pass, capped at nine provider-boundary calls and 30 minutes, with zero retries and a two-minute per-request timeout. This supersedes the earlier statement that no third pass was authorized; it does not waive any precondition. This blocker-resolution assessment made zero provider calls and consumed no E2E budget.

The prior results justify fresh review only after remediation: the original pass had no `bwrap` and no Codex Responses request reached the Hub; the supplemental pass established basic generation, authenticated routing, tool use, gateway streaming, and presence-based credential isolation, but replay, upstream-error fidelity, record verification, restart verification, and the global Codex-tree integrity check remained failed. The helper remediation adds deterministic evidence for measurement logic; it does not repair those live results. The source of the global Codex-tree changes remains unknown.

The earlier recommendation to obtain a new pass authorization and budget review is
superseded by the 2026-11-01 prompt. The existing operational requirements for
scanner approval and a redacted scan, credential provisioning, a reviewed clean
baseline, process isolation, integrity checkpoints, stop conditions, and
scenario-level evidence remain in force. The current conditional authorization
does not waive those preconditions; no additional plan revision or budget
approval is needed before the single authorized pass.

## 8. Human-Dependent Blockers and Next Decisions

- A governance owner must approve the scanner, version, scope, exclusions, verification/network behavior, and redacted output handling.
- The credential owner must revoke/rotate potentially exposed credentials and arrange secure, non-interactive injection of distinct replacements.
- The project lead must review the current E07 working-tree artifacts and authorize a clean-baseline disposition without discarding or mixing origins.
- The conditional one-pass authorization is already provided by the 2026-11-01 final acceptance prompt; do not request another budget approval. Execution remains blocked until the scanner, baseline, credentials, isolation, and other preconditions above are resolved and verified.
- The intended process environments and provider/model availability remain unverified. The current prompt conditionally authorizes only the single pass above; no standalone or unbudgeted provider probe is authorized.

## 9. SDD Evidence and Validation

- **Claim:** The original Phase 8B report recorded the Git, scanner, credential-presence, isolation, and E2E authorization blockers without disclosing secret values or implying E07 acceptance.
- **Success type:** `manual` and `command`.
- **Evidence path:** Git status/history/ancestry; presence-only environment checks; repository policy/configuration search; consistency review against the five named E07 artifacts.
- **No-test rationale at that checkpoint:** The Phase 8B report-only change altered no executable behavior, so no automated test, AI Hub process, Codex CLI scenario, provider call, or E2E run was needed or permitted in that phase. Later final-acceptance tests and their outcomes are recorded in the current preflight addendum.

The initial Git inventory and variable checks are recorded above. No external action is claimed. The independent reviewer result is reported in the session closeout; it does not authorize E2E or change E07 status.

## Previous implementation checkpoint addendum — 2026-11-01

Status: **BLOCKED — no provider-bound E2E request was made.** The current user
request authorizes one conditional final pass with a nine-call ceiling and a
30-minute limit. The two historical passes remain consumed; the conditional
pass was not started and its provider-call count remains `0/9`.

### Current checkout and baseline decision

| Check | Result |
| --- | --- |
| Branch / HEAD | `main` / `56cbd0b509494d452e2a1e9472ef3e01830792d7` |
| `origin/main` / remote `main` | `a8ac19af7cf6457e30f3eb284965c4966ecf033b` |
| Local divergence | One commit ahead, zero behind |
| Remediation commit | `40196001536b6d07084b8bbd671d646bdd674d32` is an ancestor of HEAD |
| Staged changes | None |
| `.env` status | No changed `.env` paths reported; ignored values were not read |
| Approved immutable E2E baseline | **ABSENT** |

The Phase 8B inventory recorded 29 artifacts. The initial checkout for the
conditional acceptance preflight had seven tracked modifications and seven
untracked test files. Updating the execution-plan authorization added one
tracked pending file; the current scope has eight tracked modifications and
seven untracked test files. The exact current pending paths are:

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
- `docs/validation/E07-final-e2e-execution-plan.md`

No staging, commit, push, or artifact disposition was performed. The project
lead must approve an immutable baseline and decide the provenance/disposition
of the listed E07 files before provider traffic. Do not treat the dirty tree as
clean or discard files to manufacture a baseline.

### Codex environment evidence

The generated isolated configuration uses the documented Codex shell policy:
`inherit = "none"`, explicit non-secret `set` values,
`ignore_default_excludes = false`, and
`experimental_use_profile = false`. Temporary `HOME`, `CODEX_HOME`, temporary
directories, and XDG paths are asserted by the canary. A synthetic local test
observed the target CLI version and passed the gateway-authentication and
child-environment assertions for the variables included in that run. The
provider's exact AI Hub environment variable was added to a deterministic
builder test after the canary and remains `NOT_VERIFIED` in an actual Codex
child. Before a private baseline manifest was established, one
`codex --version` probe ran in the inherited environment; it made no provider
request and its effect cannot be measured retrospectively. A later race-suite
canary detected that the global Codex tree had changed. The change's source
cannot be attributed while Codex was active; classify the
cause as `INCONCLUSIVE` and integrity as `FAIL`.

The focused E07 run stopped before launching another Codex CLI while a Codex
process was active. The full race run reached the synthetic loopback canary,
then failed its final global-tree comparison; other packages passed. Follow-up
regressions now reject a symbolic-link Codex state root, record nested symbolic
links by target without following them, reject unreadable process state, and
fail closed on unsupported platforms. The deterministic tests pass,
but no further Codex probe was made after the global-tree change was observed.
No global Codex file contents were read or copied, and none were restored or
edited by this agent. Private manifests were removed with their temporary
root.

### Credential and scanner states

Presence-only checks of the current tool process returned:

| Variable | State |
| --- | --- |
| `AIHUB_GATEWAY_API_KEY` | `ABSENT` |
| `AIHUB_CLIENT_KEY` | `ABSENT` |
| `AIHUB_PROVIDER_OPENCODEGO_API_KEY` | `ABSENT` |
| `OPENCODE_API_KEY` | `PRESENT` |

The present value was not read, copied, or used. No Hub or intended Codex
process environment exists; those launch states are `NOT_VERIFIABLE`. New,
distinct credentials must be injected via an approved non-interactive secret
facility, with only presence states reported.

Gitleaks is absent and has not been installed, executed, or approved. No
approved secret scan was performed. Snyk Code was attempted separately and
returned an unauthenticated-account result; it is not a substitute secret
scanner. A governance owner must approve the scanner/release/provenance and
redacted-output procedure, then make that approved scanner available.

### Final local checks

| Check | Result |
| --- | --- |
| Focused `TestE07` | `BLOCKED` — stable-tree preflight stopped before another Codex launch |
| `go test ./... -count=1 -race` | `FAIL` — synthetic canary reached the local mock, but final global-tree comparison detected a change; other packages passed |
| `go vet ./...` | `PASS` |
| `staticcheck ./...` | `PASS` |
| `golangci-lint run ./...` | `PASS` — 0 issues |
| `go build ./...` | `PASS` |
| `gofmt -l cmd internal` | `PASS` — empty output |
| `git diff --check` | `PASS` |
| Selected deterministic E07 tests | `PASS` — 32 tests; Codex CLI and AI Hub were not started |
| Darwin E07 test package cross-compilation | `PASS` — test binary compiled; temporary binary removed |
| E07 modified/created Go file length | `PASS` — largest E07 Go file is 278 lines |
| Repository-wide 300-line limit | `FAIL` — eight pre-existing Go files exceed 300 lines; none is part of this E07 scope |

Live flags and credential variables were removed from all final gate
processes. The temporary synthetic canary used only a loopback fake Responses
server; it made no provider-boundary call. Deterministic budget, replay,
response-fidelity, record, and restart assertions remain complementary local
evidence only. Historical real-provider outcomes and the per-AC cumulative
matrix are recorded in the E07 validation report.

The repository-wide line-limit findings are `internal/catalog/manager_test.go`
(877), `internal/catalog/store_test.go` (593),
`internal/gateway/catalog_handler_test.go` (575),
`internal/gateway/responses_handler_test.go` (454),
`internal/catalog/capability_test.go` (442), `internal/catalog/manager.go`
(397), `internal/gateway/catalog_handler.go` (386), and
`internal/catalog/store.go` (312). These existing E01–E06 files were not
changed as part of E07.

### Independent review

The first independent judge round rejected the harness with two important
findings: symlink coverage and process-detection fail-open behavior. The E07
helpers and tests were corrected. The second round returned
`APPROVED_WITH_NOTES` (`SDD APPROVED`) with both findings resolved and no open
code findings. The reviewer did not run Codex, start AI Hub, or make provider
calls. A later README review found credential exposure to `go run`, then
temporary-home and cleanup gaps. Those were corrected: build/test compilation
uses temporary home and caches without credentials, execution is brokered
directly, and temporary artifacts are removed. The independent follow-up found
no open findings in the revised README setup section. E07 remains `INCOMPLETE`
because the prior global-tree comparison failed and final E2E prerequisites
remain absent.

### Required operator action before the authorized pass

1. Approve a specific immutable baseline after reviewing the pending file list
   above; keep staging/commit/push under separate authorization.
2. Provide governance approval and verified provenance for the secret scanner,
   then complete a redacted scan covering tracked, untracked, and Markdown
   files.
3. Inject new, distinct gateway and provider credentials through an approved
   non-interactive secret facility. Prove the exact Hub and Codex process
   environments; Codex tools must receive neither credential.
4. Close other Codex CLI sessions and run the stable-tree preflight in an
   environment where no ambient Codex process is changing global state.
5. Re-run the safe synthetic process-policy canary, including the exact AI Hub
   provider-key variable, and require a passing global before/after comparison.

Only after those conditions pass may the single already-authorized E2E
pass begin. Keep its one-pass, nine-provider-call, two-minute-per-request,
zero-retry, and 30-minute limits unchanged. A subsequent integrity or
credential failure stops provider traffic; no retry or second pass is
authorized.

## Addendum — configuration and isolation evidence (2026-11-01)

This addendum supersedes the credential and canary statements above where they
conflict; the historical record is kept intact. Full detail and the updated
per-AC matrix are in
`docs/validation/E07-codex-first-harness-integration.md`. Real provider calls
in this attempt: zero.

### Production configuration

`cmd/ai-hub/main.go` loads the dotenv file (default `-env-file .env`) with
`config.LoadDotEnv` before `config.Load`, and process environment values take
precedence over file values. The OpenCode Go provider key is read only from
`AIHUB_PROVIDER_OPENCODEGO_API_KEY` (`internal/config/types.go`).
`OPENCODE_API_KEY` has no production consumer, so it cannot substitute for it.

A temporary build started with `-env-file .env` failed closed with
`gateway.api_key is invalid`, proving the file is parsed and that it provides no
usable gateway key. No provider request was attempted.

Presence-only inspection of `.env` (variable names only, no values, no lengths,
no hashes, file unmodified) found `AIHUB_PROVIDER_OPENCODEGO_API_KEY` and the
unused `OPENCODE_API_KEY` declared once each, no duplicates, and
`AIHUB_GATEWAY_API_KEY` absent. In the agent process all four E07 variables —
`AIHUB_GATEWAY_API_KEY`, `AIHUB_CLIENT_KEY`,
`AIHUB_PROVIDER_OPENCODEGO_API_KEY`, `OPENCODE_API_KEY` — were absent. The
earlier "`OPENCODE_API_KEY` present" observation described a different
checkpoint environment and is not repeated here.

### Harness credential source corrected

The approved design allowed the live setup helper to read the provider
credential from `AIHUB_PROVIDER_OPENCODEGO_API_KEY`, `OPENCODE_API_KEY`, or
`.env`. Under the current authorization only the production-consumed variable is
acceptable, because equivalence between `OPENCODE_API_KEY` and the OpenCode Go
key is unproven. The old code also copied a legacy-sourced value into
`AIHUB_PROVIDER_OPENCODEGO_API_KEY`, which would have masked a missing
production configuration and made the run look provisioned.

`e07ProviderCredential` now reads only `AIHUB_PROVIDER_OPENCODEGO_API_KEY`,
with `TestE07ProviderCredentialUsesOnlyProductionVariable` as the regression
test. The setup logs `provider_credential_origin=environment|dotenv` without a
value, so a `.env`-sourced key is never treated as provisioning evidence.

### Isolation and canary

The `TestE07CodexShellEnvironmentPolicyIsEffective` guard deliberately fails
closed while an ambient Codex process can modify global state. Four VS Code
ChatGPT extension app-servers were active; none was signalled or terminated and
no assertion was weakened. Running the canary inside a bubblewrap PID and mount
namespace with a fresh `HOME` made the before/after comparison provable:

- Codex CLI `0.164.0-alpha.11.1` (matching the E07 target).
- Local synthetic Responses endpoint observed gateway Bearer authentication.
- `exec_command` executed under `sandbox_mode = "workspace-write"` with no
  approval policy written.
- Child environment reported `AIHUB_CLIENT_KEY`, `OPENCODE_API_KEY`,
  `AIHUB_PROVIDER_OPENCODEGO_API_KEY`, and `AIHUB_GATEWAY_API_KEY` all `ABSENT`.
- Two requests, the expected bounded count.
- Global Codex tree unchanged inside the namespace.

A fresh writable `/tmp` tmpfs is required inside the namespace; otherwise the
nested Codex sandbox fails on a read-only `/tmp`. This proves the policy with
synthetic values, not a real-provider scenario.

The canary is now opt-in through `AIHUB_E07_CANARY=1`, matching the design's
"opt-in validation test helper" boundary; the default `go test ./...` run skips
it, so the suite no longer depends on host Codex processes. On its supported
Linux platform, enabling it fails closed on every precondition failure,
including a missing CLI. `AIHUB_E07_LIVE=1` remains required for the real E2E.

### Gate results

Inside the namespace with `AIHUB_E07_CANARY=1`, the focused run passed 55 tests
with 0 failures and 1 expected skip (`TestE07LiveCodexOpenCodeGo`, gated by
`AIHUB_E07_LIVE=1`); the consolidation closeout below adds two further
regression tests, for 57 passed at that point. Without the canary flag, the full
`go test ./... -count=1 -race` run passed for every package with the canary
skipped explicitly, even with the Codex CLI on `PATH`. Enabling the canary with
the CLI absent from `PATH` failed as required instead of skipping, proving the
fail-closed behavior. `go vet`, `staticcheck`, `golangci-lint` (0 issues),
`go build`, `gofmt -l cmd internal` (empty), and `git diff --check` all passed
inside the same isolation. The earlier preflight failure was caused only by the
active-Codex guard and is resolved without changing any assertion.

### Residual limitations

- The global Codex state comparison uses metadata identity, not a content
  digest, because reading the whole `~/.codex` tree would read prompt and
  session contents that FR-12 forbids reporting. Ordinary edits change
  modification time, size, or inode metadata on the validated filesystem; a
  privileged actor that restores timestamps and size could evade it.
- Nested symbolic links inside a Codex state root are recorded by target and
  never followed, so only the link entry is compared; state written through a
  linked subtree is outside the gate. A symbolic-link state root still fails
  closed, and the live helper reports `codex_tree_links=<n>` with the integrity
  outcome so the uncovered coverage is visible.
- When `AIHUB_GATEWAY_API_KEY` is absent, the live helper derives an ephemeral
  in-memory gateway key as the design permits. The README still instructs
  operators to supply a real gateway credential; the ephemeral key is not
  provisioning evidence.
- The private Codex metadata manifests are written only after the manifest
  directory is confirmed outside the repository, and the temporary root is
  removed by the single registered cleanup.

### Remaining operator actions

Only the credential rotation/provisioning, scanner approval, baseline
consolidation, and ambient-Codex-free integrity run remain. A rotated provider
key and a new `AIHUB_GATEWAY_API_KEY` must be provisioned through an approved
non-echoing facility; the gateway key is absent and must not be generated
automatically for production use. The authorized pass is still unused at `0/1`
with `0/9` provider calls.

### Consolidation closeout (2026-11-01)

The pending E07 change set is consolidated into three ordered groups for local
commit under separate authorization, with no push and an unchanged remote. The
account-home and nested-symlink corrections make the global Codex-tree
comparison cover the real `~/.codex` root even when the validation process runs
with an isolated `HOME`, while linked subtrees remain outside the comparison and
are reported by count. The opt-in canary passes inside the private namespace.
A pre-existing, timing-sensitive E01–E06 test (`TestManagerTimeoutIsClassified`)
is flaky under `-race` and is documented in the main report. Still open: committing and approving the baseline, an approved secret scan,
distinct rotated credentials, and a provider/model availability confirmation.
The authorized pass remains unused at `0/1` with `0/9` provider-boundary calls,
and E07 remains `INCOMPLETE`.
