# E07 Failure Analysis and Remediation Discovery

Status: investigation complete; remediation is proposed but not authorized.

Scope note (2026-11-02): this document preserves the historical E07
investigation. Its AC-14 findings remain observations from that scope and run;
the current E07 SPEC classifies AC-14 and FR-12 as `OUT OF SCOPE` for the AI
Hub product. E07 remains incomplete because AC-5 failed in the latest real
E2E evidence. No historical result is changed by this note.

Date: 2026-10-31.

Baseline: E06 commit 0d33ca26c15acf54fac418cb33380b5871b4d899. The E07 files
reviewed in this session are present as working-tree changes and are not part
of HEAD. No production source file was changed during this investigation.

## 1. Executive Summary

E07 remains incomplete. The additional E2E pass is consumed, and this Discovery
does not authorize another pass or implementation. The validation report
records failures for multi-turn replay, the upstream error-body comparison,
record verification, restart verification, and global Codex-tree integrity.

The strongest confirmed finding is a validation-helper defect in record outcome
expectations. The helper classifies every HTTP status of 400 or higher as an
execution failure. E05 instead classifies a fully read upstream 4xx response as
a completed execution while preserving its upstream status. The mismatch makes
the record verifier reject a matching record for the gateway's 400 probe if it
is present in the list. The aggregate report does not establish whether that
was the only or actual record mismatch. The same verifier gates the by-ID query
and is reused after restart, so those failures do not establish that E06 lost
records or failed recovery.

The six reported records are consistent with the six Responses executions
expected in the supplemental run: four Codex requests, one authenticated
gateway error probe, and one streaming request. The unauthenticated 401 is
rejected before E05, and the direct provider error comparison bypasses E05.
The count alone does not prove that all six requests were correlated or that
each record passed schema and outcome checks.

The upstream error check compares bodies from two separate provider calls. A
byte mismatch therefore cannot distinguish an AI Hub transformation from
variation between provider responses. The 400 status was observed; actual
same-response body passthrough remains undetermined. Existing deterministic
gateway tests do prove exact body forwarding for fixed upstream responses.

The multi-turn failure is undetermined. The helper reduces several independent
conditions to one FAIL and does not retain or report which condition failed.
There is no evidence of a production defect or an E01–E06 contract change.

AC-14 is not established. The global Codex tree changed in both runs and the
source was not identified. In addition, the integrity helper treats the Go
test package working directory as the repository root, so its project-tree
and environment-file checks did not cover the complete repository. Gitleaks
was unavailable and no repository secret scan ran.

## 2. Validation History

| Pass | Environment and scope | Relevant results | Integrity and disposition |
| --- | --- | --- | --- |
| Original E2E | E06 baseline; Codex 0.164.0-alpha.11.1; Linux amd64; bwrap unavailable | Zero Codex Responses requests reached the Hub. Two records were listed for the gateway error and stream probes. The 400 body comparison, record checks, and restart checks failed. | Global Codex tree comparison failed. Tool use was not verified. |
| Supplemental E2E | Same baseline and Codex release; bubblewrap 0.4.0; isolated HOME and CODEX_HOME; model deepseek-v4.1-flash | Basic generation, authenticated Codex route, sandboxed tool task, gateway-hop streaming, and credential isolation passed. Four Codex Responses requests reached the Hub. Replay and error-body comparison failed. Six records were observed; record and restart checks failed. | Global Codex tree comparison failed; project .env and reported repository state were unchanged; gitleaks did not run. Exactly one additional pass was authorized and consumed. |

The E07 report is the durable scenario record. A read-only ai-memory archive
corroborates the supplemental test exit and aggregate scenario output, but adds
no per-record values, replay booleans, provider bodies, or explanation for the
failed assertions. Earlier E02 and E04 reports show successful Codex scenarios
on older baselines; neither validates E05/E06. E05 records a narrower smoke.

The approved E07 SPEC is dated 2026-10-31 and has no semantic version field.
Its supplemental authorization permits a maximum of two real E2E passes. The
current report records both passes and marks the second consumed.

## 3. Failed Acceptance Criteria

