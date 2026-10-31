# Discovery E07 — First Harness Integration

Status: discovery recommendation for developer evaluation. This document is
not an approved SPEC, does not approve architecture decisions, and does not
authorize implementation.

Date: 2026-10-31.

Baseline before authoring this Discovery: `main` at `0d33ca2` (`Feat: E06
Implement execution records service with storage and diagnostics`), equal to
`origin/main`, with a clean working tree. The approved E06 SPEC and its
implementation are present at this HEAD.

## 1. Executive Summary

The strongest next increment is to make Codex the explicitly supported first
harness on the current E06 baseline: publish a reproducible setup path and
revalidate the full Codex → AI Hub → OpenCode Go flow after the E05 execution
lifecycle and E06 records integration.

This recommendation follows the project roadmap and approved delivery
boundaries. The kickoff names E07 “First Harness Integration”; E05 assigns
harness-level validation to E07; and the approved E06 SPEC recommends delivering
E06 before E07. Codex has already worked through the gateway: detailed E02/E04
E2E reports predate E05/E06, and the E05 SPEC records one narrow real Codex
smoke after E05. No available report validates the complete E06 baseline and
records integration end-to-end.

The current code already exposes a Responses-compatible gateway, so this E07
candidate should focus on a supported, repeatable Codex integration and current
baseline evidence. It should not introduce autonomous model selection, quotas,
learning, retries, a second provider, or a general policy engine. Those
capabilities lack one or more necessary inputs and should be sequenced after
their data and contracts are established.

## 2. Current AI Hub State

The repository is a Go 1.23, standard-library-only modular monolith. It has a
local HTTP gateway, one concrete OpenCode Go provider client, a normalized model
catalog, access controls, a coordinated execution manager, and bounded
append-only execution records. Composition and lifecycle are in `cmd/ai-hub`;
domain packages are under `internal/`.

The current request path is:

```text
Harness → authenticated gateway → execution manager → OpenCode Go → response
                                          └──────────────→ records observer
```

The Gateway supplies `provider_id = opencode-go` and currently omits
`model_id` from the execution target. It reads the accepted request body and
passes its bytes through; E02/E05 preserve the provider-ready Responses
payload. The provider client is concrete and only accepts the OpenCode Go
target. It does not route among providers.

E06 stores terminal operational metadata in JSONL and exposes authenticated
query endpoints. Schema v1 always returns `usage: null`; model identity is
`null` on the current path. The records service does not calculate cost or
affect the client response.

The README lists E01–E06 as implemented and states that routing, learning,
quotas, and multi-provider orchestration are not implemented. `docs/architecture.md`
still titles itself as the E01–E04 architecture; the README and current source
are the better descriptions of the post-E06 runtime. Updating that historical
architecture document is outside this Discovery.

## 3. Validated E01–E06 Baseline

| Epic | Repository evidence and current status | Validation boundary |
| --- | --- | --- |
| E01 — Core Foundation | Implemented in the initial `d55ddd0` baseline: bootstrap, configuration, HTTP server, lifecycle, health, logging, and ADR-001–ADR-007. ADR-008 was added with E02. There is no standalone E01 SPEC; the kickoff, architecture document, ADRs, and implementation form its record. | Covered by current package tests. No dedicated E01 validation report exists. |
| E02 — Provider Management | `ad9ce7c` adds the OpenCode Go adapter, provider lifecycle, Responses pass-through, model endpoint, and diagnostics. | `docs/validation/E02-codex-opencode-go-e2e.md` reports real Codex and OpenCode Go scenarios, but labels E02 **PARTIALLY VALIDATED** because progressive rendering inside Codex was not observable. |
| E03 — Model Catalog | `c9444ce` adds the catalog, snapshot persistence, capability claims, sync, and query API. E03 deliberately records provider facts; it does not rank, select, or route. | E04 Discovery records that no standalone E03 live-validation artifact exists. The catalog contract keeps execution verification `unverified` without execution evidence. |
| E04 — Gateway Hardening & Access Control | `3b9015d` adds bearer authentication, listener exposure checks, Host/Origin validation, request admission, and request identity. | `docs/validation/E04-gateway-hardening-access-control-e2e.md` reports E04 validated against the real upstream and Codex on the pre-E05 baseline. Progressive rendering at the Codex output hop was not instrumented. |
| E05 — Execution | `b706aa8` adds one terminal result per execution, cancellation, timeout classification, streaming coordination, and coordinated shutdown. The SPEC explicitly excludes selection, routing, quotas, and retry/fallback behavior. | The E05 SPEC records one real Codex smoke that returned `E05_E2E_OK`; it does not provide the broader scenario coverage of the E02/E04 validation reports. |
| E06 — Execution Records, Results & Usage | `0d33ca2` adds the observer, bounded JSONL store, retention, recovery, diagnostics, and record query routes. The SPEC is approved on 2026-10-31. Usage capture and provider response parsing are explicitly deferred. | Automated tests pass in the current baseline. No E06 real-harness validation report was found. |

