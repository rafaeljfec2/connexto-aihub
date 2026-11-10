# ADR-023: Auto AI-Hub Virtual Model Architecture & Routing Extensions

**Status:** Accepted  
**Date:** 2026-11-09  
**Deciders:** Connexto AI-Hub Engineering Team  
**Consulted:** E08 Intelligent Routing Audit, AI Memory Consensus, LLM Homologation Remediation  

---

## 1. Context and Problem Statement

AI-Hub currently features deterministic routing for explicitly requested models or capability-based filtering (`E08.1` to `E08.5`). When clients seek optimal dynamic execution without manually choosing a model, they need a standard OpenAI-compatible model identifier that encapsulates automatic intelligent selection.

Key architectural challenges:
1. Should `aihub/auto` be modeled as a pseudo-upstream provider or as an entry-level virtual routing intent?
2. How do we ensure observability and transparent auditing without leaking sensitive prompts or breaking standard OpenAI schemas?
3. What are the strict boundaries between eligibility filtering, ranking, and fallback?

---

## 2. Decision Drivers

- **Zero Provider Spoilage:** Avoid creating mock credentials, dummy upstream adapters, or synthetic health checks for a non-existent provider.
- **Anti-Recursion:** Prevent routing loops where an auto-selected model resolves to `aihub/auto`.
- **Pre-Commit Transparency:** Ensure clients know exactly which provider and model served their request before streaming begins.
- **Strict Fallback Semantics:** Avoid silently replacing models mid-stream or burning multiple allowances on quota exhaustion.
- **Deterministic First:** Keep request understanding structural and instant, avoiding premature external LLM classifiers.

---

## 3. Considered Options

- **Option 1: Upstream Proxy Provider (`provider: aihub, model: auto`).**  
  *Pros:* Reuses existing provider interface directly.  
  *Cons:* Requires mock credentials, pollutes catalog snapshots, risks recursive loops, and complicates lifecycle draining.

- **Option 2: Virtual Model Dynamic Resolution in Responses Gateway (Chosen).**  
  *Pros:* Directly resolves `aihub/auto` into a concrete `(ProviderID, ModelID)` before invoking the execution pipeline; cleanly separates virtual intent from physical execution.

---

## 4. Decision Outcome

Chosen: **Option 2**.

### Key Architecture Decisions:
1. **Virtual Identifier:** `aihub/auto` is registered in `GET /v1/models` with `type: "virtual"`.
2. **Gateway Interception:** In `POST /v1/responses`, if `model == "aihub/auto"`, the gateway constructs an `E08.6 RoutingContext`, evaluates `E08.2 Eligibility`, runs `E08.8 Ranked Selector`, and dispatches to the resolved concrete provider and model.
3. **Response Headers:** Injects `X-AIHub-Resolved-Model`, `X-AIHub-Resolved-Provider`, `X-AIHub-Routing-Profile`, and `X-AIHub-Routing-Reason` prior to HTTP response commitment.
4. **Body Fidelity:** Sets `model: <resolved_model>` in the OpenAI response envelope.
5. **Zero Post-Commit Fallback:** If streaming has started, model switching is strictly forbidden. Pre-commit HTTP 503 errors permit a single alternate retry.
6. **Gate 0 Enforcement:** Auto dispatch is operational only when all 10 catalog models achieve verified homologation approval.

---

## 5. Consequences

### Positive
- 100% backward compatibility with all OpenAI-compatible developer tools (Cursor, OpenCode, VS Code, cURL).
- No mock upstream provider overhead.
- Completely explainable and reproducible routing decisions.
- Privacy-safe (content-free audit trails).

### Negative / Trade-offs
- Requires careful HTTP header management before writing streaming response chunks.
- Auto dispatch requires completion of homologation Gate 0.

