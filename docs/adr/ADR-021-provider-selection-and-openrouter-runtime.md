# ADR-021 — Provider selection and OpenRouter runtime

- **Status:** accepted (Phase C2, 2026-11-02)
- **Related:** ADR-009 (provider-scoped model identity), ADR-014 (execution
  lifecycle and command contract), ADR-020 (PostgreSQL persistence)

## Context

The gateway previously executed Responses requests through the OpenCode Go
runtime only. Phase C2 adds OpenRouter while preserving the public request body,
the existing OpenCode Go default, provider administration, and execution
lifecycle contracts.

## Decision

1. `X-AIHub-Provider` selects the provider for `POST /v1/responses`. If absent,
   the gateway selects `opencode-go` for backward compatibility. For
   `GET /v1/models`, if absent, the gateway merges model entries from every
   enabled bound provider that supports model listing; with an explicit selector,
   the native upstream list is passed through unchanged. An invalid or unknown
   selector is rejected; a known provider that is disabled, lacks a credential,
   or lacks the required runtime capability is unavailable.
2. The selector is execution metadata. The JSON `model` value and request body
   are forwarded unchanged. `model_id` retains ADR-009's provider-scoped
   identity `(provider_id, model_id)` and preserves the upstream model ID.
3. The Registry binds runtime instances by provider ID. Persisted configuration
   is applied only to its matching binding. An administrable provider without
   an implementation remains unbound and cannot receive gateway traffic.
4. OpenRouter implements the provider's Responses and model-list operations
   using the configured base URL and the protected provider credential. No
   automatic fallback or OpenAI adapter is introduced.
5. OpenRouter attribution headers remain unset because C1 has no persisted
   application URL or title configuration. They can be added only with an
   explicit configuration contract.

## Rationale

The optional header is the smallest routing addition that preserves existing
clients. Keeping selection outside the opaque body avoids changing the upstream
request and preserves E05's separation between target metadata and payload.
Binding by ID prevents two configurations from sharing credentials or runtime
state.

## Consequences

- Existing Responses clients continue to use OpenCode Go without changes when
  they omit `X-AIHub-Provider`.
- Model pickers can call `GET /v1/models` once to see every enabled provider's
  models; per-provider native lists remain available with `X-AIHub-Provider`.
- Catalog identity remains provider-scoped; equal model IDs from different
  providers remain distinct.
- Provider-level availability and model-level feature support remain separate.
- OpenRouter accepts only capabilities supported by both the Responses
  contract and the selected model; see
  [Phase C2 OpenRouter compatibility](../specs/C2-openrouter-compatibility.md).