| Criterion | Evidence and interpretation |
| --- | --- |
| AC-5 — full replay | The supplemental run reports FAIL. The output does not say whether marker recall, prior-context detection, or the previous_response_id check failed. The original pass did not reach the Hub. |
| AC-7 — upstream error | HTTP 400 was observed, but the generic-envelope comparison failed. Because the two bodies came from separate provider calls, actual body passthrough is not determined. |
| AC-9 — execution records | Two records were reported in the original run and six in the supplemental run. The expected correlation and schema/outcome checks failed. The helper's 400-to-failed mapping conflicts with E05 outcome semantics. |
| AC-10 — shutdown and recovery | The post-restart check failed in both reports and reused the verifier containing the incorrect outcome map. That harness defect could produce a false failure; record loss or recovery failure is not established. |
| AC-14 — environment integrity | The global Codex tree changed in both runs. The helper's repository root is incomplete, and the supplemental report says gitleaks was unavailable. AC-14 is not established. |

The supplemental run passed basic generation, authentication, tool round trip,
gateway-hop streaming, and credential isolation. Codex stdout progressive
rendering remains NOT VERIFIED, which AC-13 allows when gateway-hop streaming
passes. The original tool scenario was NOT VERIFIED because bwrap was absent;
the supplemental tool scenario passed without a sandbox or approval bypass.

## 4. Evidence Inventory

| Source | What it establishes | Limitations |
| --- | --- | --- |
| [E07 validation report](../validation/E07-codex-first-harness-integration.md:7) | Both pass environments, aggregate outcomes, counts, consumed authorization, and category-level integrity results. | No request or execution identifiers, record fields, response bodies, file paths, or integrity hashes are retained. |
| [E07 scenario helper](../../cmd/ai-hub/e07/live/e07_live_scenarios_test.go) | Scenario ordering, request counts, pass/fail conditions, record query gate, restart flow, and aggregate-only output. | The replay result is collapsed to one boolean outcome; the GET-by-ID branch depends on the list verifier passing. |
| [E07 records helper](../../cmd/ai-hub/e07/support/e07_live_http.go) | List correlation, expected outcomes, schema/null checks, polling, and single-record retrieval. | Expected outcomes are inferred from HTTP status; no per-record diagnostic is emitted. |
| [E07 upstream probe](../../cmd/ai-hub/e07/support/e07_live_scenario_support.go:178) and [body comparator](../../cmd/ai-hub/e07/support/e07_live_scenario_support.go:207) | The direct and gateway requests are distinct provider calls; body equality is byte-for-byte. | No body is retained, so the reason for inequality cannot be recovered. |
| [E07 setup](../../cmd/ai-hub/e07/support/e07_live_setup.go:34) and [repository-root helper](../../cmd/ai-hub/e07/support/e07_integrity.go:54) | Temporary data/Codex directories, before/after snapshots, cleanup, and reported integrity categories. | os.Getwd is used as the project root; the Go test runner starts the package test binary in cmd/ai-hub/e07/live. |
| [Codex child environment](../../cmd/ai-hub/e07/support/e07_observer_support.go) | The Codex child receives an allowlisted environment with HOME and CODEX_HOME pointed at the temporary directory and only the gateway key. | No filesystem tracing identifies indirect writes by Codex or child processes. |
| [E05 execution outcome path](../../internal/execution/execution_run.go:25) and [E05 regression test](../../internal/execution/manager_test.go:20) | An upstream response status is committed unchanged; an EOF after forwarding produces OutcomeCompleted, including for an upstream 4xx response. | These deterministic tests do not reproduce the real provider's error body. |
| [E06 observer and storage](../../internal/records/service.go:84) and [E06 query routes](../../internal/gateway/records_handler.go:48) | Startup recovery, bounded queue, writer, list/get query behavior, and schema projection. | E2E output does not expose queue counters or individual recovered records. |
| [E02 validation](../validation/E02-codex-opencode-go-e2e.md:43), [E04 validation](../validation/E04-gateway-hardening-access-control-e2e.md:1), ADR-004, ADR-008, ADR-011–018 | Historical harness evidence and the approved auth, streaming, execution, observer, storage, and shutdown contracts. | E02/E04 predate E05/E06 and cannot validate this baseline. |
| Local tests run during this investigation | 15 selected E07 helper tests, 225 execution/records/gateway/server tests, and 3 composition/lifecycle tests passed. | No live test, Codex process, AI Hub/provider E2E, full repository suite, or secret scan was run. |

No E07 response payload, provider error body, per-record report, or private
before/after manifest was available in the repository. The session archive
contains only aggregate run output. Generic Codex-related files in /tmp had no
verified E07 provenance and were not used as evidence.

