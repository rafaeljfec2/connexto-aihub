# E04 — Gateway Hardening & Access Control: Design

- **Status:** implemented — review notes remediated.
- **Date:** 2026-10-31.
- **SPEC:** `docs/specs/E04-gateway-hardening-access-control.md` (approved 2026-10-31, D1–D15).
- **Baseline:** `main` at `c9444ce` (E03 closure), equal to `origin/main`.
- **Workflow:** SDD Phase 4 → Phase 5 (Red) → Phase 6 (Green) → Phase 7 (review).
- **Related ADRs:** ADR-011, ADR-012, ADR-013.

## 1. Design Goals and Invariants

1. **Fail closed.** A missing key, an unclassified listener, an ambiguous header or
   an unreadable body never falls back to unauthenticated or unbounded operation.
2. **One trust boundary.** `internal/access` owns Host/Origin decisions, gateway
   authentication, admission and request identity. Existing handlers keep their
   E02/E03 semantics.
3. **Credential separation.** The inbound gateway credential is validated and
   discarded; `internal/providers` continues to set the provider credential.
4. **Byte preservation.** Admission buffers only to enforce a bound. Accepted
   `/v1/responses` bytes are forwarded unchanged; responses stay unbuffered.
5. **Immutable policy.** Configuration and `access.Policy` are read-only after
   startup; per-request state lives only in the request context and request body.
6. **No new dependency.** Go 1.23 standard library plus existing packages only.

## 2. Package Architecture and Dependency Direction

```text
cmd/ai-hub
  ├── internal/config ──> internal/access      (pure validation/normalization reuse)
  ├── internal/access                          (standard library only)
  ├── internal/server ──> internal/access
  ├── internal/gateway ──> internal/providers   (existing, unchanged)
  ├── internal/providers ──> internal/config    (existing)
  └── internal/catalog                           (unchanged)
```

- `internal/access` imports the standard library only. It must not import
  `config`, `observability`, `server`, `gateway`, `providers` or `catalog`.
- `internal/config` imports `internal/access` for the key grammar and for
  Host/Origin parsing so startup validation and runtime enforcement cannot drift.
- `internal/server` imports both `config` and `access`.
- `internal/gateway` is not modified. The admission layer resolves the session
  and installs the header before dispatch; the existing E02 handler fallback
  remains available for direct handler use and unchanged tests.
- No cycle exists: `access` is a leaf package.

### 2.1 Why metadata omits the correlation ID

FR-053 says the metadata envelope *may* carry the correlation ID. The
correlation ID already lives in the `observability` context and is echoed/logged
by the outer middleware. `internal/access` does not import `observability` (it
would violate the stdlib-only boundary), so the envelope carries the
authenticated principal, provider session ID, accepted Host and accepted Origin.
No field is duplicated and no contract is weakened.

## 3. `internal/access` Contracts

### 3.1 Policy construction

```go
const (
    PrincipalID          = "gateway-client"
    SessionPrefix        = "aihub-ses-"
    DefaultMaxBodyBytes  int64 = 32 << 20
    MinBodyBytes         int64 = 1 << 10
    MaxBodyBytesCeiling  int64 = 128 << 20
)

type PolicyConfig struct {
    APIKey               string
    AllowedHosts         []string
    AllowedHostsExplicit bool
    AllowedOrigins       []string
    MaxBodyBytes         int64
}

func NewPolicy(cfg PolicyConfig) (Policy, error)
func (p Policy) MaxBodyBytes() int64
func (p Policy) Authorize(next http.Handler) http.Handler
```

`NewPolicy` re-validates the key and normalizes both allowlists. An empty,
non-explicit host list selects the loopback defaults `localhost`, `127.0.0.1`
and `::1`. An explicit empty host list is an error. An empty origin list allows
only requests without `Origin`.

`Policy` stores the SHA-256 digest of the configured key, the normalized
allowlists and the body limit. It contains no mutable state and is safe for
concurrent reads.

### 3.2 Pure validation helpers reused by config

```go
func ValidateAPIKey(key string) error
func ParseAllowedHosts(value string) ([]string, error)
func ParseAllowedOrigins(value string) ([]string, error)
```

- `ValidateAPIKey` enforces 32–512 ASCII characters matching the RFC 6750
  `b64token` grammar: at least one non-padding character from
  `[A-Za-z0-9._~+/-]`, only trailing `=` padding. Errors name the rule, never the
  value.
