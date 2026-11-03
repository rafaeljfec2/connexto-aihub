# AI Hub — Product and Technical Kickoff

## Product vision, architecture, and implementation status

**Document status:** Initial product baseline updated with the verified project status
**Baseline version:** 0.1
**Status date:** 2026-11-02
**Implementation strategy:** Gateway First
**First harness:** OpenAI Codex
**First provider:** OpenCode Go
**Backend:** Go
**Deployment direction:** Local-first
**Repository snapshot:** branch main, commit 5af543b, aligned with origin/main and clean when reviewed

------------------------------------------------------------------------

# Current Project Status

**Status date:** 2026-11-03. This section replaces the earlier status snapshot.
The original product and architecture plans below remain historical context;
their imperative language is not current work authorization. Current behavior
and approved scope are defined by the linked specifications, ADRs, and
validation reports.

## Delivered

- **E01–E07 — Backend foundation and harness:** The Go modular monolith,
  provider gateway, model catalog, access control, execution lifecycle,
  content-free execution records, and validation-only Codex harness are
  implemented. E07 is accepted for Codex CLI 0.164.0-alpha.11.1 on Linux
  amd64; AC-14 remains out of scope.
- **Phase B — PostgreSQL foundation:** PostgreSQL persistence, embedded
  migrations, and recovery are implemented and validated through isolated
  integration-test databases.
- **C1/C2 — Provider administration and multi-provider runtime:** PostgreSQL
  configuration, sealed provider credentials, runtime administration, provider
  diagnostics, and provider-ID-bound OpenCode Go/OpenRouter runtimes are
  implemented. C2 PostgreSQL integration validation is accepted. Operational
  evidence includes an OpenCode Go stream and one authorized OpenRouter
  Responses stream; see the C2 reports for limits and details.
- **E08 — Intelligent Routing:** E08.1–E08.5 are accepted. Routing remains
  deterministic, explainable, and based on explicit selections and trusted
  catalog/runtime eligibility. E08.6–E08.8 are deferred and unauthorized.
- **E09 — Resource Management:** Closed and published at
  9338c538eed923c380ed2b9197b8ac6522d946ee. The final PostgreSQL-enabled
  verification recorded 982 passing tests, 6 skipped live/external tests, and
  no failures. The E09 test harness used isolated per-test databases; it did
  not use or remove data from the existing ai_hub application database.

See the [README](README.md), [consolidated roadmap](docs/roadmap/AI-HUB-ROADMAP.md),
[runtime architecture](docs/architecture.md), [E01–E07 delivery history](docs/validation/E01-E07-execution-history.md),
and the individual specifications and reports for authoritative evidence. The
older [E07 readiness snapshot](docs/discovery/E07-final-acceptance-readiness.md)
is historical.

## Next authorized development front

The **Frontend MVP** is the next development front. This status update only
formalizes the direction; no frontend app or UI has been implemented. The
frontend MVP is distinct from **E11 Dashboard**, a future evolution whose scope
is not yet defined. See the [Frontend MVP preparation](docs/frontend/FRONTEND-MVP.md).

## Deferred and future work

- **E08.6–E08.8:** Deferred and not authorized. No LLM classifier is an MVP
  requirement.
- **E10 — Learning Engine:** Future work. Real-user signals are not yet
  sufficient to justify automated learning.
- **E11 — Dashboard:** Future work, separate from the smaller Frontend MVP.

## Repository snapshot

The inspected starting checkout was on main at
9338c538eed923c380ed2b9197b8ac6522d946ee, aligned with origin/main and clean.
The original kickoff below records the proposed baseline at its creation; it
does not override this current status.

------------------------------------------------------------------------

# 1. Initial Development-Agent Guidance

This section translates the original kickoff guidance. Its initial implementation goals are historical context; current approved specifications and the status above take precedence.

## 1.1 Original immediate objective

The original objective was to begin **E01 — Core Foundation** and **E02 — Provider Management** using a Gateway First strategy.

The first expected technical result was a local Go application able to:

