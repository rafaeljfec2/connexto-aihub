# ADR-008 — Streaming inactivity deadline

- **Status:** accepted
- **Date:** 2026-10-30
- **Related:** E02 provider management

## Decision

Streaming gateway responses (LLM streams) are bounded by an **inactivity**
deadline instead of the server-wide `write_timeout`. Before the first streamed
chunk the gateway arms a write deadline; after every successfully written chunk
it re-arms the deadline. Upstream silence between chunks is bounded by a reader
watchdog. Ordinary, non-streaming requests keep the existing E01 timeout
semantics.

## Rationale

- LLM responses legitimately stream for longer than any fixed write budget.
- Disconnections and stuck upstreams must not hang the handler forever.
- `http.ResponseController.SetWriteDeadline` re-arms per chunk and the E01
  `responseRecorder` already implements `Unwrap`, so the standard library
  resolves the deadline on the real connection without touching the server.

## Consequences

- Long streams stay valid as long as at least one upstream chunk arrives within the
  inactivity window (default 10 minutes). SSE comments (`: ping`) refresh the
  client connection during model silence and do not reset that watchdog.
- Inactivity expiry closes the upstream body and aborts the stream.
- No global timeout is removed; other endpoints are unaffected.
