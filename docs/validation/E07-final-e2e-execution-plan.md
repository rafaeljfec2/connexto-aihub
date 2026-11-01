# E07 Final E2E Execution Plan

Status: **PROPOSED — no execution is authorized by this document.**

Date: 2026-10-31.

## 1. Objective and Authorization Boundary

Establish the remaining E07 acceptance evidence with one consolidated,
real-provider pass over:

```text
Codex CLI → AI Hub → OpenCode Go → Provider
```

The original and supplemental passes are consumed. This plan does not authorize
a third pass. A project lead must approve this plan and the pass budget
explicitly before any provider scenario begins. If an unmet scenario fails,
do not retry or rerun it; report the result and request a new decision before
any later attempt.

The pass is limited to loopback, synthetic prompts/markers, isolated Codex and
Hub state, one selected model, and the existing E05/E06 behavior. Do not alter
production source, APIs, contracts, ADRs, global Codex files, or E01–E06
behavior. Do not install tools or bypass Codex sandbox/approval controls.

## 2. Preconditions

All conditions below must pass before the first provider-boundary request:

1. The project lead has approved this plan and the exact single-pass budget.
2. The final helper change set is reviewed and contains a hard provider-call
   cap. A call above budget must be rejected before reaching OpenCode Go.
3. The four E07 test source files exceeding 300 lines have been split within
   the authorized test-only scope; the change preserves behavior and passes
   the applicable tests and quality gates.
4. The approved secret scanner is already available. Scan tracked and
   untracked repository files without printing matched values. If no approved
   scanner is available, stop before the E2E and record the environment block.
5. Codex CLI, Go, and `bwrap` versions are recorded. Verify sandbox startup with
   a harmless local command. Use an isolated, pre-created `CODEX_HOME` beneath
   the writable runtime directory, not the shared global Codex directory.
6. Verify the provider credential and gateway credential by non-empty
   presence-only checks. The provider key is available only to AI Hub. The
   distinct gateway key is supplied to AI Hub for authentication and to Codex
   through its configured `env_key`. Before Hub startup, assert that the Hub
   process has both intended variables. Before every Codex launch, assert
   against the exact child environment passed to `exec` that the gateway key is
   present and the provider key and Hub-only variables are absent. Report only
   booleans. Enter secrets through separate non-echoing prompts or an approved
   secret facility. Never pass a value in argv, TOML, logs, or a repository
   file.
7. Confirm the selected model is available through the approved provider
   process without using a Codex scenario as a probe. If availability cannot
   be confirmed without an unbudgeted provider request, leave it unverified
   and stop for a decision.
8. Confirm loopback port 8080 is free, temporary storage is writable, the
   repository baseline is clean, and the dedicated data/work directories are
   outside the repository. Recheck port availability immediately before Hub
   startup.
9. Capture private before-state fingerprints for the complete existing global
   Codex tree, all existing repository `.env` files, tracked/untracked
   repository content, and Git state. Keep the manifest outside the repository
   with owner-only access. Do not inspect or modify the ignored `.env` values.
10. Preserve the no-support-claim wording. Do not start if isolation, scanner,
    provider, sandbox, port, or integrity preconditions fail.

## 3. Run Budget and Stop Rules

| Budget | Hard limit |
| --- | --- |
| Consolidated E2E passes | 1 invocation; no retries or repeat runs |
| Codex CLI scenario invocations | 4: one basic, two turns in one replay, one tool task |
| Provider-boundary calls | At most 9 total: 4 Codex requests, 1 upstream-error request, 1 streaming request, and at most 3 identified startup/provider-readiness calls |
| Per Codex invocation | 5 minutes |
| Per provider HTTP request | 2 minutes; streaming also has a 2-minute cap |
| Record checks | 30 seconds before stop and 30 seconds after restart |
| Total wall-clock time | 30 minutes, including setup, shutdown, cleanup, and integrity comparison |
| Automatic retries | 0; Codex request and stream retry settings remain zero |

The validation transport must enforce the nine-call ceiling before forwarding
any over-budget call. The four Codex invocations must each produce exactly the
expected Hub request count. Identify and account for every startup/readiness
call before the pass; no unplanned provider probe is allowed. If an unexpected
request would exceed the cap, reject it before provider access, stop the
remaining scenarios, and mark the pass inconclusive. No manual or automatic
retry is allowed. The prior supplemental pass reported nine provider-boundary
calls while using two separate error probes; the single same-response error
probe in this plan leaves the same hard ceiling and one fewer planned call.