The detailed E02 and E04 Codex E2E reports exercised earlier baselines: E02 at
`ad9ce7c`, and E04 at the E03 base plus an uncommitted E04 working tree. The E05
SPEC additionally records a single real Codex smoke against E05, but it
predates E06 and does not report execution-record capture. No available E2E
artifact validates the broader scenario set against the E06 baseline. This is
the material validation gap behind the E07 recommendation.

### Current automated test and coverage snapshot

During this Discovery, the read-only baseline commands completed successfully:
RTK's summary for `go test ./...` reported 601 passing tests across 12
packages; `go test -cover ./...` reported:

| Package | Statement coverage |
| --- | ---: |
| `cmd/ai-hub` | 28.0% |
| `internal/access` | 91.2% |
| `internal/app` | 100.0% |
| `internal/catalog` | 90.6% |
| `internal/config` | 89.8% |
| `internal/execution` | 88.3% |
| `internal/gateway` | 94.4% |
| `internal/health` | 95.0% |
| `internal/observability` | 93.8% |
| `internal/providers` | 92.5% |
| `internal/records` | 84.4% |
| `internal/server` | 86.7% |

These results establish a current automated baseline; they are not live Codex
E2E evidence. E02/E04 provide the detailed real-harness reports, while the E05
SPEC records only its narrow smoke. These command outputs were observed during
this Discovery and are not preserved in a separate repository validation
artifact.

## 4. Existing Capabilities and Their Limits

### Gateway and harness compatibility

The gateway accepts authenticated `POST /v1/responses` and forwards the
provider-ready body to OpenCode Go. The E02/E04 Codex E2E reports demonstrate
basic conversation, multi-turn full replay, tools, upstream errors, and
streaming at the gateway hop; the E05 SPEC records one successful Codex request.
Codex setup currently appears in validation artifacts; the README documents
API examples but not a supported Codex setup.
The repository has no harness-specific adapter or compatibility package.

### Provider and model catalog

There is one configured provider and one concrete client. The catalog keeps
provider-scoped model identity, capabilities, listing state, and raw provider
metadata. Capability claims can remain unverified; pricing data is not
normalized for policy decisions. Catalog presence does not guarantee execution
eligibility, quota, or current availability.

### Execution records and observability

E05 and E06 provide terminal outcomes, duration, request correlation, target
identity when supplied, and bounded local history. Current requests omit the
target model ID, so records cannot currently attribute executions to a model.
Records do not contain usage or cost. There are no rollup endpoints, metrics
export, OpenTelemetry, or quality feedback records. Logs and records provide
local operational evidence but do not constitute a learning system.

### Access, identity, and sessions

E04 has one static gateway principal and request-level admission. The provider
session header is per request; the Hub does not persist conversations or create
durable AI Hub sessions. This is compatible with the observed Codex full-replay
path, but is not evidence that every target harness uses the same context
model.

### Configuration and operation

Settings are environment-based. There is no operator UI, dashboard, CI
pipeline, container image, or background provider health probe. The records
queue can drop events under overload and reports process-local degradation;
history is explicitly at-most-once and best-effort.

