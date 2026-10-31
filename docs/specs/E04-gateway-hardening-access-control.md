# SPEC E04 — Gateway Hardening & Access Control

- **Status:** approved — Phase 3 developer approval recorded on 2026-10-31; implementation is authorized by this document under the SDD workflow.
- **Workflow:** SDD.
- **Phase:** 7 — Implementation complete; review notes remediated.
- **Date:** 2026-10-31.
- **Baseline:** `main` at `c9444ce` (E03 closure commit), equal to `origin/main`, clean working tree.
- **Evidence base:** `docs/architecture.md`, `docs/specs/E02-provider-management.md`, `docs/specs/E03-model-catalog.md`, `docs/design/E03-model-catalog-design.md`, ADR-001..ADR-010, `README.md`, `AGENTS.md`, and the current Go implementation inspected during Discovery.
- **Decision records:** ADR-011, ADR-012 and ADR-013 record the design decisions implemented by this SPEC.

## 1. Objective

Harden the AI Hub HTTP gateway so that it can safely proxy provider credentials and expose its API beyond an implicit local-trust assumption. E04 introduces a static gateway API key, a fail-closed network-exposure policy, Host/Origin validation, bounded request admission and an explicit request-identity contract, while preserving the E02 byte-for-byte provider passthrough, SSE streaming and E03 catalog semantics.

E04 is an access-control and admission boundary. It is not a routing, policy, quota, cost, learning or multi-provider epic.

## 2. Scope

| In scope | Out of scope |
| --- | --- |
| Static gateway API key, independent from the provider credential | OAuth, OIDC, users, organizations, RBAC, MFA or tenant isolation |
| Environment-only secret loading and validation | Key rotation UI/API, multiple concurrent gateway keys, secret-manager integration |
| Bearer authentication for every non-health route | Browser login flows, cookies or session authentication |
| Loopback default and explicit non-loopback exposure policy | TLS termination, certificate management or mTLS |
| Host and Origin validation, including DNS-rebinding defense | CORS preflight support or a browser application |
| Actual-listener exposure validation with startup failure | IP allowlists, network ACLs, VPN management or firewall automation |
| Request body limits, media-type checks, header checks and method behavior | Quotas, rate limiting, budget enforcement or abuse scoring |
| Stable local error taxonomy and secret-safe errors | Protocol translation, request/response body normalization or retries/fallback |
| `X-Request-Id`, provider session context and an in-memory metadata envelope | Durable conversation history, durable sessions or cross-turn continuity |
| Compatibility tests for E01/E02/E03 behavior | Semantic changes to E02 passthrough or E03 catalog |
| Real E2E validation with Codex CLI and OpenCode Go | A second provider, generic provider abstraction or model selection/ranking |
| Documentation, configuration examples and ADR-011..ADR-013 | E03 technical-debt refactoring, including files already above 300 lines |

## 3. Baseline and Constraints

### 3.1 Verified baseline

The following facts were verified against the repository during Discovery:

- The current HEAD is `c9444ce` (E03), it equals `origin/main`, and the working tree was clean before this SPEC was created.
- The baseline quality gates were green on 2026-10-31: `go test ./... -count=1 -race`, `go vet ./...`, `staticcheck ./...` and `golangci-lint run ./...`.
- Baseline coverage was: app 100.0%, catalog 90.6%, config 87.6%, gateway 93.3%, health 95.0%, observability 93.8%, providers 92.9%, server 83.1%. `cmd/ai-hub` has no test files and reported 0.0%.
- Routes are wired in `cmd/ai-hub/main.go:77-113`.
- The request pipeline is correlation ID → access logging → panic recovery → `http.ServeMux` (`internal/server/router.go:19-28`).
- A non-loopback listener currently only logs a warning (`internal/server/server.go:70-72`); it does not fail startup and does not authenticate callers.
- `/v1/responses` forwards `r.Body` directly to the provider adapter (`internal/gateway/responses_handler.go:65-87`) and has no request-body limit at that handler.
- Only `POST /v1/catalog/sync` has a body limit, 4 KiB (`internal/gateway/catalog_handler.go:16,195`).
- The provider adapter adds the provider `Authorization` header and `x-opencode-session`; it does not forward the inbound gateway credential (`internal/providers/client.go:74-107`).
- `X-Request-Id` is currently correlation only (`internal/server/middleware.go:13-33`); `x-opencode-session` is provider execution context and is per-request (`internal/gateway/responses_handler.go:26-31,195-213`).
- The provider client is deliberately concrete and single-provider (`internal/providers/client.go:35-42`); E04 must not generalize it.
- ADR-005 explicitly defers gateway authentication and Host/Origin validation to E04 (`docs/adr/ADR-005-loopback-without-gateway-auth.md`).
- The E02 validation report classifies the integration as **PARTIALLY VALIDATED** (`docs/validation/E02-codex-opencode-go-e2e.md:5`). No E03 live-validation artifact exists in `docs/validation/`.

### 3.2 Contracts that must remain true

1. E02 `/v1/responses` request payloads reach the provider byte-for-byte; no JSON parsing, field injection, removal, body normalization or protocol translation.
2. E02 SSE response bytes are forwarded without response buffering, with flushing per copied chunk.
3. E02 upstream status, body and `Content-Type` are preserved for upstream responses, including upstream errors.
4. E02 only mirrors `Content-Type` from upstream headers; no upstream credential or arbitrary header is reflected.
5. The inbound gateway credential is never sent upstream; the provider credential is never sent to a gateway client.
6. E03 catalog identity, capability claims, listing state and persistence behavior do not change semantically.
7. E01 health/readiness status semantics, correlation ID format, access-log privacy and ordinary timeout semantics remain intact.
8. No retry, fallback, model switching, persistence of conversation content or new provider abstraction is introduced.

### 3.3 Documentation and repository divergences

These divergences are recorded here and must not be silently “fixed” by changing E01/E02/E03 behavior:

| ID | Observation | Impact on E04 |
| --- | --- | --- |
| DIV-01 | There is no standalone `docs/specs/E01-*.md`; E01 is the kickoff baseline plus `docs/architecture.md` and the implementation. | E04 treats the kickoff and implemented architecture as the E01 baseline. It does not create or rewrite an E01 SPEC. |
| DIV-02 | ADR-002 describes an optional JSON configuration file, while the code uses defaults plus environment variables only (`internal/config/config.go:1-5,199-219`) and `.env.example:1-2` says `config.json` no longer exists. | E04 uses the implemented environment-only model. ADR-002 is stale and must be noted; correcting E01 documentation is outside this SPEC. |
| DIV-03 | The kickoff’s conceptual ADR numbering conflicts with `docs/adr` numbering (for example, ADR-009 has different meanings in the two documents). | E04 uses the actual `docs/adr` sequence: ADR-011, ADR-012 and ADR-013. |
| DIV-04 | The E03 closure commit message is “Add new model definitions to internal test data…” although the commit contains the full E03 catalog delivery (8,187 insertions across 34 files). | The commit is the authoritative E03 closure point; its message is not used as scope evidence. |
| DIV-05 | E02 is partially validated and E03 has no live-validation artifact. | E04 must not claim E02/E03 live success. E04 evidence must distinguish inherited limitations from E04 validation. |
| DIV-06 | Ten Go files currently exceed the 300-line hard limit: `internal/catalog/manager_test.go` (877), `internal/catalog/store_test.go` (593), `internal/gateway/catalog_handler_test.go` (575), `internal/config/config.go` (456), `internal/gateway/responses_handler_test.go` (454), `internal/catalog/capability_test.go` (442), `internal/catalog/manager.go` (397), `internal/gateway/catalog_handler.go` (386), `internal/server/server_test.go` (378) and `internal/catalog/store.go` (312). | The E03 catalog production/test files are technical debt and must not be refactored. `responses_handler_test.go` and `catalog_handler_test.go` are outside the E04 write set. `config.go` and `server_test.go` require behavior-preserving splits. New E04 files and modified pre-existing files must finish under the limit. |
| DIV-07 | The current non-loopback path is a warning, not a security boundary. | E04 replaces the warning with post-bind validation and startup failure. |
| DIV-08 | E02 defaults a missing `/v1/responses` `Content-Type` to `application/json` (`internal/gateway/responses_handler.go:65-68`). | E04 intentionally rejects a missing or unsupported media type with 415. Accepted requests still preserve the original header and body. |
| DIV-09 | E02 preserves any non-empty inbound `x-opencode-session` value (`internal/gateway/responses_handler.go:195-204`). | E04 validates the value and duplicate headers before it reaches the existing handler. This is an intentional admission hardening; valid E02-compatible values remain unchanged. |
| DIV-10 | E02 forwards the request body directly from the client to the provider adapter. | E04 buffers the body within the configured limit so it can reject oversized requests before the upstream call. The accepted payload bytes remain identical; transfer framing may differ. |
| DIV-11 | `docs/architecture.md:77-80` depicts panic recovery before access logging, while the existing prose and code use correlation ID → access logging → panic recovery (`internal/server/router.go:19-28`). | E04 treats the code and existing prose as authoritative. The implementation updates the architecture diagram without changing E01 behavior. |

