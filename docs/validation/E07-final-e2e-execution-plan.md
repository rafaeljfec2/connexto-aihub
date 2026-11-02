# E07 Final E2E Execution Plan

Status: **E07 remains INCOMPLETE. The authorized pass recorded below remains unused; this scope amendment does not start, renew, or expand it.**

Plan drafted: 2026-10-31. Conditional authorization recorded: 2026-11-01.

## 1. Objective and Authorization Boundary

Establish the remaining E07 acceptance evidence with one consolidated,
real-provider pass over:

```text
Codex CLI → AI Hub → OpenCode Go → Provider
```

The original and supplemental passes are consumed. The project lead's
2026-11-01 final acceptance prompt authorized one additional pass, which ran
once as documented in the validation report. A further pass was authorized
on 2026-11-02 and remained unused at the latest recorded checkpoint. That
authorization is not part of this documentation amendment and is neither
renewed nor expanded here. No host external-symlink integrity condition is an
E07 start gate because AC-14 is out of product scope. If a scenario fails
after a pass starts, do not retry or rerun it; report the result and require a
new decision before any later attempt.

The pass is limited to loopback, synthetic prompts/markers, isolated Codex and
Hub state, one selected model, and the existing E05/E06 behavior. Do not alter
production source, APIs, contracts, ADRs, global Codex files, or E01–E06
behavior. Do not install tools or bypass Codex sandbox/approval controls.

## 2. Preconditions

All conditions below must pass before the first provider-boundary request:

1. The separately recorded 2026-11-02 authorization remains unused and is
   subject to the limits in §3; this documentation amendment neither consumes
   nor renews it. All operational preconditions below still must pass before
   any separately authorized execution.
2. The final helper change set is reviewed and contains a hard provider-call
   cap. A call above budget must be rejected before reaching OpenCode Go.
3. The four E07 test source files exceeding 300 lines have been split within
   the authorized test-only scope; the change preserves behavior and passes
   the applicable tests and quality gates.
4. Codex CLI, Go, Bubblewrap, and GNU `timeout` versions are recorded. Verify
   sandbox startup with a harmless local command. Run the complete live test
   process inside private mount and PID namespaces. Bind the host root
   read-only, hide `/home` behind namespace tmpfs, and cover the account
   `~/.codex` with an empty read-only mount. Make a private tmpfs runtime the
   only writable run mount; do not bind a writable host directory. Keep network
   access available for the real provider route. Use an isolated, pre-created
   `CODEX_HOME` and dedicated XDG directories beneath that private runtime.
   Verify the effective mounts and paths before Codex starts, and prove a child
   process inherits the boundary.
5. Load the existing credentials from the repository `.env` with the
   production dotenv parser inside the read-only repository mount. Verify
   presence only. The provider key remains in the Hub test process; the
   distinct gateway key is supplied to Codex through its configured `env_key`.
   Before each Codex launch, assert against the exact child environment that
   the gateway key is present and provider/Hub-only keys are absent. Report
   booleans only. Do not rotate, regenerate, copy, print, or pass credential
   values in argv, TOML, logs, or new files.
6. Do not probe model availability or run provider diagnostics outside the
   authorized pass. The first budgeted scenario request is the availability
   observation. If it fails, record that scenario result and continue only
   within the existing call/time limits; do not retry or issue a separate
   diagnostic request.
7. Confirm loopback port 8080 is free and temporary storage is writable. Treat
   the exact reviewed E07 worktree at pass start as the immutable repository
   baseline: capture its private metadata/content manifest before setup, keep
   the repository mount read-only for the full pass, and require no delta after
   cleanup. Production paths must still match the E06 baseline. Keep dedicated
   data/work directories outside the repository. Recheck port availability
   immediately before Hub startup.
8. Capture private state only for the isolated E2E runtime and, as development
   hygiene, the authorized repository/Git scope. Keep manifests outside the
   repository with owner-only access. Do not inspect or modify ignored `.env`
   values. Do not fingerprint the host-global Codex tree or filesystem.
9. Preserve the no-support-claim wording. Do not start if process isolation,
    credential handling, provider, sandbox, or port preconditions fail.

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

