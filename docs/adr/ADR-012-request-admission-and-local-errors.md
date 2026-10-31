# ADR-012 — Gateway request admission limits and local error contract

- **Status:** accepted (E04 design, 2026-10-31)
- **Date:** 2026-10-31
- **Related:** SPEC E04, decisions D6–D8; extends ADR-008 without changing its
  streaming inactivity behaviour.

## Context

E02 forwards the `/v1/responses` body directly to the provider and has no body
limit at that handler. E03 caps catalog sync at 4 KiB inside its handler. Before
E04, an authenticated or unauthenticated caller could stream an unbounded body
into the provider path, and media-type/encoding mistakes surfaced as upstream
behaviour rather than a deterministic local contract.

## Decision

1. **Bounded admission before dispatch.** Protected POST routes are admitted
   before the existing handler. A known `Content-Length` above the effective
   limit is rejected with `413` without reading. Otherwise the gateway reads at
   most `limit + 1` bytes; more than `limit` is `413`. The original body is
   closed, the accepted bytes replace `r.Body` and `r.ContentLength` is updated.
2. **Effective limits.** `/v1/responses` uses
   `AIHUB_GATEWAY_MAX_BODY_BYTES` (default 32 MiB, range 1 KiB–128 MiB).
   `/v1/catalog/sync` uses `min(AIHUB_GATEWAY_MAX_BODY_BYTES, 4096)`, preserving
   the E03 4 KiB bound. Other routes have no body admission because their
   contract has no body.
3. **No decompression and no spooling.** `Content-Encoding` must be absent or
   exactly `identity`; anything else is `415`. Bodies are held in memory only
   for the request and never written to disk.
4. **Media types.** `/v1/responses` requires `application/json`, optionally with
   `charset=utf-8`. A missing or unsupported type is `415`; the original valid
   `Content-Type` value is preserved. `/v1/catalog/sync` keeps an optional empty
   body; a present or unknown-length body requires the same JSON media type.
5. **Header ambiguity.** Duplicate `Authorization` is `401`; duplicate `Origin`
   is `403`; duplicate `Content-Type` or `x-opencode-session` is `400`.
6. **Session admission.** On `/v1/responses`, an absent or empty
   `x-opencode-session` is generated, a valid supplied value is preserved and a
   non-empty invalid value is `400`. This is admission metadata, not JSON
   parsing.
7. **Local error envelope.** Local `400`, `401`, `403`, `413` and `415`
   responses use
   `{"error":{"type":"<type>","message":"<generic message>"}}` with
   `Cache-Control: no-store` and the outer `X-Request-Id`. `401` also carries
   `WWW-Authenticate`. Messages never include the key, bearer value, body,
   query, host/origin echo, internal URL or stack trace.
8. **Upstream provenance is preserved.** An upstream response is passed through
   byte-for-byte with its status, body and `Content-Type`. A local `401` cannot
   replace an upstream `401`, and a local envelope is never applied to an
   upstream response. Existing E02 `502`/`503` and E03 `409`/`502` meanings are
   preserved.
9. **No response buffering.** Admission does not wrap the `ResponseWriter` and
   does not alter `Flush`, `Unwrap` or `http.ResponseController` behaviour. SSE
   stays progressive under ADR-008.

## Rationale

- Reading `limit + 1` bytes proves the bound without trusting a declared length.
- Rejecting compressed bodies avoids a decompression bypass and keeps E04 free of
  a compression subsystem.
- A stable local envelope makes access/admission failures distinguishable from
  provider failures while keeping E02/E03 passthrough intact.
- `Cache-Control: no-store` prevents caches from retaining rejected security
  responses or their correlation metadata.

## Consequences

- A `/v1/responses` request without a media type now returns `415` instead of
  being defaulted to `application/json`.
- A non-empty malformed session value now returns `400` instead of being
  forwarded.
- Accepted bodies may reach the provider with different HTTP transfer framing
  after buffering; the accepted bytes themselves are unchanged.
- Catalog sync bodies above 4 KiB now fail at admission with `413`; catalog
  semantics and error classes remain unchanged.
- Large concurrent requests can consume up to the configured limit each; E04
  intentionally adds no quota, rate limit or spool subsystem.
