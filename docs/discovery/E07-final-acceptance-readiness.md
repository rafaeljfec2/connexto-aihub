# E07 Final Acceptance Readiness

Status: **INCOMPLETE — final real-provider acceptance is not established.**

Assessment date: 2026-10-31.

## 1. Decision Summary

E07 cannot close on the evidence currently available. The latest real-provider
pass passed basic generation, authenticated routing, sandboxed tool use, and
gateway-hop streaming. It failed multi-turn replay, upstream error validation,
execution-record verification, and post-restart verification. The global Codex
tree changed, and the repository secret scan did not run. The later remediation
corrected validation helpers and passed deterministic checks; it did not rerun
any real-provider scenario.

The repository currently has the approved E07 SPEC and design, two consumed
E2E passes, the failure analysis, the helper remediation, and a README procedure
that still labels Codex validation experimental. No further E2E pass is
authorized by those artifacts. A new, specific approval is required after the
execution plan is reviewed.

This assessment uses the repository as the source of truth. Prior memory
contains an independent remediation-review session with multiple review
rounds, but its retained observations do not include a final verdict. That
prior approval is therefore not counted as current acceptance evidence. This
readiness document and its execution plan require their own independent review.

## 2. Repository and Git Baseline

| Item | Verified state |
| --- | --- |
| Branch and HEAD | `main` at `a8ac19af7cf6457e30f3eb284965c4966ecf033b` |
| Remote baseline | `origin/main` equals HEAD |
| E07 remediation commit | `40196001536b6d07084b8bbd671d646bdd674d32` is an ancestor of HEAD |
| Working tree | No tracked, staged, or untracked changes before this Discovery output |
| Ignored state | An ignored `.env` exists; it was not opened or included in output |
| E06 comparison | The diff from `0d33ca26c15acf54fac418cb33380b5871b4d899` contains E07 test helpers/tests and documentation only; no production source differs |
| Diff integrity | `git diff --check` from the E06 baseline to HEAD passed |

The remediation commit changed only E07 test helpers/tests and E07 documents.
The following README commit added the experimental Codex procedure and links.
The current README explicitly says Codex support is not established and makes
no support claim.

## 3. Acceptance-Criteria Matrix

Statuses refer to the latest applicable evidence in the committed validation
report. A prior `FAIL` remains a failed observation even when the helper defect
that affected its oracle has since been corrected. A deterministic test does
not replace the real-provider proof required by an AC.