## 4. Requirements

### 4.1 Functional requirements

#### Configuration and startup

| ID | Requirement |
| --- | --- |
| FR-001 | Gateway settings are environment-only and use the names defined in §5. They are loaded by the existing defaults-plus-environment path; no JSON configuration file is introduced. |
| FR-002 | `AIHUB_GATEWAY_API_KEY` is required for every startup. It must contain 32–512 characters matching the RFC 6750 `b64token` grammar: at least one character from `[A-Za-z0-9._~+/-]`, optionally followed by trailing `=` padding. Spaces, control characters, commas, semicolons, colons, Unicode and all other characters are rejected. |
| FR-003 | The gateway key is independent from `AIHUB_PROVIDER_OPENCODEGO_API_KEY`. If the provider is enabled and both values are byte-equal, startup fails with a secret-safe configuration error. |
| FR-004 | The gateway key is never written to repository files, snapshots, logs, error messages, structured fields, command-line arguments, response bodies or the catalog. A validation error must identify the setting, never the value. |
| FR-005 | Allowed hosts, allowed origins and the maximum body size are parsed and validated before startup. Invalid syntax, wildcards, duplicate conflicting entries, an empty explicitly configured host list and out-of-range sizes fail startup. |
| FR-006 | Configuration validation aggregates all independent semantic errors, matching the existing `Config.Validate` behavior, without including secret values. Existing environment parse errors remain first-error failures and are not changed by E04. |
| FR-007 | A configuration error prevents listener creation when it can be detected statically, or prevents serving when the actual bound listener reveals an unsafe exposure. In both cases the process exits non-zero. |
| FR-008 | The existing `AIHUB_SERVER_HOST` default remains `127.0.0.1`. A loopback listener is the default. |

#### Authentication and authorization

| ID | Requirement |
| --- | --- |
| FR-009 | Every route except exactly `GET /healthz` and `GET /readyz` requires successful gateway authentication. This includes `/v1/models`, `/v1/responses`, `/diagnostics/providers`, all `/v1/catalog/*` routes and unknown paths. `HEAD /healthz` and `HEAD /readyz` are not exempt and fail closed with `401` when unauthenticated. |
| FR-010 | Authentication uses exactly one `Authorization` header with the grammar `Bearer <key>`: case-insensitive scheme, exactly one ASCII space, and no additional credentials, commas or duplicate header values. |
| FR-011 | Missing, malformed, duplicate or invalid credentials produce the same `401 unauthorized` response. The gateway must not reveal whether the key length, prefix or route was correct. |
| FR-012 | Credential comparison is constant-time over fixed-size digests. Implementations must not use ordinary string equality or leak the configured key through timing behavior. |
| FR-013 | A local `401` response includes `WWW-Authenticate: Bearer realm="ai-hub"`, `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`, the correlation ID and the generic error envelope from §6.6. |
| FR-014 | Gateway authentication never accepts a key from a query string, cookie, `X-API-Key` alias or request body. Those mechanisms are not part of the contract. |
| FR-015 | Authentication is evaluated before any request body is read and before any provider or catalog operation. An unauthenticated request must cause zero upstream calls and zero catalog mutations. |
| FR-016 | The inbound gateway `Authorization` value is never forwarded upstream. The provider adapter continues to use only the provider credential. The gateway key and provider key must be distinct. |
| FR-017 | Exact `GET /healthz` and `GET /readyz` remain unauthenticated for liveness/readiness probes, but remain subject to Host and Origin validation. Their JSON bodies and status semantics remain those of ADR-006. |
| FR-018 | A request to a protected path with a wrong method is authenticated first; after successful authentication, the existing `http.ServeMux` behavior may return `405` with an `Allow` header. Unauthenticated requests receive `401`, not route/method information. |

#### Network exposure

| ID | Requirement |
| --- | --- |
| FR-019 | Loopback binding remains the default and is permitted without an additional “allow external” flag. |
| FR-020 | A non-loopback bind is permitted only when the operator explicitly configures `AIHUB_SERVER_HOST` to a non-loopback host and explicitly provides a non-empty `AIHUB_GATEWAY_ALLOWED_HOSTS`. An unset or empty allowed-host setting on a non-loopback listener is an unsafe configuration. |
| FR-021 | The effective listener address is checked after `net.Listen`, not only the configured host string. A hostname that resolves to a non-loopback address, `0.0.0.0`, `::` or an IPv4-mapped wildcard must trigger the same policy check. |
| FR-022 | If the actual listener is non-loopback and the exposure policy is not explicit and valid, the listener is closed and startup returns an error before `Serve` accepts requests. A warning alone is not sufficient. |
| FR-023 | Host matching uses `r.Host` and an exact allowlist. Configuration entries are lowercase DNS names or IP literals without schemes, paths, user information, wildcards or ports; requests may include a port and IPv6 brackets, which are normalized away before comparison. |
| FR-024 | When the listener is loopback and `AIHUB_GATEWAY_ALLOWED_HOSTS` is unset, the default host allowlist is `localhost`, `127.0.0.1` and `::1`. If the variable is explicitly set to an empty list, startup fails. |
| FR-025 | `X-Forwarded-Host`, `Forwarded`, `X-Forwarded-Proto` and similar proxy headers are not trusted for Host/Origin decisions. A reverse proxy must preserve or rewrite the request authority, and the operator must configure the resulting host/origin explicitly. |
| FR-026 | If an `Origin` header is absent, a non-browser request is permitted to proceed to authentication and admission. If an `Origin` header is present, it must parse as an absolute `http` or `https` origin with no user information, path, query or fragment, and must exactly match the configured allowlist after canonicalization. |
| FR-027 | `Origin: null`, malformed origins, wildcard configuration and multiple Origin header values are rejected with `403 forbidden`. The gateway does not add `Access-Control-Allow-Origin` and does not implement CORS preflight in E04. |
| FR-028 | Host/Origin rejection happens before authentication and body admission. The response is generic and must not echo the rejected host or origin. |
| FR-029 | Health and readiness requests with a disallowed Host or Origin are rejected with `403`; health is not an unauthenticated bypass of the network policy. |
| FR-030 | A safe non-loopback E2E scenario must bind a non-loopback listener with authentication and an explicit allowed-host list, then exercise it from loopback/local tooling without exposing it to the public internet. |

#### Request admission and limits

| ID | Requirement |
| --- | --- |
| FR-031 | `AIHUB_GATEWAY_MAX_BODY_BYTES` bounds request bodies for protected POST routes. The default is 33,554,432 bytes (32 MiB); allowed explicit values are 1,024–134,217,728 bytes (1 KiB–128 MiB). |
| FR-032 | If a known `Content-Length` exceeds the effective limit, the gateway returns `413 payload_too_large` before reading the body or contacting upstream. For an unknown-length/chunked body, it reads at most `limit + 1` bytes and returns `413` if the limit is exceeded. |
| FR-033 | A request that exceeds the limit must never be partially forwarded to the provider. The admission layer buffers at most the configured limit plus the minimum sentinel byte, and then either forwards the exact accepted bytes or rejects the request. |
| FR-034 | Request buffering is bounded, in-memory and per request. E04 does not introduce a temp-file spool, unbounded body buffer or request-body streaming path that could bypass the limit. Response bodies are not buffered by this mechanism. |
| FR-035 | `POST /v1/responses` requires a JSON media type: `application/json` with no parameters or with `charset=utf-8` only. Missing, malformed or other media types return `415 unsupported_media_type`. The original valid `Content-Type` value is preserved when forwarded. |
| FR-036 | `POST /v1/catalog/sync` keeps its optional empty body. If the request has a body or unknown/chunked length, it must use the JSON media type from FR-035; an explicit `Content-Length: 0` may omit the header. Its effective body limit is `min(AIHUB_GATEWAY_MAX_BODY_BYTES, 4096)` to preserve E03’s 4 KiB bound. |
| FR-037 | `Content-Encoding` other than absent or `identity` is rejected with `415` on protected POST routes. E04 performs no request decompression, so compressed bodies cannot bypass the size limit. |
| FR-038 | Multiple `Authorization` values yield `401`; multiple `Origin` values yield `403`; multiple `Content-Type` values yield `400 invalid_request`; multiple `x-opencode-session` values yield `400 invalid_request`. Header names are compared case-insensitively. |
| FR-039 | The existing 64 KiB `MaxHeaderBytes` limit remains in force. A request rejected by the HTTP server for excessive headers may use the server’s standard status/body; the gateway must not add secret or internal detail. |
| FR-040 | Method mismatches on a protected path, after successful authentication, preserve the standard `405` plus `Allow` behavior. `405`, `414` and `431` are server-generated responses and are not required to use the gateway JSON error envelope. |
| FR-041 | Request-body reads honor `r.Context()` and the existing server read timeout. Client cancellation, a read timeout or an unreadable body must terminate the request without an upstream call. |
| FR-042 | The body limit applies to the bytes received from the client after transfer decoding. It is not a response limit, a total traffic quota or a concurrency limit. |