## 5. Root Cause Analysis

### A. Codex integration

#### A1. Multi-turn replay — cause undetermined

- **Scenario:** Codex resumes a two-turn conversation through the same Hub route.
- **Acceptance criterion:** AC-5.
- **Expected:** Turn two recalls the synthetic marker; its provider request contains prior-turn context and omits previous_response_id.
- **Observed:** The supplemental report marks the combined check FAIL; four Codex Responses requests reached the Hub across basic, replay, and tool scenarios. The original run never reached the Hub.
- **Evidence:** cmd/ai-hub/e07/live/e07_live_scenarios_test.go requires a thread ID, marker recall, prior context, and absence of previous_response_id. cmd/ai-hub/e07/support/e07_observer_support.go inspects the request in memory. The report stores only the combined result.
- **Cause or hypothesis:** No individual predicate result or request/response evidence survives. Codex resume behavior, provider behavior, wire shape, and an assertion/helper issue cannot be separated.
- **Classification:** UNDETERMINED for the runtime cause. The combined-only diagnostic is a TEST_HARNESS_DEFECT because it prevents root-cause attribution.
- **Architecture impact:** None demonstrated. E04 remains request-scoped; E07 does not add durable Hub sessions or protocol translation.
- **Recommended correction:** Report separate redacted booleans for first-turn success, second-turn success, Hub request count, marker recall, prior-context presence, and previous_response_id absence. Preserve no prompt, response, identifier, or payload.
- **Proof without E2E:** Unit-test the parser and observer with synthetic inputs and test every independent failure branch. A real Codex replay claim still requires an authorized E2E pass.

Basic generation, authentication, tool execution under workspace-write, and
gateway-hop SSE all passed in the supplemental report. The 401 probe did not
increase the provider counter. Codex stdout rendering is a separate
NOT VERIFIED result, not a gateway streaming failure. These passes demonstrate
basic Responses compatibility but do not resolve replay compatibility.

#### A2. Initial-pass generation and routing — cause undetermined

- **Scenario:** Run basic generation and the authenticated Codex route through
  the documented E07 setup.
- **Acceptance criteria:** AC-3 and AC-4.
- **Expected:** Codex sends an authenticated Responses request through the Hub
  and receives a successful non-empty response.
- **Observed:** The original pass reported failed generation and zero Codex
  Responses requests at the Hub. The supplemental pass reported successful
  generation and authenticated routing, with four Codex Responses requests and
  a 401 probe that did not increase the provider counter.
- **Evidence:** The E07 validation report records both outcomes; the scenario
  helper checks generation output and provider-boundary request counts.
- **Cause or hypothesis:** The original failure occurred before a request
  reached the Hub, but the preserved evidence does not identify why. The later
  pass establishes that the tested Codex version and configuration can use the
  Responses route; it does not explain the initial failure.
- **Classification:** UNDETERMINED for the original pass. No PRODUCT_DEFECT or
  persistent INTEGRATION_DEFECT is demonstrated.
- **Architecture impact:** None demonstrated. The supplemental route uses the
  existing E04 authentication and E05 Responses execution path.
- **Recommended correction:** Preserve redacted startup, route, and response
  outcome diagnostics if a future authorized validation encounters the same
  failure; do not infer its cause from the later pass.
- **Proof without E2E:** Existing helper and gateway tests can verify
  environment allowlisting, authentication, and request counting. They cannot
  explain why the original real Codex process emitted no request.

### B. Upstream error handling

#### B1. Body comparison — invalid cross-call oracle

