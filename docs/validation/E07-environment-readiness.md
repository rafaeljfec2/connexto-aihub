# E07 Environment Readiness

Status: **BLOCKED — the environment is not ready for the planned E07 E2E pass.**

Assessment time: 2026-10-31 23:45:45 America/Sao_Paulo (UTC-03:00).

## 1. Scope and Safety

This report checks the prerequisites for the proposed pass in
[`E07-final-e2e-execution-plan.md`](E07-final-e2e-execution-plan.md). It does
not authorize or perform an E2E run. No AI Hub process was started, no Codex
scenario or test was run, and no request was sent to OpenCode Go or another
provider. The checks evaluated only whether credential variables were
non-empty. No credential values were emitted, copied into artifacts, or used
in a provider request.

## 2. Context and Git State

| Item | Observed state |
| --- | --- |
| Branch and HEAD | `main`, `a8ac19af7cf6457e30f3eb284965c4966ecf033b` |
| Test-readiness remediation | Commit `40196001536b6d07084b8bbd671d646bdd674d32` exists and is an ancestor of HEAD. |
| Working tree | Dirty: modified and untracked E07 test and documentation artifacts are present. The clean-baseline precondition is not met. No production source changes are listed in the observed status. |
| Final E2E plan | `PROPOSED`; the document says it does not authorize execution. The two previously authorized passes are recorded as consumed. |
| E07 status | `INCOMPLETE`; this readiness check does not change acceptance status. |

The current tree contains the split test helpers at or below the repository's
300-line limit. The budget helper and deterministic tests described below are
present in the current checkout. Their focused test results are recorded as
passing in the remediation report; they were not rerun during this readiness
check. The current working tree remains uncommitted and must not be treated as
a clean E2E baseline.

## 3. References Consulted

- `AGENTS.md`, including the repository-wide language, scope, and workflow
  rules.
- `docs/discovery/E07-final-acceptance-readiness.md`.
- `docs/validation/E07-final-e2e-execution-plan.md`.
- `docs/validation/E07-codex-first-harness-integration.md`.
- `docs/specs/E07-codex-first-harness-integration.md`.
- `/sdd-workflow` and its phase, evidence, and reviewer references.
- Current Git status and the remediation commit ancestry.

The plan requires an already approved scanner over tracked and untracked
repository files. The repository rules and E07 documents do not name a
specific approved scanner or explicitly approve one. Historical references to
Gitleaks record that it was unavailable; they do not establish approval.

## 4. Secret Scanner

| Check | Result | Evidence |
| --- | --- | --- |
| Approved scanner named by policy | **NOT ESTABLISHED** | No specific tool approval was found in `AGENTS.md` or the E07 plan/spec/validation documents. |
| Common scanner executables | **ABSENT** | `gitleaks`, `trufflehog`, `detect-secrets`, `trivy`, and `gosec` were not available on `PATH`. |
| Scan of tracked and untracked files | **BLOCKED — NOT RUN** | No approved, available scanner could be selected. No alternative was installed or substituted. |
| Secret-free repository claim | **NOT ESTABLISHED** | No scan result exists; this report does not claim the repository is secret-free. |

Scanner verdict: **BLOCKED**.

## 5. Credential Presence

Only non-empty presence was checked in the current tool process. No value or
derived value was emitted.

| Variable | Current process | Exact future E2E process |
| --- | --- | --- |
| `AIHUB_GATEWAY_API_KEY` | **ABSENT** — no non-empty value was available. | **NOT_VERIFIABLE** — the Hub and Codex child environments were not created. |
| `OPENCODE_API_KEY` | **PRESENT** — presence only. | **NOT_VERIFIABLE** — availability to the intended Hub process and exclusion from Codex were not tested. |

The secure provisioning mechanism for a future Hub/Codex run is **NOT_VERIFIABLE**.
The gateway key must be provisioned outside chat through a secure,
non-echoing mechanism selected by the operator; the provider key must remain
limited to the Hub process.

## 6. Local Environment and Isolation