#### Error contract

| ID | Requirement |
| --- | --- |
| FR-043 | Local gateway failures use the stable envelope `{"error":{"type":"<type>","message":"<generic message>"}}` for `400`, `401`, `403`, `413`, `415`, `502` and `503`. |
| FR-044 | The stable local type/status mapping is: `invalid_request`/400, `unauthorized`/401, `forbidden`/403, `payload_too_large`/413, `unsupported_media_type`/415, `upstream_unavailable`/502 and `provider_unavailable`/503. The 503 mapping applies to E02 routes only; E03 catalog sync preserves `409 provider_disabled`. The 502 mapping applies to E02 and to E03 catalog sync when all requested providers fail. Existing E02/E03 messages for these statuses remain valid. |
| FR-045 | A local `401` and an upstream `401` remain distinguishable by provenance: the local response has the gateway’s `WWW-Authenticate` and local envelope; the upstream response is the unchanged E02 passthrough with upstream status/body/`Content-Type`. No local error wrapper is applied to an upstream response. |
| FR-046 | Local errors never include the gateway key, provider key, `Authorization` header, request body, query string, raw upstream error, internal URL, stack trace or filesystem path. Logs may contain a sanitized cause and correlation ID. |
| FR-047 | Local `400`, `401`, `403`, `413` and `415` responses include `Cache-Control: no-store` and the correlation ID established by the outer middleware. This includes admission-level `400` responses on catalog routes. |
| FR-048 | The existing panic recovery remains the final safety net. If it must emit a generic `500`, it must not add the panic value or stack trace to the response. |

#### Request and session identity

| ID | Requirement |
| --- | --- |
| FR-049 | `X-Request-Id` is correlation only. A valid inbound value may be reused; an invalid value is replaced with a fresh 128-bit hexadecimal ID. It is echoed in the response, stored in the request context and used for logs. It is never authentication, authorization or provider session identity. |
| FR-050 | `x-opencode-session` is provider execution context, not gateway authentication and not an AI Hub durable session. On `/v1/responses`, an absent or empty value is replaced by a fresh `aihub-ses-` identifier generated from 16 random bytes; a supplied value is preserved only when it is 1–128 characters from `[A-Za-z0-9._:-]`; a non-empty invalid value or duplicate session header returns `400 invalid_request` before upstream. |
| FR-051 | The session identifier is generated independently from `X-Request-Id`, is per-request, is not persisted and is not used for cross-turn continuity. E04 makes no claim that two requests with the same supplied session ID share state. |
| FR-052 | E04 introduces no durable AI Hub session identity and no new public session header. The authenticated principal is the single static gateway client; there are no users, roles, organizations or tenants. |
| FR-053 | An in-memory metadata envelope may carry only transport/identity metadata: correlation ID, authenticated principal, resolved provider session ID, accepted Host and accepted Origin. It is not added to the JSON payload, is not persisted, and does not modify provider request headers beyond the existing `x-opencode-session` behavior. |
| FR-054 | The metadata envelope is not a body-normalization mechanism. No request body field is added, removed, reordered, reformatted or re-serialized by the envelope. |

#### Compatibility and operations

| ID | Requirement |
| --- | --- |
| FR-055 | For every accepted `/v1/responses` request, the bytes delivered to the provider body must equal the bytes received from the client after HTTP transfer decoding, including whitespace, field order and duplicate JSON keys. |
| FR-056 | `/v1/responses` must preserve SSE bytes and progressive delivery. No response writer wrapper may hide `Flush`, `Unwrap` or `http.ResponseController` behavior, and no response body buffering, chunk merging or SSE parsing is introduced. |
| FR-057 | `GET /v1/models` remains the E02 provider passthrough. Its body, status and `Content-Type` behavior changes only by the new authentication and Host/Origin gates. |
| FR-058 | E03 catalog routes, domain semantics, snapshot format, counters, quarantine behavior and catalog error types remain semantically unchanged. E04 may add HTTP admission before those handlers but must not reinterpret catalog data. |
| FR-059 | E02 and E03 test suites must remain green. Test-only fixture updates needed because the gateway key is now mandatory are allowed; production behavior and assertions must not be weakened or removed. |
| FR-060 | `.env.example`, `README.md`, `docs/architecture.md` and the ADR index are updated during implementation to document the new configuration and the authenticated invocation path. E01/E02/E03 SPEC files are not rewritten. |
| FR-061 | New and modified Go source files, including tests, must not exceed 300 physical lines; Markdown files are excluded. If a file would exceed 250 lines, implementation must stop and plan an SRP split. A pre-existing Go file above 300 lines that must be modified is split without behavior change before E04 logic is added. |
| FR-062 | No new Go module dependency, framework, database, queue, metrics subsystem or OpenTelemetry subsystem is introduced. E04 uses the Go standard library and existing project packages. |
| FR-063 | E04 completion requires a real E2E validation artifact under `docs/validation/` covering the scenarios in §9.4. Mock-only evidence cannot establish the real-integration portion of the Claim. |

### 4.2 Non-functional requirements

| ID | Requirement |
| --- | --- |
| NFR-001 | Fail closed: any uncertainty in authentication, Host/Origin policy or actual listener exposure must reject the request or abort startup, never fall back to unauthenticated operation. |
| NFR-002 | Secret safety: the gateway key must not be recoverable from logs, errors, HTTP responses, configuration dumps, test output or validation artifacts. |
| NFR-003 | Streaming safety: authentication and admission must not add response buffering; accepted SSE traffic must remain progressive and bounded by the E02 inactivity policy. |
| NFR-004 | Memory safety: request buffering is bounded by the configured body limit; the implementation must not retain request bodies after the request or create unbounded per-request metadata. |
| NFR-005 | Concurrency safety: configuration and policy objects are immutable after startup; request metadata is request-scoped; the implementation must pass the race detector. |
| NFR-006 | Observability safety: existing access logs continue to omit headers, query strings and bodies; security errors are observable by status and correlation ID without logging credentials. |
| NFR-007 | Maintainability: access decisions are pure, testable functions where possible; the HTTP middleware is thin; policy parsing is isolated from request enforcement. |
| NFR-008 | Testability: Host/Origin canonicalization, key comparison, body limits and startup exposure checks must be testable without public network access; the body-preservation test must use a controllable fake upstream. |
| NFR-009 | Backward compatibility: E01 health/timeout semantics, E02 accepted passthrough and E03 catalog semantics remain observable after the new gates. |
| NFR-010 | Coverage: new E04 code has at least 80% line coverage, with justified uncovered branches. |
| NFR-011 | Evidence honesty: every E2E limitation (Codex CLI, environment, provider, sandbox or streaming rendering) is reported explicitly and is not converted into a pass. |
| NFR-012 | Portability: listener classification must handle IPv4, IPv6, IPv4-mapped IPv6 and wildcard addresses consistently on Linux; unsupported address families fail closed. |

## 5. Configuration

### 5.1 New settings

| Variable | Required | Default | Validation | Secret |
| --- | --- | --- | --- | --- |
| `AIHUB_GATEWAY_API_KEY` | Yes | None | 32–512 total characters, with at least one non-padding character from `[A-Za-z0-9._~+/-]` and only trailing `=` padding; no spaces, control characters, commas, semicolons, colons or Unicode; not equal to the configured provider key | Yes |
| `AIHUB_GATEWAY_ALLOWED_HOSTS` | Loopback: no; non-loopback: yes | Loopback names `localhost`, `127.0.0.1`, `::1` when unset | Comma-separated; 1–64 entries; DNS name or IP literal; no port, scheme, path, wildcard or user information; explicit empty value fails | No |
| `AIHUB_GATEWAY_ALLOWED_ORIGINS` | No | Empty | Comma-separated; 1–64 entries when non-empty; absolute `http`/`https` origin; no user information, path, query or fragment; no `*`; explicit empty is allowed | No |
| `AIHUB_GATEWAY_MAX_BODY_BYTES` | No | `33554432` | Integer in `[1024, 134217728]`; invalid values fail startup | No |

### 5.2 Existing setting that participates in the policy

| Variable | Existing default | E04 behavior |
| --- | --- | --- |
| `AIHUB_SERVER_HOST` | `127.0.0.1` | A non-loopback value is an explicit opt-in, but it is only valid when `AIHUB_GATEWAY_ALLOWED_HOSTS` is explicitly non-empty. The actual bound listener is re-checked after bind. |

### 5.3 Configuration semantics