- **Scenario:** Unknown-model provider 400 sent directly to the provider and through the authenticated Hub.
- **Acceptance criterion:** AC-7.
- **Expected:** The Hub preserves the provider status and generic error body; local auth errors remain distinct.
- **Observed:** The report records a Hub status of 400 and a failed generic-envelope comparison in both E2E passes.
- **Evidence:** cmd/ai-hub/e07/support/e07_live_scenario_support.go invokes the provider once directly and once through the Hub. e07SameErrorEnvelope compares the two response bodies with bytes.Equal. ADR-012 defines status/body passthrough; responses_handler_test.go:282-305 verifies exact fixed-body passthrough through the gateway. The probe is a direct HTTP client request, not a Codex CLI request.
- **Cause or hypothesis:** The comparator treats independent provider responses as if they were the same response. Any provider-generated variation can fail equality even when the Hub forwards its own response unchanged. The omitted bodies prevent determining whether this happened.
- **Classification:** TEST_HARNESS_DEFECT for the comparison design; UNDETERMINED for real-provider same-response body fidelity. No PRODUCT_DEFECT is established.
- **Architecture impact:** No contract change is justified. The gateway and E05 path already preserve the upstream status and streamed body under the approved contract.
- **Recommended correction:** Prove exact byte forwarding with a deterministic fixed-body provider transport in a local integration test. For a real provider run, compare only the generic envelope invariant or compute a same-response in-memory digest at the provider and client boundaries without retaining body content.
- **Proof without E2E:** Existing fixed-response gateway tests and a focused deterministic 400 test can verify status and exact bytes. They cannot establish what the real provider returns for an unknown model; the real-provider scenario needs a newly authorized E2E to re-establish that claim.

Status passthrough and full-body passthrough are separate assertions. The
Hub's 400 response was observed, but the helper did not retain the direct
provider status separately and combined status equality with body equality in
one result. Therefore, status preservation across the real-provider calls is
not established. The failed cross-call comparison also does not prove that the
Hub changed the body. Codex's rendering or reaction to this provider 400 was
not observed in either pass.

### C. Execution records

#### C1. Record count, outcome expectation, and query gates

- **Scenario:** Correlate observed Responses requests with E06 list/get records before and after restart.
- **Acceptance criteria:** AC-9 and AC-10.
- **Expected:** Each observed execution with an accepted observer event and successful append is queryable with the expected E05 terminal outcome, provider, null model/usage, and request correlation; orderly restart recovers those records.
- **Observed:** Two aggregate records in the original pass and six in the supplemental pass. Correlation/schema/outcome checks and the post-restart check were reported as FAIL. The report provides no per-record values.
- **Evidence:** cmd/ai-hub/e07/support/e07_live_http.go maps any response status >=400 to failed. execution_run.go:25-40 forwards the upstream response; execution_run.go:81-87 marks EOF after forwarding as completed. manager_test.go:20-52 confirms a completed upstream 429 with its status preserved. cmd/ai-hub/e07/live/e07_live_scenarios_test.go gates GET-by-ID on the list/outcome verifier and reuses the same verifier after restart.
- **Cause or hypothesis:** The gateway's unknown-model 400 is a valid upstream HTTP response, not necessarily an execution failure. Once its response body is forwarded and reaches EOF, E05 records completed. If a record for that request is present, the helper expects failed solely from status and rejects the E05 completed outcome. This is a confirmed harness defect and can explain the aggregate record and restart failures, but the report does not establish that it was the only or actual mismatch. The six-record count is consistent with all six expected executions, but cannot prove per-request coverage.
- **Classification:** TEST_HARNESS_DEFECT is confirmed for the status-to-outcome mapping and shared verifier. Whether every expected request ID was present and whether restart recovered every record remains UNDETERMINED from the aggregate report.
- **Architecture impact:** None demonstrated. The error is in the E07 validation oracle; changing E05/E06 outcome semantics or storage contracts would violate ADR-014, ADR-016, and ADR-018 without evidence.
- **Recommended correction:** Derive expected outcomes from E05 lifecycle semantics, not HTTP status. Assert list correlation, provider, outcome, model_id null, and usage null independently. Run the by-ID check even when another assertion fails, and use a fresh context for restart queries. Keep diagnostics aggregate-only.
- **Proof without E2E:** Add deterministic helper tests for upstream 2xx and 4xx responses that are fully read, plus read/transport failures. Existing execution, records, route, storage-recovery, and lifecycle suites can prove outcome projection, list/get, queue flush, and restart recovery locally.

The six supplemental records are explained by the scenario shape:

| Request | E05 execution expected? | Reason |
| --- | --- | --- |
| Four Codex Responses requests | Yes | Basic generation (one), replay (two), and tool task (one) reached the Hub. |
| Authenticated gateway 400 probe | Yes | It traverses the Responses execution manager. |
| Gateway streaming request | Yes | It traverses the Responses execution manager. |
| Unauthenticated 401 probe | No | E04 rejects it before E05 admission; the provider counter remained unchanged. |
| Direct provider error comparison | No | It calls the provider client directly and bypasses the Hub execution manager. |
| Provider startup connectivity check | No | It is not a Responses execution registered with E05. |