- Host parsing accepts lowercase/uppercase DNS names and IP literals without
  ports, schemes, paths, user information or wildcards. IPv6 brackets are
  removed. Trailing dots are removed. Entries are deduplicated; 1–64 entries are
  required when the variable is present.
- Origin parsing accepts absolute `http`/`https` origins with no user
  information, path, query or fragment. Scheme/host are lowercased, default
  ports are removed, IPv6 hosts are canonicalized and entries are deduplicated
  (1–64 when non-empty). `null`, `*` and malformed values are rejected.

### 3.3 Authorization middleware

`Policy.Authorize` implements, in order:

1. Normalize `r.Host` (strip a valid port, lowercase, remove a trailing dot,
   unbracket IPv6). A missing, malformed or non-allowlisted host is `403`.
2. Read `r.Header.Values("Origin")`. Absent is allowed. More than one value,
   `null`, `*`, a malformed origin or a non-allowlisted origin is `403`. There
   are no forwarded-header fallbacks and no CORS response headers.
3. Skip authentication only for exactly `GET /healthz` and `GET /readyz`.
4. Otherwise require exactly one `Authorization` header with the strict grammar
   `Bearer <token>` (case-insensitive scheme, one ASCII space, no commas or
   duplicate values). The token is digested with SHA-256 and compared with
   `crypto/subtle.ConstantTimeCompare`; grammar failure and digest failure
   produce the same `401` envelope with `WWW-Authenticate: Bearer realm="ai-hub"`.
5. Store `Metadata{Principal, Host, Origin}` in the request context and call the
   next handler.

Local errors set `Content-Type: application/json; charset=utf-8` and
`Cache-Control: no-store`. The outer middleware already installed `X-Request-Id`.
Rejected Host/Origin values are never echoed.

### 3.4 Admission

```go
type Admission struct { /* unexported kind and limit */ }

func ResponsesAdmission(maxBodyBytes int64) Admission
func CatalogSyncAdmission(maxBodyBytes int64) Admission
func (a Admission) Apply(next http.Handler) http.Handler
```

The zero value is “no admission” and returns the handler unchanged.
`CatalogSyncAdmission` uses `min(maxBodyBytes, 4096)`.

For an active admission:

1. Accept `Content-Encoding` absent or exactly `identity`; anything else is
   `415`.
2. Reject duplicate `Content-Type` or `x-opencode-session` values with `400`.
3. On `/v1/responses`, resolve `x-opencode-session`: absent or empty generates
   `aihub-ses-<32 hex>`; one valid value is preserved; a non-empty invalid value
   is `400`.
4. `/v1/responses` requires `application/json`, optionally with
   `charset=utf-8`, before reading; otherwise `415`. The original header value is
   preserved.
5. `/v1/catalog/sync` allows `Content-Length: 0` without a media type. A present
   or unknown-length body requires the same JSON media type; otherwise `415`.
6. A known `Content-Length` above the limit is `413` before reading. Otherwise
   read at most `limit + 1` bytes. More than `limit` bytes is `413`; a read error
   is `400`. Both occur before upstream.
7. Close the original body, replace `r.Body` with the exact accepted bytes,
   update `r.ContentLength`, attach the session ID to the metadata envelope and
   call the next handler. JSON syntax is not parsed by the gateway.

`Admission` never wraps the `ResponseWriter`; streaming interfaces are untouched.

### 3.5 Identity and metadata

```go
type Metadata struct {
    Principal         string
    ProviderSessionID string
    Host              string
    Origin            string
}

func WithMetadata(ctx context.Context, md Metadata) context.Context
func MetadataFrom(ctx context.Context) (Metadata, bool)

func NewSessionID() (string, error)
func ValidSessionID(value string) bool
```

`ValidSessionID` accepts 1–128 characters from `[A-Za-z0-9._:-]`.
`NewSessionID` reads 16 random bytes and returns `aihub-ses-` plus lowercase hex.
`X-Request-Id` is not an input to session generation. Nothing is persisted and no
payload field is added. These functions are the admission-time implementation;
the E02 handler keeps its own fallback and is not changed by E04.

## 4. Configuration Contract

```go
type Gateway struct {
    APIKey          string
    AllowedHosts    []string
    AllowedHostsSet bool
    AllowedOrigins  []string
    MaxBodyBytes    int64
}
```

`Config` gains `Gateway Gateway`. `Default()` sets `MaxBodyBytes` to 32 MiB and
leaves `APIKey` empty, so a default configuration is intentionally invalid until
the operator supplies `AIHUB_GATEWAY_API_KEY`.