Stop before provider traffic if any precondition fails. After traffic begins,
finish only the in-flight bounded operation, then check integrity before any
subsequent provider traffic. Stop immediately on credential leakage,
isolation failure, a detected global/repository integrity change,
sandbox/approval bypass requirement, unexpected call count, budget exhaustion,
or Hub lifecycle failure. A functional scenario failure does not permit retry.
Continue to independent scenarios only while safety, integrity, and remaining
call/time budgets are intact; mark skipped scenarios
`NOT RUN — environment/provider limitation`.

## 4. Ordered Scenario Plan

After all preconditions pass, run the consolidated live test exactly once from
the repository root:

```bash
AIHUB_E07_LIVE=1 go test ./cmd/ai-hub -run '^TestE07LiveCodexOpenCodeGo$' -count=1 -timeout 30m -v
```

This command was not run during this Discovery. Do not repeat it after a
failure or environmental interruption.

| ID | Related ACs | Objective and procedure | Expected result | Evidence and approval condition |
| --- | --- | --- | --- | --- |
| E07-PRE-01 | AC-1, AC-2, AC-11, AC-12, AC-14 | Verify versions, sandbox, scanner, credential presence, clean baseline, port, and isolated directories. Write a digest-only private manifest before invoking Codex. Assert the Hub environment contains its gateway and provider credentials; assert the exact Codex child environment contains the gateway credential and excludes the provider and Hub-only credentials. | Every precondition passes; no provider scenario has started. | Presence booleans and category-only readiness record. Any failure stops the pass before provider traffic. |
| E07-GEN-02 | AC-1, AC-2, AC-3, AC-4 | Start AI Hub on loopback with a dedicated data directory and provider key. Before spawning Codex, verify the exact environment allowlist passed to its process; send one basic synthetic prompt. | Non-empty successful response; exactly one Codex Responses request reaches the Hub. | CLI exit/success booleans, credential-presence booleans, and aggregate route count only. |
| E07-REPLAY-03 | AC-2, AC-5 | In one temporary Codex workspace, establish a synthetic marker in turn one and recall it in turn two. Inspect only the second provider request in memory; restore original request bytes before forwarding. | Two Hub requests; both turns succeed; marker recall passes; prior context is present; `previous_response_id` is absent. | Separate structural booleans and aggregate count. No prompt, response, request ID, or body is saved. |
| E07-TOOL-04 | AC-2, AC-6 | Ask Codex to write and read a synthetic value in a dedicated temporary workspace outside the repository, with `workspace-write` and normal approvals. | Tool round trip succeeds without bypass. | Exit/result booleans and workspace cleanup result; no tool payload in report. |
| E07-AUTH-05 | AC-3 | Issue one unauthenticated loopback request bracketed by provider counter snapshots. Keep the successful authenticated request from E07-GEN-02 as positive evidence. | Existing `401` response and provider-call delta zero. | Status and aggregate counter delta only. |
| E07-ERROR-06 | AC-7 | Send one intentionally invalid-model request through the Hub. Instrument the same upstream response and client response in memory; compare status, content type, generic-envelope classification, and streaming body digest. Do not compare separate provider calls. | Status/content type/digest match for that response; body remains a generic upstream error; Hub does not substitute local auth/gateway error. | Equality booleans only; do not emit, persist, or log response bytes or digests. Existing deterministic fixed-response test remains complementary evidence. |
| E07-STREAM-07 | AC-8 | Send one authenticated real-provider streaming request and timestamp SSE arrivals at the Hub client. Observe Codex stdout progression separately from gateway timing. | Multiple SSE events arrive before completion; CLI rendering is directly measured or `NOT VERIFIED`. | Aggregate event count and bounded timings; no response content. |
| E07-RECORDS-08 | AC-9 | Correlate all admitted scenario requests to E05 terminal observations in memory. Query E06 list and each expected ID independently. Verify provider, terminal outcome, request correlation, `model_id: null`, and `usage: null`. Exclude only the pre-E05 unauthenticated request by its captured request identity. | Every expected normal execution is queryable; each independent assertion passes. | Aggregate expected/matched/mismatch counts only; no generated IDs or record content. |
| E07-RESTART-09 | AC-10 | Gracefully stop Hub, allow E05 cleanup and the E06 two-second writer drain, restart on the same dedicated data directory, then create fresh contexts and repeat independent list/by-ID queries. | Records accepted and written under normal operation remain queryable after restart. | Aggregate recovery counts and lifecycle result only. State best-effort and loss limits. |
| E07-INT-CP | AC-1, AC-14 | After each Codex invocation and each provider-bearing scenario, and before any subsequent provider traffic, compare current global Codex/repository/environment fingerprints and Git scope to the private baseline. | No category changes between stages. If a difference appears, stop before the next provider call and mark integrity failed. | Compare digests in memory; report category outcomes only. |
| E07-INTEGRITY-10 | AC-1, AC-11, AC-12, AC-14 | After cleanup, compare the complete global Codex tree, all existing `.env` files, full repository tree, tracked file content, Git status/scope, and secret scan against the private baseline. Delete the manifest and all temporary state. | Every category passes; no unauthorized file or secret is added; no global Codex change occurs. | Category outcomes only. Never report paths, contents, credential values, hashes, or identifiers. |

