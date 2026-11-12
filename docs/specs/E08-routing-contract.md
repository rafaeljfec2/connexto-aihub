# E08 Routing Contract

**Status:** accepted — project lead approval 2026-11-03 (E08.1 + E08.1.1 refinements)

**Scope:** E08.1 deterministic routing contract and E08.1.1 refinements; no runtime behavior is implemented here.

**Revision:** E08.1.1 adds ambiguity versus evidence precedence, structural validation, snapshot consistency, expanded capability projection, and architectural boundaries for future intelligent selection documented in [SPEC E08](E08-intelligent-routing.md).

## 1. Purpose and boundaries

The routing contract converts a Responses request and a consistent snapshot of provider and catalog facts into either one provider/model target or an explainable rejection. It does not classify message meaning, alter user intent, or retry execution.

The router may inspect bounded, structural request metadata only. It must not inspect message text to infer task type, quality requirements, or user intent. The original body remains opaque except for the top-level model field and the structural capability requirements defined below.

## 2. Identity and inputs

### 2.1 Provider and model identity

- Provider identity is the configured provider ID.
- Model identity is the exact provider_id/model_id pair from the catalog (ADR-009). canonical_id is not an identity key.
- The Responses JSON model string is the requested model_id. Preserve its value verbatim; do not trim, normalize, or substitute aliases.
- X-AIHub-Provider retains its existing parsing contract: multiple values or a value over 128 bytes is invalid; a nonblank value is resolved as supplied. An absent or one blank value is an implicit compatibility default, not an explicit provider constraint. It applies to requests with neither selector; when a model is explicit, resolve the model-only case below.

The current gateway extracts a nonblank, bounded JSON model string into execution metadata, but forwards the request body unchanged and does not validate the model against the catalog. E08.4 will make model parsing part of routing input validation. A present model that is not one nonblank JSON string, or exceeds the existing 4,096-byte metadata bound, is an invalid request. Duplicate top-level model keys are invalid because their meaning is parser-dependent.

### 2.2 Routing request and context

| Conceptual field | Source | Contract |
| --- | --- | --- |
| request_id | Existing X-Request-Id correlation context | Reuse the validated or generated correlation ID. |
| explicit_provider_id | X-AIHub-Provider | Optional hard constraint; source is provider_header. |
| explicit_model_id | Top-level JSON model | Optional hard constraint; source is request_model. |
| required_capabilities | Structural Responses request fields | A deduplicated set from the mapping in §3.3; never derived from message text. |
| max_output_tokens | Top-level Responses request field | Optional positive bound compared only with known catalog limits.output. |
| provider_facts | Provider administration and runtime registry | Provider ID, configured enabled flag, credential-presence boolean, bound Responses-adapter presence, runtime enabled flag, and diagnostic health state. Credential values are never exposed to the router context. |
| model_facts | Catalog view | Provider/model identity, presence, freshness, capability claims, and known limits. |

The only existing explicit user routing preferences are the provider header and request model. No persisted per-user model preferences or quality ranking exist in the current system. The opencode-go default is a compatibility rule, not a user preference.

Routing context excludes prompt text, input values, message contents, response data, provider session IDs, credentials, and the complete request body.

### 2.3 Structural request validation

Structural validation runs before eligibility and selection. It rejects malformed routing inputs as `invalid_request`. It does not coerce, trim, normalize, or substitute invalid values.

Validation is separate from eligibility: a structurally valid request may still be rejected because no eligible model exists, evidence is insufficient, or multiple candidates remain.