| Variable | Parsing | Validation |
| --- | --- | --- |
| `AIHUB_GATEWAY_API_KEY` | string | `access.ValidateAPIKey`; must differ from an enabled provider key |
| `AIHUB_GATEWAY_ALLOWED_HOSTS` | `access.ParseAllowedHosts`; sets `AllowedHostsSet` | non-empty when set; non-loopback `server.host` requires it to be set |
| `AIHUB_GATEWAY_ALLOWED_ORIGINS` | `access.ParseAllowedOrigins` | explicit empty is valid; otherwise 1–64 canonical origins |
| `AIHUB_GATEWAY_MAX_BODY_BYTES` | `strconv.ParseInt` | 1,024–134,217,728 |

Environment syntax/parse failures keep the existing first-error behavior.
Semantic failures (key grammar, key equality, ranges, explicit-empty hosts,
non-loopback without explicit hosts) are aggregated by `Config.Validate` and
never include the key value. The new variables are added to `knownEnvKeys`.

Static loopback classification for `server.host`: an IP for which
`net.ParseIP(...).IsLoopback()` is true, or `localhost` case-insensitively, is
loopback. Every other hostname or IP requires an explicit allowed-host list.
The post-bind check remains authoritative.

## 5. Server and Router Wiring

### 5.1 Router signature

```go
type Route struct {
    Method    string
    Pattern   string
    Handler   http.Handler
    Admission access.Admission
}

func NewRouter(logger *slog.Logger, policy access.Policy, live, ready http.Handler, additional ...Route) http.Handler
```

`NewRouter` becomes fail-closed by construction: there is no variant that omits
access control. The pipeline is:

```text
withCorrelationID
  -> withRequestLog
    -> withRecovery
      -> policy.Authorize
        -> http.ServeMux
          -> route.Admission.Apply
            -> existing handler
```

Health and readiness handlers stay registered as `GET` patterns. `HEAD` is
matched by `http.ServeMux` to the GET handler only after authentication, which
satisfies “HEAD is protected”. Unknown paths reach `Authorize` before the mux and
therefore return `401` when unauthenticated.

### 5.2 Listener exposure

```go
type Option func(*options)
func WithNonLoopbackAllowed(allowed bool) Option
func New(cfg config.Server, handler http.Handler, logger *slog.Logger, opts ...Option) *Server
```

`Start` keeps binding synchronously. After `net.Listen`:

- A loopback `*net.TCPAddr` is always accepted.
- A non-loopback `*net.TCPAddr` is accepted only with
  `WithNonLoopbackAllowed(true)`.
- Any other address type is rejected.

On rejection the listener is closed and `Start` returns an error before `Serve`.
`main` passes `cfg.Gateway.AllowedHostsSet && len(cfg.Gateway.AllowedHosts) > 0`.
The former warning-only path is removed.

### 5.3 `main` wiring

1. `config.Load()` validates the gateway key, lists and body limit.
2. `access.NewPolicy` builds the immutable policy from `cfg.Gateway`.
3. `/v1/responses` receives `access.ResponsesAdmission(policy.MaxBodyBytes())`.
4. `/v1/catalog/sync` receives `access.CatalogSyncAdmission(policy.MaxBodyBytes())`.
5. `server.New` receives the non-loopback option described above.
6. No secret is logged; startup errors name settings, not values.

## 6. File Write Set and 300-Line Budget

### 6.1 New production files (each must stay ≤300 lines)

| File | Responsibility | Estimated lines |
| --- | --- | --- |
| `internal/access/access.go` | package doc, constants, `Policy`, `PolicyConfig`, `NewPolicy`, metadata types | 170 |
| `internal/access/key.go` | key grammar, bearer parsing, digest comparison | 120 |
| `internal/access/host.go` | host entry/request normalization and allowlist | 170 |
| `internal/access/origin.go` | origin parsing/canonicalization and allowlist | 160 |
| `internal/access/authorize.go` | Host/Origin + authentication middleware | 150 |
| `internal/access/admission.go` | admission constructors, header/media/session checks, bounded read | 230 |
| `internal/access/session.go` | session ID generation and validation | 60 |
| `internal/access/errors.go` | local error envelope and status/type constants | 90 |
| `internal/config/types.go` | existing types/constants moved from `config.go` | 180 |
| `internal/config/environment.go` | existing environment application and data-dir resolution | 140 |
| `internal/config/validate.go` | existing validation helpers | 150 |
| `internal/config/gateway.go` | gateway settings, parsing and validation | 170 |