Thus four plus one plus one yields six expected execution records. The original
pass had no Codex requests, so the gateway 400 and stream account for two
expected records. The supplemental aggregate provider-call count of nine also
includes non-execution provider traffic and must not be interpreted as nine
records.

#### C2. E05/E06 observer, queue, writer, storage, and lifecycle

- **Scenario:** Deliver finished E05 executions to the E06 JSONL store, query the in-memory index, stop, and recover from the same directory.
- **Acceptance criteria:** AC-9 and AC-10; E06 ADR-017 and ADR-018.
- **Expected:** The observer is called after E05 cleanup and before deregistration; it enqueues without blocking; the writer appends accepted records; the index exposes successful writes; shutdown waits for E05, then drains records for up to two seconds before provider release; restart rebuilds the index from JSONL.
- **Observed:** E2E assertions failed, and the same verifier ran before and after restart. Its incorrect outcome expectation could produce a false failure, but the aggregate report does not identify the specific mismatch. No report evidence identifies queue overflow, storage failure, degraded history, or a data-directory mismatch.
- **Evidence:** internal/execution/execution_terminal.go:30-45 publishes the observer before removing the execution. internal/records/service.go:121-151 closes admission and waits for the bounded drain. cmd/ai-hub/main.go:173 and cmd/ai-hub/e07/support/e07_live_hub.go:86 register modules in reverse shutdown order; cmd/ai-hub/lifecycle/lifecycle.go:32 performs the lifecycle stop. The helper reuses the same temporary data directory for restart. The local execution/records/gateway/server suites and three lifecycle/composition tests passed during this investigation.
- **Cause or hypothesis:** C1's helper defect could explain some or all of the aggregate record and restart failures; the report does not establish that it was the actual or sole cause. A separate E06 defect is not demonstrated. Actual completeness after restart remains unproven because per-record comparisons were not preserved.
- **Classification:** TEST_HARNESS_DEFECT for the E2E verifier; UNDETERMINED for actual per-record recovery completeness. No PRODUCT_DEFECT is supported by available evidence.
- **Architecture impact:** No change to the approved E05→E06 observer boundary, 256-entry best-effort queue, single writer, JSONL segments, in-memory index, two-second drain, or module order is indicated.
- **Recommended correction:** Fix C1 and independently report list and single-ID query results before and after restart. Include an aggregate count and schema/outcome mismatch counts; never publish IDs or record contents.
- **Proof without E2E:** Existing tests cover observer placement, queue overflow, writer flush, JSONL recovery, index lookup, list/get routes, and lifecycle ordering. Focused local composition tests can verify the same dedicated data directory is reused.

### D. Environment integrity and security

#### D1. Global Codex tree changed — source undetermined

- **Scenario:** Compare the global Codex tree before and after E07 setup, version probe, and Codex child runs.
- **Acceptance criterion:** AC-14.
- **Expected:** The complete pre-existing global Codex tree remains unchanged; Codex state is confined to the dedicated temporary HOME and CODEX_HOME.
- **Observed:** The global tree comparison failed in both runs. The supplemental report states that HOME and CODEX_HOME were isolated. No source path or change detail was retained.
- **Evidence:** E07PrepareLiveSetup in cmd/ai-hub/e07/support/e07_live_setup.go:34 fingerprints the global tree; e07CodexEnvironment sets both child variables to the temporary Codex directory and uses an environment allowlist. The report states that the source of the tree change was not established.
- **Cause or hypothesis:** The helper itself writes Codex configuration, Hub data, and tool files under temporary paths. It reads the global Codex tree for comparison. An indirect write by Codex or a child, an executable wrapper, or concurrent external activity are hypotheses only; no file-write trace distinguishes them.
- **Classification:** UNDETERMINED for the source of the change. The failed integrity result remains a confirmed security/integrity blocker.
- **Architecture impact:** No AI Hub contract or product-code change is indicated. The isolated Codex process boundary must be understood before another validation.
- **Recommended correction:** Do not restore or edit global files. No retained private manifest is available in the report or repository; if separate private evidence still exists, inspect its differences without exposing paths or contents. A future controlled trace would need new authorization and isolated state.
- **Proof without E2E:** Static inspection confirms the helper's direct file writes target the temporary root. Unit tests can prove environment allowlisting and fingerprint behavior, but cannot prove the real Codex binary made no indirect global write. AC-14 requires a future authorized environment-integrity validation.