Several Go files exceed the project's 300-line source limit. The E04 SPEC
documented these baseline exceptions and identified the catalog files as E03
technical debt. Current examples include `internal/catalog/manager_test.go`
(877 lines), `internal/catalog/store_test.go` (593),
`internal/gateway/catalog_handler_test.go` (575),
`internal/gateway/responses_handler_test.go` (454),
`internal/catalog/capability_test.go` (442), `internal/catalog/manager.go`
(397), `internal/gateway/catalog_handler.go` (386), and
`internal/catalog/store.go` (312). The E04 SPEC explicitly kept E03 catalog
debt out of its scope. These files are recorded here and remain untouched.

## 5. Problems and Opportunities

1. **Current E06 harness compatibility is not established.** E05 changed the
   execution path and E06 added asynchronous observation and shutdown draining
   after the detailed E02/E04 E2E reports. E05 records one successful smoke
   request, but no E06-baseline E2E verifies records capture or the broader
   Codex scenarios.
2. **The first harness path is not reproducibly documented.** The prior E2E
   reports include isolated Codex configuration, but the README has no supported
   harness setup section or single reproduction path.
3. **Model attribution and usage evidence are absent.** The execution target
   omits the model today, and E06 usage is always null. This blocks trustworthy
   per-model usage reporting and weakens future quota, cost, and learning work.
4. **Catalog facts do not yet drive execution.** E03 provides model metadata,
   but there is no eligibility or selection boundary connecting catalog views
   to execution targets.
5. **Provider diversity and harness coverage are narrow.** One provider is
   configured. Codex has historical E2E evidence; other harnesses do not have
   equivalent repository validation.
6. **Operational analytics are limited.** E06 supports bounded record queries,
   but not aggregate trends or exported metrics. No repository evidence
   demonstrates that a dashboard or distributed telemetry system is needed.

## 6. Evidence in Code and Documentation

| Claim | Evidence |
| --- | --- |
| The product roadmap names E07 First Harness Integration and permits changing order as dependencies become clearer. | [`AI-Hub-Product-Architecture-Kickoff.md`](../../AI-Hub-Product-Architecture-Kickoff.md), §§3–6 and 9.1–9.2. |
| E02/E04 have detailed real Codex evidence from earlier baselines; E05 records a narrower smoke; no E2E artifact covers the E06 baseline. | [`E02 validation`](../validation/E02-codex-opencode-go-e2e.md), [`E04 validation`](../validation/E04-gateway-hardening-access-control-e2e.md), [`E05 SPEC`](../specs/E05-execution.md), §14, and Git history for `ad9ce7c`, `3b9015d`, `b706aa8`, and `0d33ca2`. |
| E05 explicitly assigns harness-level validation to E07. | [`E05 SPEC`](../specs/E05-execution.md), §11. |
| Approved E06 places E06 before E07 and keeps records independent from future routing. | [`E06 SPEC`](../specs/E06-execution-records.md), §§3, 6, and 11. |
| The current gateway omits model metadata while forwarding the Responses payload. | [`responses_handler.go`](../../internal/gateway/responses_handler.go), `responsesHandler.ServeHTTP`; [`execution/types.go`](../../internal/execution/types.go), `Target` and `Command`. |
| E06 schema v1 has no usage and allows model ID to be absent. | [`records/record.go`](../../internal/records/record.go), `Record`; [`records/service.go`](../../internal/records/service.go), `OnExecutionFinished`; [`records_handler.go`](../../internal/gateway/records_handler.go), `newExecutionRecordDTO`. |
| Provider and model choices are not made by the current provider manager or catalog query. | [`providers/manager.go`](../../internal/providers/manager.go), `Statuses`; [`catalog/query.go`](../../internal/catalog/query.go), `Manager.List`; [`E03 SPEC`](../specs/E03-model-catalog.md), §§1 and 4.5. |
| E06 drains after execution shutdown and before provider release. | [`cmd/ai-hub/main.go`](../../cmd/ai-hub/main.go), module composition; [`docs/design/E06-execution-records-design.md`](../design/E06-execution-records-design.md), §3.2; [`ADR-018`](../adr/ADR-018-execution-terminal-observation.md). |
| Current product limits include no routing, quotas, learning, metrics export, or multi-provider orchestration. | [`README.md`](../../README.md), current status and Known limitations; [`E05 SPEC`](../specs/E05-execution.md), §3.2; [`E06 SPEC`](../specs/E06-execution-records.md), §3.2. |
| Existing E03 source files exceed the project line limit and are documented as out of scope for E04. | [`E04 SPEC`](../specs/E04-gateway-hardening-access-control.md), §3.3; current line counts checked against the working tree. |

