# ADR-011 — Gateway API key authentication and network exposure policy

- **Status:** accepted (E04 design, 2026-10-31)
- **Date:** 2026-10-31
- **Related:** SPEC E04, decisions D1–D5, D12, D15; supersedes the deferred-auth
  intent of ADR-005 without editing ADR-005.

## Context

ADR-005 deferred gateway authentication to E04 while binding to loopback. E02
then introduced a provider credential and a byte-for-byte Responses proxy, so
loopback is no longer a sufficient trust boundary: any local process, browser
origin or DNS-rebinding page could reach the provider credential path. E04 must
make authentication mandatory and make non-loopback exposure an explicit,
validated operator decision.

## Decision

1. **Mandatory static key.** Every startup requires
   `AIHUB_GATEWAY_API_KEY`, including loopback startups. The key is 32–512
   ASCII characters matching the RFC 6750 `b64token` grammar and must differ
   from the enabled OpenCode Go provider key. Missing, malformed or equal keys
   fail startup without disclosing the value.
2. **One authenticated principal.** A successful request represents the single
   static gateway client. There are no users, roles, organizations or tenants.
3. **Constant-time verification.** The presented token is SHA-256 digested and
   compared with the configured digest using `crypto/subtle`. The gateway does
   not use ordinary string equality for authentication.
4. **Strict bearer grammar.** Exactly one `Authorization` header is accepted,
   with the form `Bearer <key>`: case-insensitive scheme, exactly one ASCII
   space and no commas, duplicates, query parameters or cookies. Missing,
   malformed, duplicate and invalid credentials produce the same `401`
   `unauthorized` envelope with `WWW-Authenticate: Bearer realm="ai-hub"`.
5. **Health exemption is exact.** Only `GET /healthz` and `GET /readyz` skip
   authentication. `HEAD` is protected. Both methods remain subject to
   Host/Origin validation.
6. **Host is authoritative.** Host matching uses `r.Host` and an exact,
   normalized allowlist. A request port and IPv6 brackets are normalized away.
   `X-Forwarded-Host`, `Forwarded` and similar headers are not trusted.
7. **Origin is exact and optional.** An absent `Origin` is allowed for
   non-browser clients. A present `Origin` must canonicalize to an exact
   configured `http`/`https` origin. `null`, `*`, malformed values, multiple
   values, credentials, paths, queries and fragments are rejected with `403`.
   No CORS headers and no preflight handling are introduced.
8. **Loopback default; explicit non-loopback opt-in.** The default listener is
   loopback. A non-loopback `AIHUB_SERVER_HOST` requires an explicitly
   configured, non-empty `AIHUB_GATEWAY_ALLOWED_HOSTS`. The actual bound
   listener is re-checked after `net.Listen`; a non-loopback or wildcard bound
   address without that explicit policy closes the listener and fails startup.
   A warning-only mode no longer exists.
9. **Loopback defaults.** When the allowed-host variable is unset, the policy
   uses `localhost`, `127.0.0.1` and `::1`. An explicitly empty value fails
   startup.
10. **Out of scope (D15, expressly approved).** TLS termination, rate limiting,
    key rotation, CORS/preflight, mTLS, IP allowlists and secret managers remain
    out of E04. A non-loopback deployment must use a trusted network, VPN or a
    TLS-terminating reverse proxy; plaintext non-loopback HTTP is an explicitly
    documented operational risk.

## Rationale

- Authentication is the primary control; Host allowlisting alone can be spoofed
  by a client that already knows the key.
- Constant-time digest comparison prevents prefix and length leakage from
  ordinary string comparison.
- Exact Host/Origin matching addresses DNS rebinding and cross-origin access
  without inventing a browser/CORS feature that E04 does not support.
- A post-bind check catches hostnames that resolve to a non-loopback address and
  wildcard binds, which static configuration alone cannot prove.
- Keeping TLS and key rotation out preserves a coherent, testable increment; the
  residual risks are explicit rather than silently accepted.

## Consequences

- Every client must send a gateway bearer token, including local Codex CLI and
  curl invocations.
- A non-loopback bind now fails closed unless both the host list and the listener
  check pass.
- Reversing the ADR-005 assumption is recorded here; ADR-005 remains in place as
  historical context and is not rewritten.
- A stolen static key remains valid until the operator changes the environment
  and restarts the process. Rotation is a later decision.