### Same-response upstream oracle

The real error check must observe one response at both sides of the Hub. A
validation-only transport wrapper may hash bytes as they are read upstream and
at the client boundary, retaining only the digest in memory. It must compare
the same response instance, status, and content type only after both bodies
have been read completely through EOF with no read error, then discard the
digests. If either observation is incomplete or ends with an error, classify
body fidelity as `NOT VERIFIED`; do not infer equality from matching partial
digests. It must not make a direct provider call and compare its body with a
separate Hub call. The committed fixed-response test proves exact byte
forwarding for deterministic 400/429 cases; the real-provider check proves the
same-response route for the selected provider response.

## 5. Evidence Handling and Scenario Outcomes

1. Use only synthetic, non-sensitive prompts and markers. Keep credentials in
   their intended process environments; do not write them to command text,
   process arguments, TOML, files, or logs.
2. Capture request/response bodies and generated IDs only in process memory
   where required for replay, response hashing, and record correlation. Do not
   serialize them or include them in failure messages.
3. Emit only versions, environment class, statuses, aggregate counts,
   structural booleans, category results, and bounded timing summaries.
4. Classify each scenario as `PASS`, `FAIL`, `NOT VERIFIED`, or
   `NOT RUN — environment/provider limitation`. Any required scenario not
   marked `PASS` keeps E07 incomplete; the Codex stdout streaming observation
   may be `NOT VERIFIED` only when gateway streaming passes.
5. The evidence report must state the exact AI Hub revision, Go/Codex/bwrap
   versions, OS class, selected model, each result, limitations, and call/time
   totals. No support claim is permitted until every required AC passes.

## 6. Cleanup and Post-Run Integrity

1. Stop Codex and AI Hub gracefully. Verify that E05 execution cleanup finished
   before E06's bounded two-second writer drain completes.
2. Reuse only the dedicated temporary data directory for the planned restart;
   after recovery queries, remove it and the temporary Codex home, tool
   workspace, gateway key state, and private manifest.
3. Do not remove or restore any global Codex file. If any integrity category
   fails, preserve the category failure, stop, and leave the source
   `UNDETERMINED` unless private evidence establishes it.
4. Compare `git status`, tracked/untracked content, ignored project `.env`
   fingerprints, and production paths against the pre-run manifest. Confirm
   that only approved E07 files are present and that production source remains
   identical to E06.
5. Review scanner output and report for secret values, payloads, paths, hashes,
   and IDs. Store only the redacted scan result. Remove temporary evidence
   after its category result is confirmed.

## 7. Final Local Verification and Reporting

After cleanup, clear the live-test flag and remove live credentials from the
test environment. Then, after any authorized test-helper changes, run the
AC-12 gates:

```text
go test ./cmd/ai-hub/ -count=1 -run TestE07
go test ./... -count=1 -race
go vet ./...
staticcheck ./...
golangci-lint run ./...
go build ./...
gofmt -l cmd internal
git diff --check
```

Keep the existing tests intact. Record actual command outcomes; do not infer
that a green local suite proves any real-provider AC. Have an independent
reviewer inspect the final report, evidence redaction, full AC matrix, and
change scope. Do not commit, push, publish, or add a support claim as part of
this plan without separate authorization.

## 8. Failure and Inconclusive Conditions

The pass is **inconclusive** if it cannot reach a scenario because a required
credential, provider/model, sandbox, scanner, port, or isolated directory is
unavailable; if the call/time cap is reached; if the provider response is
ambiguous; or if integrity comparison cannot be completed. Mark affected
scenarios `NOT RUN — environment/provider limitation` or `NOT VERIFIED` as
appropriate. Do not convert an environmental limitation into a pass.

If a required behavior assertion fails, record `FAIL`, continue only with
independent scenarios while budgets and safety conditions remain valid, and
do not repeat the pass. If any integrity or credential-isolation assertion
fails, stop provider traffic immediately. Any later attempt requires a new
human decision and a newly reviewed budget.

## 9. Evidence Sources

- [E07 SPEC](../specs/E07-codex-first-harness-integration.md)
- [E07 design](../design/E07-codex-first-harness-integration-design.md)
- [E07 failure analysis](../discovery/E07-failure-analysis.md)
- [E07 validation report](E07-codex-first-harness-integration.md)
- [Final acceptance readiness](../discovery/E07-final-acceptance-readiness.md)