| AC | Requirement | Existing evidence | Evidence still needed | Method and environment | Current result | Objective closure condition |
| --- | --- | --- | --- | --- | --- | --- |
| AC-1 | Reproducible README setup with isolated Codex and Hub state, environment-only credential separation, presence-only checks, and passing AC-14 integrity evidence. | README creates dedicated `CODEX_HOME` and `AIHUB_DATA_DIR`, configures the loopback Responses provider, and uses separate terminals. The supplemental pass reported credential isolation PASS, but global Codex integrity failed. | A fresh full setup and before/after integrity comparison for global Codex state, all project `.env` files, and repository state. | One approved pass; separate Hub and Codex environments; private digests and category-only reporting. | **FAIL** — the AC-14 dependency failed in the latest pass. | All setup steps work; the Hub and Codex receive the gateway key through their intended variables, only the Hub receives the provider key, and all AC-14 categories pass. |
| AC-2 | Identify the tested Codex release/model and route every scenario Codex Responses request through AI Hub. | The latest pass records Codex `0.164.0-alpha.11.1`, model `deepseek-v4.1-flash`, and four Codex requests through the Hub. | Reconfirm versions and route counts in the final consolidated pass. | Isolated Codex process; aggregate provider-boundary counter and Hub request observer. | **PASS** | Report exact versions/model and show every Codex request counted at the Hub with no direct provider route. |
| AC-3 | Accept authenticated Codex traffic; reject an unauthenticated request with `401` and zero provider-call delta. | The latest pass reports four authenticated Codex requests and an unauthenticated `401` with no provider-counter increase. | Repeat the bounded auth probe and retain only status and aggregate counter delta. | Local loopback Hub and validation-only provider transport counter. | **PASS** | Authenticated request succeeds; unauthenticated status is `401`; provider-call delta is zero. |
| AC-4 | Complete basic generation through real Codex, AI Hub, and OpenCode Go. | Latest pass reports a non-empty successful Codex response through the Hub. | Repeat once in the final pass and retain only success/non-empty booleans. | Real Codex CLI and available OpenCode Go provider/model. | **PASS** | A Codex response succeeds through the Hub, and the corresponding request is observed at the Hub. |
| AC-5 | Prove two-turn replay, synthetic marker recall, prior context in the second provider request, and absence of `previous_response_id`. | Latest pass reports replay FAIL. Its combined assertion did not preserve which predicate failed. The remediated helper now reports the predicates separately and has deterministic tests. | One real two-turn run with each structural predicate reported separately; no prompt or response retained. | Same Codex conversation, synthetic marker, in-memory request observer restoring bytes before forwarding. | **FAIL** — runtime cause remains undetermined. | Both turns succeed; marker recall, prior context, expected request count, and `previous_response_id` absence all pass. |
| AC-6 | Complete a bounded local tool round trip without bypassing sandbox or approvals. | Latest pass reports a temporary-workspace write/read task passed with `workspace-write` and normal controls. | Repeat once in a fresh temporary workspace; retain outcome only. | Codex `workspace-write`, isolated workspace outside the repository, sandbox available. | **PASS** | Codex writes and reads the synthetic value in the temporary workspace without a bypass. |
| AC-7 | Preserve the real upstream client-error status and generic body; keep local auth errors distinct. | Latest pass returned status `400`, but the cross-call generic-body comparison failed. The remediation removes cross-call equality and adds a deterministic fixed-upstream test for exact bytes, content type, status, and request bytes. | One real Hub request with same-response observations at provider and Hub boundaries; no body content or digest published. | Real provider; in-memory streaming digests/status/content type for that single response, plus existing deterministic passthrough test. | **FAIL** — real same-response fidelity is not established. | Same request's upstream and Hub status/content type/digest match, its body is a generic error envelope, and local unauthenticated `401` remains distinct. |
| AC-8 | Prove progressive gateway-hop SSE delivery and report Codex stdout rendering separately. | Latest pass observed 142 SSE data events before completion. Codex stdout progression was NOT VERIFIED, which AC-8 permits. | Repeat gateway measurement; independently record Codex rendering as observable or NOT VERIFIED. | Real provider; timestamps/counts at the Hub client connection; no inference from gateway to CLI. | **PASS** | Multiple SSE events arrive before completion; Codex rendering is reported separately. |
| AC-9 | Query records for observed executions and verify provider, E05 outcome, correlation, `model_id: null`, and `usage: null`. | Latest pass observed six records, but correlation/schema/outcome checks failed. The former helper inferred outcomes from HTTP status; remediation now derives expectations from E05 terminal observations and evaluates checks independently. | Real request-to-record correlation and independent list/get assertions for every expected execution. | Validation observer correlates IDs only in memory; authenticated E06 list/get API; redact identifiers from report. | **FAIL** — record completeness is not established; the previous failure may include a false-negative oracle. | Every expected execution is queryable with matching correlation/provider/outcome and null semantics; by-ID results are checked independently. |
| AC-10 | Gracefully stop and restart Hub on the same data directory and retrieve written records afterward. | Latest pass reports post-restart verification FAIL. The old shared verifier could fail because of the outcome-mapping defect. | One successful orderly restart followed by independent fresh-context list/get queries for the same records. | Same isolated data directory; graceful E05/E06 lifecycle; new request contexts after restart. | **FAIL** — recovery completeness is not established. | Records written in normal operation remain queryable after restart; report states the two-second drain and best-effort loss limits. |
| AC-11 | Report exact revision/environment/scenario results without secrets, content, or generated IDs. | The committed report includes both pass environments, scenario outcomes, versions, aggregate counts, limitations, and no credential/prompt/response/identifier values. | Add the final pass results using the same redaction rules. | Manual report review and approved secret scan; no raw payloads or IDs. | **PASS** for the existing report; final report update pending. | Final report contains required metadata/results and no prohibited content or generated identifiers. |
| AC-12 | Keep production source unchanged and pass Go tests, quality, build, formatting, and diff checks without weakening existing tests. | Remediation report records focused E07 tests, full race tests, vet, staticcheck, golangci-lint, build, gofmt, and diff checks as PASS. Git confirms no production source change since E06. | Re-run the required local gates after any future helper changes; confirm no production diff. | Go 1.23 repository checks; no provider calls required. | **PASS** on the committed remediation evidence. | Every required command passes on the final E07 change set; production diff remains empty and existing tests remain intact. |
| AC-13 | Add a Codex support claim only after all required scenarios pass; CLI rendering may remain NOT VERIFIED if gateway streaming passes. | README labels the procedure experimental, states E07 incomplete, and makes no Codex support claim. | Keep this wording until AC-1 through AC-12 and AC-14 are satisfied. | Review README against the final report. | **PASS** | No support claim is published before every required criterion passes. |
| AC-14 | Prove global Codex tree, existing `.env` files, unrelated tracked files, repository secrets, and authorized Git scope unchanged. | Both real passes report a changed global Codex tree. The supplemental pass reports project `.env` and Git categories unchanged, but the earlier helper missed the repository root/dirty content. Gitleaks was unavailable. Remediated snapshot tests cover these helper defects; no new live comparison or secret scan exists. | Fresh before/after snapshots using corrected full-root helper; approved scanner over tracked and untracked files; category-only report. | Full global Codex tree and repository snapshot; private manifest outside repository; no global-file repair; approved scanner available before run. | **FAIL** | Every required integrity category and the secret scan passes; no paths, values, contents, hashes, or IDs appear in evidence. |