| Field / condition | Routing behavior |
| --- | --- |
| JSON body not parseable | `invalid_request` |
| Top-level `model` absent | Valid; treated as no explicit model (automatic selection may apply under §4). |
| Top-level `model` JSON `null` | `invalid_request` |
| Top-level `model` empty string or whitespace-only | `invalid_request` |
| Top-level `model` not a JSON string | `invalid_request` |
| Duplicate top-level `model` keys | `invalid_request` (parser-dependent semantics) |
| Top-level `model` length over 4,096 bytes | `invalid_request` (existing execution-record bound) |
| Top-level `max_output_tokens` absent | Valid; no output limit check beyond capability mapping |
| Top-level `max_output_tokens` JSON `null` | Valid; treated as absent |
| Top-level `max_output_tokens` zero or negative | `invalid_request` |
| Top-level `max_output_tokens` not a JSON number | `invalid_request` |
| Top-level `max_output_tokens` fractional or non-integer | `invalid_request` |
| `max_output_tokens` present and structurally valid but above a known catalog `limits.output` for a candidate under evaluation | Eligibility exclusion with `limit_exceeded` for that candidate |
| `max_output_tokens` present and structurally valid but catalog `limits.output` is unknown for a candidate that would otherwise match | Eligibility exclusion with `limit_unknown` for that candidate; may contribute to `routing_unavailable` when no eligible candidate remains |

The current gateway may forward some malformed model metadata with an empty observed model ID (see `requestedModelID` in `internal/gateway/responses_handler.go`). E08.4 adopts the table above for routing admission. That is a documented behavior change for invalid model shapes only; valid absent-model requests remain supported for automatic selection.

Duplicate or invalid provider selector values remain governed by the existing X-AIHub-Provider contract.

### 2.4 Routing snapshot consistency

Each routing decision uses one **routing snapshot**: a coherent view of provider configuration, runtime availability, catalog presence, catalog freshness, model capabilities, and provider health captured for that decision.

**Snapshot contents**

| Fact class | Source (current architecture) | Snapshot rule |
| --- | --- | --- |
| Provider configuration | Provider registry / administration settings | Enabled flag, credential presence, bound adapter |
| Runtime availability | Provider adapter `Enabled()` | Effective execution gate |
| Diagnostic health | Provider manager health state | Context only; not an eligibility gate |
| Catalog entries | Catalog manager in-memory index | Provider-scoped model entries |
| Catalog sync state | Catalog manager sync metadata | Presence, freshness, last outcome |
| Derived capabilities | Catalog entry claims | Closed E03 vocabulary only |

**Consistency rules**

1. Build the snapshot once per routing decision before candidate enumeration.
2. Do not mix provider facts from time T₁ with catalog facts from time T₂ when both influence the same uniqueness or coverage conclusion.
3. Under concurrent administrative changes (provider reconfigure, catalog sync, health transition), prefer a single critical section or equivalent sequence: capture registry-visible provider statuses and a catalog manager read view in one pass, then evaluate eligibility and selection without interleaved mutation of the captured view.
4. If a catalog sync completes mid-request, the decision uses either the pre-sync or post-sync view consistently; never a hybrid per candidate.

**Implementation note and limitation**

The monolith does not today expose a cross-package version token for “registry + catalog” atomicity. E08.2 should implement snapshot capture using existing `catalog.Manager` RWMutex read paths and registry status reads in one goroutine before evaluation. If a future revision introduces an explicit generation counter, record it in decision evidence. This contract does not require distributed transactions, global locks, or new infrastructure.

**Pending decision (non-blocking for E08.1.1)**

Whether rejected routing decisions must echo the catalog sync generation or provider health check version in content-free evidence is deferred to E08.5 observability design.

## 3. Eligibility

### 3.1 Provider checks

A provider candidate is eligible only when all of these facts hold:

1. Its provider ID is known and has a runtime binding.
2. Its adapter implements the Responses operation.
3. Its configured provider is enabled.
4. A credential is configured.
5. The bound runtime reports enabled.

The existing provider record distinguishes Enabled from HasCredential; runtime Enabled() is the effective execution gate. Report provider_disabled and credential_missing separately when the corresponding facts are available. A known administrative record without a bound Responses adapter is provider_unbound or responses_unsupported, not a model-catalog failure.

Eligibility is read-only. It does not issue network probes or start catalog synchronization.

### 3.2 Model checks

A model candidate is eligible only when:

1. Its exact provider-scoped entry exists.
2. Catalog presence is present.
3. Catalog freshness is fresh.
4. Every required normalized capability has a catalog claim with state supported.
5. If max_output_tokens is present, limits.output is known and at least that value.