- The existing dotenv loader may populate the process environment before `config.Load`; this does not make the setting file-based. Secrets must never be committed in `.env.example`.
- `AIHUB_GATEWAY_ALLOWED_HOSTS` absent means “use loopback defaults” only for a loopback listener. It does not authorize a non-loopback listener.
- `AIHUB_GATEWAY_ALLOWED_ORIGINS` absent or empty means “no browser origin is allowed”; non-browser requests without `Origin` remain allowed after authentication.
- Entries are normalized deterministically for comparison: lowercase scheme/host, default ports removed for origins, hostname lowercased, optional trailing dot removed, IPv6 brackets removed for host entries, and request port stripped for Host matching.
- Duplicate normalized entries are de-duplicated. A configuration that becomes empty after normalization fails.
- Existing environment parse errors retain the current first-error behavior. E04 semantic range, equality and cross-field errors are aggregated through `Config.Validate` and reported without secret values.
- Unknown `AIHUB_*` variables continue to produce the existing warning; their values are never logged.

### 5.4 Example

```env
# Required: 32-512 characters from [A-Za-z0-9._~+/-]. Generate a private value,
# for example with `openssl rand -base64 48`; never commit the real value.
AIHUB_GATEWAY_API_KEY=

# Only needed when AIHUB_SERVER_HOST is non-loopback or when the default
# loopback names are insufficient. Must be explicit for non-loopback binds.
AIHUB_GATEWAY_ALLOWED_HOSTS=localhost,127.0.0.1,::1

# Empty by default. Add exact origins only for a browser client.
AIHUB_GATEWAY_ALLOWED_ORIGINS=

# 32 MiB default; the accepted range is 1 KiB to 128 MiB.
AIHUB_GATEWAY_MAX_BODY_BYTES=33554432
```

## 6. HTTP Contract

### 6.1 Middleware order

The effective pipeline must be:

```text
X-Request-Id validation/generation
        |
        v
access logging
        |
        v
panic recovery
        |
        v
Host/Origin validation
        |
        v
gateway authentication (except exact GET /healthz and GET /readyz; HEAD is protected)
        |
        v
route-specific header/media/body admission
        |
        v
http.ServeMux and existing handlers
        |
        v
provider/catalog behavior
```

Precedence rules:

1. Correlation ID is established first.
2. Access logging and panic recovery surround every security decision.
3. Host/Origin failures win over authentication failures.
4. Authentication failures win over route, media-type and body-size failures for protected routes.
5. Only exact `GET /healthz` and `GET /readyz` skip authentication; `HEAD` is protected. Both methods remain subject to Host/Origin validation.
6. All security decisions complete before any provider call, catalog mutation or request-body read beyond the minimum needed for admission.
7. New response writers must preserve `http.Flusher`, `Unwrap` and `http.ResponseController` behavior. If a wrapper cannot preserve them, it must not wrap streaming responses.

### 6.2 Route protection matrix

| Method | Path | Authentication | Host/Origin | Body limit | Media type |
| --- | --- | --- | --- | --- | --- |
| GET | `/healthz` | No | Yes | No | No |
| GET | `/readyz` | No | Yes | No | No |
| HEAD | `/healthz` | Yes | Yes | No | No |
| HEAD | `/readyz` | Yes | Yes | No | No |
| GET | `/v1/models` | Yes | Yes | No | No |
| POST | `/v1/responses` | Yes | Yes | Yes | `application/json` |
| GET | `/diagnostics/providers` | Yes | Yes | No | No |
| GET | `/v1/catalog/models` | Yes | Yes | No | No |
| GET | `/v1/catalog/models/{provider}/{model...}` | Yes | Yes | No | No |
| GET | `/v1/catalog/providers` | Yes | Yes | No | No |
| POST | `/v1/catalog/sync` | Yes | Yes | Yes, capped at 4 KiB | JSON if body present |
| Any | Unknown path | Yes | Yes | No | No |

### 6.3 Authentication contract

Accepted header:

```http
Authorization: Bearer <gateway-api-key>
```

Rejected forms include, but are not limited to: no header, empty token, `Basic`, `Bearer` with no token, multiple spaces, tabs, a comma-separated list, duplicate headers, a query parameter, a cookie, or a key that violates the FR-002 `b64token`/padding grammar.

Local rejection:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json; charset=utf-8
Cache-Control: no-store
WWW-Authenticate: Bearer realm="ai-hub"
X-Request-Id: <correlation-id>

{"error":{"type":"unauthorized","message":"authentication required"}}
```

The gateway does not distinguish “missing” from “invalid” in status, type or message. Timing behavior must not reveal a valid prefix or key length.

### 6.4 Host and Origin contract

- Host is read from `r.Host`; URL host or proxy-forwarded headers are not trusted.
- The request Host may include a port; matching uses the normalized hostname/IP.
- A missing, malformed or non-allowlisted Host returns `403 forbidden`.
- An absent `Origin` is allowed for non-browser clients after Host validation.
- A present `Origin` must exactly match a configured origin after canonicalization. `null`, `*`, malformed values, multiple values, credentials, paths, queries and fragments are rejected with `403`.
- No CORS response headers are emitted. Preflight `OPTIONS` is not implemented; a browser client that requires preflight is out of scope.
- Health and readiness are not exempt from Host/Origin.

Local rejection:

```http
HTTP/1.1 403 Forbidden
Content-Type: application/json; charset=utf-8
Cache-Control: no-store
X-Request-Id: <correlation-id>