## 4. Historical Evidence Reconciliation

Classifications use the SPEC's evidence categories: **VALID ON CURRENT
BASELINE**, **VALID PARTIALLY**, **OBSOLETE**, **INSUFFICIENT**, and **ABSENT**.

| Evidence stage | Classification | Current use and limitation |
| --- | --- | --- |
| OpenCode Go protocol spike | **OBSOLETE** for E07 acceptance | Historical protocol evidence supports the integration approach, but predates the E05/E06 runtime and does not prove any current AC. |
| E02 provider validation | **OBSOLETE** for the current baseline | Real Codex scenarios were reported on the E02 revision; it predates gateway hardening, execution lifecycle, and records changes relevant to E07. Preserve as history only. |
| E04 gateway validation | **OBSOLETE** for E07 runtime acceptance | Earlier authenticated Codex evidence remains useful context for the E04 contract, but was not run on E05/E06. |
| E05 smoke | **VALID PARTIALLY** | Proves one narrow real Codex smoke on E05; it predates E06 records and does not cover E07's required matrix. |
| E06 baseline | **VALID PARTIALLY** | Establishes the production baseline, contracts, and automated tests. No real Codex/E06 records E2E was completed before E07. |
| E07 original E2E pass | **VALID PARTIALLY** | Current E06 revision; the 401 probe and gateway SSE were observed. Codex requests did not reach the Hub, `bwrap` was absent, and records/restart/integrity criteria failed or were unverified. |
| E07 supplemental E2E pass | **VALID PARTIALLY** | Same E06 production revision as current HEAD. Basic generation, auth, tool, credential isolation, and gateway SSE are usable evidence. Replay, error fidelity, records, restart, and global integrity remain failed or unverified. |
| E07 failure analysis | **VALID PARTIALLY** | Correctly identifies confirmed helper-oracle defects and undetermined runtime causes for those runs. Its proposed helper fixes are now present; old runtime diagnoses remain unresolved. |
| E07 validation remediation | **VALID PARTIALLY** | Focused deterministic evidence and local quality-gate results apply to the committed helper/test change. It expressly makes no real-provider acceptance claim. |
| Remediation commit `4019600` | **VALID PARTIALLY** | Verified on `main`; contains the E07 helper/tests and supporting documents, with no production change. A commit proves artifact presence and lineage, not real acceptance. |
| Prior independent remediation review | **INSUFFICIENT** as retained evidence | Memory shows multiple reviewer rounds but not the final verdict body; no durable verdict was found in the repository. Do not infer a result from the review prompts. |