unsupported is an objective incompatibility. unknown, an absent claim, an unknown limit needed for the request, stale catalog data, missing presence, or a missing entry is insufficient evidence and cannot be treated as compatible.

Absence of a model entry proves model_not_found only when every provider that passes the provider checks has a fresh accepted catalog snapshot. Providers already ineligible because they are disabled, credentialless, unbound, or lack Responses support cannot produce a candidate and do not make uniqueness uncertain. If any other relevant provider has never synchronized or has only stale/failed catalog evidence, the router cannot prove that an explicit model ID does not exist or is unique; return routing_unavailable unless an explicit provider constraint resolves the request. Eligibility is read-only and does not trigger provider probes or catalog synchronization.

Positive capability claims may be unverified; they remain catalog evidence, not a guarantee of execution. execution_verification is not promoted by routing.

Do not use listing_state as a routing gate. It combines model presence/freshness with provider health, while C2.6 defines health and execution availability as separate facts. Use the separate presence and freshness dimensions above.

### 3.3 Request feature projection

The MVP recognizes only structural request features that map to E03's closed capability vocabulary. Projection inspects JSON structure only; it never reads message or input text.

#### 3.3.1 Mapping table

| Responses request feature | Required catalog capability | Classification when feature present |
| --- | --- | --- |
| Text in `input` string or `input_text` item | `text_generation` | Known mappable capability |
| `input_image` item | `image_input` | Known mappable capability |
| `input_audio` item | `audio_input` | Known mappable capability |
| `input_file` or file-attachment item types documented by OpenAI Responses | *(none in E03 vocabulary)* | Known feature, **requirement_unmapped** — do not infer `file_input` |
| `tools[]` with `type: function` | `function_tools` | Known mappable capability |
| `tools[]` with `type: custom` | *(none for eligibility)* | Client-executed freeform tools pass through on the wire; do not project `custom_tools` because E03 keeps that claim `unknown` for every entry |
| Nonempty requested reasoning configuration | `reasoning` | Known mappable capability |
| `text.format.type: json_schema` | `structured_output` | Known mappable capability (JSON schema constrained output) |
| `text.format.type: json_object` | `structured_output` | Known mappable capability (JSON object mode uses the same normalized claim until E03 adds a distinct capability) |
| Other `text.format.type` values | *(none)* | Known feature without normalized mapping → **requirement_unmapped** unless E03 extends vocabulary |
| Top-level `stream: true` | *(none for model capability)* | Transport/SSE contract via gateway and adapters; **not** a model-level `streaming` eligibility gate (E03 keeps `streaming=unknown`) |
| Requested audio output modalities in structural fields | `audio_output` | Known mappable when modality fields are present; absent catalog claim → insufficient evidence |
| Temperature, top_p, and other pass-through parameters | *(none)* | No eligibility effect |

#### 3.3.2 Outcome classes per projected requirement

For each projected requirement, eligibility uses exactly one of:

| Class | Meaning | Router action |
| --- | --- | --- |
| Known compatible | Capability claim state `supported` | Requirement satisfied for that candidate |
| Known incompatible | Capability claim state `unsupported` | Exclude candidate (`capability_unsupported`) |
| Insufficient evidence | Claim `unknown`, absent, stale catalog, or missing entry | Cannot treat as compatible; may yield `routing_unavailable` or exclusion with `capability_unknown` |
| Requirement unmapped | Structural feature recognized but no E03 capability exists | Reject route with `requirement_unmapped` (valid request, no compatible resource) |

Do not conflate SSE transport with verified per-model streaming support. Do not invent capabilities beyond the closed E03 vocabulary.

Parameters with no catalog-backed compatibility rule remain pass-through parameters and do not affect eligibility. Do not estimate input tokens from byte length or invent a tokenizer. Cost, latency, quota, quality, and availability estimates are not eligibility inputs.

### 3.4 Insufficient evidence versus ambiguity

Eligibility and selection distinguish **insufficient evidence** from **ambiguity among known eligible candidates**. The precedence is deterministic:

