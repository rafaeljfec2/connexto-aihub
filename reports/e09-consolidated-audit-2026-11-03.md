# E09 Consolidated Audit and Closure — 2026-11-03

## 1. Executive Summary

**Original audit recommendation: `NEEDS_FIX`. Final disposition after E09-FIX: `CLOSED — APPROVED_WITH_NOTES`.** The original audit recorded four IMPORTANT and six MINOR findings. The follow-up resolved all ten and added regression coverage. The final PostgreSQL-enabled Go runs each recorded 982 passing tests, 6 skipped live/external tests, and no failures, including the full race run. `go vet`, builds, `pnpm check`, and `pnpm build` also passed.

The database tests ran against the development PostgreSQL instance using the repository's isolated per-test databases. Read-only inventories found no pre-existing `aihub_it_*` databases before execution and no test databases left afterward; the application database `ai_hub` was not used for test migrations or fixtures. No paid provider calls were made. Sections 2–14 preserve the original audit snapshot; this closure addendum records the later remediation and is authoritative for final E09 status.

## 2. Scope and Method

The audit covers E09.1, E09.1.1, E09.2, E09.3, E09.3.1, E09.4, and E09.5. It compares the current code and migrations with `docs/specs/E09-resource-management.md`, `docs/adr/ADR-023-resource-management-admission.md`, `docs/specs/E09.4-usage-analytics.md`, the E05/E06 contracts, prior E09 review evidence in AI Memory, and the user-provided consolidated audit criteria.

