# SPEC E09 — Resource Management

**Status:** Approved — project lead authorized E09.1 Resource Admission Core on 2026-11-03. Later E09 increments require separate authorization.
**Date:** 2026-11-03
**Scope:** Full E09 MVP specification. Only E09.1 implementation is authorized as of 2026-11-03.
**Architecture ADR:** [ADR-023](../adr/ADR-023-resource-management-admission.md) — accepted.

## 1. Intent

Define the smallest Resource Management increment that can enforce durable request limits after E08 selects a provider/model and before E05 starts execution. The MVP must remain content-free, preserve existing execution contracts, survive process restart, and make concurrent admission decisions consistently.

## 2. Findings from the current repository

- The product is a local-first Go modular monolith. PostgreSQL is an optional persistence mode with embedded transactional migrations; the file mode remains available. PostgreSQL configuration already enables the separate `/api/` admin namespace.
- `POST /v1/responses` authenticates and bounds the request in E04, routes it in E08, then calls `execution.Manager.Execute`. That boundary is the Resource Admission integration point. E05 owns execution registration, streaming, cancellation, timeout, terminal arbitration, and shutdown; it has no policy or quota responsibility.
- E06 observes only a terminal summary (`execution_id`, request ID, provider/model, outcome, status, timing). Its schema-v1 record codec rejects non-null usage, and the observer is asynchronous, bounded, best-effort, and at-most-once. It cannot be the quota ledger.
- The OpenCode Go and OpenRouter adapters return the upstream `http.Response` for E05 to stream. Neither parses successful Responses bodies or extracts structured usage. Usage may appear in forwarded `response.completed` data, but today it remains opaque to the Hub. Catalog data is model metadata, not execution consumption evidence.
- The configured gateway identity is one shared static principal; there is no authenticated end-user or tenant identity for per-user quotas. `X-Request-Id` is correlation data, not an idempotency key.
- E08 MVP and C2 PostgreSQL integration validation are accepted. At discovery start, the checkout was clean at `main`, HEAD `2c6e06eeb2702767a114756cedd42bb90f3f8555`.

Relevant code anchors:

- `internal/gateway/responses_handler.go`: calls `routeResponses`, resolves the selected provider, then calls `execution.Manager.Execute`.
- `internal/gateway/responses_routing.go`: performs E08 structural validation, selection, and payload preparation.
- `internal/execution/manager.go` and `internal/execution/types.go`: own lifecycle admission and expose terminal observations without response bytes or usage.
- `internal/providers/client.go` and `internal/providers/openrouter.go`: return successful upstream responses without parsing usage fields.
- `internal/records/service.go`, `internal/records/convert.go`, `internal/records/codec.go`, and `internal/gateway/records_handler.go`: implement bounded best-effort E06 capture and enforce schema-v1 `usage: null`.
- `cmd/ai-hub/main.go`, `cmd/ai-hub/gateway_routes.go`, and `internal/db/migrations/`: compose PostgreSQL, management routes, and embedded migrations.
- `internal/access/access.go`: distinguishes `/api/` administrative authentication from the gateway principal.

## 3. Proposed MVP boundary

### 3.1 Essential

- Count one **admitted Responses execution** after successful E08 selection and before E05 execution starts. This is a Hub admission unit; it is not proof that a provider received a request or that inference completed.
- Configure and enforce one daily UTC request ceiling at each of these scopes: global, provider, and provider/model pair. A request is allowed only when every applicable enabled policy has capacity; an allowed admission increments all applicable period counters atomically.
- Provide authenticated administration to create/update, enable/disable, and inspect policies, plus read usage and current quota state without execution content or credentials.
- Use PostgreSQL as the durable source of truth for policies and period counters. Resource Management is available only in PostgreSQL persistence mode; file-only deployments keep their existing behavior and do not expose E09 administration.
- Reject over-limit requests before `execution.Manager.Execute` with a stable `429 resource_limit_exceeded` error. Reject an admission storage failure with a stable `503 resource_admission_unavailable` error. Neither rejection starts a provider execution.
- Keep all provider/model accounting keys provider-scoped, consistent with E03/E08 identity `(provider_id, model_id)`.

### 3.2 Deferred

- Token quotas and exact input/output token accounting. They require a separately approved usage contract and reliable extraction; request byte length and estimated tokenization are not acceptable substitutes.
- Monetary budgets. No normalized, verified cost value or pricing source is currently consumed by the adapters.
- Per-user or per-tenant limits. The gateway currently exposes a shared client principal, not caller identities.
- Concurrency ceilings. Active execution count is a separate, process-local E05 concern and must not be represented as a period quota.
- Rolling windows, provider-defined billing periods, automatic refunds, retries, fallback, dashboards, and distributed coordination.

### 3.3 Explicitly out of scope

- Changing E06 schema v1 or making its `usage` field non-null.
- Parsing, buffering, or rewriting provider response streams to capture usage in this increment.
- Redis, Kafka, Kubernetes, microservices, or a general-purpose policy framework.
- Interrupting, canceling, or rerouting executions that have already passed admission.

## 4. Capability assessment

