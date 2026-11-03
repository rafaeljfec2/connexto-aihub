# ADR-022 — Deterministic intelligent routing

- **Status:** accepted (E08.1 + E08.1.1, project lead approval 2026-11-03)
- **Date:** 2026-11-03
- **Related:** ADR-009 (provider-scoped model identity), ADR-014 (execution lifecycle), ADR-017 (content-free execution records), ADR-021 (provider selection and OpenRouter runtime), E03 Model Catalog, E05 Execution, C2 OpenRouter compatibility
- **Contracts:** [E08 Routing Contract](../specs/E08-routing-contract.md), [SPEC E08](../specs/E08-intelligent-routing.md)

## Context

The current Responses Gateway routes to the provider selected by X-AIHub-Provider, or to opencode-go when the header is absent. It captures the request model as execution metadata but forwards the request body unchanged. E03 records provider-scoped model IDs, conservative capabilities, and independent presence, freshness, provider-health, and execution-verification facts. It does not rank models or decide eligibility.

The provider registry has OpenCode Go and OpenRouter Responses adapters. Provider runtime availability is distinct from health: a successful model-list probe does not establish inference or model capability. E05 executes one explicit target and prepared payload; it owns cancellation, timeout, streaming, terminal arbitration, and shutdown. E06 schema v1 records request ID and the selected provider/model for terminal executions without request content.

C2 PostgreSQL integration and persistence evidence remains open. C2 is not fully accepted until it passes, but that validation debt does not block E08.

E08.1.1 refines the contract with structural validation, snapshot consistency, expanded capability projection, and explicit precedence between routing_unavailable and routing_ambiguous. It also documents an evolutionary path toward task classification and optimization profiles informed by public GitHub Copilot and Cursor documentation, without adopting proprietary algorithms.

## Decision

### MVP routing (E08.2–E08.5)

1. **Start deterministic.** E08 operational increments use explicit request selections, eligibility facts, and documented compatibility rules. No LLM classifier, machine learning, hidden scores, or inferred message meaning in the MVP path.
2. **Separate eligibility from selection.** The Eligibility Engine removes objectively ineligible candidates. The Deterministic Selector applies precedence and unique-candidate rules. Eligibility never ranks; selection never guesses compatibility.
3. **Absolute priority for explicit selections.** An explicit provider constrains the route. An explicit model constrains the exact model ID. When both are present, only their exact pair is acceptable; conflicts reject without provider switch.
4. **Resolve model-only requests only by unique catalog match** after complete fresh catalog coverage across providers that pass provider eligibility. Multiple eligible matches are ambiguous; incomplete coverage is routing_unavailable, which prevails over ambiguity when coverage is insufficient (contract §3.4).
5. **Preserve the no-header compatibility default** for automatic selection: prefer one eligible opencode-go model; otherwise select a non-default provider only when exactly one eligible candidate remains and relevant catalogs are complete and fresh. Lexical order is evidence-only, never a tie-break.
6. **Use conservative eligibility** with present/fresh catalog entries and positive capability claims for required features. Distinguish unsupported, unknown, stale, and unmapped requirements.
7. **Do not use health as fallback.** Health is decision context only; runtime enabled gates execution.
8. **Single-target execution** without post-start fallback or provider switching.
9. **Prepare only the selected model field** in E08.4 when routing supplies a model; preserve bytes when the explicit model already matches.
10. **Keep decision evidence content-free** without changing E06 schema v1 in E08.

### Evidence and validation (E08.1.1)

11. **Structural validation before eligibility.** Invalid model shapes, duplicate model keys, and invalid max_output_tokens reject as invalid_request without silent coercion (contract §2.3).
12. **One routing snapshot per decision.** Capture registry and catalog facts coherently; do not mix moments when proving uniqueness (contract §2.4). No new distributed infrastructure required; E08.2 implements capture with existing manager/registry reads.
13. **Capability projection uses E03 vocabulary only.** Map json_schema and json_object to structured_output; treat file input without E03 mapping as requirement_unmapped; do not treat SSE stream flag as model streaming capability.

### Evolution (documented, not authorized for implementation)

14. **Task classification may arrive later (E08.6+).** Bounded structural metadata and optional classifiers may inform selection only under a future approved contract. A classifier must not override explicit provider or model selections.
15. **Optimization Profiles (Efficiency, Balance, Intelligence) are product evolution (E08.7+), not MVP.** Document trade-offs without weights, formulas, or fictitious scores.
16. **Ranking requires verifiable signals (E08.8+).** Intelligent routing may rank eligible candidates only when signals and profiles are explicitly defined and approved; until then, ambiguity is the correct outcome.
17. **Decisions remain explainable and audit-stable** with stable reason codes and selection origins across deterministic and future intelligent modes.
18. **E08 does not depend on E09, E10, or E11.** Resource management, learning, and dashboard work are out of scope.
19. **External product docs are informative only.** GitHub Copilot Auto and Cursor Router informed the evolutionary SPEC; they are not acceptance criteria for the deterministic MVP.

### Governance

20. **C2 acceptance stays independent.** E08 may proceed while PostgreSQL integration evidence is outstanding. E08 does not declare C2 complete.

## Alternatives considered

| Alternative | Decision |
| --- | --- |
| LLM-based request classifier in MVP | Rejected for E08.2–E08.5. Deferred to optional E08.6+ with separate approval. |
| Choose the first sorted model when candidates tie | Rejected. Return routing_ambiguous when coverage is complete. |
| Treat incomplete catalog as ambiguity | Rejected. routing_unavailable prevails when uniqueness cannot be proved (E08.1.1). |
| Use listing_state=current as the sole model gate | Rejected. Use presence/freshness and provider availability separately. |
| Block every provider whose health is unknown or unhealthy | Rejected. Health is not proof of inference availability. |
| Automatically switch providers after an execution failure | Rejected. Contradicts E05/C2 contracts. |
| Copy Copilot/Cursor proprietary routing algorithms | Rejected. Only public concepts inform evolution; no undisclosed weights or classifiers. |
| Persist routing decisions in a new store or endpoint | Deferred. E08.5 bounded logs only unless E06 schema change is separately approved. |
| Global locks or distributed transactions for snapshots | Rejected for E08.2. Single-process coherent read sequence is sufficient initially. |

## Consequences

- The same request, routing snapshot, and explicit selections produce the same deterministic decision.
- Automatic selection works only when documented rules leave one candidate; otherwise ambiguity or insufficient evidence is reported instead of invented ranking.
- E08.4 changes gateway behavior for structurally invalid routing fields and for inserting an automatically selected model; valid absent-model requests remain supported.
- Future intelligent routing must extend selection without breaking eligibility contracts or explicit user constraints.
- Copilot/Cursor Auto modes illustrate user-facing optimization profiles but do not obligate AI Hub to match their behavior or model pools.
- No routing runtime is implemented by this ADR. E08.1.1 artifacts require project-lead approval before E08.2 implementation proceeds.
- PostgreSQL integration and persistence evidence remains a C2 validation debt and must pass before C2 is accepted.