1. **Structural invalidity** → `invalid_request` (§2.3).
2. **Requirement unmapped** → `requirement_unmapped` / `no_eligible_model` per §6.3.
3. **Relevant catalog coverage incomplete** for a conclusion that requires global uniqueness (model-only match, implicit default with cross-provider uniqueness, or proving `model_not_found`) → **`routing_unavailable`**. Incomplete coverage means any relevant provider that passes provider checks lacks fresh accepted catalog evidence, or required capability/limit facts remain unknown for a candidate that would otherwise match.
4. **Two or more eligible candidates** after applying §4 selection rules when catalog coverage **is** complete for the relevant scope → **`routing_ambiguous`**. Lexical enumeration order does not pick a winner.
5. **Exactly one eligible candidate** under complete evidence → select.

**Precedence rule:** When both incomplete coverage and multiple eligible candidates could be argued, **`routing_unavailable` prevails** if any relevant provider catalog is not fresh and accepted for the decision scope. Do not return `routing_ambiguous` based on candidates that cannot be proven unique because coverage is incomplete. Do not return `routing_unavailable` when coverage is complete and two or more candidates are genuinely eligible — that case is always `routing_ambiguous`.

#### 3.4.1 Conceptual test scenarios

| Scenario | Expected code | Rationale |
| --- | --- | --- |
| Explicit model `M` eligible under OpenRouter and OpenCode Go with complete fresh catalogs | `routing_ambiguous` | Two known eligible provider-scoped pairs |
| Explicit model `M` eligible only under OpenRouter; OpenCode Go catalog stale | `routing_unavailable` | Cannot prove absence or uniqueness under OpenCode Go |
| Explicit model `M` absent from all fresh catalogs; every relevant provider fresh | `model_not_found` | Negative evidence is complete |
| Provider-only OpenRouter; two eligible models with complete catalog | `routing_ambiguous` | Multiple eligible models under explicit provider |
| Neither selector; one eligible opencode-go model; complete catalogs | Select opencode-go | Compatibility default |
| Neither selector; zero opencode-go eligible; exactly one eligible OpenRouter model; all relevant catalogs fresh | Select OpenRouter | Unique global candidate with complete coverage |
| Neither selector; two eligible models on opencode-go with complete catalog | `routing_ambiguous` | Multiple eligible models |
| Request requires `json_schema` format; every candidate has `structured_output=unknown` or is excluded | `routing_unavailable` when catalog coverage is complete but no candidate can satisfy the requirement; `no_eligible_model` when at least one candidate is excluded only by `capability_unsupported` |
| Request includes `input_file`; any catalog | `requirement_unmapped` |

## 4. Deterministic precedence and selection

Evaluate one provider/catalog snapshot per request. Enumerate candidates in ascending provider_id/model_id order so the eligibility result is stable. This order is for reproducible output only; it is never a winner-selection tie-break.

| Provider selection | Model selection | Required behavior |
| --- | --- | --- |
| Explicit provider | Explicit model | Require the exact pair to exist and be eligible. If the model exists only under another provider, reject with provider_model_conflict. Never switch provider. |
| Explicit provider | No explicit model | Restrict candidates to that provider. Select only if exactly one eligible model remains; otherwise return no-candidate or routing_ambiguous. |
| No explicit provider | Explicit model | Search exact model_id matches across provider-scoped catalog entries. Select only if exactly one eligible pair matches and every relevant provider catalog is fresh and accepted. Duplicate eligible IDs are ambiguous; incomplete catalog coverage is routing_unavailable. The implicit default does not override an explicit model. |
| No explicit provider | No explicit model | Apply the deterministic compatibility rule: if exactly one eligible opencode-go model exists, select it. Otherwise, use other providers only if the opencode-go catalog is complete and fresh and all other relevant provider catalogs are fresh and accepted; select only if exactly one eligible candidate remains across all providers. In every other multi-candidate case return routing_ambiguous. If opencode-go has no eligible candidate and another provider is uniquely eligible, this is an intentional pre-execution provider selection change from the current hard default. |

An explicit provider or model is a hard constraint, not a preference that may be silently relaxed. If it is unavailable or cannot be satisfied, reject the request. An explicit provider does not cause a different provider to be selected. An explicit model without a provider can identify a provider only when its provider-scoped catalog match is unique.