One external hard watchdog must start before E07 preflight/setup and wrap the
entire isolated supervisor, including test-binary compilation, namespace
startup, `e07PrepareLiveSetup`, scenarios, shutdown, integrity comparison, and
cleanup. Use `/usr/bin/timeout --signal=KILL 30m` as the outermost command; do
not start a new command outside that supervisor after the timer begins. The
supervisor must keep every writable run artifact (test binary, Go build cache,
Codex state, Hub data, workspaces, manifests, and temporary files) on the
private namespace tmpfs; the repository and source toolchain remain
read-only. This makes watchdog expiry a terminal stop that discards the
namespace state without touching the host repository or Codex tree. If the
watchdog fires, classify the pass as `INCONCLUSIVE`, mark after-state integrity
`NOT VERIFIED`, stop all provider traffic, and do not retry. The watchdog is
the hard 30-minute ceiling; the per-request and per-invocation limits below
remain independently enforced.

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
finish only the in-flight bounded operation, then check private-state and
repository integrity before any subsequent provider traffic. Stop immediately
on credential leakage, isolation failure, a private-state or repository
integrity change, sandbox/approval bypass requirement, unexpected call count,
budget exhaustion, or Hub lifecycle failure. A functional scenario failure
does not permit retry.
Continue to independent scenarios only while safety, integrity, and remaining
call/time budgets are intact; mark skipped scenarios
`NOT RUN — environment/provider limitation`.

## 4. Ordered Scenario Plan

After all preconditions pass, start the single 30-minute watchdog before any
preflight/setup command and execute the entire pass through the verified
Bubblewrap mount and PID boundary. Build the test binary inside the supervisor,
with credentials absent and `HOME`, `GOCACHE`, `TMPDIR`, and other writable Go
paths under the private tmpfs. Do not source the runtime or test binary from
host `/tmp`. Bind the repository read-only, make the private tmpfs the only
writable run mount, and pass an explicit environment allowlist for `PATH`,
private `HOME`, `CODEX_HOME`, `CODEX_SQLITE_HOME`, `TMPDIR`, `TEMP`, `TMP`, XDG
directories, `GOCACHE`, `GOMODCACHE`, and `AIHUB_E07_PRIVATE_ROOT`. Keep
`GOMODCACHE` read-only. No credential values enter the supervisor or build
environment. The live helper loads the existing repository `.env` with the
production dotenv parser from the read-only repository mount; values remain in
the Hub test process and the Codex child receives only its configured gateway
key. The versioned launcher defines the complete Bubblewrap argument vector,
the namespace-local account view, and the environment allowlist:

```bash
./scripts/e07-live-supervisor.sh --run-live
```

The launcher requires the explicit `--run-live` mode; `--preflight` runs only
local checks with a synthetic Responses endpoint and the live-test flag unset.
The GNU `timeout` clock includes environment discovery, Bubblewrap startup,
test-binary compilation, `e07PrepareLiveSetup`, graceful Hub shutdown, final
integrity comparison, and normal cleanup. The Go test timeout is an inner
guard; the external watchdog remains authoritative. Do not run `go test` with
credentials present in the environment. Do not repeat the live invocation
after a failure, watchdog expiry, or environmental interruption.