The current `main` production code remains equal to the committed E06 baseline.
That makes passed runtime observations relevant to the same production route;
it does not repair failed observations or promote helper-only tests into live
acceptance evidence.

## 5. Remediation Helpers and Test Evidence

The committed helper changes address the validation defects identified in the
failure analysis:

| Area | Repository evidence | What it establishes | What it does not establish |
| --- | --- | --- | --- |
| Repository root and snapshot | `cmd/ai-hub/e07_integrity_test.go`; `e07_integrity_remediation_test.go` | Git-root/module fallback, nested/root `.env` discovery, dirty and untracked content coverage, ignored production-file detection, symlink cycle protection, and private manifest checks have deterministic tests. | A real before/after run with an approved secret scan; the historical source of global Codex changes. |
| Record expectations and API checks | `cmd/ai-hub/e07_live_http_test.go`; `e07_records_test.go` | Outcomes derive from E05 terminal observations; list, schema, correlation, by-ID, and restart checks are independent and bounded. | Completeness/correlation of real provider executions or recovery after a new real restart. |
| Upstream response | `cmd/ai-hub/e07_passthrough_test.go`; `e07_live_scenario_support_test.go`; `e07_records_test.go` | Fixed upstream status, content type, exact bytes, and request forwarding are proven deterministically; independent provider responses are not byte-compared. | Same-response real-provider body fidelity. |
| Multi-turn replay | `cmd/ai-hub/e07_live_scenario_support_test.go`; `e07_replay_test.go`; `e07_observer_test.go` | Synthetic parser/observer conditions are reported separately, payload bytes are restored, and summaries remain structural. | Whether the real Codex/provider replay now passes. |
| Scope and history | `git show 40196001536b6d07084b8bbd671d646bdd674d32`; `git diff 0d33ca2..HEAD` | The remediation is committed, test/documentation-only, and current production source is unchanged from E06. | Any unmet real-provider AC. |

The validation report records local passes for focused E07 tests, full Go race
tests, vet, staticcheck, golangci-lint, build, formatting, and diff checks. No
tests or E2E checks were run in this Discovery. No secret scanner was run.

## 6. Environment Readiness

Credential checks below report presence only. No value was read, printed, or
used for a provider request.

| Dependency | Current result | Evidence and limitation |
| --- | --- | --- |
| Go | **READY** | Go `1.23.0 linux/amd64` is available. |
| Codex CLI | **READY, WITH NOTE** | `codex-cli 0.164.0-alpha.11.1`; an isolated version probe succeeded. The probe warned that helper aliases could not be created when `CODEX_HOME` lived under `/tmp`; a writable `XDG_RUNTIME_DIR` is available for the next isolated probe. |
| `bwrap` and sandbox startup | **READY** | `bwrap` is available and a harmless isolated `true` command succeeded. The prior validation report records `bubblewrap 0.4.0`. No Codex scenario was run. |
| Isolated `HOME` and `CODEX_HOME` | **READY for setup; scenario NOT VERIFIED** | Temporary directories were created and the isolated version probe worked. Use the runtime directory to avoid the `/tmp` helper-alias warning. |
| OpenCode Go credential | **READY, presence only** | `OPENCODE_API_KEY` is present in the current environment. Its value was not read or used; provider/model reachability remains unverified. |
| AI Hub gateway credential | **MISSING in current process** | `AIHUB_GATEWAY_API_KEY` is not set. A future pass must use a distinct non-echoing prompt or an approved secret facility. |
| Loopback port 8080 | **READY at check time** | Binding `127.0.0.1:8080` was available during the read-only readiness check; recheck immediately before any future run. |
| Temporary data/work directories | **READY** | The temporary directory is writable, and an isolated runtime directory is available. |
| AI Hub start/stop/restart | **NOT VERIFIED** | The Hub was not started in this Discovery. Previous real restart verification failed; the helper's deterministic lifecycle tests do not establish this live result. |
| Selected OpenCode Go model availability | **NOT VERIFIED** | The prior pass used `deepseek-v4.1-flash`; no provider call was made to establish present availability. |
| Approved repository secret scanner | **MISSING** | `gitleaks`, `trivy`, `detect-secrets`, and `trufflehog` are not installed, and no scanner configuration was found. Do not install a tool in this Discovery. |
| Git/repository preflight | **READY at check time** | Tracked working tree is clean on `main`; the existing ignored `.env` was not opened. Repeat the full private snapshot before any authorized pass. |