#### D2. Repository and .env integrity scope — helper defect

- **Scenario:** Prove that existing project .env files and unrelated tracked files were unchanged.
- **Acceptance criterion:** AC-14.
- **Expected:** Before/after comparison covers the repository root, all existing .env files, and the complete relevant working tree.
- **Observed:** The E2E report marks project .env, worktree, and Git state as PASS. The helper obtains root with os.Getwd. Go's test runner executes the package test binary in the package source directory, so the documented go test ./cmd/ai-hub/e07/live invocation makes root cmd/ai-hub/e07/live, not the repository root. The existing dirty README is outside that fingerprint. Git status equality alone cannot detect content changes to a file that was already dirty.
- **Evidence:** E07PrepareLiveSetup in cmd/ai-hub/e07/support/e07_live_setup.go:34 fingerprints root and loads root/.env, then repeats the same scope for subsequent checks. go help testflag documents the package source-directory working directory. e07GitState records only git status output.
- **Cause:** The helper assumes the package working directory is the repository root and treats an unchanged Git status string as a content-integrity check.
- **Classification:** TEST_HARNESS_DEFECT.
- **Architecture impact:** None; this is validation instrumentation only.
- **Recommended correction:** Resolve the repository root from Git metadata or a stable module marker. Fingerprint every project .env and all relevant file content, including already-dirty files, with a private manifest outside the repository. Keep emitted results category-only.
- **Proof without E2E:** Use temporary fixture repositories with root and nested .env files, a pre-dirty tracked file, an untracked file, and symlinks. Assert both unchanged and changed fixtures are detected. No provider call or Codex launch is needed.

The fingerprint helper follows a symlink when the root itself is a symlink but
records only the link target for symlinks encountered below a directory. It
does not hash nested symlink target contents. It also fingerprints file mode,
size, and content rather than all filesystem metadata. No evidence ties these
limitations to the observed global Codex-tree change; they further limit an
unqualified claim that the entire tree was unchanged.

#### D3. Secret scan unavailable

- **Scenario:** Scan repository state for accidentally added credentials.
- **Acceptance criterion:** AC-14 and the FR-12 evidence path.
- **Expected:** A secret scan covers the E07 repository change set and reports only a redacted category result.
- **Observed:** The E07 report says gitleaks was unavailable and the scan was NOT RUN. The current environment also has no gitleaks executable; repository search found no configured alternative scanner.
- **Evidence:** E07 validation report lines 48-60; the repository references gitleaks only as an unavailable scanner in validation evidence.
- **Cause:** Required scanner tooling was absent from the validation environment.
- **Classification:** ENVIRONMENT_LIMITATION.
- **Architecture impact:** None.
- **Recommended correction:** Make an approved secret scanner available in the environment and scan tracked and untracked E07 files without printing matched secret values. Do not install tooling as part of this Discovery.
- **Proof without E2E:** Run the scanner locally against the repository change set and retain only its redacted pass/fail result. This does not require Codex, AI Hub, or the provider.

## 6. Architecture Impact

No change to E01–E06 architecture is supported by the evidence. The confirmed
defects concern E07's outcome oracle, body comparator, diagnostics, and
integrity instrumentation. E04 authentication passed in the supplemental run;
the combined request-correlation/schema/outcome check did not, and its
individual correlation result is undetermined. E05 correctly distinguishes a
completed upstream HTTP response from transport/read failure. E06 remains
best-effort and at-most-once, with an asynchronous writer and bounded shutdown
drain. The E07 composition uses the approved lifecycle order and same dedicated
data directory on restart.

No model routing, provider selection, durable sessions, protocol translation,
new provider, storage backend, microservice, or production refactor is
recommended. No ADR or historical contract change is needed on current
evidence. If a later investigation demonstrates a required contract change,
record it as a separate decision before implementation.

## 7. Security and Integrity Findings

- Both E2E reports record a changed global Codex tree. The latest helper uses
  temporary HOME and CODEX_HOME, but no path-level diff or process write trace
  explains the change. Do not modify or restore global files as remediation.
- The Codex child allowlist excludes provider and Hub-only credentials and
  supplies only the gateway credential. The report's presence-only credential
  checks passed.
- E07 config, Hub data, and tool workspace are created below a temporary root;
  the Hub uses a dedicated data directory. No helper source write to the global
  Codex tree was found.