The initial product kickoff is a vision and proposed sequence, not a current
implementation contract. Approved SPECs, current code, and current README
claims take precedence when they differ.

## 7. Architectural and Functional Alternatives

### A. Revalidate and productize Codex as the first harness

Run the real Codex → current AI Hub → OpenCode Go path against the E06 baseline
and publish a minimal, reproducible setup. This extends the detailed E02/E04
E2E and narrow E05 smoke to the current records path, closing the known revision
gap. It creates an explicit support boundary and avoids a new provider or router.
The trade-off is that it primarily consolidates compatibility and operation;
it does not deliver automatic model choice.

### B. Capture model attribution and provider-reported usage

Add structured execution evidence from the requested model and provider
Responses result. A request may contain a model field and the provider can
return usage in Responses output, but the current Gateway does not extract the
requested model and the approved E06 contract intentionally does not inspect
either direction. Any future design must distinguish the requested target from
a provider-reported resolved model. This would unlock better usage history and
later resource management. The trade-offs are provider-specific parsing, record
schema evolution, privacy review, and risk to byte-preserving streaming.

Possible evidence sources include a bounded read-only observer of the existing
Responses stream or a provider account/usage API. The repository establishes
the former data shape for OpenCode Go, but contains no account-usage API
contract. Any external quota API option would need separately verified provider
documentation before selection.

### C. Deterministic model selection and routing

Select a target from explicit preferences and catalog facts, with a reason
visible to the operator. This moves toward the central product goal, but the
current route fixes one provider, omits target model metadata, and does not
normalize price or eligibility. It also needs a request contract for replacing
the model in a provider-ready payload while preserving E02 passthrough for
requests that remain direct.

### D. Add a second provider before routing

Use a second concrete adapter to validate provider boundary reuse before
introducing provider-independent routing. This would expand available
resources, but creates integration and protocol-mapping work before the Hub can
choose between them. Existing E02/E03 intentionally defer a generic abstraction
until a second provider exists; this is a prerequisite if that provider is
selected, not a reason to add a speculative framework in advance.

### E. Start quotas, learning, retries, or a general policy engine

These address real product goals, but current evidence lacks reliable usage or
balance inputs, explicit user quality feedback, task labels, and a multi-target
set. Retrying after response bytes are committed can duplicate provider work
and spend. A general policy engine would precede the concrete decisions it is
supposed to represent. These are not recommended as the next standalone
increment.

## 8. Dependencies and Recommended Sequence

| Capability | Required foundation | Relationship to candidate E07 |
| --- | --- | --- |
| First harness integration | Current E05/E06 compatibility evidence, safe credentials, repeatable harness configuration | Directly aligns with the E05/E06 roadmap boundary and can be completed with the existing provider. |
| Model attribution and provider usage | A read-only request/response observation contract, usage provenance, record schema evolution, streaming and privacy guarantees | Valuable follow-up; not required to prove Codex compatibility. |
| Deterministic selection/routing | Resolved target contract, available model set, mapping from model to provider, explicit user preferences, auditable decision reasons | Should follow clear target and eligibility contracts; selection must not silently violate direct passthrough behavior. |
| Resource management | Trusted usage or quota source, reset periods, allowance semantics, and user limit policy | Depends on data and provider contract work; it constrains automatic routing. |
| Learning and evaluation | Task/result labels, explicit quality signals, enough observations, privacy and retention rules | Depends on better attribution and evaluation signals; execution success alone is not quality. |
| Multi-provider routing | At least two supported provider adapters and consistent target/capability mapping | Add a second adapter before claiming provider-independent routing. |
| Durable context | A defined session ownership and privacy model plus harness-specific protocol evidence | Independent of the first Codex HTTP integration; avoid adding persistence without a demonstrated need. |