| ID | Related ACs | Objective and procedure | Expected result | Evidence and approval condition |
| --- | --- | --- | --- | --- |
| E07-PRE-01 | AC-1, AC-2, AC-11, AC-12 | Verify versions, sandbox, credential presence, reviewed worktree scope, port, and isolated directories. Write private runtime/repository manifests before invoking Codex. Assert the Hub test process contains its gateway and provider credentials; assert the exact Codex child environment contains the gateway credential and excludes provider and Hub-only credentials. | Every operational precondition passes; no provider scenario has started. | Presence booleans and category-only readiness record. Any operational failure stops the pass before provider traffic. |
| E07-GEN-02 | AC-1, AC-2, AC-3, AC-4 | Start AI Hub on loopback with a dedicated data directory and provider key. Before spawning Codex, verify the exact environment allowlist passed to its process; send one basic synthetic prompt. | Non-empty successful response; exactly one Codex Responses request reaches the Hub. | CLI exit/success booleans, credential-presence booleans, and aggregate route count only. |
| E07-REPLAY-03 | AC-2, AC-5 | In one temporary Codex workspace, establish a synthetic marker in turn one and recall it in turn two. Inspect only the second provider request in memory; restore original request bytes before forwarding. | Two Hub requests; both turns succeed; marker recall passes; prior context is present; `previous_response_id` is absent. | Separate structural booleans and aggregate count. No prompt, response, request ID, or body is saved. |
| E07-TOOL-04 | AC-2, AC-6 | Ask Codex to write and read a synthetic value in a dedicated temporary workspace outside the repository, with `workspace-write` and normal approvals. | Tool round trip succeeds without bypass. | Exit/result booleans and workspace cleanup result; no tool payload in report. |
| E07-AUTH-05 | AC-3 | Issue one unauthenticated loopback request bracketed by provider counter snapshots. Keep the successful authenticated request from E07-GEN-02 as positive evidence. | Existing `401` response and provider-call delta zero. | Status and aggregate counter delta only. |
| E07-ERROR-06 | AC-7 | Send one intentionally invalid-model request through the Hub. Instrument the same upstream response and client response in memory; compare status, content type, generic-envelope classification, and streaming body digest. Do not compare separate provider calls. | Status/content type/digest match for that response; body remains a generic upstream error; Hub does not substitute local auth/gateway error. | Equality booleans only; do not emit, persist, or log response bytes or digests. Existing deterministic fixed-response test remains complementary evidence. |
| E07-STREAM-07 | AC-8 | Send one authenticated real-provider streaming request and timestamp SSE arrivals at the Hub client. Observe Codex stdout progression separately from gateway timing. | Multiple SSE events arrive before completion; CLI rendering is directly measured or `NOT VERIFIED`. | Aggregate event count and bounded timings; no response content. |
| E07-RECORDS-08 | AC-9 | Correlate all admitted scenario requests to E05 terminal observations in memory. Query E06 list and each expected ID independently. Verify provider, terminal outcome, request correlation, `model_id: null`, and `usage: null`. Exclude only the pre-E05 unauthenticated request by its captured request identity. | Every expected normal execution is queryable; each independent assertion passes. | Aggregate expected/matched/mismatch counts only; no generated IDs or record content. |
| E07-RESTART-09 | AC-10 | Gracefully stop Hub, allow E05 cleanup and the E06 two-second writer drain, restart on the same dedicated data directory, then create fresh contexts and repeat independent list/by-ID queries. | Records accepted and written under normal operation remain queryable after restart. | Aggregate recovery counts and lifecycle result only. State best-effort and loss limits. |
| E07-INT-CP | AC-1 | After each Codex invocation and each provider-bearing scenario, and before any subsequent provider traffic, compare isolated Codex-state snapshots and confirm the exact child credential allowlist. | Private Codex state remains isolated and the provider credential is absent from every Codex child. | Report structural booleans and aggregate outcomes only. |
| E07-REPO-HYGIENE-10 | Development hygiene; no AC | Compare the reviewed E07 repository/Git scope with its private pre-run baseline. Do not inspect ignored `.env` values, fingerprint the host-global Codex tree, or run a secret scanner as an E07 acceptance step. | Any out-of-scope repository change is investigated as a development-hygiene issue, not reported as a product acceptance result. | Keep repository/Git outcomes separate from product AC results. Never report secrets or private manifest contents. |

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
3. Do not inspect, remove, or restore any global Codex file. Host-global state
   is outside E07 product acceptance and is not fingerprinted.
4. As development hygiene, compare `git status`, authorized tracked/untracked
   source scope, and production paths against the pre-run manifest. Confirm
   that only approved E07 files are present and that production source remains
   identical to E06. Do not inspect ignored `.env` values.
5. Review the report and E07 diff for secret values, payloads, paths, hashes,
   and IDs. Do not claim a secret-free repository without evidence. Remove
   temporary evidence after its development-hygiene result is confirmed.

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
credential, provider/model, sandbox, port, or isolated directory is
unavailable; if the call/time cap is reached; or if the provider response is
ambiguous. Mark affected
scenarios `NOT RUN — environment/provider limitation` or `NOT VERIFIED` as
appropriate. Do not convert an environmental limitation into a pass.

If a required behavior assertion fails, record `FAIL`, continue only with
independent scenarios while budgets and safety conditions remain valid, and
do not repeat the pass. If any integrity or credential-isolation assertion
fails, stop provider traffic immediately. Any later attempt requires a new
human decision and a newly reviewed budget.

### AC-14 scope disposition

AC-14 is `OUT OF SCOPE` for the AI Hub product. The Hub centralizes provider
access and does not execute Codex or administer its host environment. The
global `~/.codex` tree, external symlink destinations, project `.env` files,
host filesystem fingerprints, and repository secret scanning are not E07
acceptance gates. Historical observations remain in the validation report and
must not be rewritten as PASS. Repository/Git scope checks remain development
hygiene; credential isolation and gateway security remain product validation.

## 9. Evidence Sources

