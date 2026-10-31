# ADR-009 — Provider-scoped model identity, capability claims and listing semantics

- **Status:** accepted (E03 design, 2026-10-30)
- **Date:** 2026-10-30
- **Related:** E03 model catalog, SPEC E03 rev 5

## Decision

1. **Identity** is the pair `(provider_id, model_id)`. The upstream model ID is
   preserved verbatim. `canonical_id` is vendor metadata: it is exposed but is
   never a key and never merges models across providers.
2. **Capability claims** are four-dimensional:
   `state ∈ {supported, unsupported, unknown}`,
   `provenance = provider_catalog`,
   `confidence ∈ {explicit, derived, none}`,
   `verification = unverified` (E03 records no execution evidence).
3. The normalized capability vocabulary is closed to `text_generation`,
   `streaming`, `function_tools`, `custom_tools`, `reasoning`, `image_input`,
   `audio_input`, `audio_output` and `structured_output`. Everything else stays
   in `provider_metadata`.
4. **Modality enumerations never produce `unsupported`.** A present modality
   yields `supported`/`derived`; an absent one yields `unknown`/`none`. Only an
   explicit boolean `false` yields `unsupported`/`explicit`.
5. **Listing is not availability.** `presence`, `freshness`, `provider_state`
   and `execution_verification` are separate dimensions. The derived
   `listing_state ∈ {current, stale, missing, unknown}` describes what the
   catalog knows. `current` never asserts that a request will succeed.
6. There is no model-level `enabled` field. Enablement is policy and belongs to
   a later epic with real behavior.
7. **Provider state is consumed read-only.** After the provider module starts,
   its state is `disabled`, `healthy` or `unhealthy`; `unknown` is not produced
   in the current lifecycle. If `unknown` is ever observed, it means missing
   information, not failure: an automatic sync is skipped with no error class,
   and the catalog never rewrites or "corrects" provider state.

## Rationale

- `canonical_id` differs from `id` in 28/28 sampled models and is missing in one;
  it is provider-scoped vendor vocabulary, not a stable AI Hub key.
- The OpenCode Go model payload and its models.dev schema document
  `modalities.input` as "Supported input modalities" but guarantee neither
  exhaustiveness nor completeness. Inferring `unsupported` from absence would
  assert more than the contract supports, so the design prefers `unknown`.
- Boolean capability fields are explicit declarations; a `false` is a declared
  negative, distinct from an absent field.
- The gateway validated that DeepSeek streams and supports custom tools, but
  that evidence is per-request, not a catalog fact. Without a verification
  record, claiming `verified` would be dishonest.

## Consequences

- The catalog may understate modalities that a model supports but the vendor
  dataset omits; this is accepted and documented (SPEC §5.3).
- Consumers must not treat `listing_state=current` as eligibility or as an
  execution guarantee; routing (E08) must consume the separate dimensions and,
  later, verification evidence.
- Adding a normalized capability requires a SPEC revision because the vocabulary
  is a contract.
- Recording verified capabilities later requires a verification source and a
  schema extension; `verification` is already present to carry it.
- `provider_metadata` remains the lossless escape hatch for vendor-specific
  detail.
- Missing provider information (`unknown`) is never converted into a failure
  state; it is skipped defensively and left visible as `unknown`.