1. Start reliably.
2. Load and validate configuration.
3. Expose a local HTTP server.
4. Provide health checks.
5. Record structured logs and correlation IDs.
6. Configure the OpenCode Go provider.
7. Protect provider credentials.
8. Check connectivity.
9. Query available models when supported.
10. Run a test request with a compatible model and protocol.

The original kickoff explicitly deferred a complete intelligent routing engine, learning, dashboard, and support for every harness. Its immediate aim was to prove the foundation and real connectivity.

## 1.2 Original working principles

- Inspect the existing repository before coding; do not assume it is empty.
- Compare existing decisions, code, and documentation with the kickoff before replacing anything.
- Work incrementally and use logical commits when the environment permits.
- Add automated tests from the start.
- Avoid speculative abstractions without a demonstrated need.
- Keep the initial deployment a modular monolith; do not introduce microservices.
- Do not add Redis, Kafka, Kubernetes, or distributed infrastructure without a proven requirement.
- Never store secrets in code, versioned files, logs, or HTTP responses.
- Do not claim an integration works without real provider validation when credentials are available.
- Distinguish implemented, simulated, estimated, and unverified capabilities.
- When a decision is unspecified, choose a simple option that preserves future growth and record the decision.
- Update documentation and acceptance criteria as implementation evidence changes the requirements.
- Do not anticipate every later epic.

## 1.3 Original incremental process

For each increment:

1. Inspect the current repository.
2. Identify the exact scope.
3. Record relevant questions and risks.
4. Propose the smallest implementation.
5. Define or refine observable acceptance criteria.
6. Implement.
7. Run applicable unit, integration, and static checks.
8. Manually validate the flow when needed.
9. Update documentation, backlog, and architecture decisions.
10. Report completed work, remaining work, and dependencies.

------------------------------------------------------------------------

# 2. Product Vision

## 2.1 What AI Hub is

AI Hub is a Personal AI Orchestrator: a personal platform that centralizes access to and management of AI models, providers, subscriptions, usage allowances, and development environments.

Users connect their harnesses to a gateway they control. AI Hub manages communication with providers and may progressively decide which model to use for each task based on quality, capabilities, availability, limits, costs, and user-defined policies.

The product is intended to become more than an LLM proxy. It is an orchestration layer for the AI resources available to a user.

**Core idea:** users should not have to administer models one by one; they should be able to administer a system that manages those models for them.

## 2.2 Problem to solve

Users rely on tools and services with separate models, credentials, limits, credits, allowances, and integration methods. Priority ecosystems include OpenCode, OpenCode Go, OpenRouter, OpenAI Codex, Claude / Claude Code, and Cursor.

The product aims to reduce the need to:

- Select a model manually for every task.
- Switch among provider settings in many places.
- Monitor limits across multiple dashboards.
- Spend paid credits when suitable free resources or allowances are available.
- Reconfigure every harness whenever a provider is added.
- Use premium models for simple tasks that a free model can handle adequately.

## 2.3 Value proposition

AI Hub is intended to provide:

- One connection point for multiple harnesses.
- Independence from a single provider.
- A central catalog of models and capabilities.
- Model selection based on task and policy.
- Priority for suitable free models.
- Selective use of premium resources and subscription allowances.
- Centralized limits, costs, and usage management.
- Learning from execution outcomes.
- Transparency about decisions.

> One connection. Every model. Intelligent execution.

------------------------------------------------------------------------

# 3. Product Principles

## 3.1 User-driven

The user defines policies, permissions, limits, and autonomy. The system should not impose a single operating mode.

## 3.2 Configurable autonomy

The intended autonomy levels are:

1. **Manual:** the user selects the model and provider.
2. **Assisted:** the system recommends, and the user confirms.
3. **Semi-autonomous:** the system chooses within granted permissions and requests confirmation for configured conditions.
4. **Autonomous:** the system decides and executes within authorized policies.

Autonomy is independent from financial and security policies. Even autonomous operation must respect explicit limits, permissions, and restrictions.

## 3.3 Free-first

The intended default is to prioritize the best suitable free model for each task. The goal is not merely to choose the cheapest model, but the best available option within the user's policies and resources.

## 3.4 Quality-driven

Real quality on the user's tasks should guide decisions. External rankings and benchmarks may inform decisions but do not replace local evaluation.