| Prerequisite | Result | Evidence and limit |
| --- | --- | --- |
| Codex CLI | **PASS** | `codex --version` reported `codex-cli 0.164.0-alpha.11.1`; this was a version query only. |
| Go | **PASS** | `go version` reported `go1.23.0 linux/amd64`. |
| `bwrap` | **PASS** | `bwrap --version` reported `bubblewrap 0.4.0`. |
| Local sandbox startup | **PASS, limited** | A harmless isolated `/bin/true` command completed under `bwrap`. No Codex sandbox scenario was run. |
| Writable runtime/temp directory | **PASS** | The runtime directory was writable; temporary isolated directories were created and removed. |
| `HOME` and `CODEX_HOME` separation | **PASS, setup only** | An empty environment with separate temporary `HOME` and `CODEX_HOME` directories was verified. Codex was not launched with them. |
| Loopback port 8080 | **FREE AT CHECK TIME** | A read-only listener check found no listener. Recheck immediately before any separately authorized run. |
| 30-minute end-to-end time limit | **NOT_VERIFIABLE** | The plan sets this limit; confirming the full run fits requires the prohibited E2E execution. |
| Hub startup, shutdown, and restart | **NOT_VERIFIED** | The Hub was not started, as required by this gate. |
| Provider/model availability | **NOT_VERIFIABLE** | No provider request was made. The plan requires stopping if availability cannot be confirmed without an unbudgeted call. |

## 7. Integrity Checkpoint and Hard Call Budget

The private integrity-manifest helper and its deterministic tests are present
in the checkout. The remediation report records the manifest and redaction
tests as passing. No before-state manifest was created for a live pass in this
check, and the dirty working tree prevents a clean baseline claim.

The shared provider-call budget is implemented in the current test harness:

- `cmd/ai-hub/e07/support/e07_provider_support.go` sets the maximum to nine;
  reservation is mutex-protected and rejects excess calls before forwarding.
- `cmd/ai-hub/e07/provider/e07_provider_budget_test.go` covers rejection before the
  underlying transport, a 50-call concurrent attempt, redirect accounting,
  readiness/startup accounting, shared accounting across observers/restart,
  and no budget consumption for an unauthenticated request.
- The remediation report records the focused deterministic checks as passing.
  They were inspected but not rerun in this environment-readiness check.

Hard call budget evidence: **PASS for implementation and recorded deterministic
tests; no live provider call was made.** This is not E2E evidence.

## 8. Readiness Matrix and Remaining Blocks

| Gate condition | Result |
| --- | --- |
| Approved scanner available and scan completed | **BLOCKED** |
| No unresolved scanner finding | **NOT_VERIFIABLE** — no scan ran |
| Gateway credential available | **BLOCKED** |
| Provider credential in intended Hub process; excluded from Codex | **NOT_VERIFIABLE** |
| Codex CLI, Go, and `bwrap` available | **PASS** |
| Local isolation and sandbox startup | **PASS, limited to harmless setup probes** |
| Temporary resources available | **PASS** |
| Port 8080 available | **PASS at check time; recheck required** |
| Integrity checkpoint mechanism | **IMPLEMENTATION PRESENT; live checkpoint NOT CREATED** |
| 30-minute run capacity and model availability | **NOT_VERIFIABLE without the separately authorized run or approved evidence** |
| Nine-call hard budget | **PASS for implementation and recorded deterministic tests** |
| Clean baseline required by the plan | **BLOCKED — working tree is dirty** |
| E2E execution authorization | **NOT GRANTED by the proposed plan; no execution attempted** |

Overall Environment Readiness Gate: **BLOCKED**. The environment does not meet
the approval criteria because the approved scanner is unavailable, the gateway
credential is absent, the exact future process environments are unverified,
and the required clean baseline is not present. The plan is also still
proposed, and its previous two-pass budget is recorded as consumed.

## 9. Next Actions

1. Establish which secret scanner is approved under project governance and
   make that scanner available without installing it as part of this gate.
2. Provision the gateway credential through a secure non-echoing mechanism in
   the intended Hub and Codex process environments. Verify provider-key
   isolation in those exact child environments without emitting values.
3. Resolve the current working-tree changes through the normal project
   workflow, then repeat the clean-baseline and integrity preflight.
4. Recheck port availability and temporary isolation immediately before any
   separately authorized run. Confirm model availability only through an
   approved mechanism that does not exceed the call budget.
5. Obtain a separately reviewed and approved execution plan/budget before any
   E2E. This report does not authorize that run; E07 remains `INCOMPLETE`.

## 10. Explicit Non-Execution Confirmation

No E2E scenario or automated test was executed. The AI Hub was not started.
No request was sent to OpenCode Go or any provider, and the nine-call budget
was not consumed.