- [E07 SPEC](../specs/E07-codex-first-harness-integration.md)
- [E07 design](../design/E07-codex-first-harness-integration-design.md)
- [E07 failure analysis](../discovery/E07-failure-analysis.md)
- [E07 validation report](E07-codex-first-harness-integration.md)
- [Final acceptance readiness](../discovery/E07-final-acceptance-readiness.md)

## 10. Historical E2E Readiness Checkpoints — Preserved Record

The sections below preserve the readiness record and plan that existed before
the 2026-11-02 architectural scope disposition. They are historical evidence,
not current acceptance criteria or operational gates. The recorded E2E
outcomes remain unchanged.

### Reviewed working-tree baseline

| Item | Current value |
| --- | --- |
| Branch / HEAD | `main` / `a4027c24ffdb788136a60956b4f270834f99cce9` |
| Staged changes | None |
| Tracked E07 modifications | 10 files listed below |
| Untracked E07 tests | 10 files listed below |
| Untracked E07 supervisor | `scripts/e07-live-supervisor.sh` |
| `.env` | Not listed by Git; ignored values were not displayed or changed |
| Commit / push | None |

Tracked modifications:

- `cmd/ai-hub/e07_codex_env_policy_support_test.go`
- `cmd/ai-hub/e07_integrity_test.go`
- `cmd/ai-hub/e07_integrity_tree_test.go`
- `cmd/ai-hub/e07_live_hub_test.go`
- `cmd/ai-hub/e07_live_scenarios_test.go`
- `cmd/ai-hub/e07_live_setup_test.go`
- `cmd/ai-hub/e07_provider_observer_test.go`
- `cmd/ai-hub/e07_validation_helpers_test.go`
- `docs/validation/E07-codex-first-harness-integration.md`
- `docs/validation/E07-final-e2e-execution-plan.md`

Untracked E07 tests:

- `cmd/ai-hub/e07_codex_mount_boundary_test.go`
- `cmd/ai-hub/e07_codex_mount_support_test.go`
- `cmd/ai-hub/e07_codex_resume_test.go`
- `cmd/ai-hub/e07_integrity_git_test.go`
- `cmd/ai-hub/e07_integrity_hash_test.go`
- `cmd/ai-hub/e07_integrity_mount_test.go`
- `cmd/ai-hub/e07_integrity_secret_test.go`
- `cmd/ai-hub/e07_live_credentials_test.go`
- `cmd/ai-hub/e07_live_evidence_test.go`
- `cmd/ai-hub/e07_live_integrity_support_test.go`

The supervisor is part of the authorized validation scope and remains
untracked until the user authorizes a commit. Include it in any working-tree
scope review; do not describe it as versioned.

### Isolated execution environment

The local preflight passed on Linux `amd64` with Bubblewrap 0.4.0 and Go
`1.23.0`. Use Bubblewrap's mount and PID namespaces without an explicit UID-0
mapping; that variant prevented the nested Codex sandbox from starting. The
tested namespace-local passwd view maps the executing UID/GID `1000:1000` to
`e07-homologation` and `/home/e07-homologation`. It is not a host OS account;
the numeric UID is shared with the host, and all setup is recreated inside the
namespace for this one run.

The exact AC-14 account root is `/home/e07-homologation/.codex`, resolved by
the same account lookup used by the integrity helper. Mount that path empty
and read-only inside the private tmpfs home. Mount the developer's `/home`
behind tmpfs so the real `/home/rafael/.codex` and other developer home data
are inaccessible in the E2E view. Bind the repository read-only at
`/tmp/e07/repository`; do not mount a writable repository path. Mount the
per-run tmpfs runtime at `/tmp/e07/runtime`, with `HOME`, `CODEX_HOME`,
`CODEX_SQLITE_HOME`, XDG config/data/cache/state/runtime, and `TMPDIR`/`TEMP`/
`TMP` below it. Keep the test binary, Codex executable, and Go module cache
read-only. Preserve network access for the authorized provider route.

The E07 focused suite passed in this layout with the Codex CLI's normal
`workspace-write` sandbox and approval behavior. Child processes remained in
the same mount and PID namespaces. The host-global Codex root was not visible
from the namespace. Do not interpret this local preflight as the AC-14
before/after result from the next real pass.

### Credentials and scanner

The production dotenv parser loaded the read-only repository `.env` in the
namespace and reported presence only:

| Variable | Current preflight |
| --- | --- |
| `AIHUB_GATEWAY_API_KEY` | `PRESENT` |
| `AIHUB_PROVIDER_OPENCODEGO_API_KEY` | `PRESENT` |