| Signal | OpenCode Go adapter | OpenRouter adapter | E09 MVP treatment |
| --- | --- | --- | --- |
| Request accepted by the Hub | Known locally after policy admission | Known locally after policy admission | Count as `admitted_requests`; do not call it a provider request count |
| Provider/model identity | Known execution target | Known execution target | Attribute by provider ID and provider-scoped model ID |
| Input/output tokens | Not parsed from successful response bytes | Not parsed from successful response bytes | Unknown; no token quota |
| Financial cost | No structured cost is extracted or normalized | No structured cost is extracted or normalized | Unknown; no monetary quota |
| Completion/failure | E05 reports terminal outcome | E05 reports terminal outcome | Do not refund an admitted request on provider error, cancellation, timeout, or stream failure |

No live provider calls were made for this discovery. The matrix describes the checked adapter contracts, not undocumented upstream guarantees.

## 5. Request flow and admission semantics

```text
E04 validation/authentication
  → E08 routing and provider/model selection
  → E09 atomic policy check and request-count reservation
  → E05 execution lifecycle
  → selected provider
  → E06 content-free terminal observation (unchanged)
```

- Apply current enabled policies at admission time. A policy change affects later admissions; an already admitted execution continues under its original decision.
- A successful admission consumes one request unit immediately, including when the provider subsequently fails, times out, or the client cancels. This conservative rule avoids refund races and does not claim that provider usage occurred.
- A quota rejection is pre-execution and therefore has no E06 execution record. It may produce a bounded structured log containing scope, period, and reason only.
- If the admission transaction fails, do not call E05. Do not silently bypass an enabled policy when PostgreSQL is unavailable.
- `X-Request-Id` is not used for deduplication. A client retry is a new admission and consumes another unit. The current Hub has no provider retry/fallback path. Any internal retry of the admission transaction must reuse one stable internal admission identifier so an uncertain commit cannot increment twice, as required by [ADR-023](../adr/ADR-023-resource-management-admission.md).

## 6. Minimal domain and persistence model

| Concept | MVP meaning | Needed? |
| --- | --- | --- |
| `ResourceScope` | `global`, `provider`, or `model`; model scope includes both provider ID and model ID | Yes, to define policy identity |
| `ResourcePolicy` | Scope, positive daily request limit, enabled flag, and update timestamp; at most one policy per scope key | Yes |
| `UsagePeriod` | UTC calendar day and the aggregate count for one policy | Yes, to define reset and durable usage |
| `AdmissionDecision` | Allowed or rejected with a stable reason and reset time; internal result, not an E06 execution record | Yes |
| `UsageRecord` | Provider-reported token/cost event | No; such evidence is not currently extracted |

Persist policies and daily counters in additive PostgreSQL tables through the existing embedded migration chain. Check all applicable policies and update all corresponding counters in one transaction. Persist the stable internal admission identifier and decision in the same transaction; retrying that admission after an uncertain commit must return the prior decision without incrementing again. Lock/update policy-period rows in deterministic scope order so concurrent calls cannot both take the last request unit. A rejection or database error must leave every counter unchanged. Exact SQL, identifier-retention policy, and lock strategy belong to implementation design; they must preserve the idempotency guarantee. Serializable transactions or equivalent conditional updates are acceptable if they meet these guarantees.

PostgreSQL remains local-first when run on the operator's machine. The MVP does not claim multi-process quota coordination in file mode or across independent databases.

## 7. HTTP contracts

All administration routes use the existing `/api/` admin authentication and no-store responses. They exist only when PostgreSQL-backed administration is configured.

| Method and path | Purpose |
| --- | --- |
| `GET /api/resource-policies` | List policy definitions and enabled state |
| `PUT /api/resource-policies` | Upsert one policy by `(scope, provider_id, model_id)`; validate scope fields and positive daily request limit |
| `GET /api/resource-usage` | Read daily UTC admitted counts and quota state, filterable by provider/model and bounded period range |

Policy and usage responses contain scope IDs, limits, enabled state, UTC period start/reset, admitted count, and remaining count. They contain no credentials, prompts, responses, provider sessions, or raw upstream errors. Disabling is an explicit policy update; no policy deletion or arbitrary historical correction is in the MVP.

`POST /v1/responses` keeps the current E08 error envelope. E09 adds `429 resource_limit_exceeded` for a full applicable quota and `503 resource_admission_unavailable` when an enabled admission cannot be decided safely. The error exposes a stable type and, when known, the next UTC reset time; it does not expose internal SQL errors or policy secrets.

## 8. Failure and consistency rules