There are no quality, cost, latency, or quota weights. The router never scores candidates, sorts and picks the first, or uses an LLM. When evidence leaves more than one candidate, ambiguity is the defined result.

### 4.1 Selection origin

selection_origin is one of:

- explicit_provider_and_model
- explicit_provider
- explicit_model
- compatibility_default
- automatic_unique_candidate

The decision's criteria contains stable rule identifiers, not natural-language summaries of message content.

## 5. Health state and availability

| Provider fact | Meaning for eligibility |
| --- | --- |
| Configured disabled | Ineligible: provider_disabled. |
| Credential absent | Ineligible: credential_missing. This is distinguishable from the configured enabled flag without reading or exposing the secret. |
| Runtime disabled | Ineligible: provider_unavailable. |
| Health unknown | Does not block an enabled runtime; no current probe result exists. |
| Health healthy | Does not prove inference or model capability and does not improve a capability claim. |
| Health unhealthy | Does not by itself block an enabled runtime; the model-list probe is not an inference result. |
| Validation label reachable | Probe response wording, not a Manager health state. |

Health is reported as decision context only. It does not trigger an implicit alternate provider. A candidate selected before execution remains the target even if opening or streaming fails.

## 6. Results and error classes

### 6.1 Eligibility result

The in-memory result contains:

- Candidates evaluated in stable order.
- Candidates eligible after all provider and model checks.
- A bounded list of reason codes per excluded candidate.
- Whether a candidate was excluded by objective incompatibility or insufficient information.
- The absence of candidates as an explicit result, never as an empty success.

Minimum reason codes: provider_unknown, provider_disabled, credential_missing, provider_unbound, responses_unsupported, provider_unavailable, model_not_found, model_missing, catalog_stale, capability_unsupported, capability_unknown, limit_exceeded, limit_unknown, and requirement_unmapped.

### 6.2 Routing decision

The decision contains selected provider_id and model_id, selection_origin, ordered criteria, the eligibility result, and request_id. A rejected decision contains no selected target and one stable rejection code. It contains no request content.

### 6.3 Failure categories

| Condition | Category and behavior |
| --- | --- |
| Malformed provider selector or malformed explicit model field | Invalid input; reject as invalid_request. |
| Unknown explicit provider | Missing resource; preserve provider_not_found. |
| Explicit model ID absent from every relevant fresh accepted catalog snapshot | Missing resource; model_not_found. |
| Explicit model ID absent while any relevant provider catalog is never-synced, stale, or incomplete | Insufficient routing evidence; routing_unavailable. |
| Explicit model exists only under a different explicit provider | Selection conflict; provider_model_conflict. |
| Explicit provider disabled, credentialless, unbound, or without Responses support | Operational unavailability; preserve provider_unavailable semantics where currently applicable and return the specific internal reason. |
| Structural request feature requires a normalized capability that E03 does not define (for example file input) | Valid request with unmapped requirement; **`requirement_unmapped`** (§3.3.2). |
| No eligible model because requirements are explicitly unsupported on every candidate | Valid request with no compatible resource; **`no_eligible_model`**. |
| No eligible model because catalog or capability facts are unknown/stale | Insufficient routing evidence; routing_unavailable. |
| More than one candidate remains under the selection rules with **complete** relevant catalog coverage | Ambiguity; routing_ambiguous. |
| More than one candidate appears eligible only because catalog or capability facts are unknown/stale | Insufficient routing evidence; routing_unavailable (§3.4). |
| Selected provider fails after execution starts | Return the existing execution result/error. Do not select or invoke another provider. |

HTTP status mapping for new routing error codes is finalized in E08.4 against the existing gateway error envelope. That mapping must preserve the distinctions above and the existing statuses for invalid provider selectors, unknown providers, and unavailable providers.

## 7. Request preparation boundary

E08.4 places routing before execution.Manager. execution.Target remains the selected provider/model identity; the manager remains responsible for execution lifecycle only.

If the request already names the selected model, E08 preserves the original body bytes. If no model was supplied and routing selects one, the gateway must prepare a provider-ready Responses body whose top-level model is the selected model ID while preserving every other JSON value. It must not rewrite unrelated parameters or use target metadata to silently overwrite a user-selected model.