## 3.5 Selective premium use

Users may permit premium resources or allowances associated with services such as Cursor, Codex, and Claude. Their use should depend on configurable policies and evidence of meaningful benefit.

A subscription must not be assumed to permit API use through an external gateway. Technical and contractual integration constraints must be checked for each product.

## 3.6 Learning-first

The system should learn from execution results using available evidence, including task type and complexity, provider and model, duration, reported usage and cost, completion or failure, retries and fallbacks, user evaluation, execution context, and capabilities used.

The system must not infer model superiority from a small sample or weak signals. Evaluation should account for the quantity and quality of evidence.

## 3.7 Provider-agnostic

The central domain should not depend on one vendor. Provider integrations should be isolated in adapters where the architecture and evidence warrant them.

## 3.8 Harness-agnostic

The core should support different harnesses and protocols while preserving the features available to each integration.

## 3.9 Cost-aware

Every decision should respect the user's financial policies. Unauthorized spending must be blocked, including in autonomous mode.

## 3.10 Transparent

Users should be able to understand why a model was selected, which policies applied, and what information supported the decision.

------------------------------------------------------------------------

# 4. Harnesses and Providers

The product distinguishes four concepts:

- **Harness:** the environment that performs work and interacts with the user.
- **Provider:** the service that exposes access to models.
- **Model:** the language model actually used.
- **Gateway:** the central entry point and orchestration layer.

Conceptual flow:

    Codex (Harness)
        |
        v
    AI Hub (Gateway)
        |
        v
    Decision Engine
        |
        v
    OpenCode Go (Provider)
        |
        v
    DeepSeek (Model)

## 4.1 Priority harnesses

The initial target harnesses are OpenCode, Codex, Claude / Claude Code, and Cursor. Implementation order may change based on feasibility and compatibility.

## 4.2 Priority providers

The initial provider and ecosystem candidates are OpenCode Go, OpenRouter, Anthropic, OpenAI, and other providers added later.

A harness name must not be confused with the provider it uses. OpenCode can act as a harness while OpenCode Go is a model provider.

## 4.3 Integration limits

Not every harness accepts an external endpoint or permits a subscription allowance to be used outside its product. Verify official documentation and test each integration. Do not assume that:

- Every subscription exposes a public API.
- Every allowance can be used through an external gateway.
- Every provider supports the same protocols.
- Every model from one provider supports the same capabilities.
- Generic OpenAI compatibility preserves harness-specific features automatically.

------------------------------------------------------------------------

# 5. Operating Modes

## 5.1 Manual

The user explicitly selects the model and provider. AI Hub may still centralize policies, limits, consumption, costs, and records.

## 5.2 Assisted

The system analyzes a request and recommends an option. The user confirms before execution when configured.

## 5.3 Semi-autonomous

The system selects and executes within granted permissions, but requests confirmation for configured cases such as paid usage or exceeding a limit.

## 5.4 Autonomous

The system classifies a task, chooses a resource, executes, monitors, and performs a fallback when allowed. Every decision remains constrained by user policy.

------------------------------------------------------------------------

# 6. Product Intelligence

Product intelligence is intended to evolve progressively.

## 6.1 Task Intelligence

Task Intelligence identifies the nature and complexity of a task. Initial categories may include Plan, Coding, Debug, Review, Research, Chat, Documentation, and Fast / Simple tasks. The list may evolve with use.

## 6.2 Resource Intelligence

Resource Intelligence maintains knowledge of providers, available models, capabilities and protocols, availability, limits and allowances, costs, speed and latency, and historical quality evidence.

Each value should have clear provenance: confirmed by API, configured manually, estimated, or potentially stale.

## 6.3 Decision Engine

The Decision Engine combines task type and complexity, required capabilities, historical quality, availability, user policies, allowances and limits, permitted cost, and autonomy preferences. It should produce a justifiable and traceable execution decision.

## 6.4 Hybrid routing

The intended strategy is hybrid:

1. Apply deterministic rules when they are sufficient.
2. Use an LLM classifier when the request is ambiguous or requires interpretation.
3. Combine classification with resources, evaluations, and user policy in the Decision Engine.

An LLM classifier must never override security policies or financial limits.