- The environment-integrity helper does not cover the repository root or root
  .env files under the documented Go test command. Git status equality does
  not prove content equality for pre-existing dirty files.
- Gitleaks was unavailable and no secret scan ran. Therefore the no-secret
  repository claim is not established by this E2E evidence.
- No secret value, prompt, response, provider-session ID, request ID, execution
  ID, global path, or integrity hash is included in this Discovery.

## 8. Remediation Priorities

| Priority | Remediation | Potential files or modules | Tests and local evidence | E2E needed? | E01–E06 regression risk |
| --- | --- | --- | --- | --- | --- |
| P0 | Resolve the global Codex-tree change before any further E07 validation; correct repository-root, .env, and dirty-file integrity coverage. Preserve category-only reporting and never repair global files. | cmd/ai-hub/e07/support/e07_live_setup.go; cmd/ai-hub/e07/support/e07_validation_helpers.go; cmd/ai-hub/e07/support/e07_validation_support.go | Temporary-tree tests for root resolution, all .env locations, already-dirty content, symlinks, and private manifests. Inspect retained evidence if it exists; none was found in the report or repository. | A real environment-integrity claim needs a newly authorized validation. The consumed pass cannot be repeated under the existing authorization. | None if limited to test helpers; accidental changes to environment or shared files carry integrity risk. |
| P0 | Run a secret scan when approved scanner tooling is available. | Repository scan configuration or validation environment; no production module. | Redacted scan of tracked and untracked E07 files; record only category result. | No. | None. |
| P1 | Correct record expectations so execution outcome is based on E05 terminal semantics rather than HTTP status. Do not let one mismatch suppress list/get or restart assertions. | cmd/ai-hub/e07/support/e07_live_http.go; cmd/ai-hub/e07/live/e07_live_scenarios_test.go; existing execution/records tests. | Unit cases for completed upstream 4xx, transport failure, body-read failure, list correlation, null fields, by-ID lookup, and restart recovery. | A real record/restart claim needs a newly authorized validation after helper corrections. | None if changes remain in validation helpers and tests. |
| P1 | Replace the cross-call byte comparison with a valid same-response passthrough oracle. Preserve the status/body distinction and do not retain body content. | cmd/ai-hub/e07/support/e07_live_scenario_support.go; cmd/ai-hub/e07/support/e07_integrity.go; internal/gateway response tests only if existing deterministic coverage is insufficient. | Deterministic injected-provider test proves exact bytes/status; real-provider helper tests generic envelope classification or same-response digest logic. | Required to re-establish the real-provider scenario; needs new authorization. | Low; avoid changing production response behavior. |
| P1 | Split replay checks and emit only redacted structural outcomes. | cmd/ai-hub/e07/live/e07_live_scenarios_test.go; cmd/ai-hub/e07/support/e07_live_codex.go; cmd/ai-hub/e07/support/e07_observer_support.go. | Synthetic tests for JSON event parsing, each condition, request-body restoration, and redaction. | Required to establish real Codex replay; needs new authorization. | None if limited to test helpers. |
| P2 | Make single-record query, pre-stop list query, and post-restart list/get checks independent and use a fresh context for each phase. | cmd/ai-hub/e07/support/e07_live_http.go; cmd/ai-hub/e07/live/e07_live_scenarios_test.go. | Existing gateway list/get tests plus focused helper tests with missing, delayed, and recovered rows. | Yes for real-provider query/restart evidence; needs new authorization. | Low; tests only. |
| P3 | Keep Codex stdout progressive rendering as NOT VERIFIED unless the tested CLI exposes a measurable signal. Revisit only when direct timing evidence is available. | Validation documentation only. | Review report wording against AC-13. | No, unless a new measurable claim is introduced. | None. |

P0 items are integrity gates. P1 helper corrections should precede any future
authorized E2E pass. This Discovery itself performs no such pass.

## 9. Proposed Tests and Verification Strategy

1. **Record outcome mapping:** Verify that a fully forwarded upstream 400 or
   429 is recorded as completed with the original upstream status; transport
   and body-read failures retain their E05 failure outcomes. Assert that HTTP
   status and terminal outcome are separate fields.
2. **Record delivery and recovery:** Use the existing observer, queue, writer,
   JSONL recovery, index, GET-by-ID, list-route, and lifecycle tests. Add a
   focused E07 helper test proving that list failure does not suppress by-ID
   diagnostics and that a new context is used after restart.
