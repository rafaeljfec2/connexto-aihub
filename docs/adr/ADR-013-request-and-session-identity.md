# ADR-013 — Request and session identity

- **Status:** accepted (E04 design, 2026-10-31)
- **Date:** 2026-10-31
- **Related:** SPEC E04, decisions D9–D10; formalizes and preserves E02 D3.

## Context

E01 introduced `X-Request-Id` for correlation and logging. E02 introduced
`x-opencode-session` as provider execution context and generated
`aihub-ses-<hex>` values when the header was absent. Those two identifiers can be
confused: correlation is observability metadata, while the provider session is
execution context. E04 adds an authenticated principal and must not imply a
durable AI Hub session or cross-turn continuity it does not implement.

## Decision

1. **`X-Request-Id` is correlation only.** A valid inbound value is reused;
   otherwise a fresh 128-bit hexadecimal identifier is generated. It is echoed,
   stored in the request context and used by logs. It is never authentication,
   authorization or provider session identity.
2. **`x-opencode-session` is provider execution context.** On `/v1/responses`,
   an absent or empty value is replaced by a fresh `aihub-ses-` identifier built
   from 16 random bytes. A supplied value is preserved only when it is 1–128
   characters from `[A-Za-z0-9._:-]`. A non-empty invalid value or duplicate
   header is `400 invalid_request` before upstream.
3. **Independent entropy.** Session generation never derives from
   `X-Request-Id`, and the two values are generated independently.
4. **Per-request lifetime.** The session identifier is not persisted and is not
   used for cross-turn continuity. Two requests carrying the same supplied
   session ID do not acquire shared state through AI Hub.
5. **No durable AI Hub session.** E04 introduces no durable session, no new
   public session header and no users/roles/tenants. The authenticated principal
   is the single static gateway client.
6. **In-memory metadata envelope.** The request context may carry only
   transport/identity metadata: authenticated principal, resolved provider
   session ID, accepted Host and accepted Origin. The correlation ID remains in
   the observability context. The envelope is never serialized into the provider
   JSON payload, never persisted and never changes provider headers beyond the
   existing `x-opencode-session` behaviour.
7. **No payload mutation.** The envelope is not a normalization mechanism. No
   request field is added, removed, reordered, reformatted or re-serialized.

## Rationale

- Keeping the two identifiers separate prevents correlation IDs from becoming
  accidental session keys and preserves the E02 provider contract.
- A bounded, documented session grammar rejects ambiguous values instead of
  forwarding them to a provider whose interpretation is unknown.
- An in-memory, transport-only envelope provides observability and testability
  without inventing durable conversation state.
- Omitting the correlation ID from the envelope keeps `internal/access`
  standard-library-only and avoids duplicating observability context.

## Consequences

- Clients that send malformed session values now receive `400` instead of having
  the value forwarded.
- The generated session remains per-request; multi-turn behaviour relies on the
  client replaying full context, not on AI Hub session state.
- E04 makes no claim of durable sessions, cross-turn continuity or per-user
  attribution.