## 6.5 Learning Engine

The system should record outcomes and improve decisions over time. Possible signals include success or failure, explicit user feedback, response time, consumption, result quality, corrections or retries, and model suitability for the task.

The evaluation and learning mechanism must be specified in its own approved work. It is not part of the delivered E01–E07 runtime.

------------------------------------------------------------------------

# 7. High-Level Architecture

## 7.1 Initial architectural direction

| Topic | Initial direction |
| --- | --- |
| Deployment | Local-first |
| Architecture | Modular monolith |
| Backend | Go |
| Dashboard | React + Vite, planned |
| Integration | Compatible protocols plus provider-specific adapters |
| Routing | Hybrid rules and LLM when needed, planned |
| Execution | Synchronous or asynchronous based on task |
| Persistence | To be selected in detailed design |
| Observability | OpenTelemetry was an initial direction |
| Streaming | Preserve streaming when supported |
| Distribution | Local application; Docker or binary to be decided |

The implemented service remains a local-first Go modular monolith. The initial OpenTelemetry direction is not a claim that OpenTelemetry instrumentation is implemented. Current package and lifecycle details are maintained in [architecture.md](docs/architecture.md).

## 7.2 Local-first

AI Hub is intended to run primarily on the user's machine, with locally protected credentials, a local gateway and dashboard, local persistence, direct communication with external providers, and no mandatory dependency on the product's own cloud infrastructure.

Local-first does not mean local-only. Backup, synchronization, and remote access may be considered later, but are not initial requirements.

## 7.3 Modular monolith

The application is one Go system with explicit internal module boundaries. Microservices should not be introduced prematurely.

## 7.4 Conceptual layers

### Integration Layer

Receives requests from harnesses and exposes compatible protocols. Its responsibilities include HTTP APIs, protocol adapters, gateway authentication, request normalization, and streaming.

### Orchestration Layer

Coordinates request and execution lifecycles. Future responsibilities may include a Request Manager, Execution Manager, session context, and fallback. The current execution lifecycle does not make routing decisions or persist conversations.

### Intelligence Layer

Understands tasks, makes decisions, and learns. Planned components include a Task Classifier, Decision Engine, Model Evaluator, and Learning Engine.

### Resource Layer

Manages providers, models, and policies. Planned responsibilities include provider and model registries, quota management, policy evaluation, and provider adapters.

### Infrastructure Layer

Provides cross-cutting support such as storage, scheduling, event processing, telemetry, and secrets.

The dashboard is intended to consume system APIs and should not duplicate business rules.

## 7.5 Future conceptual flow

    Harness
       |
       v
    Integration Layer
       |
       v
    Orchestration Layer
       |
       +----> Task Intelligence
       |
       +----> Decision Engine
       |          |
       |          +----> Policy Engine
       |          |
       |          +----> Resource Manager
       |
       v
    Execution
       |
       v
    Provider Adapter
       |
       v
    LLM Provider
       |
       v
    Response / Stream
       |
       +----> Usage Tracking
       |
       +----> Evaluation
       |
       +----> Learning Engine

This is the product's target flow, not the current runtime. The delivered increments implement the gateway, provider route, execution lifecycle, and records needed for connectivity and observability.

------------------------------------------------------------------------

# 8. Execution, State, and Persistence

## 8.1 Intended execution modes

| Operation | Intended strategy |
| --- | --- |
| Chat | Synchronous with streaming |
| Coding | Synchronous with streaming when supported |
| Classification | Synchronous on demand |
| Benchmarking | Asynchronous |
| Result evaluation | Asynchronous |
| Learning | Asynchronous |
| Catalog refresh | Asynchronous |
| Reports | Asynchronous |

## 8.2 Gateway and persisted state

The gateway should not depend on process memory to recover an execution or session. This does not mean the entire system is stateless: configuration, history, policies, metrics, and evaluations may require persistence.

Current execution is in-process. AI Hub does not provide durable conversation state or cross-turn session continuity in E01–E07.

## 8.3 Current persistence

The initial baseline left persistence technology open. Current implementations use a versioned JSON snapshot for the model catalog and append-only JSONL records for executions. Records are bounded by retention and storage limits. No database has been adopted.