3. **Upstream body oracle:** Send one deterministic provider response through
   the gateway and assert exact status and bytes. Keep an additional real
   provider probe limited to status and generic error classification, or
   compare same-response digests in memory without persisting content.
4. **Replay diagnostics:** Feed synthetic Codex JSON events and request bodies
   through the parser/observer. Assert independent outcomes for marker recall,
   prior context, and previous_response_id, while proving payload bytes are
   restored before forwarding and no content is emitted.
5. **Integrity snapshots:** Build temporary fixture repositories with root
   .env files, nested .env files, pre-existing dirty tracked files, untracked
   files, directories, and symlinks. Prove that changes are detected without
   outputting paths, contents, or hashes.
6. **Secret scan:** Run an approved scanner over tracked and untracked files
   when available. Gitleaks is not installed in the current environment.
7. **Future live pass:** Only after the P0 and P1 work is reviewed, obtain a new
   explicit authorization for one bounded Codex/provider validation. The
   earlier supplemental pass is consumed; this report grants no new pass.

Current local verification performed for this Discovery:

- 15 selected deterministic E07 helper tests passed.
- 225 tests across internal/execution, internal/records, internal/gateway,
  and internal/server passed.
- Three command-composition and lifecycle shutdown tests passed.
- No live E2E, Codex process, AI Hub/provider integration, full repository
  quality suite, or secret scan was run.

## 10. Dependencies and Risks

- Integrity coverage and the source of the global Codex-tree change must be
  resolved before relying on AC-14. The report has no path-level diff, and no
  private before/after manifest was found in the repository.
- Correct record expectations must land before evaluating whether all six
  records were captured or recovered. Otherwise the same false failure can
  recur across capture, by-ID, and restart checks.
- The live provider's error response may vary between independent requests.
  A deterministic local test proves gateway forwarding but does not replace
  real-provider evidence required by AC-7.
- Replay diagnostics must be fixed before another Codex run; the current
  failure output cannot distinguish Codex behavior, provider behavior, or the
  test oracle.
- Any future real E2E requires new authorization. Do not infer approval from
  the consumed supplemental pass or from this remediation plan.
- Test-only corrections should not affect E01–E06. A production or contract
  change is not justified by current evidence and would require a separate
  scope decision.

## 11. Open Questions

1. Which global Codex-tree entry changed, and which process or concurrent
   actor changed it? The before/after source report contains no path-level data.
2. Which replay predicate failed: marker recall, prior-context presence, or
   previous_response_id absence? The helper does not report these separately.
3. Did all six supplemental execution request IDs appear in the six records?
   The aggregate count does not establish correlation completeness.
4. Did any record field besides the 400 outcome expectation mismatch? No
   per-record diagnostic was retained.
5. Did the provider return different generic error bodies across the two
   independent probes, or did the Hub response differ? The bodies were not
   retained and cannot be compared now.
6. Can an approved secret scanner be made available in the next validation
   environment without installing tooling during this Discovery?

No architectural decision is pending on current evidence.

## 12. Recommended Next Steps

1. Keep E07 status INCOMPLETE and retain the no-support-claim wording.
2. Resolve the global integrity finding without touching or restoring global
   Codex files. If no separate private before/after manifest is available,
   record that the source is undetermined for these runs.
3. Correct the E07 outcome oracle, body comparison, replay diagnostics, and
   integrity root discovery in test-only changes, then add the focused tests in
   Section 9.
4. Run the local E07 helper, E05/E06, gateway, query, lifecycle, and approved
   secret-scan checks; preserve only redacted aggregate evidence.
5. Submit the corrected validation harness and updated analysis to an
   independent reviewer.
6. Request new authorization before any future real Codex/provider E2E. Do not
   alter E01–E06 contracts or begin another epic without that separate scope.

## 13. Explicit Non-Goals

- Implementing any proposed remediation in this session.
- Running Codex against OpenCode Go, starting AI Hub for integrated
  validation, or consuming a third E2E pass.
- Restoring, changing, or repairing global Codex configuration, files, or
  caches.
- Changing E01–E06 production behavior, APIs, ADRs, or acceptance criteria.
- Adding model routing, model selection, quota management, learning, a new
  provider, a new harness, a microservice, or an unrelated refactor.
- Publishing secrets, request/response content, generated identifiers,
  global paths, or integrity hashes.
- Committing or pushing changes.