This is an explicit E08.4 behavior change from the current byte-for-byte pass-through: automatic model selection requires the selected model to reach the provider request. It does not change streaming, cancellation, timeout, response, or execution-record semantics.

## 8. Content-free decision evidence

The routing decision may carry provider/model IDs, source, stable criteria and reason codes, bounded eligible/excluded counts, and the existing request correlation ID. It must not carry prompts, messages, response content, full request bodies, provider session IDs, credentials, or raw provider errors.

E06 schema v1 already records request_id, selected provider_id, and model_id for executions. E08 does not change that schema. E08.5 may emit a bounded structured routing_decision log event through the existing logger for successful and rejected decisions; this is operational logging, not a durable query API. Rejected routes have no E06 execution record. Any durable queryable decision history requires a separately approved E06 schema and storage contract.

## 9. Decision examples

1. X-AIHub-Provider: openrouter plus model: anthropic/claude-sonnet-4.5 selects only that exact OpenRouter pair if eligible.
2. X-AIHub-Provider: opencode-go plus a model found only under OpenRouter returns provider_model_conflict; it does not route to OpenRouter.
3. An explicit model that exists under exactly one provider may select that provider when the header is absent.
4. The same explicit model ID present and eligible under two providers returns routing_ambiguous; provider-scoped identity prevents merging them.
5. With an explicit provider and no model, exactly one eligible model under that provider is selected; two eligible models return routing_ambiguous.
6. With neither selector, one eligible OpenCode Go candidate wins the compatibility rule. If OpenCode Go has no eligible model and exactly one eligible model exists elsewhere with complete relevant catalog coverage, that provider is selected before execution. Two eligible OpenCode Go models return routing_ambiguous, even if one sorts first.
7. A stale catalog or unknown required capability is not treated as a compatible model. If no other eligible candidate exists, return routing_unavailable.
8. After an eligible OpenRouter target is selected, an upstream 401 or stream failure is returned through the existing lifecycle; OpenCode Go receives no retry.

## 10. References

### Repository and platform

- [ADR-009 — Provider-scoped model identity and capability claims](../adr/ADR-009-model-catalog-identity-and-semantics.md)
- [ADR-014 — Execution lifecycle and command contract](../adr/ADR-014-execution-lifecycle-and-command-contract.md)
- [ADR-017 — Execution records storage](../adr/ADR-017-execution-records-storage.md)
- [ADR-021 — Provider selection and OpenRouter runtime](../adr/ADR-021-provider-selection-and-openrouter-runtime.md)
- [E03 Model Catalog](E03-model-catalog.md)
- [E05 Execution](E05-execution.md)
- [C2 OpenRouter compatibility](C2-openrouter-compatibility.md)
- [OpenAI Responses API create reference](https://developers.openai.com/api/reference/typescript/resources/responses/methods/create)
- [OpenAI Responses streaming guide](https://developers.openai.com/api/docs/guides/streaming-responses)
- [OpenAI Responses tools guide](https://developers.openai.com/api/docs/guides/tools)

### External product concepts (informative only)

These sources informed the evolutionary routing vision in SPEC E08. They are **not** requirements for the deterministic MVP.

| Product | Official reference | Documented concepts used for AI Hub evolution |
| --- | --- | --- |
| GitHub Copilot | [About auto model selection](https://docs.github.com/en/copilot/concepts/models/auto-model-selection) | Auto selection from allowed models; task complexity and system health as routing inputs; Efficiency / Balance / Intelligence tiers; policy-respecting model pools; transparency of selected model; no published proprietary weights |
| Cursor | [Cursor Router docs](https://cursor.com/docs/cursor-router) | Auto routing by task fit; Cost / Balance / Intelligence optimization modes; policy and blocked-model handling; SDK `auto-smart` with `optimize_for` |
| Cursor | [How Cursor Router works (blog)](https://cursor.com/blog/how-cursor-router-works) | High-level description of complexity routing and task taxonomy — **architectural inspiration only**; algorithms and thresholds are proprietary and not reproduced here |