Catalog state is stored under AIHUB_DATA_DIR. Execution records contain operational metadata only. Schema-v1 usage is null; records are best-effort and at-most-once, so they do not claim complete billing history.

------------------------------------------------------------------------

# 9. Gateway-First Implementation Plan

Gateway First prioritizes reliable request ingress and provider communication. Intelligent routing is intended to build on that core later.

## 9.1 Original kickoff epic status (historical)

This table records the proposed status when the original roadmap was written.
For the current implementation state, use [Current Project Status](#current-project-status)
and the [consolidated roadmap](docs/roadmap/AI-HUB-ROADMAP.md).

| Epic | Name | Current status |
| --- | --- | --- |
| E01 | Core Foundation | Implemented; no standalone E01 specification or separate E01 E2E report. |
| E02 | Provider Management | Implemented for OpenCode Go. Its original real-provider report remains partially validated with stated limitations. |
| E03 | Model Catalog | Implemented; automated contract coverage and real catalog observation during E04 validation. |
| E04 | API Gateway and access control | Implemented and validated in the documented local E2E scope. |
| E05 | Execution and streaming | Implemented; lifecycle, error, timeout, cancellation, and shutdown contracts are specified and tested. |
| E06 | Execution records | Implemented; content-free bounded records, query API, and recovery. Usage values are not captured. |
| E07 | First harness integration | Complete for AC-1–AC-13 in the validated Codex CLI and Linux environment; AC-14 out of scope. |
| E08 | Intelligent Routing | Not implemented. |
| E09 | Resource Management | Not implemented. |
| E10 | Learning Engine | Not implemented. |
| E11 | Dashboard | Not implemented. |

The original roadmap may change as dependencies become clearer. Current implementation order and scope are governed by approved specifications and repository records.

## 9.2 Initial milestones

| Milestone | Intended result | Current state |
| --- | --- | --- |
| M1 — Connectivity | Gateway communicates with a provider | Delivered through E01–E02 and subsequent gateway work. |
| M2 — Compatibility | A harness uses the gateway | Codex CLI path validated within E07's stated environment and support boundary. |
| M3 — Reliability | Streaming, errors, traceability, and tests are established | Core runtime and E07 criteria are covered; usage accounting, cross-harness support, and product routing remain future work. |

## 9.3 Original first increment

The initial integration target was:

    Codex
       |
       v
    Local AI Hub
       |
       v
    Provider Adapter
       |
       v
    OpenCode Go
       |
       v
    Compatible model
       |
       v
    Response to Codex

The first selected combination was OpenAI Codex as harness and OpenCode Go as provider, with a DeepSeek model as a candidate subject to availability and compatibility checks. E07 later validated a specific Codex CLI and model configuration; see its final report for exact boundaries.

------------------------------------------------------------------------

# 10. E01 — Core Foundation

## 10.1 Objective

Establish an executable foundation with configuration, startup, HTTP serving, modular lifecycle, and controlled shutdown.

## 10.2 Initial capabilities

| ID | Capability | Description |
| --- | --- | --- |
| E01-C01 | Application Bootstrap | Controlled startup and shutdown |
| E01-C02 | Configuration | Load and validate configuration |
| E01-C03 | HTTP Server | Local HTTP server |
| E01-C04 | Module Management | Organize and initialize modules |
| E01-C05 | Health Monitoring | Check application health |
| E01-C06 | Logging | Structured, traceable logs |

## 10.3 Initial functional requirements

- RF-001: Start as a local application.
- RF-002: Load startup configuration.
- RF-003: Reject invalid configuration before inconsistent startup.
- RF-004: Expose health endpoints over HTTP.
- RF-005: Support graceful shutdown.
- RF-006: Give every request a correlation identifier.
- RF-007: Report useful startup failure information.

## 10.4 Initial acceptance criteria

### AC-001 — Startup

Given valid configuration, when the user starts AI Hub, the server starts and exposes its health endpoint.

### AC-002 — Invalid configuration

Given an invalid required setting, when the application starts, it reports the problem and stops startup.

### AC-003 — Health check

Given a running server, when a client requests the health endpoint, the response indicates application availability.

### AC-004 — Graceful shutdown

Given a server handling requests, when it receives a shutdown signal, it stops accepting new requests and finishes work within the configured limit.

### AC-005 — Correlation ID

Given an incoming HTTP request, when it passes through the server, a correlation identifier is available in logs.

## 10.5 Initial suggested structure

    ai-hub/
    ├── cmd/
    │   └── ai-hub/
    │       └── main.go
    ├── internal/
    │   ├── app/
    │   ├── config/
    │   ├── server/
    │   ├── health/
    │   └── observability/
    ├── pkg/
    ├── configs/
    ├── docs/
    ├── go.mod
    └── README.md

Do not create empty directories without a need. This structure represents planned responsibilities and should be adapted to the actual code.

------------------------------------------------------------------------

# 11. E02 — Provider Management

## 11.1 Objective

Integrate, configure, and administer providers without coupling the AI Hub core to one vendor. OpenCode Go was selected as the first provider.

## 11.2 Initial capabilities

| ID | Capability | Description |
| --- | --- | --- |
| E02-C01 | Provider Registry | Register providers |
| E02-C02 | Provider Configuration | Configure endpoints and parameters |
| E02-C03 | Credential Management | Manage credentials securely |
| E02-C04 | Provider Adapter | Define an integration boundary |
| E02-C05 | Provider Connectivity | Check connectivity |
| E02-C06 | Provider Lifecycle | Enable, disable, and update providers |

## 11.3 Initial functional requirements

- RF-008: Allow provider configuration.
- RF-009: Configure name, endpoint, protocol, and credential.
- RF-010: Protect locally held credentials.
- RF-011: Isolate provider integrations.
- RF-012: Check provider connectivity.
- RF-013: Identify supported protocols.
- RF-014: Enable and disable a provider.
- RF-015: Record integration errors without exposing credentials.

## 11.4 Initial provider contract proposal

The original conceptual Go contract was:

    type Provider interface {
        ID() string
        Capabilities() ProviderCapabilities
        ListModels(ctx context.Context) ([]Model, error)
        Execute(ctx context.Context, request Request) (Response, error)
        Stream(ctx context.Context, request Request) (Stream, error)
    }

This was a proposal, not a final contract. Request and response types would be internal normalized types; adapters would convert to external protocols; capabilities would be explicit; unsupported operations would remain optional; provider-specific behavior should not spread through the central domain; and extensibility should not create unnecessary abstractions.

The current repository has one validated concrete provider adapter. The architecture documents why it does not introduce a speculative provider interface before a second adapter is needed.

## 11.5 Initial OpenCode Go settings

| Field | Initial value or direction |
| --- | --- |
| Provider ID | opencode-go |
| Name | OpenCode Go |
| Base URL | To be confirmed against current official documentation |
| Desired protocol | Responses |
| Authentication | API key |
| Credential variable | OPENCODE_API_KEY |

The kickoff did not treat the endpoint, protocol support, or model availability as confirmed. The original candidate was a DeepSeek model, whose availability and Responses compatibility required proof before claiming integration completion. Current settings and evidence live in the provider configuration and research/specification documents.

## 11.6 Initial acceptance criteria

### AC-006 — Provider configuration

Given a valid configuration, when the user registers a provider, the system makes it available for use.

### AC-007 — Credential protection

Given a configured API key, when the system persists or logs information, the key does not appear in logs or API responses and is protected in local storage.

### AC-008 — Connectivity

Given a configured provider, when the user requests a connection check, the system reports whether communication succeeded and classifies relevant failures.

### AC-009 — Capabilities

Given a reachable provider, when its capabilities are queried, the system identifies the protocols and operations it actually supports.

### AC-010 — Isolation

Given two providers, when one integration changes, the other should not need changes unless a shared contract changes.

### AC-011 — Disablement

Given a registered provider, when the user disables it, new executions do not use it.

------------------------------------------------------------------------

# 12. Codex + OpenCode Go Integration

## 12.1 Target flow

    Codex
       |
       | OpenAI Responses API
       v
    AI Hub Responses Gateway
       |
       v
    OpenCode Go Adapter
       |
       | Validated upstream protocol
       v
    OpenCode Go
       |
       v
    Selected model
       |
       v
    Response / Stream
       |
       v
    Codex

The implemented route preserves the upstream Responses protocol and streams responses. The gateway injects transport headers for provider authentication and provider-session identity; credentials are not logged or mirrored into client responses.

## 12.2 Protocol compatibility

The Codex configuration used by this project supports custom Responses providers. The initial preference was Responses-to-Responses when the selected endpoint and model support it.

The current E02/E05 path does not translate Chat Completions to Responses. The request body is forwarded byte-for-byte. The current lifecycle does not extract the model from that body. A future feature that changes the selected target or payload must define an explicit mapping boundary and acceptance evidence.

The current implementation does not send previous_response_id. Translated models may require the harness to replay the full conversation context. E02 does not provide cross-turn durable AI Hub session continuity.

## 12.3 Capabilities to validate

Integration evidence should distinguish authentication, model identification, request/instruction formats, streaming, response completion, errors and HTTP status, timeout and cancellation, reported usage, tool calling, reasoning metadata, context limits, and model/endpoint compatibility.

A successful simple text request alone does not prove full compatibility.

## 12.4 Real integration validation

The initial validation plan was to load credentials safely, query a model catalog if available, select a proven compatible model, send a minimal request, receive a response, validate streaming and error paths, and check that credentials never appear in logs.

When credentials are unavailable, simulated HTTP servers can validate deterministic behavior, but the real integration must remain explicitly unverified. The final E07 real-provider result and its limitations are recorded in the linked validation report.

------------------------------------------------------------------------

# 13. Later Epics

The following work was outside the original E01/E02 implementation scope. The
descriptions below are original kickoff proposals, not current status or
authorization. Current status is in
[Current Project Status](#current-project-status) and the
[consolidated roadmap](docs/roadmap/AI-HUB-ROADMAP.md).

## E08 — Intelligent Routing

- Classify tasks.
- Apply routing policies.
- Select models.
- Explain decisions.

## E09 — Resource Management

- Manage allowances, quotas, credits, costs, limits, and permissions.

## E10 — Learning Engine

- Evaluate outcomes.
- Track historical performance.
- Compare models.
- Improve decisions over time.

## E11 — Dashboard

- Configure providers, models, policies, and autonomy.
- Monitor history and metrics.

Earlier planned epics E03–E07 are delivered as described in the Current Project Status section; their remaining limits are in the relevant specifications and reports.

------------------------------------------------------------------------

# 14. Architectural Risks and Safeguards

## 14.1 Quotas and allowances

Some providers do not expose APIs for subscription allowances. The system must distinguish official values, manual configuration, and estimates.

## 14.2 Subscriptions and external access

Do not assume Cursor, Codex, or Claude allowances can be consumed through an external gateway. Verify each product's technical capabilities and terms.

## 14.3 Streaming and fallback

Do not switch models in the middle of a stream without an explicit strategy. Unless a safe continuation mechanism is defined, fallback should happen before emitting content to the client.

## 14.4 Sessions and context

Cross-model continuity may require session affinity, history, or context transformation. Providers must not be assumed to accept identical context.

## 14.5 Security

- Never expose credentials.
- Avoid recording sensitive content without need.
- Protect a local endpoint from unintended network access.
- Require gateway authentication before exposing the service beyond localhost.
- Apply timeouts and body limits.
- Validate endpoints and TLS certificates.
- Do not let AI classification bypass security or budget policies.

Current gateway access controls and provider credential handling are specified in E02 and E04. AC-14's former host-environment integrity requirement is outside the current AI Hub product boundary; this does not waive provider credential protection or gateway security.

## 14.6 Observability and provenance

Every consumption, quota, cost, or performance value should state its source and reliability. Estimates must not be presented as official values.

------------------------------------------------------------------------

# 15. Initial Architecture Decisions

The initial direction was recorded as follows. Current accepted decisions are maintained in the numbered ADRs under [docs/adr](docs/adr/README.md); the original list below is historical and does not replace those records.

| Initial ADR | Initial direction |
| --- | --- |
| ADR-001 | Local-first deployment |
| ADR-002 | Modular monolith |
| ADR-003 | Distinguish harness, gateway, provider, and model |
| ADR-004 | Protocol compatibility plus provider-specific adapters |
| ADR-005 | Hybrid routing with rules and an LLM when needed |
| ADR-006 | Synchronous and asynchronous execution according to task |
| ADR-007 | Defer persistence technology to detailed design |
| ADR-008 | Gateway First implementation strategy |
| ADR-009 | Codex + OpenCode Go as the first integration combination |

The current repository contains ADR-001 through ADR-018, including decisions about the standard-library HTTP server, configuration, logging, lifecycle, access control, model identity, streaming, catalog persistence, request admission, execution lifecycle, records, and terminal observation. Consult the ADR index for current accepted status.

------------------------------------------------------------------------

# 16. Initial Definition of Done

An increment is complete only when:

- Its defined scope is implemented.
- Applicable acceptance criteria are satisfied.
- Relevant automated checks pass.
- Important negative cases are covered.
- Logs do not expose credentials.
- Documentation is current.
- Known limitations are recorded.
- Real validation is marked as completed or pending.
- Simulated behavior is not presented as complete functionality.

This definition may evolve as the product develops.

------------------------------------------------------------------------

# 17. Original Immediate Execution Plan

This plan describes the original E01/E02 kickoff and has been completed or superseded by the current implementation records. It is retained as historical context, not as an instruction to start another increment.

## Step 1 — Inspect the environment

- Check whether a repository and code already exist.
- Identify language, structure, tools, and conventions.
- Read project instructions and documentation.
- Analyze existing files before replacing them.

## Step 2 — Establish E01

- Define application bootstrap.
- Implement configuration and validation.
- Implement the local HTTP server.
- Implement health checks.
- Implement structured logs and correlation IDs.
- Implement graceful shutdown.
- Add tests.

## Step 3 — Implement E02 incrementally

- Define only the minimum provider concepts.
- Configure the OpenCode Go provider.
- Protect its credential.
- Implement its adapter and connectivity checks.
- Query models if supported.
- Add unit and integration tests.

## Step 4 — Validate protocol and model

- Check current official provider documentation.
- Confirm the upstream endpoint and Responses support.
- Confirm the candidate DeepSeek model is available.
- Make a real request when credentials are available.
- Validate streaming and record limitations.

## Step 5 — Close the increment

Report implementation, final code structure, decisions, checks, real validation, acceptance outcomes, risks, and next steps. The original plan said not to begin E03 before reviewing E01/E02; E03 and subsequent increments have since been delivered under their own project records.

------------------------------------------------------------------------

# 18. Original Initial Non-goals

At the initial E01/E02 stage, the following were explicitly deferred to avoid premature complexity:

- LLM task classifier.
- Complete Decision Engine.
- Learning Engine.
- Automated model benchmarking.
- Advanced quota management.
- Simultaneous integration with every harness.
- Complete dashboard.
- Distributed system or microservices.
- Kubernetes.
- External queues without a proven need.
- Cloud synchronization.
- Switching models during an active streaming response.

These remain future product capabilities where they are not already covered by the delivered E01–E07 scope. Their current status is listed above.

------------------------------------------------------------------------

# 19. First-Cycle Expected Result and Actual Boundary

The first cycle originally expected a functional local Go application with validated configuration, controlled startup, an HTTP server and health checks, structured logs and traceability, an initial provider boundary, protected OpenCode Go credentials, a provider adapter, connectivity checks, a model catalog when available, and documented evidence of model/protocol compatibility.

The repository now contains the E01–E07 implementation summarized at the top of this document. This does not mean the complete AI Hub product is delivered: intelligent routing, resource policies, usage accounting, learning, and dashboard work remain outside the current runtime as noted above.

------------------------------------------------------------------------

# 20. Closing Guidance

Treat the original kickoff as a product and architecture baseline, not an immutable specification. The project should evolve through small, specified increments. Define enough for the next approved scope, validate it with suitable evidence, and update project documentation when new facts justify a change.

The immediate E01/E02 mission in the original document is historical. Current project state, approved specifications, ADRs, and dated validation evidence are the references for future work.
