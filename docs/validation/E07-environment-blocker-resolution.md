# E07 Environment Blocker Resolution

Status: **BLOCKED — environment readiness and a new E2E authorization are not established.**

Assessment date: 2026-11-01.

## 1. Scope and Evidence

This phase reconciles the current E07 environment blockers and prepares a proposal for a later readiness review. It does not run tests or E2E, start AI Hub or Codex, contact a provider, consume a provider-call budget, rotate credentials, or change files other than this report. E07 remains **INCOMPLETE**.

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
| `docs/validation/E07-final-e2e-execution-plan.md` | Untracked | `EXPECTED_E07_ARTIFACT` | E07 proposed, unauthorized future-pass plan. |

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

For later provisioning, use an already approved local secret manager or process-credential broker to inject secrets at process launch without interactive prompts. Do not put values in shell history, command text, arguments, environment files, TOML, logs, fixtures, or reports. Build explicit child-environment allowlists: AI Hub receives `AIHUB_GATEWAY_API_KEY` and `OPENCODE_API_KEY`; the Codex child receives only its configured gateway `env_key` and required non-secret runtime variables. Do not inherit the provider-bearing parent environment into Codex. Check presence in each exact child environment immediately before spawn and record only `PRESENT`, `ABSENT`, or `NOT_VERIFIABLE`.

## 5. Process Isolation Requirements

Before any separately approved run, prove against the exact launch environments that the Hub has both intended keys and Codex has the gateway key only. Keep the Hub and Codex in separate processes with explicit environment allowlists; use isolated, pre-created `HOME`/`CODEX_HOME`, Hub data, and tool-workspace directories outside the repository. Keep Codex loopback-only and preserve its normal sandbox/approval controls. Presence in this agent's shell is not evidence of isolation. Any mismatch stops the preflight before Hub startup or provider traffic.

## 6. Working-Tree Reconciliation and Clean-Baseline Proposal

Preserve all 27 initial E07 artifacts and this report for review; no deletion or restoration is proposed. Every test and document needs human provenance/scope review because uncommitted Git state cannot establish its author. The ten modified and thirteen new Go test files appear to organize integrity, provider-budget, fidelity, and record-validation helpers. The three initial untracked reports and the modified harness report document E07 evidence and readiness. The proposed final execution plan remains only a proposal.

To reach a clean baseline, the project lead should review the changes by origin, decide which E07 artifacts are retained, and authorize any separate local commits. Do not combine changes from distinct origins. If the retained artifacts are approved and committed, verify `git status` is empty and the intended commit is the documented E07 baseline; otherwise use a separately reviewed clean worktree at an approved revision. Do not stage or commit as part of this phase. The current tree, including this report, is **not clean**.

Potential later commit boundaries are the Go validation-helper/test changes and the E07 evidence documents, but only if review confirms each group has a common origin. Split either group further when provenance differs; do not infer a shared origin from directory or filename.

## 7. E2E Plan and Budget State

`E07-final-e2e-execution-plan.md` remains **PROPOSED** and explicitly authorizes no execution. The approved original-plus-supplemental allowance is consumed: the report records five provider-boundary calls in the original pass and nine in the supplemental pass. The final plan's one-pass limit and nine-call ceiling are proposed values for a possible third pass, not current authorization. This blocker-resolution phase made zero provider calls and consumed no E2E budget.

The prior results justify fresh review only after remediation: the original pass had no `bwrap` and no Codex Responses request reached the Hub; the supplemental pass established basic generation, authenticated routing, tool use, gateway streaming, and presence-based credential isolation, but replay, upstream-error fidelity, record verification, restart verification, and the global Codex-tree integrity check remained failed. The helper remediation adds deterministic evidence for measurement logic; it does not repair those live results. The source of the global Codex-tree changes remains unknown.

Do not edit or approve a new execution plan in this phase. A later plan revision should include:

1. A new-pass justification tied to the unresolved acceptance criteria and a summary of both consumed passes and the helper-remediation evidence.
2. Current scanner approval and redacted scan evidence, credential rotation/provisioning evidence, exact process-isolation checks, and a reviewed clean Git baseline.
3. Remaining E07 failures and environment gaps, plus a fresh authorization boundary that does not inherit either earlier pass approval.
4. A newly reviewed budget. A candidate ceiling is one pass, four Codex scenario invocations, at most nine provider-boundary calls (four Codex requests, one error probe, one streaming request, and up to three identified readiness calls), zero retries, two minutes per provider request, and 30 minutes total. This is a new proposal only; it requires explicit approval even if the numeric ceiling matches the prior draft.
5. Stop conditions for missing prerequisites, credential leakage/isolation failure, unexpected calls, budget/time exhaustion, integrity changes, sandbox bypass, or Hub lifecycle failure. No retry or third attempt without another human decision.
6. Private before/after integrity checkpoints for the complete global Codex tree, existing project `.env` files, tracked and untracked repository content, and Git scope. Report category outcomes only and delete private manifests after comparison.
7. The existing E07 acceptance criteria, per-scenario evidence, cleanup, and independent review requirements, with E07 kept `INCOMPLETE` until every required criterion is satisfied.

## 8. Human-Dependent Blockers and Next Decisions

- A governance owner must approve the scanner, version, scope, exclusions, verification/network behavior, and redacted output handling.
- The credential owner must revoke/rotate potentially exposed credentials and arrange secure, non-interactive injection of distinct replacements.
- The project lead must review the current E07 working-tree artifacts and authorize a clean-baseline disposition without discarding or mixing origins.
- A project lead must separately review and approve a new E2E plan and exact fresh call/time budget after the blockers above are resolved.
- The intended process environments and provider/model availability remain unverified. No probe is authorized by this report.

## 9. SDD Evidence and Validation

- **Claim:** This report records the current Git, scanner, credential-presence, isolation, and E2E authorization blockers without disclosing secret values or implying E07 acceptance.
- **Success type:** `manual` and `command`.
- **Evidence path:** Git status/history/ancestry; presence-only environment checks; repository policy/configuration search; consistency review against the five named E07 artifacts.
- **No-test rationale:** This change is documentation-only and changes no executable behavior. No automated test, AI Hub process, Codex CLI scenario, provider call, or E2E run is needed or permitted for this phase.

The initial Git inventory and variable checks are recorded above. No external action is claimed. The independent reviewer result is reported in the session closeout; it does not authorize E2E or change E07 status.