| Case | Required behavior |
| --- | --- |
| Provider retry or fallback | None exists in E05/E08. One successful E09 admission is counted once; no provider switch is allowed. |
| Client retry / repeated request ID | Treat each HTTP request as a new admission. Request ID is correlation only. |
| Provider error, cancellation, timeout, stream failure | Keep the admission counted; do not interrupt or refund it. |
| Failure before admission commit | Roll back and do not start E05. |
| Retry after uncertain commit acknowledgement | Reuse the stable internal admission identifier and return the committed decision without another increment; never use `X-Request-Id` for this purpose. |
| Admission commit succeeds but process stops before E05 starts | The daily unit remains consumed. It is an admitted slot, not verified provider consumption. |
| Usage persistence failure | E09 counters are synchronous and authoritative; return 503 before execution. E06's independent best-effort record loss does not alter quota state. |
| Last unit under concurrent requests | Exactly one transaction succeeds; all other requests receive the limit rejection. |
| Process restart | Re-read policy and counters from PostgreSQL; do not reconstruct quota from E06 records or memory. |
| Policy changes during execution | Apply new state to future admissions only; preserve already admitted work. |
| Provider reports no usage | Continue request-count accounting; token and cost remain unknown. |
| Different provider periods/units | Ignore provider billing periods in MVP; use one UTC calendar day and one Hub admission unit. |

## 9. Verification plan

Automated tests should cover:

- Below-limit admission, exact-limit admission, and deterministic `429` at the next request.
- Global, provider, and provider/model policy intersection and isolation between providers sharing a model ID.
- Disabled and updated policies, period boundary at UTC midnight, and reset-time reporting.
- Concurrent contenders for one remaining unit; assert exactly one succeeds and rejected transactions do not increment other applicable scopes.
- Retry with the same internal admission identifier after a committed transaction whose acknowledgement was lost; assert the decision is returned without a second counter increment.
- Database errors and transaction rollback before execution; assert the provider opener is not called.
- Provider failure, cancellation, and timeout after admission; assert the request unit remains consumed and E05 behavior is unchanged.
- PostgreSQL-backed restart recovery and policy/usage query authentication, bounds, and content-free output.
- Regression compatibility with E05 lifecycle tests, E06 schema-v1 `usage: null`, E08 routing/error behavior, and existing file persistence mode.

Use unit tests for policy validation and API mapping, plus isolated PostgreSQL integration tests for atomicity, concurrency, migration, and restart. A live provider call is not required: deterministic adapter fakes establish that admission precedes the provider opener and that no usage value is fabricated. No tests were run during this discovery-only task.

## 10. Acceptance criteria

- **AC-1:** After successful E08 routing, every applicable enabled daily policy is checked before E05 starts; a rejected or indeterminate admission never invokes the provider.
- **AC-2:** One transaction enforces global, provider, and provider/model request ceilings and increments all applicable counters once on success or none on rejection/error. A retry with the same internal admission identifier returns the committed decision without a second increment.
- **AC-3:** Concurrent requests cannot exceed the final available unit; the result is durable across process restart.
- **AC-4:** One UTC day is the only supported period. The reset boundary and `remaining_requests` are deterministic.
- **AC-5:** Counts mean Hub admissions. Provider failures, cancellation, timeout, and client retries follow §8; no usage is inferred from request/response bytes.
- **AC-6:** Admin policy and usage routes require the existing admin credential and return bounded content-free data.
- **AC-7:** Quota and storage failures map to stable local error types without leaking SQL errors or credentials.
- **AC-8:** E05 streaming/cancellation/shutdown, E06 schema-v1 records, E08 selection, and file-mode behavior remain compatible; E06 `usage` stays `null`.

## 11. Decisions and risks for approval

1. Approve PostgreSQL as a prerequisite for E09, leaving file-only deployments without quota administration/enforcement. This is the smallest existing durable transactional backend and is available for local deployment.
2. Approve the quota unit as an admitted Hub execution, consumed before the provider call and not refunded for later failures. This is measurable and enforceable with current contracts but is not a count of provider-billed requests.
3. Approve one UTC daily period and global/provider/provider-model scope only. The shared gateway principal cannot support per-user limits.
4. Accept that a process stop immediately after admission may consume one unit without starting provider execution. This is safer than undercounting and avoids unreliable refunds.
5. Token, cost, and concurrency policies remain separate future decisions. Token/cost work requires explicit provider evidence and likely a separately approved E06 usage contract; do not relax E06 v1 silently.

The PostgreSQL admission, accounting, and internal idempotency guarantees are recorded in proposed [ADR-023](../adr/ADR-023-resource-management-admission.md). The SPEC and ADR are both pending project-lead approval; neither authorizes implementation by itself.

## 12. Evidence

- **Claim:** E09 request ceilings are enforced atomically before execution and remain correct under concurrency and restart without claiming unknown provider usage.
- **Success type:** `test` and `manual`.
- **Evidence path:** Unit tests prove policy and HTTP behavior; isolated PostgreSQL integration tests prove transaction atomicity, last-unit concurrency, migration, and restart recovery; regression tests prove E05/E06/E08 compatibility. Manual review confirms adapter fakes are used and no paid/live provider call is needed.
- **Budget:** Run the focused resource package tests, resource PostgreSQL integration tests against an isolated disposable database, then `go test ./... -count=1 -race`, `go vet ./...`, and `go build ./...`. No live-provider call is required.
- **Affordance:** None expected; the existing isolated PostgreSQL integration harness and provider opener seam should make the required state observable. If a deterministic concurrent PostgreSQL harness is unavailable, add a test-only fixture rather than weakening AC-3.