The current environment is not ready for a complete acceptance run because the
gateway credential and approved scanner are missing, provider/model reachability
and Hub lifecycle are unverified, and the historical global-tree finding is
unresolved. These gaps do not authorize a probe or another E2E pass.

## 7. Risks, Gaps, and Decisions

1. **Replay cause remains undetermined.** The prior failed run did not preserve
   individual predicate results. The corrected helper can distinguish them in
   a future authorized run, but no payload may be retained.
2. **Real error-body fidelity remains undetermined.** The old comparison used
   separate provider calls. The deterministic test proves fixed-response
   forwarding; the real test must compare the same response at both in-memory
   observation points.
3. **Record completeness and restart recovery remain undetermined.** Six
   aggregate records do not prove request correlation. The old outcome oracle
   could produce a false failure. Recheck list and by-ID assertions separately
   before and after restart.
4. **Global Codex-tree change source remains undetermined.** Do not repair or
   restore global files. A new run can establish its own before/after result;
   it cannot retroactively identify the old change without retained evidence.
5. **Secret-free repository status is not established.** No approved scanner
   is available. Do not claim AC-14 until a scanner runs over tracked and
   untracked files and its redacted result is recorded.
6. **Project source-length rule is not met by four committed E07 test files:**
   `e07_integrity_remediation_test.go` (420 lines), `e07_integrity_test.go`
   (515), `e07_live_http_test.go` (345), and `e07_records_test.go` (516). They
   exceed the supplied 300-line hard limit. This Discovery does not refactor
   them; the authorized test-helper implementation phase must split them before
   E07 closure and rerun the applicable local gates.
7. **The prior review verdict is not durable in the retained archive.** The
   current readiness artifacts will receive an independent review before this
   Discovery is reported complete.

No production behavior, API, ADR, or E01–E06 contract change is indicated by
current evidence. Any future evidence that requires such a change needs a
separate scope decision.

## 8. Recommendation

Keep E07 **INCOMPLETE** and keep the README support-claim wording unchanged.
Review the companion
[`E07 final E2E execution plan`](../validation/E07-final-e2e-execution-plan.md),
resolve its environment prerequisites, address the four overlength helper
files within an authorized test-only change, and obtain explicit approval for
one bounded final pass. Do not use the already consumed two-pass budget as
authorization for another run.

## 9. Evidence Sources

- [Approved E07 SPEC](../specs/E07-codex-first-harness-integration.md) — all
  fourteen acceptance criteria and their required evidence.
- [E07 design](../design/E07-codex-first-harness-integration-design.md) —
  validation-only composition, isolation, and scenario order.
- [Initial E07 Discovery](E07-discovery.md) — E01–E06 history and baseline
  evidence.
- [E07 failure analysis](E07-failure-analysis.md) — prior failure results,
  helper defects, and limits of those observations.
- [Committed E07 validation report](../validation/E07-codex-first-harness-integration.md)
  — both real-provider passes and remediation evidence.
- [Current README](../../README.md) — experimental setup and current support
  wording.
- [E07 readiness execution plan](../validation/E07-final-e2e-execution-plan.md)
  — proposed procedure and budget for the only remaining consolidated pass.