The values were not printed, copied, hashed, or measured. Do not rotate,
replace, or duplicate either credential. The live launcher receives no
credential values; the Hub test process loads the existing `.env` through the
production parser. The Codex child must receive only the configured gateway
key through `AIHUB_CLIENT_KEY`; the provider key and Hub-only variables must be
absent from its exact environment. The existing scanner decision remains
`NOT RUN — operator-approved exception`; do not claim the repository is
secret-free. Under the literal SPEC, this leaves the AC-14 no-secret-added
subcondition unverified.

### Scenario allocation and limits

| Scenario | Expected provider-boundary calls |
| --- | ---: |
| Basic generation | 1 |
| Two-turn replay | 2 |
| Sandboxed tool round trip | 1 |
| Authentication rejection | 0 additional |
| Same-response upstream error | 1 |
| Gateway streaming | 1 |
| Identified startup/readiness requests | Up to 3 |
| **Hard maximum** | **9** |

Run the consolidated E2E test exactly once after approval. Total elapsed time
must not exceed 30 minutes; each provider request must not exceed two minutes;
automatic and manual retries remain zero. Build the test binary without live
credentials, then launch it once inside the verified namespace with
`AIHUB_E07_LIVE=1`. Keep all provider credentials out of the outer launcher
environment and all output presence-only/category-only.

Stop before provider traffic if the exact namespace identity or monitored root,
repository read-only mount, `.env` read-only access, private runtime paths,
credential presence, child environment allowlist, sandbox behavior, port, or
integrity baseline cannot be verified. After the first provider call, stop on
credential leakage, any repository/private-state integrity delta, a sandbox or
approval bypass requirement, an unexpected call, budget/time exhaustion, or a
Hub lifecycle failure. Do not retry a failed scenario or restart the pass.

### Acceptance matrix before the proposed pass

| AC | Current disposition | Evidence expected from the proposed pass |
| --- | --- | --- |
| AC-1 | `NOT ESTABLISHED` | Recheck exact process/credential isolation and all literal AC-14 conditions. |
| AC-2 | `PASS` | Preserve prior routing evidence; verify route count in the new pass. |
| AC-3 | `PASS` | Preserve prior auth evidence; confirm rejection has zero provider-call delta. |
| AC-4 | `PASS` | Preserve prior basic-generation evidence. |
| AC-5 | `REAL E2E FAIL; LOCAL CAUSE CORRECTED` | Two Hub requests, prior context present, no `previous_response_id`, marker recalled. |
| AC-6 | `PASS` | Preserve prior sandboxed tool result. |
| AC-7 | `PASS` | Preserve prior same-response error evidence. |
| AC-8 | `PASS` | Recheck gateway-hop streaming; Codex stdout may remain `NOT VERIFIED`. |
| AC-9 | `PASS` | Recheck correlated list and by-ID records. |
| AC-10 | `PASS` | Recheck orderly shutdown, restart, and recovery. |
| AC-11 | `PASS` | Report versions, outcomes, budget, and limitations without payloads or IDs. |
| AC-12 | `PASS` | Preserve the current full local gate evidence. |
| AC-13 | `PASS` | Keep the README support claim absent. |
| AC-14 | `FAIL — literal clause` | Compare the namespace account root and repository categories; the scanner exception still prevents proving the no-secret-added subcondition. |

Historical readiness statement, recorded before the 2026-11-02 task
authorization: the proposed pass could provide new AC-5 evidence but could not
make AC-14 pass under the scanner decision. A new human authorization was
required before that proposed provider pass. The current authorization and
start gate are recorded in the addendum below.

### Reconciled preflight and AC-12 evidence — 2026-11-01

The current working tree adds `scripts/e07-live-supervisor.sh` to the ten
untracked E07 test files listed above. The script is part of the review scope,
not a versioned artifact. No files are staged; no commit or push was made.

The previous preflight failure was caused by remounting the isolated `/tmp`
tmpfs read-only. Codex's nested shell sandbox needs a writable temporary mount
while building its Bubblewrap command. The supervisor now keeps `/tmp` as an
empty writable tmpfs and retains the repository as a read-only bind mount. The
empty namespace `/home` is also writable for sandbox setup; its `.codex` root
remains a separate empty read-only mount. These writable locations are
namespace tmpfs mounts and do not expose host files.