{"error":{"type":"forbidden","message":"request host or origin is not allowed"}}
```

The rejected value is not echoed.

### 6.5 Admission contract

For protected POST routes:

1. Reject non-identity `Content-Encoding` with `415`.
2. Reject duplicate `Content-Type` or session headers with `400`.
3. For `/v1/responses`, resolve `x-opencode-session`: absent or empty generates a new value; one syntactically valid value is preserved; a non-empty malformed value returns `400`.
4. For `/v1/responses`, require a JSON media type before reading the body; reject missing or unsupported media types with `415`.
5. For `/v1/catalog/sync`, allow an explicit empty body; if a body or an unknown-length body is present, require the JSON media type.
6. Reject an oversized known `Content-Length` with `413` before reading.
7. Otherwise read at most `effective_limit + 1` bytes. If more than `effective_limit` bytes are available, return `413` and do not call upstream.
8. Pass exactly the accepted bytes to the existing handler, preserving body bytes and the original valid `Content-Type` string.

Effective limits:

- `/v1/responses`: `AIHUB_GATEWAY_MAX_BODY_BYTES`.
- `/v1/catalog/sync`: `min(AIHUB_GATEWAY_MAX_BODY_BYTES, 4096)`.
- All other routes: no request-body admission limit because their contract has no body.

The body limit is enforced on bytes received after HTTP transfer decoding. E04 does not decompress content and does not spool bodies to disk.

In this SPEC, “malformed” at admission level means malformed or duplicated admission headers, invalid media metadata or an invalid session value. It does not mean JSON syntax validation. `/v1/responses` JSON syntax is validated by the upstream provider and its bytes remain unparsed locally. `/v1/catalog/sync` JSON syntax remains validated by the existing E03 handler. HTTP framing rejected by `net/http` remains a server-generated response outside the local-envelope guarantee.

Intentional behavior deltas from the current E02 boundary are limited to admission and access control: a missing `/v1/responses` media type becomes `415` instead of being defaulted; non-empty invalid session values are rejected with `400` instead of being forwarded; and accepted request bodies may be forwarded with different transfer framing after bounded buffering. The accepted body bytes, provider auth separation and response passthrough remain unchanged.

### 6.6 Error taxonomy

| Status | `error.type` | Trigger | Body/provenance |
| --- | --- | --- | --- |
| 400 | `invalid_request` | Admission-invalid headers or media metadata, malformed/duplicate session header, or existing handler validation failure (including catalog sync JSON syntax) | Local generic envelope; never an upstream body |
| 401 | `unauthorized` | Missing, malformed, duplicate or invalid gateway credential | Local envelope + `WWW-Authenticate` |
| 403 | `forbidden` | Disallowed Host/Origin or multiple Origin values | Local generic envelope |
| 405 | `method_not_allowed` | Method mismatch after successful authentication | Server-generated; may carry `Allow` and may not use the JSON envelope |
| 413 | `payload_too_large` | Effective request-body limit exceeded | Local generic envelope |
| 415 | `unsupported_media_type` | Missing/unsupported JSON media type or non-identity `Content-Encoding` | Local generic envelope |
| 502 | `upstream_unavailable` | Provider connection/setup failure on E02, or all requested providers failed on E03 catalog sync | Existing E02/E03 safe envelope for the route |
| 503 | `provider_unavailable` | Provider disabled on E02 routes, preserved from E02; E03 catalog sync keeps `409 provider_disabled` | Existing E02 safe envelope |
| Upstream status | Upstream-defined | Any upstream HTTP response | Exact E02 passthrough: status, body and `Content-Type`; no local wrapper |

`404`, `414`, `431` and a last-resort `500` remain possible server/handler responses. They must not reveal secrets or internal details. E04 does not require every server-generated status to use the gateway JSON envelope. The local `400` is not a claim that `/v1/responses` JSON syntax is parsed by the gateway.

### 6.7 Identity contract

| Identifier | Owner | Lifetime | Meaning and constraints |
| --- | --- | --- | --- |
| `X-Request-Id` | AI Hub observability | One HTTP request | Correlation only; valid inbound values are reused, otherwise generated; echoed in response; never authentication, authorization or provider session identity. |
| Authenticated principal | AI Hub access control | One request | Constant single-client principal derived from successful static-key authentication; no users, roles or tenants. |
| `x-opencode-session` | Provider integration / AI Hub | One provider execution request | Provider execution context; absent/empty value generated as `aihub-ses-<32 hex>`; valid supplied value preserved; non-empty malformed or duplicate value rejected with `400`; per-request only; not derived from `X-Request-Id`. |
| AI Hub durable session | Not implemented | N/A | E04 does not create, persist or claim a durable AI Hub session. Cross-turn continuity is not implemented. |

The in-memory metadata envelope may contain the correlation ID, principal, provider session ID and accepted transport identity. It is never serialized into the provider payload and is never persisted.

### 6.8 Payload and streaming compatibility

- The `/v1/responses` request body is buffered only to enforce the admission limit, then forwarded as the exact accepted byte sequence. The adapter’s existing body handling and header injection remain unchanged.
- The provider adapter continues to set the provider `Authorization` header and `x-opencode-session`; it never forwards the client’s gateway `Authorization`.
- Response status, body bytes and `Content-Type` from the provider are preserved as in E02. Only the existing allowlisted response header is mirrored.
- SSE responses are copied and flushed per chunk. The new middleware must not wrap the response writer in a way that hides `http.Flusher` or `Unwrap`.
- No protocol translation, body normalization, JSON re-serialization, model substitution, retry or fallback is introduced.
- Upstream errors remain upstream errors. A local `401` is not allowed to replace an upstream `401` or vice versa.

### 6.9 Health and readiness

- `GET /healthz` and `GET /readyz` remain unauthenticated.
- `HEAD /healthz` and `HEAD /readyz` are not exempt; unauthenticated requests receive `401`.
- Both still require an allowed Host and, when present, an allowed Origin.
- Their response bodies, status codes and `Cache-Control: no-store` behavior remain ADR-006 semantics.
- A missing gateway key prevents startup, so health endpoints are not available as an unauthenticated fallback for a misconfigured process.

## 7. Architecture and Package Boundaries

### 7.1 Package responsibilities

| Package/path | E04 responsibility | Constraints |
| --- | --- | --- |
| `internal/config` | Add typed gateway settings, parsing, normalization and cross-field validation | No secret values in errors; retain existing precedence and aggregation |
| `internal/access` (new) | Pure Host/Origin canonicalization, bearer parsing, constant-time credential verification, body-admission policy and request metadata | Standard library only; no provider/catalog imports; split files to remain under 300 lines |
| `internal/server` | Insert the access middleware in the correct pipeline position and validate the actual listener before serving | Preserve correlation/logging/recovery/streaming behavior; keep existing `NewRouter` behavior available where possible |
| `internal/gateway` | Existing handlers consume the buffered request body; no auth logic added to handlers | Preserve byte-for-byte passthrough and E03 catalog behavior |
| `internal/providers` | No semantic changes; provider credential remains the only upstream credential | Do not add a provider abstraction or second provider |
| `cmd/ai-hub` | Wire configuration, access policy and server; keep lifecycle order explicit | No secret logging; no new module lifecycle semantics |
| `docs/` and root documentation | Update README, architecture, env example and ADR index; add E04 validation artifact during verification | Do not rewrite E01/E02/E03 SPECs; keep repository artifacts in English |

### 7.2 Expected affected files

The Design phase may refine names, but the expected write set is:

| File | Expected change |
| --- | --- |
| `cmd/ai-hub/main.go` | Construct access policy and pass it to the HTTP server/router |
| `internal/config/config.go` | Reduce from 456 to at most 300 lines through a behavior-preserving SRP split before adding E04 settings |
| `internal/config/{types,environment,validate}.go` (new split files, names finalized in Design) | Receive the existing log types, environment/key application and validation helpers without semantic change |
| `internal/config/gateway.go` (new) | Gateway settings, parsing, normalization and cross-field validation |
| `internal/config/{gateway,gateway_secret}_test.go` (new, split as needed) | Gateway validation, key alphabet, secret-safety and aggregation tests; each file must stay below 300 lines; existing small config tests may receive fixture-only updates |
| `internal/access/*.go` (new) | Authentication, Host/Origin, admission and metadata helpers plus tests |
| `internal/server/server.go` | Actual-listener validation and fail-closed startup |
| `internal/server/router.go` | Inject/execute access middleware without hiding streaming interfaces |
| `internal/server/middleware.go` | Preserve correlation/logging/recovery behavior; no secret-bearing fields |
| `internal/server/server_test.go` | Reduce from 378 to at most 300 lines through a behavior-preserving test split before E04 tests are added |
| `internal/server/{server_lifecycle,router}_test.go` (new split files, names finalized in Design) | Preserve the existing start/stop, correlation, logging and recovery assertions |
| `internal/server/{access_pipeline,access_listener,access_streaming}_test.go` (new, split as needed) | Pipeline order, exact-GET/HEAD health behavior, listener exposure, streaming-wrapper, auth-gated integration and error/secret capture tests; each must stay below 300 lines |
| `.env.example` | Document the four new settings without a real secret |
| `README.md` | Authenticated invocation examples, configuration and limitations |
| `docs/architecture.md` | New pipeline, trust boundary and access-control description |
| `docs/adr/README.md` | Register ADR-011..ADR-013 |
| `docs/adr/ADR-011-*.md` | Gateway authentication and network exposure policy |
| `docs/adr/ADR-012-*.md` | Request admission limits and local error contract |
| `docs/adr/ADR-013-*.md` | Request/session identity |
| `docs/validation/E04-*.md` | Real E2E evidence and explicit limitations (created during verification) |

The E02/E03 production handlers are expected to remain unchanged because the admission layer can replace `r.Body` before dispatch. If implementation proves that a handler change is unavoidable, it must preserve the E02/E03 contracts and be called out as a scope exception before coding.

The 300-line work plan is mandatory, not a Design-time preference:

- Ten pre-existing Go files are above the limit. The E04 decision set is: `internal/config/config.go` (456), `internal/server/server_test.go` (378), `internal/gateway/responses_handler_test.go` (454), `internal/gateway/catalog_handler_test.go` (575), and the six remaining E03 catalog files (`manager_test.go` 877, `store_test.go` 593, `capability_test.go` 442, `manager.go` 397, `catalog_handler.go` 386 and `store.go` 312).
- The first two are in the E04 write set. Design must freeze their split layout and the implementation must complete the behavior-preserving split before adding E04 behavior or tests. These are E01/E02-structure changes only; public APIs and assertions remain unchanged.
- The two oversized gateway test files are outside the E04 write set. They must not be modified. E04 integration coverage goes in new, focused test files below 300 lines. Shared fixtures, if needed, go in a new helper file or an existing file already below 300.
- If Design discovers another file above the limit that must be modified, it must include that file in the split plan and must not add E04 logic until the split is complete. E03 production files remain out of scope.

### 7.3 Dependency direction

```text
cmd/ai-hub
  ├── internal/config
  ├── internal/access
  ├── internal/server
  ├── internal/gateway
  ├── internal/providers
  └── internal/catalog

internal/server ──> internal/access (middleware types/helpers)
internal/gateway ──> internal/providers (existing direction, unchanged)
internal/access ──> standard library only
```

No import cycle may be introduced. `internal/access` must not import `internal/providers` or `internal/catalog`.

### 7.4 Test-first and modularity constraints

- Tests are written before production code, must fail for the intended reason, and must not be relaxed to obtain green.
- New access helpers should be pure where possible; the HTTP middleware remains an orchestrator.
- The 300-line limit applies to Go source files, including tests; Markdown documentation is excluded.
- Any Go file approaching 250 lines triggers an SRP split before further work; no new or modified Go file may exceed 300 lines.
- The required splits of `internal/config/config.go` and `internal/server/server_test.go` are E04 prerequisites, not E03 technical-debt refactoring.
- E03 files already above the limit are not refactored in E04.

## 8. Security and Threat Model

### 8.1 Assets

- The OpenCode Go provider credential.
- The gateway API key.
- Request and response content, including prompts, model output and tool-call payloads.
- The E03 catalog snapshot and provider diagnostics.
- Process configuration and logs.

### 8.2 Threats and controls

| Threat | Control |
| --- | --- |
| Any local process or browser page reaches an unauthenticated provider proxy | Mandatory gateway key for every non-health route; loopback is not treated as a trust boundary |
| Browser cross-origin access | Exact Origin allowlist when Origin is present; no CORS headers; no preflight support |
| DNS rebinding | Exact Host allowlist based on `r.Host`; no trust in forwarded host headers; Origin validation |
| Credential brute force or timing analysis | 32–512-byte key, fixed-size digest comparison, constant-time comparison, generic `401` |
| Secret disclosure in logs/errors | No header/body/query logging; secret-safe config errors; no key echo; validation artifact redaction |
| Provider key delivered to clients | Inbound gateway key never forwarded; provider adapter overwrites `Authorization` with the provider credential; equality with provider key rejected |
| Oversized request consumes memory or provider bandwidth | Known-length precheck; bounded `limit + 1` read; `413` before upstream; per-request in-memory bound |
| Request decompression bypass | Non-identity `Content-Encoding` rejected; no decompression |
| Header smuggling/duplicate security headers | Explicit duplicate handling for Authorization, Origin, Content-Type and session header; existing 64 KiB header bound |
| SSE broken by middleware | No response buffering; preserve `Flush`/`Unwrap`; SSE compatibility tests |
| Accidental external exposure | Actual-listener check; non-loopback requires explicit host list; startup fails closed; no warning-only mode |
| Health endpoint used as a route-discovery oracle | Health is unauthenticated only for exact `GET`; Host/Origin still enforced; unknown/protected paths return `401` |
| Upstream error confused with local auth error | Provenance contract and separate integration tests; upstream response passed through unchanged |

### 8.3 Residual risks

- E04 does not terminate TLS. A non-loopback plaintext HTTP deployment can expose the gateway key and payloads on the network. Operators must use a trusted network, VPN or a TLS-terminating reverse proxy; this is a documented deployment constraint, not an E04 feature.
- E04 does not implement rate limiting, lockout or key rotation. A stolen key remains valid until the operator changes the environment and restarts the process.
- A single static key provides no per-user attribution, revocation or least-privilege authorization.
- Request bodies up to the configured limit are held in memory; concurrent large requests can increase memory pressure. E04 intentionally does not introduce quotas or a spooling subsystem.
- Health endpoints remain unauthenticated and can reveal process liveness to an allowed Host/Origin.
- Host allowlisting is not a network ACL; a client that knows the key can spoof an allowed Host. Authentication remains the primary control.
- The local process environment may be readable by the same OS user. E04 does not add OS-level secret isolation.

## 9. Test Strategy

### 9.1 Unit tests

| Area | Required cases |
| --- | --- |
| Configuration | Defaults; valid key boundaries (32/512); `b64token` alphabet and optional trailing `=`; rejection of comma, colon, semicolon, spaces, control characters and Unicode; too short/long; missing key; provider-key equality; secret not in error; allowed-host parsing and normalization; wildcard rejection; explicit empty list; origin canonicalization; default ports; `null`/wildcard/malformed origins; body-limit boundaries and invalid values; aggregated semantic validation errors; existing parse first-error behavior unchanged |
| Bearer parsing | Missing header; valid key; lowercase/uppercase scheme; extra spaces/tabs; empty token; duplicate header; comma-separated value; wrong key; provider key used as gateway key; characters outside `b64token`; constant-time comparator behavior |
| Host/Origin | Lowercase; optional port; IPv6 bracket; trailing dot; malformed host; multiple Origin values; absent Origin; allowed origin; scheme/port mismatch; URL credentials/path/query/fragment; wildcard config |
| Admission | Known Content-Length over limit; exact limit; limit+1; chunked over limit; empty body; `Content-Encoding`; duplicate Content-Type; JSON media-type parameters; missing media type; catalog 4 KiB cap |
| Session/identity | Valid supplied session preserved; absent/empty generated; malformed non-empty value → 400; over-128-character value → 400; duplicate header → 400; generation failure; generated ID differs from request ID; no persistence; no payload injection |
| Error mapping | Every local status/type pair; `WWW-Authenticate`; `Cache-Control: no-store` for local 400/401/403/413/415; correlation header; no secret/body/host echo |
| Metadata | Envelope contains only allowed fields; no body mutation; no provider header beyond existing session header |

### 9.2 Integration tests

Use `httptest` servers and a controllable fake upstream that records method, headers, body bytes and timing.

Required cases:

1. Middleware order: invalid Host → 403; missing key → 401; both → 403; valid Host but missing key → 401; valid key but oversized body → 413; all with a correlation ID and access log.
2. Health/readiness: exact `GET` is unauthenticated with a valid Host; invalid Host/Origin → 403; `HEAD` requires authentication; wrong method on health is not exempt.
3. Protected routes: all E02/E03/diagnostics routes reject missing/invalid auth with zero upstream/catalog effects; valid auth reaches the expected handler.
4. Credential separation: fake upstream sees the provider key in `Authorization`, never the gateway key; the client receives no provider credential.
5. Body admission: oversized known and chunked requests produce 413 before upstream; exactly-at-limit passes; `/v1/catalog/sync` is capped at 4 KiB; no partial upstream request exists.
6. Media type: `/v1/responses` missing/wrong type → 415 before upstream; valid `application/json; charset=utf-8` preserves the original header; catalog empty body remains accepted.
7. Passthrough: request body bytes exactly equal for `/v1/responses` (including whitespace/key order and invalid JSON syntax); `/v1/models` status/body/Content-Type preserved; upstream 4xx/5xx exact passthrough.
8. Streaming: SSE bytes and progressive arrival preserved through the full middleware; `Flush`/`Unwrap`/`ResponseController` still work; client disconnect and inactivity behavior unchanged.
9. Session: valid supplied session reaches the fake upstream unchanged; absent/empty value produces `aihub-ses-...`; correlation ID does not change; malformed/overlength and duplicate session headers return 400 with zero upstream calls.
10. Listener exposure: bind a non-loopback/wildcard address with no explicit allowed hosts → startup error, listener closed, no request served; valid explicit policy starts; loopback defaults work.
11. Secret safety: capture logs and responses across success/failure paths; no key, Bearer value, body, query or raw upstream error appears. Local 400 responses on catalog routes include `Cache-Control: no-store`.
12. Recovery/logging: panic inside a protected handler is recovered and logged without leaking the key; access log still omits headers/query/body.

### 9.3 Regression tests

- Run the full existing E01/E02/E03 suites with `-race`.
- Assert `/v1/models` behavior remains E02 passthrough.
- Assert `/v1/responses` body preservation and SSE framing remain byte-identical for accepted requests.
- Assert catalog snapshot, listing dimensions, counters, quarantine and error classes are unchanged.
- Assert E01 health/readiness and ordinary timeout semantics remain intact.
- Update only test fixtures required by the now-mandatory gateway key; do not weaken or remove assertions.

### 9.4 Real E2E validation

The E2E artifact must be created under `docs/validation/` and must explicitly classify each item as validated, not validated, environment-limited, Codex-CLI-limited or upstream-limited.

This artifact is part of the Phase 6 Green evidence package. Phase 6 cannot close while any required E2E scenario is unclassified. Phase 7 reviews and revalidates the evidence; it does not create it for the first time.

Prerequisites and isolation:

- Use a temporary `CODEX_HOME` and a temporary workspace; do not modify the developer’s global `~/.codex` or project `.env`.
- Use the real OpenCode Go upstream and a real provider credential supplied through the process environment.
- Use a gateway key distinct from the provider key; never print either value.
- Configure Codex CLI with `base_url = "http://<allowed-host>:<port>/v1"`, `wire_api = "responses"` and `env_key` pointing to `AIHUB_CLIENT_KEY`.
- First prove that the configured Codex CLI sends `Authorization: Bearer <AIHUB_CLIENT_KEY>` (or document the actual mechanism and stop if it cannot authenticate). Do not assume this without evidence.

Required scenarios:

| Scenario | Minimum evidence |
| --- | --- |
| Basic generation | Codex CLI prompt → AI Hub → real provider → answer; authenticated request; no secret in logs |
| Multi-turn | `codex exec` followed by `codex exec resume --last`; full context replay; no claim of gateway durable session |
| Tool calling | Codex performs a bounded file/tool operation through the real path; record sandbox limitations |
| Streaming | SSE reaches the gateway progressively; record whether Codex CLI exposes progressive rendering or buffers it |
| Upstream error | A real upstream error (for example, unknown model) preserves upstream status/body through E04 |
| Missing/invalid gateway auth | 401 local envelope and zero upstream calls |
| Wrong provider key as gateway key | 401; provider key is not accepted as gateway auth |
| Host rejection | Disallowed Host → 403 before auth/upstream |
| Origin rejection | Disallowed/malformed Origin → 403; allowed exact origin can proceed if configured |
| Health/readiness | `GET /healthz` and `/readyz` succeed without auth with allowed Host; invalid Host → 403 |
| Safe non-loopback exposure | Bind non-loopback with explicit allowed hosts and auth; exercise locally without public exposure; verify unsafe configuration fails startup |
| E01/E02/E03 regression | Existing suite green; catalog reads remain authenticated and semantically unchanged |

If the real provider, Codex CLI, sandbox or network is unavailable, the artifact must state the scenario as not validated and E04 must not claim full completion.

### 9.5 Quality gates

All must pass before E04 completion:

```bash
gofmt -w <changed-go-files>
go test ./... -count=1 -race
go vet ./...
staticcheck ./...
golangci-lint run ./...
go test ./... -cover -count=1
go build ./...
```

- At least 80% line coverage on new E04 code; uncovered branches justified in the evidence.
- No new dependency.
- No Go source file over 300 physical lines (new or modified, tests included); Markdown is excluded. Pre-existing oversized files in the E04 write set must be split without behavior change. E03 debt files are exempt from refactoring but do not justify new violations.
- Scope check: only E04-related files are modified.

## 10. Acceptance Criteria

| ID | Criterion | Verification |
| --- | --- | --- |
| AC-01 | A valid configuration with a 32–512 character `b64token` gateway key starts; missing, too-short, too-long, out-of-alphabet (including comma/colon/semicolon/Unicode), or provider-equal keys fail startup without leaking the value. | Config unit tests + startup integration test |
| AC-02 | All protected routes reject missing, malformed, duplicate or invalid Bearer credentials with the same 401 envelope and `WWW-Authenticate`, before upstream/catalog activity. | Router integration tests + fake upstream |
| AC-03 | Valid authentication reaches the existing handlers; the upstream sees the provider credential, never the gateway credential. | Fake-upstream capture |
| AC-04 | `GET /healthz` and `GET /readyz` remain unauthenticated but enforce Host/Origin and preserve ADR-006 bodies/status; `HEAD` is not exempt and returns 401 without a valid key. | Router integration tests |
| AC-05 | Disallowed/malformed Host, `Origin: null`, wildcard, malformed, multiple or non-allowlisted Origin produce 403 without echoing input; IPv4, IPv6 and wildcard classification follow NFR-012. | Host/Origin unit + integration tests |
| AC-06 | A non-loopback actual listener without explicit non-empty allowed hosts fails startup and does not serve; a valid explicit configuration serves. | Real-listener startup test |
| AC-07 | Known and chunked bodies over the effective limit return 413 before any upstream call; exactly-at-limit bodies are accepted; catalog sync keeps its 4 KiB cap. | Admission unit + integration tests |
| AC-08 | `/v1/responses` rejects missing/wrong media types with 415; valid original Content-Type is preserved; non-identity Content-Encoding is rejected. | Media-type integration tests |
| AC-09 | Duplicate Authorization/Origin/Content-Type/session headers follow the defined statuses and never reach a handler ambiguous. | Header unit + integration tests |
| AC-10 | Local error statuses/types/messages match §6.6; local 400/401/403/413/415 include `Cache-Control: no-store`; E03 keeps 409 `provider_disabled` and 502 `upstream_unavailable` when all requested providers fail; no secret/internal detail is present. | Error contract table tests + log capture |
| AC-11 | Upstream success and error status/body/Content-Type are byte-preserved; a local 401 cannot replace an upstream 401 and vice versa. | Fake-upstream passthrough tests |
| AC-12 | `/v1/responses` accepted request bytes are identical to the client body; `/v1/models` remains E02 passthrough. | Byte-capture tests |
| AC-13 | SSE body bytes, flush behavior and progressive arrival survive the full middleware; client disconnect/inactivity semantics remain E02-compatible. | Real-listener streaming tests |
| AC-14 | `X-Request-Id` remains correlation-only; valid `x-opencode-session` is preserved, absent/empty is generated independently, and malformed/duplicate values return 400; no durable session or cross-turn continuity is claimed. | Unit + integration + E2E artifact |
| AC-15 | The metadata envelope contains only transport/identity fields and never changes the provider JSON payload or persists data. | Body-byte and context tests |
| AC-16 | E01/E02/E03 suites are green, including race detection; catalog behavior is semantically unchanged. | Full regression suite |
| AC-17 | Real E2E covers basic, multi-turn, tools, streaming, upstream errors, auth, Host/Origin and safe non-loopback exposure; limitations are classified honestly. | `docs/validation/E04-*.md` |
| AC-18 | `go vet`, `staticcheck`, `golangci-lint`, `go build` and coverage gates pass; new code coverage is ≥80%. | Quality-gate output |
| AC-19 | README, `.env.example`, architecture, ADR index and ADR-011..ADR-013 document the new behavior without including a real secret. | Documentation inspection |
| AC-20 | No new dependency, no provider abstraction, no routing/classification/policy/learning/retry/fallback, no E03 refactoring, and no new or modified Go file over 300 lines is present in the final diff; the required `config.go` and `server_test.go` splits preserve behavior. | Scope check + import/diff/line-count inspection |
| AC-21 | Admission-level malformed is limited to headers, media metadata and session values; an otherwise admitted `/v1/responses` body with invalid JSON syntax is forwarded unchanged for upstream validation, while catalog sync JSON syntax remains an E03 handler concern. HTTP framing rejected by `net/http` remains server-generated. | Fake-upstream boundary test + catalog regression |

### 10.1 Traceability summary

| Requirement group | Acceptance criteria |
| --- | --- |
| Configuration/startup (FR-001..FR-008) | AC-01, AC-06, AC-18, AC-19 |
| Authentication (FR-009..FR-018) | AC-02, AC-03, AC-04, AC-06 |
| Network exposure (FR-019..FR-030) | AC-05, AC-06, AC-17 |
| Admission/limits (FR-031..FR-042) | AC-07, AC-08, AC-09, AC-13, AC-21 |
| Error contract (FR-043..FR-048) | AC-10, AC-11 |
| Identity/envelope (FR-049..FR-054) | AC-14, AC-15 |
| Compatibility/E2E (FR-055..FR-063) | AC-12, AC-13, AC-16, AC-17, AC-18, AC-19, AC-20, AC-21 |
| NFR-001..NFR-012 | AC-05, AC-06, AC-10, AC-13, AC-15, AC-18, AC-20 |

## 11. Evidence Block

| Field | Value |
| --- | --- |
| **Claim** | After E04, AI Hub starts only with a valid gateway credential and a fail-closed exposure policy; every protected route is authenticated and Host/Origin-gated; admission-invalid headers/media/session or oversized requests are rejected before upstream; accepted Codex/OpenCode traffic preserves E02 request bytes, SSE delivery and upstream-error passthrough; and real E2E evidence proves the authenticated path without exposing either credential. |
| **Success type** | `test` (primary automated evidence), plus `command` and `manual` for the real E2E boundary. |
| **Evidence path** | TDD unit and integration suites in §9.1–§9.3; `go test ./... -count=1 -race`, `go vet ./...`, `staticcheck ./...`, `golangci-lint run ./...`, `go build ./...`; real E2E artifact under `docs/validation/E04-*.md` produced in Phase 6; redacted logs and capture evidence. |
| **Budget** | Every FR group has at least one focused test; every acceptance criterion has a named verification; the fake upstream captures body/header/zero-call evidence; the real listener proves startup failure; the real E2E matrix covers all scenarios in §9.4. No generic suite-only pass is accepted. |
| **Affordance** | A controllable fake upstream (captures bytes, headers and call counts), an injectable/real-listener probe for exposure classification, a temporary Codex home and a temporary validation workspace. These are test affordances, not durable production features. |
| **TDD exception** | None requested. Tests must be written first and fail for the intended reason before implementation. |

## 12. Edge Cases

1. Key exactly 32 or 512 characters; trailing `=` padding; key with a comma, colon, semicolon, newline, space, tab, Unicode or control character; key equal to the provider key.
2. Duplicate `Authorization`, `Origin`, `Content-Type` or `x-opencode-session` headers with different casing.
3. `Host` with an IPv6 bracket, IPv6 zone, port, trailing dot, uppercase letters or malformed port.
4. `Origin` with a default port, non-default port, user information, path, query, fragment, uppercase scheme/host or `null`.
5. Host/Origin path trailing slash or HTTP method that causes `ServeMux` redirects; only exact health paths are exempt.
6. `Content-Length: 0` versus chunked empty body on `/v1/catalog/sync`.
7. Body exactly at the limit, limit+1, known-length over limit and unknown-length over limit.
8. Client aborts while the body is being buffered; read timeout; request context cancellation.
9. `Content-Encoding: gzip`, `identity`, absent and duplicate values.
10. Provider disabled versus provider unhealthy versus provider transport failure; E02 keeps 502/503, E03 sync keeps 409 `provider_disabled` and keeps 502 `upstream_unavailable` when all requested providers fail; local statuses remain distinct from upstream statuses.
11. Upstream 401/403/413/415 must not be replaced by local access-control envelopes.
12. SSE stream with long reasoning pauses, rapid chunks, client disconnect and upstream silence longer than the inactivity window.
13. Health request with a valid Host but no auth; health request with an invalid Origin but no auth; `HEAD` health without auth; wrong-method health request.
14. Non-loopback hostname resolving to loopback, wildcard or multiple addresses; IPv4-mapped IPv6 loopback.
15. `AIHUB_GATEWAY_ALLOWED_HOSTS` unset on loopback versus explicitly empty; non-loopback with loopback-only host entries.
16. Startup failure after the listener binds; listener must be closed and no request served.
17. Aggregated semantic configuration errors with multiple simultaneous invalid fields; none may include the secret; the existing first-error behavior for environment parse failures remains unchanged.
18. Log capture across success, 401, 403, 413, 415, 502, 503 and panic paths.
19. Test fixture compatibility after the key becomes mandatory.
20. A request that would exceed the body limit but is unauthenticated must return 401, not 413 or 415.
21. `/v1/responses` body with invalid JSON syntax passes admission byte-for-byte and is rejected or accepted upstream, not locally; catalog sync malformed JSON remains an E03 handler validation case.
22. Empty, over-128-character, invalid-character and duplicate `x-opencode-session` headers; only empty/absent is generated.

## 13. Risks, Dependencies and Limitations

### 13.1 Risks

| ID | Risk | Mitigation/decision |
| --- | --- | --- |
| R-01 | Static key compromise has no revocation or rotation mechanism in E04. | Document the operational rotation-by-environment-and-restart procedure; keep rotation out of scope. |
| R-02 | Non-loopback plaintext HTTP exposes credentials and content on the network. | Document the trusted-network/VPN/TLS-terminating-proxy requirement; do not claim TLS support. |
| R-03 | In-memory body buffering can consume memory under concurrent large requests. | Default 32 MiB, hard maximum 128 MiB, per-request bound, no response buffering; no quota subsystem. |
| R-04 | Codex CLI may not send the expected `Authorization` value from `env_key` in all versions. | Make E2E prove the mechanism first; if it cannot authenticate, record a Codex-CLI limitation and do not claim E2E success. |
| R-05 | E02 and E03 inherited validation is incomplete. | Keep inherited limitations explicit in the E04 artifact; do not reinterpret them as E04 failures or successes. |
| R-06 | Existing tests assume no gateway key. | Update only test fixtures required by mandatory authentication; preserve assertions and production behavior. |
| R-07 | ADR-002 is stale and conflicts with environment-only configuration. | Record the divergence; use implemented environment-only semantics; do not silently rewrite E01 history. |
| R-08 | Host/Origin validation can break existing browser or reverse-proxy clients. | Default non-browser path remains usable; document exact configuration; no CORS in E04. |
| R-09 | `405`/`414`/`431`/last-resort `500` may not use the JSON envelope. | Document the exception; keep status/`Allow`/safe bodies deterministic. |
| R-10 | Non-loopback actual-listener detection may vary by OS/address family. | Require IPv4/IPv6/wildcard tests and fail closed on unsupported address forms. |
| R-11 | No rate limiting or lockout means brute-force/log-flood attempts remain possible. | 32-byte minimum key and bounded admission reduce trivial attacks; quotas/rate limits are explicitly out of scope. |
| R-12 | Health remains unauthenticated. | Keep bodies non-sensitive; enforce Host/Origin; document the residual liveness disclosure. |
| R-13 | The catalog sync 4 KiB behavior may change from a handler `400` to an admission `413`. | This is an intentional admission-contract change; catalog semantics are unchanged and the change is tested. |

### 13.2 Dependencies

- Go 1.23 standard library and existing packages.
- The existing `http.ServeMux` routing and `http.Server` timeout behavior.
- OpenCode Go availability and a real provider credential for E2E.
- A Codex CLI build that can be configured with a custom Responses provider and an environment-backed key.
- A local listener that can bind a non-loopback/wildcard address for the startup test; no public exposure is required.
- E03 snapshot directory and existing catalog behavior for regression tests.

### 13.3 Known limitations

- No TLS, per-user identity, RBAC, key rotation, mTLS or secret manager.
- No CORS/preflight; browser clients are not a supported E04 use case unless they can perform a non-preflighted request with an allowed exact Origin and Bearer header (which typical browsers cannot for a custom Authorization header).
- No rate limiting, quotas or cost controls.
- No durable AI Hub session or cross-turn continuity.
- No retry, fallback or model switching.
- No metrics/OpenTelemetry subsystem.
- No multi-provider or generic provider abstraction.
- E02 and E03 live limitations remain as documented in their existing artifacts.

## 14. Decision Register and Required ADRs

### 14.1 Decisions proposed by this SPEC

| ID | Decision | Status |
| --- | --- | --- |
| D1 | A static gateway key is mandatory for all startups, including loopback, because loopback is not a trust boundary and E02 already proxies a provider credential. | approved 2026-10-31 |
| D2 | The gateway key and provider key are distinct values; equality is a startup error. | approved 2026-10-31 |
| D3 | Health/readiness remain unauthenticated only for exact `GET`; `HEAD` is protected. Both methods are Host/Origin-gated. | approved 2026-10-31 |
| D4 | Non-loopback exposure requires an explicit non-loopback host, an explicit non-empty allowed-host list and a post-bind actual-listener check. | approved 2026-10-31 |
| D5 | Host and Origin matching is exact, normalized and wildcard-free; no CORS or forwarded-host trust. | approved 2026-10-31 |
| D6 | Request bodies are bounded and buffered only for admission; responses remain unbuffered. | approved 2026-10-31 |
| D7 | `/v1/responses` requires JSON; catalog sync keeps an optional empty body; non-identity content encoding is rejected. | approved 2026-10-31 |
| D8 | Local admission/auth errors use the stable envelope; upstream errors keep E02 passthrough. | approved 2026-10-31 |
| D9 | `X-Request-Id` is correlation only; `x-opencode-session` is per-request provider context; no durable AI Hub session exists. | approved 2026-10-31 |
| D10 | The metadata envelope is in-memory, transport/identity-only, non-persistent and cannot alter the provider payload. | approved 2026-10-31 |
| D11 | E04 uses only the standard library and existing packages; no provider abstraction or new dependency. | approved 2026-10-31 |
| D12 | ADR-005 is superseded in intent by ADR-011, but ADR-005 is not edited during E04; ADR-011 records the replacement. | approved 2026-10-31 |
| D13 | E03 technical debt and E03 semantic changes are out of scope. | approved by the E04 brief |
| D14 | OAuth/RBAC, per-user identity and durable sessions are out of scope. | approved by the E04 brief |
| D15 | TLS termination, rate limiting, key rotation and CORS/preflight are out of scope. | approved 2026-10-31 |

### 14.2 ADRs required for implementation

| ADR | Required content | Relationship |
| --- | --- | --- |
| ADR-011 — Gateway API key authentication and network exposure policy | Mandatory static key, constant-time comparison, health exemption, loopback default, non-loopback opt-in, actual-listener validation and Host/Origin rules; record D15 exclusions only if the developer approves D15 | Revisits/replaces ADR-005’s deferred-auth assumption; does not edit ADR-005 |
| ADR-012 — Gateway request admission limits and local error contract | Body limit, buffering rationale, Content-Type/encoding rules, 413/415 semantics, local error envelope, upstream-error passthrough boundary | Extends ADR-008 without changing streaming inactivity behavior |
| ADR-013 — Request and session identity | `X-Request-Id` correlation semantics, `x-opencode-session` provider context, generated prefix, no durable session, metadata envelope | Formalizes and preserves E02 D3 |

ADR files must be created during the E04 Design/implementation phase after SPEC approval. They are not created in this phase.

## 15. Non-goals

The following are explicitly excluded from E04:

- Task classification or a Task Classifier.
- Model selection, ranking, scoring, eligibility or intelligent routing.
- A Policy Engine, quotas, budget, cost control or learning engine.
- Retry, fallback or model switching during execution or streaming.
- Persistent conversation history or durable AI Hub sessions.
- A second provider or a generic provider abstraction.
- Dashboard, metrics subsystem or OpenTelemetry.
- Protocol translation, body normalization, request rewriting or response rewriting.
- E03 semantic changes or E03 technical-debt refactoring.
- OAuth, users, organizations, RBAC, MFA or tenant isolation.
- Key rotation UI/API, multiple gateway keys or a secret manager.
- TLS termination, mTLS or certificate management.
- CORS preflight support or a browser UI.
- Rate limiting, abuse scoring or IP allowlists.

## 16. Approval Record

This SPEC was approved by the developer on 2026-10-31, including decisions
D1–D15 and the express approval of D15's exclusions. The approval authorizes
the SDD sequence below. It does not authorize commit or push; LIM-03 remains in
force.

The approved SDD sequence is:

1. Phase 3: developer approval of this SPEC and its evidence path.
2. Phase 4: Design, including the exact package/file split, middleware injection point, ADR-011..ADR-013 and implementation order.
3. Phase 5: failing tests (Red) with visible evidence.
4. Phase 6: implementation (Green), bounded green loop, real E2E validation, quality gates and the complete Claim evidence package.
5. Phase 7: independent `senior-code-review-judge` review, revalidation of the Phase 6 evidence and final report.

No commit or push is authorized by this SPEC. LIM-03 remains in force. The
approved scope deliberately keeps TLS termination, rate limiting, key rotation
and CORS/preflight out of E04; non-loopback exposure remains conditional on
explicit configuration, authentication and listener validation, with the
operational requirement of a trusted network, VPN or TLS-terminating proxy.