### 6.2 Mandatory behavior-preserving splits before E04 logic

| Current file | Lines | Split result |
| --- | --- | --- |
| `internal/config/config.go` | 456 | Orchestrator ≤200 lines (`Config`, `Default`, `Load`, `UnknownEnvVariables`); `types.go`, `environment.go`, `validate.go` receive the existing code unchanged. Public API and error text remain identical. |
| `internal/server/server_test.go` | 378 | `server_test.go` keeps helpers and router/middleware tests; `server_lifecycle_test.go` receives Start/Stop, address-in-use, cancellation, deadline and exposure tests. No assertion is removed. |

The split is completed and the full baseline suite is green before any E04
behavior is added. `responses_handler_test.go` (454) and
`catalog_handler_test.go` (575) are outside the write set and are not modified.
E03 catalog files above the limit are not refactored.

### 6.3 New test files (each ≤300 lines)

| File | Coverage |
| --- | --- |
| `internal/access/key_test.go` | grammar boundaries, bearer forms, constant-time comparison, provider-key equality helper |
| `internal/access/host_test.go` | DNS/IPv4/IPv6, ports, trailing dots, wildcard/scheme/path rejection |
| `internal/access/origin_test.go` | canonicalization, default ports, duplicates, `null`/`*`/path/userinfo rejection |
| `internal/access/authorize_test.go` | precedence, health exemption, HEAD, metadata, error headers |
| `internal/access/admission_test.go` | limits, chunked, media types, encoding, duplicates, session resolution, body bytes |
| `internal/access/session_test.go` | generation, validity boundaries, independence from correlation |
| `internal/access/metadata_test.go` | envelope fields and absence |
| `internal/config/gateway_test.go` | defaults, env parsing, normalization, aggregation, secret safety |
| `internal/config/gateway_secret_test.go` | key grammar boundary table and no-value-in-error assertions |
| `internal/server/access_pipeline_test.go` | full pipeline precedence, auth-gated routes, zero upstream calls |
| `internal/server/access_streaming_test.go` | SSE/flush/`Unwrap` through the new middleware |
| `internal/server/access_listener_test.go` | non-loopback refusal, listener closed, explicit policy accepted |
| `internal/gateway/responses_admission_test.go` | accepted body bytes, session to upstream, media type preservation |

Shared fixtures live in a small `*_test.go` helper file below 300 lines or in an
existing helper file that is already below the limit. Test descriptions are in
English.

## 7. Implementation Order

1. **Split only:** `config.go` → orchestrator + `types.go` + `environment.go` +
   `validate.go`; `server_test.go` → router tests + `server_lifecycle_test.go`.
   Run the full baseline suite; no E04 behavior yet.
2. **Red tests:** add the access, config, server and gateway test files listed in
   §6.3 and the fixture updates required by the mandatory key. Capture the
   failing output.
3. **Access core:** errors, key, host, origin, session, metadata, policy.
4. **Admission:** bounded body read, media/encoding/session checks.
5. **Config:** `Gateway` type, env application, validation, `knownEnvKeys`.
6. **Server:** `Route.Admission`, required policy in `NewRouter`, listener option
   and post-bind check.
7. **Gateway:** no production change. Admission supplies a session header before
   dispatch; the existing handler fallback remains for direct handler use.
8. **Wiring:** `cmd/ai-hub/main.go`.
9. **Docs:** `.env.example`, `README.md`, `docs/architecture.md`, ADR index.
10. **Green loop:** focused tests, then `go test ./... -count=1 -race`,
    `go vet`, `staticcheck`, `golangci-lint`, coverage, `go build`.
11. **Real E2E:** artifact under `docs/validation/`.
12. **Phase 7:** independent `senior-code-review-judge`, fixes, revalidation.

## 8. Test Design and Evidence

### 8.1 Red inventory (minimum)

- Key grammar at 31/32/512/513 characters, padding, comma/colon/semicolon,
  spaces, control characters, Unicode and provider equality.
- Bearer: missing, duplicate, `Basic`, empty, multiple spaces, tab, comma list,
  wrong key, provider key used as gateway key.
- Host: lowercase, port, IPv6 bracket, trailing dot, wildcard, scheme, path,
  userinfo, malformed, missing.
- Origin: absent, exact, scheme/port mismatch, default-port normalization,
  `null`, `*`, multiple values, userinfo/path/query/fragment.
- Precedence: bad Host + no key → 403; good Host + no key → 401; good key +
  oversized body → 413; unauthenticated oversized body → 401.
