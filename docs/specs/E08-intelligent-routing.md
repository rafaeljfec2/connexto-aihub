# SPEC E08 — Intelligent Routing

**Status:** accepted — project lead approval 2026-11-03 (E08.1 + E08.1.1 refinements)

**Revision:** E08.1.1 — contract refinements, evolutionary routing vision, external product concepts (informative)
**Related:** [Routing Contract](E08-routing-contract.md), [ADR-022](../adr/ADR-022-deterministic-intelligent-routing.md)

## 1. Problem

AI Hub currently routes Responses requests to an explicitly selected provider or to the opencode-go compatibility default. It forwards the request model to that provider without resolving it against the provider-scoped catalog. E08 adds deterministic selection of an eligible provider/model while retaining explicit user intent and the existing execution lifecycle.

The project currently has provider availability and health diagnostics, a provider-scoped model catalog with conservative capability claims, a byte-preserving Responses gateway, content-free schema-v1 execution records, OpenCode Go and OpenRouter adapters, and no silent execution fallback. C2 PostgreSQL integration evidence remains a validation debt; C2 is not fully accepted until that evidence passes, but it does not block E08.

## 2. MVP objective

For a request, a consistent provider/runtime snapshot, a catalog snapshot, and the explicit selections present in the request, return one eligible provider_id/model_id pair or an explainable rejection. Identical inputs and snapshots must produce the same decision.

E08 uses deterministic rules only for the operational MVP (E08.2–E08.5). The catalog has no quality, cost, latency, or quota ranking in that phase, so the selector abstains with routing_ambiguous when more than one candidate remains after the documented compatibility rule. It never chooses the first sorted model as an arbitrary winner.

Later increments (E08.6–E08.8) may add task classification and optimization profiles **only** with verifiable signals and without breaking explicit user constraints or eligibility facts documented here.

## 2.1 Evolutionary architecture (product vision)

The long-term intelligent router follows a pipeline that keeps responsibilities separable:

```text
Request Understanding  →  Eligibility Engine  →  Intelligent Selector  →  Provider + Model  →  Execution Lifecycle
     (E08.6+)                 (E08.2)              (E08.3 MVP; E08.7–E08.8+)         (E08.4)            (E05/E06)
```

| Stage | Responsibility | MVP (E08.2–E08.5) | Future (E08.6+) |
| --- | --- | --- | --- |
| Request Understanding | Derive bounded task metadata from structural request fields and explicit preferences | Not implemented; routing is content-blind | May classify task type, complexity, reasoning/tool/modality needs, optimization preference — **never** from message text in E08.6 contract without separate approval |
| Eligibility Engine | Remove candidates that fail objective provider, catalog, capability, and limit checks | Full contract in [Routing Contract §3](E08-routing-contract.md) | Same facts; may consume additional **explicit** policy flags when approved |
| Intelligent Selector | Choose among eligible candidates | Deterministic precedence and unique-candidate rules only | May rank using optimization profile and verifiable signals (E08.7–E08.8) |
| Gateway integration | Validate structure, build snapshot, prepare provider-ready body | E08.4 | Extends observability (E08.5) |

**Request Understanding (future boundary).** May eventually emit metadata such as task category, estimated complexity, reasoning or tool requirements, modality requirements, and explicit optimization preference. An LLM or ML classifier is **out of scope** for E08.1.1 and E08.2–E08.5. A future classifier must not override explicit provider or model selections and must not inspect message content unless a future SPEC explicitly defines bounded structural features only.

**Eligibility Engine.** Eliminates candidates using verifiable facts only (provider enabled, credential, Responses adapter, catalog presence/freshness, capability claims, known limits, availability policies). Eligibility never ranks survivors.

**Intelligent Selector (future).** Among eligible candidates, apply explicit preferences, optimization profile, and verifiable task signals; explain the decision with stable codes. Selection must not depend on arbitrary sort order. E08.3 implements deterministic selection as the baseline that must continue to work when no classifier is present.

### Optimization profiles (future product concepts)

Three profiles document intended trade-offs for E08.7+. They are **not** implementable algorithms in E08.1.1 and must not introduce weights, scores, or formulas here.

| Profile | Intent | Constraints |
| --- | --- | --- |
| **Efficiency** | Prefer economical, fast models when they meet minimum capability | Must not select models that fail eligibility for required capabilities |
| **Balance** | Trade off cost, quality, and latency using verifiable signals only | Must not invent metrics the catalog or runtime does not expose |
| **Intelligence** | Prefer higher-capability models for complex tasks | Must not assume a model is “better” without catalog or approved verification signals |

## 2.2 External product concepts (informative)