The approved E05 SPEC states that E09 eligibility/limits constrain automatic
E08 selection, while E06 records results and E07 performs harness-level
validation. The initial kickoff roadmap is explicitly provisional; this
dependency is the stronger current sequencing evidence.

## 9. Risks and Trade-offs

| Risk | Impact | Discovery treatment |
| --- | --- | --- |
| E2E evidence is mistaken for current E05/E06 validation | Compatibility or shutdown defects may go unnoticed | Re-run the harness path against the actual current revision and report each scenario separately. |
| Codex hides progressive SSE rendering details | A full user-visible streaming claim cannot be proven from the CLI output alone | Measure progressive delivery at the gateway hop; label CLI rendering as unverified when the harness exposes no instrumentation. |
| Harness-specific behavior is confused with provider behavior | Wrong support claims or unnecessary protocol translation | Keep the Codex harness contract distinct from the OpenCode Go provider adapter. |
| Credentials or generated content leak during validation | Security and privacy impact | Use isolated configuration and temporary secrets; do not store tokens, prompts, responses, or real credentials in the repository. |
| E06 records are best-effort | A completed harness request may not have a retained record after overflow, write failure, or crash | Validate normal capture and accurately report degraded/loss semantics; do not claim exactly-once history. |
| Usage is assumed from a successful response | Cost/quota decisions could be wrong | Keep `usage: null` semantics explicit until a separately approved contract captures provider-reported values and provenance. |
| Routing or retries are added during harness work | Could change payloads, choose an unapproved model, duplicate provider charges, or alter committed streams | Keep E07 candidate scope to compatibility and setup; defer selection and retries. |
| Historical docs differ from the current baseline | Discovery could repeat stale plans as current truth | Prefer current code, approved SPECs, README, and commit history; list document drift as debt, not a reason to rewrite historical artifacts. |

## 10. Prioritized Opportunities

Prioritization weighs demonstrated need, alignment with approved boundaries,
readiness from current code, dependencies, and risk.

| Priority | Opportunity | Assessment |
| --- | --- | --- |
| 1 | Codex first-harness integration on the E06 baseline | Recommended E07. It is explicitly assigned by E05 and closes a real gap between the current HEAD and the older E2E baselines. Scope can remain local and bounded. |
| 2 | Model attribution and provider-reported usage evidence | Strong next foundation for cost/usage reporting and later resource policies; requires a separate streaming, privacy, and schema decision. |
| 3 | Second provider adapter | Necessary before true provider-independent routing; should be justified by a target provider and protocol evidence. |
| 4 | Deterministic model selection and routing | High product value, but currently lacks stable model attribution, normalized eligibility inputs, and an approved selection contract. |
| 5 | Quotas, limits, and cost management | Important free-first control, but currently lacks authoritative usage/allowance inputs and defined billing semantics. |
| 6 | Quality evaluation and learning | A strategic goal, but no task labels or explicit quality feedback currently enter the Hub. |
| 7 | Aggregate analytics, dashboard, and configuration UX | No documented operational pain establishes that a dashboard or metrics stack is the next constraint; existing local records and environment configuration remain usable. |
| 8 | Retry/fallback, generalized policy engine, durable context, additional harnesses | Defer until target, eligibility, idempotency, privacy, and harness-specific needs are supported by evidence. |

## 11. Evidence-Based E07 Recommendation

Recommend E07 as **Codex First Harness Integration on the current E06 baseline**.

This preserves the original E07 intent while adapting it to the actual project
state. Codex already reached OpenCode Go through the gateway in earlier real
E2E work, and the E05 SPEC records one successful real request, so E07 should
not claim that no harness integration exists. Instead, it should make that
path reproducible and validate the committed records lifecycle and broader
scenario set not covered by those earlier artifacts.