- Health: exact GET without auth; disallowed Host → 403; HEAD without auth → 401;
  wrong method without auth → 401.
- Admission: known/chunked overflow, exactly-at-limit, `limit+1`, empty catalog
  body, JSON parameters, missing media type, non-identity encoding, duplicate
  headers, read error.
- Session: preserved, generated, empty, malformed, over-128, duplicate,
  independence from `X-Request-Id`, no payload mutation.
- Errors: every local type/status, `WWW-Authenticate`, `Cache-Control: no-store`,
  correlation header, no host/origin/secret echo.
- Listener: wildcard/IPv4/IPv6 non-loopback refusal, listener closed, explicit
  policy accepted, unsupported address type refused.
- Compatibility: `/v1/responses` body byte equality (including invalid JSON
  syntax), `/v1/models` passthrough, upstream 4xx/5xx passthrough, SSE bytes and
  progressive flush, catalog 409/502 semantics.

### 8.2 Integration harness

- A fake upstream `httptest.Server` records method, path, headers, body bytes and
  call count.
- A probe handler exposes `access.MetadataFrom` without changing production code.
- Real listeners are used for exposure and streaming tests.
- Log buffers assert absence of the gateway key, provider key, `Bearer`, bodies
  and query strings.

### 8.3 Real E2E plan (Phase 6)

Artifact: `docs/validation/E04-gateway-hardening-access-control-e2e.md`.

Isolation: temporary `CODEX_HOME`, temporary workspace, temporary `AIHUB_DATA_DIR`;
process environment only; project `.env` is read by the application through
`-env-file .env` but is never modified. Gateway and provider keys are distinct
and never printed.

Codex CLI configuration uses `base_url = "http://127.0.0.1:<port>/v1"`,
`wire_api = "responses"` and `env_key = "AIHUB_CLIENT_KEY"`. The first E2E step
proves the `Authorization` mechanism before any scenario is marked validated.

Scenarios: basic generation, multi-turn (`exec` + `resume --last`), bounded tool
calling, streaming observation, real upstream error passthrough, missing/invalid
gateway key, provider key used as gateway key, Host/Origin rejection, health and
readiness, safe non-loopback exposure (explicit hosts + auth, exercised locally),
unsafe startup failure, and the existing E01/E02/E03 regression suite.

If the provider, Codex CLI, network or sandbox prevents a scenario, the artifact
records the exact limitation and E04 does not claim that scenario as validated.

## 9. Risks and Open Decisions

| Item | Design position |
| --- | --- |
| Static key compromise | Rotation stays out of scope (D15); document environment-and-restart rotation. |
| Plaintext non-loopback | TLS stays out of scope (D15); require a trusted network, VPN or TLS-terminating proxy. |
| Memory pressure from 32 MiB bodies | Per-request bound and hard 128 MiB ceiling; no spool or quota subsystem. |
| Codex CLI auth mechanism | Prove it first in E2E; a limitation is reported, not converted into a pass. |
| Reverse proxies | Forwarded headers are never trusted; the operator configures the resulting Host/Origin explicitly. |
| 4 KiB catalog behavior | An oversized catalog body is an admission `413`; catalog semantics are unchanged. |

No decision in this design is pending. D1–D15 are approved; any deviation found
during implementation is escalated instead of being decided silently.

## 10. Traceability

| SPEC area | Design section |
| --- | --- |
| FR-001..FR-008, AC-01, AC-06 | §4, §5.2 |
| FR-009..FR-018, AC-02..AC-04 | §3.3, §5.1 |
| FR-019..FR-030, AC-05, AC-06 | §3.3, §4, §5.2 |
| FR-031..FR-042, AC-07..AC-09, AC-21 | §3.4 |
| FR-043..FR-048, AC-10, AC-11 | §3.3, §3.4 |
| FR-049..FR-054, AC-14, AC-15 | §2.1, §3.5 |
| FR-055..FR-063, AC-12, AC-13, AC-16..AC-20 | §1, §6, §7, §8 |
| NFR-001..NFR-012 | §1, §5.2, §8 |

## 11. Phase 4 Exit Criteria

- This design is approved and consistent with the approved SPEC.
- ADR-011, ADR-012 and ADR-013 are created and registered.
- The split of `internal/config/config.go` and `internal/server/server_test.go`
  is frozen as described in §6.2.
- The write set is limited to E04-related files.
- Phase 5 may now write failing tests; no production code is written before the
  Red evidence exists.