Official documentation from GitHub Copilot and Cursor informed the evolutionary vision. **Facts** below are paraphrased from public docs; **inferences** are AI Hub architectural proposals.

### GitHub Copilot (documented facts)

Sources: [About auto model selection](https://docs.github.com/en/copilot/concepts/models/auto-model-selection), [Supported models](https://docs.github.com/en/copilot/reference/ai-models/supported-models), [Change chat model](https://docs.github.com/en/copilot/how-tos/copilot-in-your-ide/chat-with-copilot/change-the-chat-model).

| Fact (public documentation) | AI Hub inference (proposal) |
| --- | --- |
| Users can choose **Auto** instead of a fixed model | AI Hub may later expose an explicit “automatic selection” mode distinct from naming a model ID |
| Auto considers availability, system health, and task complexity (with task optimization) | Future router may combine diagnostic health and structural task metadata — health remains non-fallback in MVP |
| Efficiency / Balance / Intelligence tiers adjust routing preference | Maps to Optimization Profiles (E08.7); not MVP requirements |
| Auto respects plan and administrator model policies | Future policy engine may filter eligible pools before selection |
| Selected model can be surfaced to the user | Aligns with explainable, content-free routing evidence |

Copilot does **not** publish proprietary classifier weights, datasets, or internal algorithms. AI Hub must not claim parity with Copilot routing behavior.

### Cursor (documented facts)

Sources: [Cursor Router](https://cursor.com/docs/cursor-router), [Help: Cursor Router](https://cursor.com/help/models-and-usage/cursor-router), [Models & pricing](https://cursor.com/docs/models-and-pricing).

| Fact (public documentation) | AI Hub inference (proposal) |
| --- | --- |
| **Auto** with Cost / Balance / Intelligence modes steers routing | Optimization Profiles (E08.7) |
| Router sends simple work to efficient models and complex work to capable models | Future Intelligent Selector (E08.8) with eligibility preserved |
| Blocked models are skipped; over-blocking can disable routing | Future policy filters on eligible set only |
| SDK exposes `auto-smart` with `optimize_for` | Potential future client contract; not E08 MVP |

Cursor blog posts describe ML classifiers and internal metrics. Those details are **not** treated as AI Hub requirements. Do not reproduce proprietary thresholds or model pools.

## 3. Scope and architecture (MVP)

The routed path is:

1. Validate structural routing fields (E08.4; contract §2.3).
2. Capture one routing snapshot (contract §2.4).
3. Read the existing correlation ID and bounded structural request metadata.
4. Build a routing context without message content or credentials.
5. Evaluate provider and model eligibility from the snapshot.
6. Apply deterministic precedence in the Routing Contract.
7. Return a selected target and a provider-ready payload to the existing Responses Gateway/execution lifecycle, or a typed rejection.

The gateway remains responsible for HTTP errors and writing the response. execution.Manager continues to execute one explicit target and prepared payload; it does not select, retry, or fall back.

## 4. Functional requirements

| ID | Requirement |
| --- | --- |
| FR-1 | Preserve the existing X-AIHub-Provider parsing and explicit provider behavior. A blank or absent header is an implicit opencode-go compatibility rule only when no model is explicitly requested; it is not a hard provider constraint in the model-only case. |
| FR-2 | Treat a valid top-level JSON model as an exact explicit model ID. An explicit provider/model pair must match exactly or be rejected. |
| FR-3 | When only a model is explicit, resolve only a unique eligible provider-scoped match after confirming complete fresh catalog coverage for relevant providers. Duplicate model IDs across eligible providers are ambiguous; incomplete coverage is insufficient evidence. |
| FR-4 | When only a provider is explicit, restrict eligibility to that provider; choose only if one model remains eligible. |
| FR-5 | When neither provider nor model is explicit, use the deterministic default/unique-candidate rule in the Routing Contract; prove catalog coverage before selecting a unique non-default candidate; never choose lexicographically among tied models. |
| FR-6 | Exclude disabled, credentialless, unbound, or Responses-incompatible provider runtimes with distinct reason codes where the current provider facts permit. |
| FR-7 | Require fresh, present catalog entries and positive claims for every required catalog-backed capability. Treat unknown, absent, or stale evidence as insufficient, not compatible. Declare an explicit model missing only when all relevant provider catalogs are fresh and accepted. |
| FR-8 | Keep health independent from runtime availability. unknown, healthy, and unhealthy do not themselves select a different provider or claim model compatibility. |
| FR-9 | Return typed, explainable outcomes for malformed input, missing provider/model, provider/model conflict, no eligible candidate, insufficient evidence, and ambiguity. |
| FR-10 | Preserve the selected provider/model pair through the execution command and E06 schema-v1 target fields. |
| FR-11 | When automatic model selection supplies a model absent from the request, prepare a provider-ready body with that top-level model while preserving all other JSON values. Preserve original body bytes when the explicit model already equals the selected model. |
| FR-12 | A provider execution failure after selection is terminal for that provider. Do not retry with or switch to another provider or model. |
| FR-13 | Keep any routing decision event content-free and correlated with the existing request ID. Do not add durable storage, a query endpoint, a migration, or a new persistence mechanism in E08. |
| FR-14 | Keep routing candidates, output ordering, error reasons, and selection origins deterministic for equivalent snapshots. |
| FR-15 | Distinguish routing_unavailable (incomplete catalog or unknown capability evidence) from routing_ambiguous (two or more eligible candidates with complete relevant coverage). routing_unavailable prevails when coverage is incomplete (contract §3.4). |
| FR-16 | Apply structural validation before eligibility; reject invalid model and max_output_tokens shapes as invalid_request without silent coercion (contract §2.3). |
| FR-17 | Use one routing snapshot per decision; do not mix registry and catalog facts from different moments (contract §2.4). |
| FR-18 | Map structural Responses features to E03 capabilities per contract §3.3, including json_object, json_schema, modalities, tools, and requirement_unmapped cases. |

## 5. Non-functional requirements

- **Compatibility:** Preserve Responses semantics, SSE forwarding, cancellation, timeout, sanitization, and provider-ID/model-ID identity. Document the bounded behavior change required to map an automatically selected model into the outgoing request body.
- **Explainability:** Each selected target is justified by explicit source and stable criteria; each rejection has one stable reason code and supporting eligibility reasons.
- **Privacy:** Routing inspects structural metadata only. No request content, credentials, provider session, response content, or raw upstream errors enter decision records or logs.
- **Conservative evidence:** Do not infer unsupported capabilities, provider availability from health, token counts from byte lengths, or cost/quality/latency data.
- **Determinism:** Candidate enumeration and decision rules are stable and use one input snapshot per request.
- **Resilience:** Routing occurs once before execution. The existing execution lifecycle retains sole ownership of timeout, cancellation, streaming, terminal arbitration, and shutdown.

## 6. Acceptance criteria

- **AC-1:** Every combination of explicit provider and explicit model follows the precedence matrix, including a provider/model conflict rejection.
- **AC-2:** A model-only request selects a unique eligible provider-scoped match only with complete fresh catalog coverage; duplicate matches are ambiguous and incomplete coverage is routing_unavailable.
- **AC-3:** A provider-only request selects exactly one eligible model or returns a typed no-candidate/ambiguous result.
- **AC-4:** A request with neither selector uses opencode-go as the compatibility preference when exactly one eligible model exists there; otherwise it selects only a globally unique eligible candidate when relevant catalog coverage is complete or reports ambiguity/no candidate/insufficient evidence. If opencode-go has no eligible model and another provider is uniquely eligible, that provider may be selected before execution only when coverage is complete.
- **AC-5:** Provider disabled, credential missing, runtime unavailable, health unknown/healthy/unhealthy, model absent/missing/stale, capability supported/unsupported/unknown, incomplete provider catalog coverage, and insufficient limits produce the documented eligibility outcomes.
- **AC-6:** Candidate ordering is stable, but ordering never resolves a tie.
- **AC-7:** The selected provider and model are present in the execution target; a newly auto-selected model is reflected in the provider-ready request body without changing other request values.
- **AC-8:** A post-selection provider failure produces no second provider invocation.
- **AC-9:** Routing evidence includes only bounded IDs, source, rule/reason codes, eligibility summaries, and correlation ID; it contains no prompt, messages, response, credential, provider session, or complete request.
- **AC-10:** Existing E05, E06, C2, Responses, SSE, cancellation, timeout, and no-fallback contracts remain intact except for the explicit E08.4 request-model mapping and documented routing rejection cases.

- **AC-11:** Invalid model shapes, duplicate model keys, and invalid max_output_tokens values reject as invalid_request; absent model remains valid for automatic selection.
- **AC-12:** routing_unavailable and routing_ambiguous follow §3.4 precedence, including conceptual scenarios in the contract.
- **AC-13:** json_schema and json_object both require structured_output; file input without E03 mapping yields requirement_unmapped; stream flag does not gate on streaming capability.
- **AC-14:** One snapshot per routing decision; snapshot rules in §2.4 are reflected in E08.2 design without new infrastructure.

## 7. Increment decomposition

| Increment | Deliverable | Authorization |
| --- | --- | --- |
| E08.1 | Routing Contract, E08 SPEC, ADR-022 (including E08.1.1 refinements) | Documentation; project-lead approval pending |
| E08.2 | Eligibility Engine evaluating provider/model facts, capability projection, and reason codes | **In progress** — implementation authorized 2026-11-03 |
| E08.3 | Deterministic Selector: precedence, compatibility rule, unique-candidate selection, ambiguity | **Accepted** 2026-11-03 — see `docs/decisions/e08.3-formal-acceptance-2026-11-03.md` |
| E08.4 | Responses Gateway integration, typed error mapping, structural validation, request preparation | **Implemented** (local) — see `docs/decisions/e08.4-gateway-routing-integration.md` |
| E08.5 | Content-free routing decision events and end-to-end tests | Not started |
| E08.6 | Task classification from bounded structural metadata (no LLM classifier in initial E08.6 contract without separate approval) | **Future vision — not authorized by this SPEC** |
| E08.7 | Optimization Profiles (Efficiency, Balance, Intelligence) as explicit preference inputs | **Future vision — not authorized** |
| E08.8 | Runtime-aware intelligent routing using verifiable signals and profiles atop eligibility | **Future vision — not authorized** |

E08.2 returns an ordered eligible set and explicit reasons without ranking policy. E08.3 selects only when the contract yields one unambiguous candidate. E08.6–E08.8 must not alter execution lifecycle contracts or E06 schema v1 without separate approval. The deterministic selector remains the baseline when no classifier runs.

E08 does **not** depend on E09 Resource Management, E10 Learning Engine, or E11 Dashboard.

## 8. Out of scope

- LLM-based classification, machine learning, or automatic learning in E08.2–E08.5 (future E08.6+ is documented but not authorized here).
- Dynamic cost, latency, quality, or quota optimization in the MVP increments.
- E09 resource management, E10 learning, or E11 dashboard work.
- New providers, endpoints, headers, database migrations, Redis, queues, or infrastructure.
- Automatic retry, fallback, or provider switching after execution begins.
- Durable queryable routing history or changes to E06 schema v1.
- Runtime implementation in E08.1.

## 9. Known limitations and explicit behavior changes

1. The catalog has no ranking metric. Multiple eligible models produce an ambiguity error, including when one sorts before another. A later preference or ranking contract must be explicit and separately approved.
2. A unique explicit model match without a provider may select a provider other than the old no-header opencode-go default, but only after all providers that pass provider eligibility have complete fresh catalog coverage. This is intentional: an explicit model constrains the destination. Duplicate matches remain ambiguous; incomplete coverage is insufficient evidence.
3. If no model is supplied and routing selects one, E08.4 must add that model to the outgoing request body. This changes the current byte-for-byte pass-through only for this automatic-selection case; other request values remain unchanged.
4. With neither selector, an eligible non-default provider may be selected before execution only when opencode-go is unavailable by provider facts or its complete fresh catalog has no eligible model, and exactly one eligible candidate remains across providers with complete relevant catalog coverage. This intentionally evolves the historical hard default into a compatibility preference; it is not execution fallback.
5. The streaming catalog claim remains unknown; SSE is treated as the existing Responses transport contract, not as model-level verification.
6. Routing errors introduced by E08 need an HTTP mapping in E08.4. Existing provider-selector, not-found, and provider-unavailable mappings remain the baseline.
7. C2 remains incomplete until PostgreSQL integration and persistence acceptance passes against an explicitly approved isolated test instance. E08 does not consume or relax that gate.
8. E08.1.1 documents Copilot/Cursor concepts for evolution only; commercial Auto modes are not MVP acceptance criteria.

## 10. E08.1 documentation evidence

**TDD-EXCEPTION:** E08.1 and E08.1.1 create only contract/specification/ADR documents. No runtime behavior or testable implementation changes.

### E08.1.1 evidence (2026-11-03)

| Field | Value |
| --- | --- |
| Claim | Routing Contract, E08 SPEC, and ADR-022 remain consistent after E08.1.1 refinements and evolutionary vision; MVP determinism preserved. |
| Prior session | AI Memory `01a19ef9-f3af-7353-8cc8-c28d33273c0e` / `sessions/01a19ef9-f3af-7353-8cc8-c28d33273c0e.md` |
| External docs | GitHub Copilot auto model selection; Cursor Router docs and help (URLs in Routing Contract §10) |
| Success type | manual and command |
| Evidence path | Cross-check against gateway, catalog, execution implementations; internal Markdown links; `git diff --check`; independent review |
| Budget | All three artifacts; E08.1.1 refinements in contract §2.3, §2.4, §3.3, and §3.4 (ambiguity vs evidence, structural validation, capability projection, snapshot consistency) |
| Affordance | None |