The recommendation is deliberately narrower than intelligent orchestration.
It creates a dependable harness boundary and current end-to-end evidence before
the project adds policies or autonomous choices that depend on reliable model,
usage, and provider data.

## 12. Candidate Scope and Explicit Non-Goals

### Candidate scope

- Treat Codex using the Responses API as the first supported harness; use the
  existing OpenCode Go adapter as the provider.
- Provide one reproducible, authenticated setup path that does not require
  committing secrets or modifying global user configuration.
- Validate the present E05/E06 route end-to-end: basic responses, multi-turn
  full replay, tool round trip, upstream error handling, gateway authentication,
  streaming at the observable gateway hop, execution record capture, and
  shutdown/record drain where the harness flow can exercise it.
- Publish scenario-level evidence and distinguish PASS, NOT VERIFIED, and
  environment/provider limitations.
- Describe the current semantics honestly: no durable Hub conversation,
  `model_id` may be null, `usage` is null, and records are at-most-once.

### Explicit non-goals

- A second harness or provider, generic provider framework, or protocol
  translation layer.
- Task classification, model selection, ranking, routing, or automatic
  fallback/retry.
- Usage parsing, local cost calculation, quotas, allowance enforcement, or
  billing truth.
- Durable sessions, conversation storage, quality evaluation, learning,
  dashboard, metrics export, or OpenTelemetry.
- Changes to E01–E06 behavior, APIs, ADRs, or historical documents.

## 13. Success and Validation Criteria

The future E07 delivery should be considered successful only when:

1. A developer can follow the documented setup from an isolated configuration
   and connect Codex to the authenticated local AI Hub without placing secrets
   in tracked files or editing global Codex settings.
2. Real end-to-end scenarios exercise the current committed E05/E06 baseline,
   not an earlier E02/E04 checkout. Each scenario records its revision,
   environment class, result, and limitation.
3. Basic generation, multi-turn replay, and a tool round trip succeed through
   AI Hub. Upstream errors and authentication behavior remain consistent with
   the existing gateway contract.
4. Streaming is measured at the Hub boundary. Codex-side progressive rendering
   is claimed only if that harness version exposes measurable evidence.
5. Finished executions produce queryable records under normal operation, and
   the test evidence does not imply records are guaranteed after queue overflow,
   storage failure, or process crash.
6. No request/response content, provider key, gateway key, or generated
   session identifier is published in the validation artifact.
7. Existing automated tests remain green; E07 adds only the tests or opt-in
   validation needed for its accepted harness scenarios.

For Discovery itself, success means all current-state claims link to source,
historical validation is not presented as current E05/E06 evidence, alternatives
are compared, and the recommendation remains a candidate rather than an
approved SPEC.

## 14. Questions Requiring Developer Decision

1. Is Codex the first harness to support formally, consistent with the existing
   real E2E evidence and original roadmap? **Recommendation: yes.**
2. Should “support” in E07 include a maintained README setup path in addition
   to current-baseline E2E evidence? **Recommendation: yes.**
3. Is gateway-hop streaming evidence sufficient when Codex CLI does not expose
   progressive rendering? **Recommendation: accept the explicit limitation.**
4. Should the E07 acceptance scope validate only normal record capture and
   shutdown drain, while preserving E06's `usage: null` and best-effort
   guarantees? **Recommendation: yes.**
5. Should model attribution and usage capture be considered a separate
   follow-up Discovery/SPEC after E07, rather than being folded into harness
   validation? **Recommendation: yes.**

## 15. Recommended Next Steps

1. Review this Discovery and decide whether the recommended Codex integration
   is the next approved epic.
2. If accepted, write a separate E07 SPEC that freezes the supported Codex
   scenarios, reproducible configuration, security/privacy constraints, and
   observable end-to-end evidence.
3. Review and approve that SPEC before designing or implementing E07, following
   the project SDD workflow.
4. Keep model attribution/usage evidence, second-provider integration, routing,
   resource management, and learning as separately sequenced opportunities.
5. Do not create ADRs or modify protected baseline artifacts as part of this
   Discovery.