The supervisor now rejects `--supervised` unless the process is the child of
`/usr/bin/timeout` with the exact `--signal=KILL 30m` command line. A negative
check using a 60-minute timeout and the internal marker was rejected before
setup. `--preflight` invokes
`TestE07ReadOnlyDotEnvCredentialPreflight` under its explicit environment
allowlist. That test calls the production dotenv parser on the mounted
read-only `.env` and logs only the presence of the configured gateway and
provider keys. The values are neither logged nor hashed. The preflight does
not enable `AIHUB_E07_LIVE` and does not issue provider-boundary calls.

After these changes, the isolated `--preflight` completed successfully,
including the local Codex shell-policy and resume canaries, mount checks,
credential-presence check, secret-file fingerprint exclusion, and symlink
integrity tests. The complete AC-12 gates passed on the current working tree:
race tests, `go vet`, `staticcheck`, `golangci-lint`, build, formatting, and
`git diff --check`. The direct supervised-mode negative check returned status
2 before repository discovery. All touched or added Go and shell files remain
below the 300-line repository limit.

### Literal AC-14 host-root reconciliation

A distinct host Linux account is not an AC-14 requirement. The current
supervisor starts its 30-minute watchdog before setup, then runs the host
integrity helper outside Bubblewrap both before and after the sandboxed child.
The helper resolves the current UID's home from the host account database and
fingerprints that account's actual `~/.codex` tree. It also discovers and
fingerprints existing project `.env` files. File reads require `O_NOATIME`,
`O_NOFOLLOW`, and stable before/open/after metadata checks. Access time is
checked for stability during each helper read and included in before/after
equality; any observed change fails the integrity comparison. The production
dotenv parser may update access time while reading an environment file, which
must remain a reported failure rather than be excluded from the comparison.
Symlinks in the Codex tree are captured through `lstat` metadata without
dereferencing them; referent contents outside that tree are not covered. The
current host snapshot fails closed on any Codex-tree symlink until its target
can be observed safely. Any project symlink prevents safe `.env` discovery
and fails closed.
Unsupported flags, unreadable entries, or concurrent changes result in
`NOT VERIFIED`/failure.
The before/after digests remain in process memory and the supervisor emits
category outcomes only. The existing owner-only `.env` remains readable by the
operator process without changing its mode, ACL, or contents.

The Bubblewrap mount checks remain complementary evidence: they establish
that the E2E child cannot access the developer home through its namespace
view. They do not replace the host-side comparison. The previously observed
global metadata changes remain historical results with writer
`UNDETERMINED`; ambient app-server processes were never proven responsible.

The local helper tests and supervisor diff must be reviewed before another
real pass. They do not establish a real before/after result. AC-14 remains
`FAIL` until a real pass establishes this host-side comparison, the other
literal integrity categories, and an approved secret scan with no findings.
No separate account, permission change, privileged dotenv broker, or
replacement of the namespace account view is required for the host-side
comparison. The scanner condition remains a separate unresolved unit.

### Original conditional scenario budget — superseded by 2026-11-02 authorization

This section records the readiness condition before the 2026-11-02 user
request. The current authorization and execution gate are recorded in the
addendum below.

After the host-side snapshot helper passes its focused regression checks and
review, and the scanner condition is resolved through its separate approval
unit, the operator may authorize one real E2E evidence pass under this plan.
No distinct host account, privileged dotenv broker, or alternate identity
mapping is required. The pass is limited to the scenarios and allocation
above: at most 9 provider calls, 30 minutes total, 2 minutes per request, and
zero retries. It can establish AC-5 through the AI Hub route and recheck
previously passing scenarios. It cannot complete E07 or make AC-14 pass while
the scanner subcondition remains unverified. This correction does not start or
authorize a new real pass.

### 2026-11-02 authorization and integrity gate disposition

The user-provided AC-14 resolution request authorizes one additional E2E pass
under this plan's existing scenario and budget limits. The authorization is
not consumed: no pass or preflight was started and no provider-boundary call
was made. A local synthetic Docker experiment established a read-only
`noatime` bind view inside a disposable Ubuntu 24.04 container. The equivalent
Bubblewrap remount on the host failed with exit `32` and `permission denied`
even as UID 0 inside the unprivileged user namespace. The supervisor now fails
closed when a host Codex-tree snapshot encounters a symlink whose external
target cannot be observed safely. Do not start the authorized E2E pass until
the host-side target-integrity precondition can be established. The
operator-approved scanner exception remains recorded and unresolved; no
secret-free claim is made.
