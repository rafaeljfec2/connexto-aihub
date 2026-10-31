# ADR-016 — Execution timeout and HTTP error mapping

- **Status:** accepted
- **Date:** 2026-10-31
- **Related:** SPEC E05, ADR-008, ADR-012

## Context

The provider transport has separate dial, TLS-handshake, response-header, and
stream-inactivity bounds. E02 currently maps every error while opening a
Responses request to `502`, while E05 approves `504` only for a local timeout
waiting for upstream response headers.

## Decision

1. The OpenCode Go adapter observes successful request completion through
   `httptrace.WroteRequest`. A timeout error after the request was fully written
   is classified as the local response-header timeout. Dial, TLS, request-write,
   and other transport errors remain transport failures unless the adapter can
   identify the response-header deadline.
2. The adapter returns a typed, credential-free response-header-timeout error
   for that recognized case. The execution manager records `timed_out(opening)`;
   the gateway maps it to `504 Gateway Timeout` with code
   `execution_timeout` only before response commitment.
3. Other open failures are `failed(transport)` and map to the existing `502`
   local error contract. Upstream HTTP status, permitted headers, and body are
   passed through unchanged; upstream `4xx` and `5xx` are not local failures.
4. Once the gateway commits upstream status and headers, a stream inactivity
   timeout, upstream read error, or client write failure ends the stream and
   records the execution result. No JSON replacement response is written.
5. Preserve ADR-008's inactivity deadline and introduce no total execution
   timeout.

## Rationale

- `httptrace.WroteRequest` distinguishes a deadline after the request has been
  sent from a connect or TLS timeout without relying on error-message parsing.
- Keeping the provider adapter responsible for transport-specific
  classification avoids conflating independent timeout bounds in the manager.
- The gateway remains the sole owner of HTTP commitment and local error
  envelopes.

## Consequences

- Provider tests must cover a delayed response header and a separate dial or
  connection failure.
- The approved `504` behavior is narrow; unknown timeout causes continue to
  use `502`.
- A committed streaming response can only be terminated, never replaced.