The review inspected resource admission, administration, concurrency policies and leases, the Responses execution path, provider usage capture and normalization, analytics aggregation, migrations 0003–0005, related tests, route wiring, authorization, and error handling. AI Memory records for E09.1.1, E09.4 Round 2, and E09.5 were read. Targeted queries did not surface standalone E09.2, E09.3, or E09.3.1 session pages; no claims below rely on those unavailable pages as current evidence. Provider units were checked against the public [OpenCode Inference API](https://dev.opencode.ai/v2/docs/console/api/inference) and [OpenRouter usage guidance](https://openrouter.ai/blog/insights/cost-vs-quality-tradeoff-framework-for-agent-models/). The E09.5 scope included no paid provider invocation, so OpenRouter live behavior was assessed from code, fixtures, and documented contracts only.

## 3. Git State

### Initial state recorded by the first audit pass

- Branch: `main`
- HEAD: `e9cb1d2b7a1e328f9dce1960b10a198e28c10b4d` (`feat(analytics): add provider usage analytics endpoints and service integration`)
- Upstream: `origin/main`; the branch was aligned with its upstream.
- Working tree: clean; no tracked changes or untracked files were present before this report was created, as recorded in the first audit pass.

### Resumed audit baseline

- Branch and HEAD were still `main` at `e9cb1d2b7a1e328f9dce1960b10a198e28c10b4d`, equal to `origin/main`.
- No tracked changes were present. This in-progress report was the only untracked file and has been preserved.

The historical E09.5 acceptance entry records an earlier state at `faaba39` with a local diff. The live repository state used for this audit is the later `e9cb1d2` commit, already present on `origin/main`.

### Final state at the time of the original audit

- HEAD and branch remain unchanged.
- This report remains the sole untracked file; no tracked file is modified.
- No implementation files, migrations, or pre-existing reports were modified. No commit, push, PR, or merge was performed.

## 4. E09.1–E09.5 Conformance Matrix at Initial Audit

| Increment | Result | Evidence and remaining limitation |
| --- | --- | --- |
| E09.1 — Resource Admission Core | Implemented; current database verification incomplete | Admission checks lock applicable policies, evaluate every scope, and commit counters and the admission row in one transaction (`internal/resources/pgstore/admit.go:60-150`). Historical E09.1/E09.1.1 acceptance records PostgreSQL-backed scenarios. PostgreSQL tests did not run in this audit because `AIHUB_TEST_DATABASE_URL` is unset. |
| E09.1.1 — Integration Verification | Historically accepted; not re-established against PostgreSQL in this audit | Prior evidence records routing → admission → E05 ordering, 429/503 behavior, idempotency, atomic scope counters, and file-mode behavior. Current database-backed tests were skipped. |
| E09.2 — Resource Administration | Implemented; current database verification incomplete | The PostgreSQL composition wires resource policy administration and admission only in PostgreSQL mode (`cmd/ai-hub/main.go:145-153`). Handler tests exist. Database integration could not be rerun here. |
| E09.3 — Concurrency Management | Implemented; current database verification incomplete | `Acquire` locks enabled policies in a stable order, checks active leases, and inserts a lease transactionally (`internal/concurrency/pgstore/acquire.go:65-128`). The E05 observer releases leases (`internal/concurrency/release_observer.go:14-28`). Historical review remediation records renewal, expiry, release retry, and administration coverage. |
| E09.3.1 — Concurrency Integration Verification | Historically verified; not re-established against PostgreSQL in this audit | Prior evidence records pipeline order, quota preservation on concurrency rejection, two independent pools, lease expiry/renewal, terminal E05 paths, and admin HTTP. The prior two-pool test was not a separate-process test. |
| E09.4 — Usage Analytics | Implemented with open test gaps | Admission analytics filter `allowed = true`; service and SQL layers propagate `provider_id`/`model_id`. The two MINOR filtered-distribution test items from the independent E09.4 review are still absent. |
| E09.5 — Provider Usage Metering | Implemented with IMPORTANT data-integrity findings | Capture, normalization, idempotent storage, and analytics are present. Cancellation can prevent persistence of captured usage. Partial aggregates may expose zero when a metric is unknown; the API contract does not define this case. PostgreSQL end-to-end tests were skipped in this audit. |

### Criterion-level audit evidence

| Area | Evidence | Result | Gap or limit |
| --- | --- | --- | --- |
| E09.1 scopes and atomic counters | internal/resources/pgstore/admit.go:26-36, 63-150 locks applicable policies in stable order and commits counters plus the admission row in one transaction. E09.1.1 scenarios A, D, and E recorded successful multi-scope, idempotency, and rejection checks. | Code path and historical disposable-PostgreSQL evidence agree with ADR-023. | The current run had no AIHUB_TEST_DATABASE_URL; database behavior was not re-executed at this HEAD. |
| E09.1 concurrent admission and quota rejection | internal/resources/pgstore/integration_test.go; internal/gateway/responses_pipeline_e0911_integration_test.go; internal/gateway/resource_admission_handler.go:20-53; internal/gateway/responses_handler.go:148-159. | Historical tests recorded exactly one winner for the remaining quota, 429 before execution, and 503 on admission unavailability. | Current PostgreSQL-backed concurrency and counter assertions were skipped. |
| E09.2 GET/PUT, enable/disable, and usage | internal/gateway/resource_admin_handler.go:100-227; internal/gateway/resource_admin_handler_test.go:169-230. | Handler and unit-test paths cover policy list/upsert, usage, enable/disable, malformed input, and admin credential enforcement. | PostgreSQL service integration was not run in this audit. |
| E09.2 payload/query validation and no-store | internal/gateway/resource_admin_handler.go:89-95, 232-265; internal/gateway/resource_admin_handler_test.go:150-230; internal/access/authorize.go:71-74. | Unknown payload fields/query parameters are rejected; success and error responses use no-store; /api routes use the separate admin credential. | No live server or PostgreSQL mode was started for this audit. |
| E09.2 persistence mode | cmd/ai-hub/main.go:145-152 constructs resource services in the PostgreSQL composition; cmd/ai-hub/gateway_routes.go:50-57 wires optional admin services. | Static wiring preserves file-mode behavior without E09 resource administration routes. | File-mode runtime was not separately launched; evidence is code composition and existing tests. |
| E09.3 scoped limits and atomic acquisition | internal/concurrency/pgstore/acquire.go:27-62, 65-147 locks applicable policies, counts unexpired leases, and inserts within the acquisition transaction. | Code and historical integration scenarios support global, provider, and provider/model limits. | Current PostgreSQL acquisition tests were skipped. |
| E09.3 idempotency, renewal, expiry, and release | internal/concurrency/pgstore/acquire.go:20-62, 149-169; internal/concurrency/service.go:143-184; internal/concurrency/pgstore/e0931_integration_test.go. | Historical tests cover repeated IDs, renewal, expiry cleanup, release idempotency, and stale-peer protection. | The E09.3.1 two-instance test uses independent pools in one process, not separate OS processes. |
| E09.3 terminal lifecycle and rejected admission | internal/concurrency/release_observer.go:14-28; internal/gateway/responses_concurrency_lifecycle_e0931_integration_test.go:48-199. E09.3.1 scenarios B and E01-E07. | Historical coverage includes normal completion, provider failure, opening timeout, client cancel, inactivity, transport failure, and shutdown; a concurrency rejection leaves daily quota unchanged. | Current PostgreSQL lease lifecycle tests were skipped. |
| E09.3 persistence timeout and shutdown behavior | internal/concurrency/release_observer.go:22-26; internal/concurrency/service.go:143-169; internal/gateway/concurrency_renewal.go:11-42; internal/execution/execution_terminal.go:42-45; cmd/ai-hub/main.go:148-155. | Release runs synchronously inside the E05 observer; renewal runs on a goroutine whose stop function waits for completion. Both database calls use context.Background without a deadline. | No blocked-database test exists in the reviewed E09.3 evidence. A stalled pool/query can delay the handler and execution drain; see IMPORTANT-04. |
| E09.4 totals, UTC series, and breakdown semantics | internal/analytics/service.go:20-74; internal/analytics/pgstore/store.go; docs/specs/E09.4-usage-analytics.md:21-30, 52-56. | Admission totals use allowed rows; UTC day filling and separate global/breakdown semantics are implemented. | Current PostgreSQL aggregation was not executed. |
| E09.4 filtered distribution | internal/gateway/analytics_admin_handler.go:120-150; internal/analytics/service.go:49-74; internal/analytics/pgstore/store.go:124-138. | Filter propagation is present in handler, service, and SQL. | The two prior Phase 7 MINOR test gaps remain: no dedicated filtered-distribution HTTP test and no PostgreSQL test for filtered distribution. |
| E09.4 quota/concurrency snapshots, JSON, and empty days | internal/gateway/analytics_admin_handler.go:37-92, 153-190; internal/analytics/service_test.go:60-110; internal/gateway/analytics_admin_handler_test.go:68-113. | Stable JSON fields, gap-filled admissions days, quota utilization, and concurrency snapshots have implementation/unit-test coverage. | Database-backed snapshot semantics were not re-executed; no current finding indicates a contract defect. |
| E09.5 capture, provider fields, and statuses | `internal/metering/capture/capture.go:20-130`; `internal/metering/normalize.go:50-285`; `internal/metering/normalize_test.go:11-64`; `internal/metering/capture/capture_test.go:9-55`. | Fixtures exercise OpenCode Go `response.completed`, OpenRouter `response.done`, status classification, explicit token zero, and final-event selection. Code maps cached input and persists cost only when present. Provider usage documentation cited in §2 defines `usage.cost` in USD; OpenRouter's optional currency is normalized to that unit when omitted (`parseReportedCost`). | Current tests do not assert cached-input persistence or omitted-currency behavior. No live OpenRouter request was made, as required; fixtures cannot establish paid upstream runtime behavior. |
| E09.5 provider-usage analytics summary and distribution | `internal/gateway/analytics_admin_handler.go:193-276`; `internal/analytics/service.go:75-125`; `internal/analytics/pgstore/provider_usage.go:11-160`; `internal/gateway/responses_provider_usage_e095_integration_test.go:139-183`. | Both endpoints accept range and provider/model filters; the service rejects `model_id` without `provider_id`; SQL applies bound filters, groups distribution by provider and provider/model, and sums token values only for `reported` and `partial` rows. The summary daily series is gap-filled; distribution returns observed provider/model/day groups. JSON field names are explicit in the handlers. | The current HTTP integration test covers only the summary endpoint. There is no provider-usage distribution HTTP test or filtered PostgreSQL distribution test. All relevant current database tests were skipped, so filters, group results, status sums, and distribution JSON were not executed. The brief does not define whether distribution groups should be gap-filled. See MINOR-06. |
| E09.5 idempotency and best-effort persistence | `internal/metering/pgstore/store.go:24-57`; `internal/metering/service.go:74-105`; `internal/gateway/responses_handler.go:168-187`. | Insert is idempotent by execution ID; the service logs persistence errors without returning them to the handler; E05 terminal outcomes are returned with nil Go error, so the handler attempts metering when a terminal result includes a capture. Raw prompt/completion fields are absent from migration 0005. | No dedicated current HTTP test injects a persistence failure or verifies trusted usage on a provider-error outcome. PostgreSQL-backed pipeline tests were skipped; cancellation-context risk is IMPORTANT-03 below. |
| E06 compatibility and privacy | docs/specs/E09-resource-management.md:15-16, 52-55; internal/db/migrations/0005_provider_execution_usage.up.sql; internal/metering/pgstore/store.go:30-52. | E09.5 stores normalized usage separately; no E06 schema or provider response contract was changed at this HEAD. No prompt, completion, credential, or raw provider payload column was found. | Static inspection only for database contents; PostgreSQL integration was unavailable. |

## 5. End-to-End Flow Audit

The current handler order is E04 request admission → E08 route selection → E09 concurrency acquire → E09 resource admission → E05 execution → response capture → metering persistence → analytics. Concurrency is acquired before daily resource admission, and the lease is released when E05 publishes its terminal result. A resource rejection releases its already-acquired concurrency lease (`internal/gateway/responses_handler.go:148-159`).

| Scenario | Evidence | Result and gap |
| --- | --- | --- |
| Concurrency rejection occurs before daily quota admission and provider execution | Historical E09.3.1 scenario B; current handler order at `internal/gateway/responses_handler.go:148-159` | Pass in historical integration evidence; PostgreSQL execution not repeated now. |
| Resource limits apply atomically across enabled scopes | Transaction and policy/counter locks in `internal/resources/pgstore/admit.go:82-150`; historical E09.1 integration evidence | Code path is consistent with the ADR; current database test run unavailable. |
| Lease release across E05 terminal outcomes | `internal/concurrency/release_observer.go:14-28`; historical E09.3.1 lifecycle scenarios | Pass in historical integration evidence; current database test run unavailable. |
| Lease release and renewal under a stalled database | `internal/concurrency/release_observer.go:22-26`; `internal/concurrency/service.go:143-169`; `internal/gateway/concurrency_renewal.go:26,39-42`; `internal/execution/execution_terminal.go:42-45` | No request deadline bounds these database calls, release runs synchronously before E05 deregistration, and renewal shutdown waits for its goroutine. This is IMPORTANT-04; no blocked-database test exists. |
| Provider usage is captured without changing forwarded response bytes | `internal/gateway/responses_handler.go:168-187`, `:271-283`; E09.5 fake-provider test | Code observes chunks before forwarding and stores normalized fields only. Its PostgreSQL integration test was skipped now. |
| Usage from an E05 terminal error is considered for persistence | `internal/execution/manager.go:99-100`; `internal/execution/execution_terminal.go:30-45`; `internal/gateway/responses_handler.go:181-187` | E05 encodes completed/failed/canceled/timed-out outcomes in Result and returns a nil Go error after execution starts; when the capture has provider usage, the handler attempts metering for those results. A dedicated test for trustworthy usage on provider-error/read-failure outcomes is absent. |
| Usage survives cancellation after terminal provider usage was captured | `internal/gateway/responses_handler.go:181-187`; `internal/metering/service.go:54-66`; `internal/metering/pgstore/store.go:25-55` | Finding IMPORTANT-03: persistence receives the request context, which can already be canceled after client disconnect, and the insert can fail before storing captured usage. |
| Usage persistence failure leaves a valid HTTP response unchanged | `internal/metering/service.go:74-105`; `internal/gateway/responses_handler.go:181-187` | The service logs and suppresses repository errors, so the handler does not replace the response for a persistence error. No current HTTP-level test injects a failed usage repository; current PostgreSQL pipeline tests were skipped. |
| SSE capture avoids duplicate intermediate usage; analytics separates admission and usage counts | `internal/metering/capture/capture.go:44-130`; `internal/metering/capture/capture_test.go:9-55`; migrations 0003 and 0005; `internal/analytics/pgstore/provider_usage.go:14-23` | The tap selects a final usage event; admissions and execution-usage rows use separate tables and counters. Current PostgreSQL aggregation was not executed. |
| Persisted usage is distinguishable from admissions and execution outcomes | `internal/metering/types.go`; migration 0005; analytics summary counts status classes separately | Separate tables and counters are present. Finding IMPORTANT-02 concerns whether a partial metric with no known values may be exposed as zero. |
| Prompt, completion, credentials, or raw provider payload are persisted | Metering row fields in `internal/db/migrations/0005_provider_execution_usage.up.sql` and normalized insert in `internal/metering/pgstore/store.go` | No raw prompt, completion, or credential field is persisted. Capture bytes remain transient in the stream tap. |

## 6. Persistence Audit

Migrations 0003–0005 and their down files were reviewed statically. Migration 0003 defines scoped resource policies, period counters, and admission IDs; migration 0004 defines concurrency policies and expiring leases; migration 0005 defines one usage row per `execution_id`, nullable token/cost fields, a constrained status, and time/provider/model/status indexes. The counter foreign key cascades on policy deletion; the down files drop dependent tables before their parents. These are additive tables, but compatibility against an existing database and rollback execution were not tested. The migration runner applies each migration transactionally and serializes runners with an advisory lock (`internal/db/migrate.go:52-56`).

No disposable PostgreSQL target was configured for this audit: `AIHUB_TEST_DATABASE_URL` was unset, `pg_isready` reported no response on the default socket, and the Docker CLI was unavailable in this WSL environment. `internal/db/dbtest/dbtest.go` skips opt-in database tests when the URL is unset. Therefore migration application/rollback, transaction behavior, restart recovery, filtered SQL results, idempotency under PostgreSQL, and the E09.5 storage pipeline were not verified in this run. Skipped tests are not counted as passing database evidence.

## 7. Security and Robustness Audit

- `/api/` routes are dispatched through the access policy and require the separate admin API key; production constructs the E09 admin and metering services in PostgreSQL mode (`cmd/ai-hub/gateway_routes.go:25-57`, `cmd/ai-hub/main.go:145-155`, `internal/access/authorize.go:73`).
- Admin JSON success and error responses set `Cache-Control: no-store` (`internal/gateway/resource_admin_handler.go:89-96`).
- SQL values use bound parameters. Dynamic SQL fragments interpolate only positional parameter numbers, while filter values remain arguments (`internal/analytics/pgstore/store.go:124-138`, `internal/analytics/pgstore/provider_usage.go:141-156`).
- The usage table contains execution/provider/model identifiers, metrics, status, source, and timestamp; it has no prompt, response, or credential columns.
- No paid provider calls or live OpenRouter usage capture were performed. No external SCA/SAST scan was run; the security review was code- and configuration-based.
- SQL values are bound parameters, and provider/model filters remain provider-scoped. Error responses use stable local types rather than SQL details.
- The response path has E05 inactivity and cancellation handling. E09.3 release and renewal persistence has no operation deadline, as documented in IMPORTANT-04.
- No CRITICAL or IMPORTANT security-specific vulnerability was confirmed in the inspected paths. The data-integrity and availability findings below remain material to closure.

## 8. Regression and Verification

| Command | Result |
| --- | --- |
| `go test ./... -count=1` | PASS — Go JSON output recorded 903 passing test events and 68 skipped test events. The `dbtest` helper skips PostgreSQL tests when `AIHUB_TEST_DATABASE_URL` is unset; 47 skipped events were in E09 resource/concurrency/analytics/gateway/metering packages. |
| `go test ./... -count=1 -race` | PASS — all reported package runs passed; PostgreSQL-backed tests were skipped for the same unset database URL. |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `pnpm check` | PASS — script runs `go vet ./...` and `go test ./... -count=1 -race`. |
| `pnpm build` | PASS — script runs `go build ./...`. |
| `git diff --check` | PASS for the tracked diff; no tracked files changed. This command does not inspect the untracked report. |
| Audit report whitespace check | PASS — `rtk proxy awk '/[[:blank:]]$/ { print NR ": trailing whitespace"; bad=1 } END { exit bad }' reports/e09-consolidated-audit-2026-11-03.md` returned no matches. |

### Historical test-count reconciliation

- **939** is quoted for E09.3.1 at `d710b2a` in the draft audit; the standalone E09.3.1 session record was not surfaced in the current AI Memory search, so its exact selection and database environment could not be independently re-read.
- **892** is recorded in `reports/e09.4-usage-analytics-delivery-2026-11-03.md` for the earlier E09.4 delivery state; that report does not record whether `AIHUB_TEST_DATABASE_URL` was set.
- **903** is recorded in E09.5 acceptance evidence and reproduced as passing test events on this audit's E09.5-inclusive HEAD with the database URL unset. The current Go JSON run separately records 68 skipped events, including 47 E09 package integration skips.

The counts come from different source snapshots and test selections. The E09.4 report omits its database-test environment, and the E09.3.1 count could not be checked against its original session record. E09.5 also added metering code and tests. The available evidence does not support a precise arithmetic reconciliation; the current run confirms 903 passes and 68 skips but does not establish current PostgreSQL behavior.

## 9. Independent Review

- **SDD Phase 7 senior-code-review-judge, Round 1:** `APPROVED_WITH_NOTES`. The reviewer confirmed IMPORTANT-01, IMPORTANT-03, MINOR-01, and MINOR-02; confirmed the partial-aggregate behavior but required the report to qualify the missing E09.5 specification and semantic ambiguity; and identified IMPORTANT-04 for unbounded concurrency release/renewal persistence. The reviewer also required explicit evidence or limitation statements for usage on provider errors and response behavior after persistence failure.
- **Business-rule-guardian, Round 1:** rejected completion at that checkpoint because audit criteria lacked enough traceable evidence, the named AI Memory record and reread were outstanding, and the untracked report had not been checked separately from `git diff --check`.
- Both prior reviews were read-only. The report was updated with criterion-level evidence, provider-error/persistence-failure limits, database skip counts, and initial/resumed Git state.
- **Fresh final audit judge, Round 1:** `NEEDS_FIX`. It identified missing provider-usage distribution coverage and three editorial inconsistencies (whitespace status, USD source, and “changed files” wording).
- **Fresh final audit judge, Round 2:** `NEEDS_FIX`. The Round 1 report corrections were confirmed, but the required named AI Memory record and report closure were still pending at that point.
- **AI Memory evidence:** created and reread `default/rafaeljfec2-connexto-aihub/sessions/e09-consolidated-audit-2026-11-03.md`. It contains the initial/resumed/final Git state, sources, commands and results, scenario findings, review rounds, limitations, and `NEEDS_FIX` recommendation. The memory read-back returned the expected title and current test/review evidence.
- **Final reviewer revalidation, Round 3:** `APPROVED` by the independent `senior-code-review-judge`. The reviewer reread the named AI Memory page, confirmed the Git/whitespace evidence and report corrections, and found no material residual report issues. This verdict approves the audit report; the E09 closure recommendation remains `NEEDS_FIX`.

## 10. Findings by Severity at Initial Audit

### IMPORTANT-01 — E09 source files violate the 300-line hard limit

The user-provided repository rules prohibit source files above 300 physical lines. The audited E09 source and test files on this HEAD include these files above the limit:

| File | Lines |
| --- | ---: |
| `internal/gateway/responses_pipeline_e0911_integration_test.go` | 452 |
| `internal/gateway/analytics_admin_handler.go` | 383 |
| `internal/resources/pgstore/integration_test.go` | 377 |
| `internal/gateway/responses_handler.go` | 320 |
| `internal/gateway/responses_pipeline_e093_integration_test.go` | 306 |

Additional audited E09 source/test files on this HEAD exceed the 250-line refactoring trigger: `internal/metering/normalize.go` (286), `internal/gateway/resource_admin_handler.go` (280), `internal/execution/manager.go` (257), and `internal/gateway/responses_concurrency_lifecycle_e0931_integration_test.go` (251). This violates the prescribed file-size/refactoring protocol and the stated definition of done. Refactoring is required before closure; it was not performed because the audit explicitly forbids automatic fixes.

### IMPORTANT-02 — Partial usage aggregates can expose zero for unknown metrics

`internal/analytics/pgstore/provider_usage.go:14-16`, `:42-44`, and `:76-78` include `reported` and `partial` rows, sum nullable token fields, and coalesce a NULL sum to `0`. A `partial` record can have a NULL token field; if no included row has a value for that metric, the aggregate returns zero. The accepted E09.5 normalization evidence stores absent token values as nullable fields, and the user audit brief requires avoiding fictitious zeros. There is no local E09.5 SPEC defining whether aggregate zero means measured zero or no known values. The response also reports `partial_executions`, but that count does not identify which metric fields are unknown. Existing tests cover partial normalization but not partial-field aggregate semantics (`internal/metering/normalize_test.go:29-35`; no partial aggregate case exists in `internal/analytics/pgstore/integration_test.go`). This confirmed behavior creates a data-quality risk; the exact public contract needs a project-lead decision before changing response semantics.

### IMPORTANT-03 — Cancellation can discard already captured terminal usage

`responseSink.WriteChunk` observes each chunk before writing it (`internal/gateway/responses_handler.go:271-275`). E05 returns terminal outcomes in a Result and a nil Go error after execution starts (`internal/execution/manager.go:99-100`), so the handler proceeds to metering and passes `r.Context()` (`internal/gateway/responses_handler.go:181-187`). On client disconnect, that request context can already be canceled. The metering service forwards it to PostgreSQL (`internal/metering/service.go:54-66`; `internal/metering/pgstore/store.go:25-55`), causing an insert of previously captured reliable usage to fail. This violates the audit brief's requirement to record reliable provider usage on cancellation/error paths. Best-effort persistence prevents a valid response from being replaced by a persistence error, but it does not guarantee the captured usage is attempted with a usable context.

### IMPORTANT-04 — Unbounded concurrency persistence can delay terminal completion

`internal/concurrency/release_observer.go:22-26` calls `Service.Release` synchronously with `context.Background()`. Release retries transient storage errors and passes the same context to the repository without a deadline (`internal/concurrency/service.go:143-169`). E05 invokes the observer before removing the execution from its active registry (`internal/execution/execution_terminal.go:42-45`), and production composes this observer for PostgreSQL mode (`cmd/ai-hub/main.go:148-155`). The renewal goroutine also calls `RenewLease(context.Background(), ...)`, while its stop function waits for that goroutine (`internal/gateway/concurrency_renewal.go:26,39-42`). If pool acquisition or a database operation stalls, a terminal request can remain blocked, and E05 drain can wait for the execution to leave the registry. No blocked-database deadline test was found. This is an availability and shutdown robustness gap; remediation is outside this audit authorization.

### MINOR-01 — E09.4 filtered-distribution tests remain absent

The prior independent E09.4 review left two MINOR items: a dedicated HTTP distribution test with `provider_id`/`model_id`, and a PostgreSQL integration test for filtered distribution. They remain unresolved. `internal/gateway/analytics_admin_pg_integration_test.go:70-79` exercises distribution without filters, and `internal/analytics/pgstore/integration_test.go` has no filtered distribution case. The handler, service, and SQL filter propagation exist, so this is a verification gap rather than a confirmed filter implementation defect.

### MINOR-02 — Existing E09.4 report is not in English

`reports/e09.4-usage-analytics-delivery-2026-11-03.md` is written in Portuguese, while the current repository instructions require all Markdown artifacts to be in English. This audit did not rewrite the existing report because the authorized scope allows documenting findings only.

### MINOR-03 — README still says routing and quotas are not implemented

`README.md:6-11` says routing and quotas are not implemented, although E08 routing and E09 resource management are present in the current branch. This stale status can mislead readers about the current product scope. Updating the README is outside this audit's documentation-only finding authorization and was not done.

### MINOR-04 — E09.5 provider-error and persistence-failure paths lack dedicated HTTP tests

The service suppresses persistence errors (`internal/metering/service.go:74-105`), and the handler attempts metering for terminal E05 results when execution returns a result without a Go error (`internal/gateway/responses_handler.go:181-187`). No dedicated HTTP test injects a repository failure and asserts the already-valid response remains unchanged, or supplies trusted usage alongside a provider-error terminal result. The relevant PostgreSQL pipeline tests were skipped in this audit. Code behavior is visible by inspection, but these two user-requested paths remain unverified by a dedicated current test.

### MINOR-05 — Cached-token and omitted-currency normalization lack focused assertions

`internal/metering/normalize.go:102-110,171-179,281-285` maps cached-token fields and only stores cost when a cost value exists; it defaults a missing currency to `USD`. The provider documentation defines `usage.cost` in US dollars, so the default follows the provider unit rather than estimating a value. Current normalization tests do not assert cached-token persistence or the omitted-currency case. OpenRouter live behavior was not exercised. This is a test gap, not a confirmed incorrect amount.

### MINOR-06 — Provider-usage distribution has no dedicated HTTP or PostgreSQL test

`ProviderUsageDistribution` accepts provider/model filters, the service validates that `model_id` requires `provider_id`, and the PostgreSQL store groups usage by provider and provider/model (`internal/gateway/analytics_admin_handler.go:231-276`; `internal/analytics/service.go:102-125`; `internal/analytics/pgstore/provider_usage.go:72-160`). The current E09.5 HTTP integration test covers only `/provider-usage`, not `/provider-usage/distribution` (`internal/gateway/responses_provider_usage_e095_integration_test.go:139-183`), and no filtered provider-usage distribution integration test exists. Database-backed tests were skipped in this audit. This is a verification gap; no filter defect was established by inspection.

## 11. Gaps and Limitations at Initial Audit

- PostgreSQL migrations and integration tests could not execute in the current environment; no configured disposable database, responsive local PostgreSQL, or Docker CLI was available.
- OpenRouter paid/live behavior was not exercised, as required by the audit scope.
- E09.4 filtered-distribution tests remain absent.
- Provider-usage distribution grouping, filter behavior, and JSON response were not executed against PostgreSQL; no dedicated HTTP test exists.
- E09.5 partial aggregation semantics need a project decision; cancellation can lose captured usage.
- E09.3 release and renewal persistence lack bounded database contexts.
- The valid-response-on-usage-persistence-failure and usage-on-provider-error paths have implementation evidence but no dedicated current end-to-end test.
- Cached-token and omitted-currency normalization are not directly asserted by current focused tests; USD defaults rely on provider contract semantics.
- AI Memory did not surface standalone E09.2, E09.3, or E09.3.1 session pages in targeted searches; the report uses current repository evidence and labels older results as historical where available.
- E09 source file-size rules are not satisfied.
- Historical acceptance and test counts cannot replace database validation against the current commit and configuration.
- Existing E09.4 and README documentation contain the language and status inconsistencies recorded as MINOR findings.

## 12. Residual Risks at Initial Audit

Until IMPORTANT-01 through IMPORTANT-04 are resolved and verified, the codebase violates the file-size rule, partial usage aggregates have ambiguous zero semantics, cancellation may lose captured usage, and concurrency persistence may delay request completion or shutdown. The lack of a current PostgreSQL run leaves migration and database behavior for E09.1–E09.5 unconfirmed in this audit. Earlier acceptance records remain useful historical evidence, but do not replace the current findings or verification gaps.

## 13. Project-Lead Decisions Required at Initial Audit

1. Authorize a follow-up scope to refactor the oversized E09 files and address cancellation-safe usage persistence and bounded concurrency lease operations.
2. Define whether a partial usage aggregate with no reported value for a metric may return zero or must preserve unknownness; any response-contract change needs separate authorization.
3. Decide whether to close both E09.4 filtered-distribution test gaps and add the missing provider-error/persistence-failure checks before E09 closure.
4. Provide or authorize use of a disposable PostgreSQL test target so migrations 0003–0005 and database integration tests can run against the audited commit.
5. Decide whether to update the stale README status and translate the Portuguese E09.4 delivery report under the English-only repository rule.

## 14. Initial Audit Recommendation

**`NEEDS_FIX`** — E09 is integrated and non-database Go/pnpm verification passes, but it is not ready for closure under the stated requirements. IMPORTANT-01 violates the file-size constraint; IMPORTANT-02 leaves partial aggregate semantics ambiguous; IMPORTANT-03 can lose usage captured before cancellation; IMPORTANT-04 can block terminal completion and shutdown during an unbounded database operation. PostgreSQL-backed verification was not possible. The two E09.4 admission-distribution test gaps, provider-usage distribution test gap, provider-error/persistence-failure test gap, and documentation inconsistencies remain. The report was the only repository artifact changed; no implementation fixes, migration changes, commits, pushes, PRs, or merges were made.

## 15. E09-FIX Closure Addendum — 2026-11-03

The findings in §§10–14 describe the state at the time of the initial audit. The project lead subsequently authorized E09-FIX to resolve all four IMPORTANT and six MINOR findings. The following status supersedes the initial recommendation in §14.

| Finding | Final status | Resolution evidence |
| --- | --- | --- |
| IMPORTANT-01 — Go file-size limit | Resolved | Cohesive response, analytics, pipeline-test, and resource-policy helpers/scenarios were extracted. All changed or added Go files are below 300 lines; the largest is `internal/gateway/analytics_admin_handler.go` at 299 lines. |
| IMPORTANT-02 — Unknown usage metrics exposed as zero | Resolved | Nullable SQL sums and `*int64` analytics fields preserve unknown values as JSON `null`, while explicit zero remains `0`. PostgreSQL integration and HTTP serialization tests cover unknown-only, zero, mixed, partial, and empty cases. |
| IMPORTANT-03 — Usage persistence after client cancellation | Resolved | Metering persistence uses `context.WithoutCancel` with a bounded, configurable timeout. Tests cover canceled request context, timeout, repository failure, terminal provider error, and single persistence. |
| IMPORTANT-04 — Unbounded lease persistence | Resolved | Release retries and renewal use bounded contexts; renewal propagates cancellation. Tests cover retry, timeout, cancellation, and execution drain with a pending release. |
| MINOR-01 — E09.4 filtered admission distribution | Resolved | HTTP and PostgreSQL integration coverage now includes provider/model filters, model-only rejection, and empty results. |
| MINOR-02 — E09.4 delivery report language | Resolved | `reports/e09.4-usage-analytics-delivery-2026-11-03.md` is now in English. |
| MINOR-03 — Stale README status | Resolved | README now reflects E08 routing and E09 resource management in PostgreSQL mode. |
| MINOR-04 — Provider-error and persistence-failure HTTP coverage | Resolved | HTTP tests verify terminal usage on provider error and that persistence failure does not alter the main response. |
| MINOR-05 — Usage normalization assertions | Resolved | Tests assert cached tokens, omitted currency defaulting to USD, absent/null values, explicit zero, and invalid values. No paid provider calls were made. |
| MINOR-06 — Provider-usage distribution coverage | Resolved | HTTP and PostgreSQL tests cover distribution filters, partial rows, `null` versus zero, and empty results. |

### Final validation

The PostgreSQL-enabled runs used `go test -json ./... -count=1 -p 1` and `go test -json ./... -count=1 -race -p 1`. Each reported **982 passed, 6 skipped, 0 failed**. The six skips are live/external E07, catalog-sync, and provider-smoke tests; no E09 PostgreSQL test was skipped. `pnpm check` also passed with PostgreSQL enabled and packages serialized; `go vet ./...`, `go build ./...`, `pnpm build`, `gofmt -d`, and `git diff --check` passed. The six live/external checks were not invoked against providers, and no paid request was made.

The three initial PostgreSQL failures were stale test assumptions, not production regressions: migration idempotence expectations were updated to include version 5; rollback now reverses migration 0005 before 0004–0002 and verifies reapplication through version 5; and the E09.5 no-usage fixture now supplies a valid model. The three focused database tests then passed, followed by both full PostgreSQL-enabled suites.

### Database data-preservation evidence

The configured development PostgreSQL instance was reachable as database `ai_hub`; the DSN and credentials were withheld from logs. Before execution, a read-only catalog query found zero existing databases with the test helper's `aihub_it_` prefix. `internal/db/dbtest.Pool` created isolated databases under that prefix for integration tests and removed them during cleanup. After the focused tests, both full suites, and `pnpm check`, the read-only inventory again found zero `aihub_it_*` databases. The application database was not used for migrations or test fixtures, and no pre-existing database was destroyed.

### Independent review and final disposition

- The final `senior-code-review-judge` follow-up returned `APPROVED_WITH_NOTES`: it found no new defect, marked the PostgreSQL limitation and three test failures resolved, and noted that its own Round 2 shell did not repeat the database suite because the test URL was not present in that agent's environment. The PostgreSQL results above were executed by the authoring session and provided to the reviewer.
- The business-rule review found no unmet E09 rule; its required database evidence and AI Memory record were completed.
- AI Memory session `sessions/E09-FIX-complete-2026-11-03.md` records the increments, tests, reviews, and database cleanup evidence and was reread after update.

**Final E09 disposition: `CLOSED — APPROVED_WITH_NOTES`.** The note is limited to the independent reviewer's inability to repeat the PostgreSQL run in its own process; the complete PostgreSQL-enabled verification passed in the authoring session. No PR or merge is implied by this documentary closure.

### Git disposition

The project lead authorized committing and pushing the E09-FIX changes to `main`. The final closure state is `main` tracking `origin/main`, with the implementation, tests, documentation, and this report committed and pushed; the working tree has no remaining staged, modified, or untracked E09 task files. No PR or merge was created. The exact resulting HEAD is recorded in Git history and the AI Memory session.
