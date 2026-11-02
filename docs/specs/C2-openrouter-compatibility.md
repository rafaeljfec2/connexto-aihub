# Phase C2 OpenRouter compatibility

## Supported contract

The adapter calls `GET /api/v1/models` and `POST /api/v1/responses` relative to
the configured provider base URL. It sends the protected provider API key as a
Bearer credential. Responses request bytes and the supplied content type pass
through unchanged, including the upstream model identifier. The AI Hub execution
lifecycle owns timeout, cancellation, and progressive SSE forwarding.

The provider endpoint and credential come from the existing PostgreSQL-backed
provider configuration. Credentials remain sealed at rest and are never added
to logs. The adapter does not retry or fall back to another model or provider.

## Compatibility limits

- Model features vary. A model listed by OpenRouter is not proof that it
  supports every Responses input, tool, modality, or output option. The adapter
  does not infer or normalize unsupported features. Responses execution errors
  preserve the upstream HTTP status while replacing the upstream error body
  with a generic JSON error. The native model-list route passes successful
  provider responses through and applies the same error-body sanitization.
- The current execution record stores bounded metadata only. It does not retain
  prompts, responses, or OpenRouter response state. Multi-turn continuity must
  be supplied by the caller in each request; the AI Hub does not manage
  `previous_response_id` state.
- The request body is passed through. OpenRouter's Responses schema specifies
  `store` as `false`; callers should omit it or send `false`. The adapter does
  not rewrite a conflicting value.
- OpenRouter documents `HTTP-Referer` and `X-OpenRouter-Title` for attribution.
  They are not sent because this phase has no application URL/title settings.
- Provider-level health is based on the configured endpoint and credential.
  It does not guarantee that a particular model or feature is available.

## References

- [OpenRouter Create a response API](https://openrouter.ai/docs/api/api-reference/responses/create-responses)
- [OpenRouter Models API](https://openrouter.ai/docs/api/api-reference/models/get-models)
- [OpenRouter API key and attribution headers](https://openrouter.ai/docs/app-attribution)

## Operational validation status

The [C2.1 operational validation report](../validation/C2.1-operational-validation-2026-11-02.md)
records the evidence collected on 2026-11-02. Phase C2 remains incomplete:

| Criterion | Result |
| --- | --- |
| OpenCode Go real Responses request | PASS |
| Progressive streaming through OpenCode Go | PASS |
| PostgreSQL server restart and AI Hub recovery | PASS |
| Provider configuration persistence across restart | PARTIAL — the OpenRouter credential was added after the restart test |
| OpenRouter authenticated model listing | PASS — HTTP 200, 415 models |
| OpenRouter real Responses request | BLOCKED — two zero-price model requests returned HTTP 429 |
| OpenRouter progressive streaming | NOT VERIFIED — no successful Responses stream |
| OpenRouter model ID preservation | NOT VERIFIED — both Responses requests returned HTTP 429 |
| Provider isolation and default-selection checks | NOT VERIFIED |
| Credential secrecy during validation | PASS |

The OpenRouter credential was subsequently configured through the protected
provider administration API. Its presence is confirmed without exposing the
value. The runtime diagnostic still reported `enabled=true` with state
`disabled` after the update, while the authenticated model-list request returned
HTTP 200. Do not mark Phase C2 complete until a successful OpenRouter Responses
request and the remaining approved isolation checks have evidence.
