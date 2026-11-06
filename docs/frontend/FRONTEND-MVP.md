# Frontend MVP Preparation

**Status:** Preparation only. No frontend application, UI components, API, or
frontend dependencies are created by this document.

This document is the technical starting point for the next authorized
development front. It maps the existing backend as inspected on 2026-11-03.
Endpoint contracts below reflect the current Go router and handlers; they do
not authorize new routes or contract changes.

## Repository and stack

- The backend is a Go modular monolith rooted at the repository root.
- The JavaScript workspace uses pnpm 12.16.1, Node.js 22 or newer, and
  Turborepo 2.11.10.
- The pnpm workspace includes apps/* and packages/* only. There is no frontend
  app or shared UI package in the current checkout.
- The root has only Turborepo as a JavaScript development dependency. React,
  Vite, TypeScript, a test runner, and a design system are not installed as
  workspace packages.
- ADR-019 records React + Vite as the expected direction and apps/frontend as
  the expected location. Treat this as a direction, not an existing
  implementation or a decision to add dependencies in this preparation task.
- Root pnpm scripts currently run Go build, vet, and test commands. Turborepo
  has frontend task names prepared, but there are no JavaScript workspace tasks
  to run.
- Local development currently starts the Go server. A frontend development
  server, proxy, deployment topology, and serving strategy have not been
  implemented.

See [ADR-019](../adr/ADR-019-hybrid-monorepo-workspace-tooling.md),
[development workspace](../development-workspace.md), and
[package.json](../../package.json).

## Security boundary

The backend uses two distinct static bearer credentials:

- **AIHUB_GATEWAY_API_KEY** protects gateway routes such as /v1/* and
  /diagnostics/providers.
- **AIHUB_ADMIN_API_KEY** protects /api/* administrative routes. Provider
  administration and E09 administration are wired in PostgreSQL mode; the
  admin key is required in that mode.

The admin key must differ from both the gateway and provider credentials. The
two backend bearer keys are not interchangeable. They must not be embedded in
JavaScript, compiled into a browser bundle, stored in browser local/session
storage, or returned to browser code. Provider credentials must remain
server-side as well.

Only exact GET /healthz and GET /readyz routes bypass bearer authentication.
Host and Origin validation still apply. The access policy accepts only
configured exact Origins; it does not emit CORS headers or handle preflight
requests. An allowed Origin therefore does not make browser cross-origin fetch
work.

There is no existing user authentication, browser session service, role model,
or BFF. Before a browser consumes protected routes, decide on a trusted
same-origin server-side proxy/BFF and its credential handling. If administrative
operations will be exposed beyond a single trusted local operator, define user
authentication and authorization separately; the current static admin key is
not a user-management system.

The frontend must not weaken gateway access policy, introduce CORS, or expose
either static key as a shortcut. Relevant sources are
[ADR-011](../adr/ADR-011-gateway-authentication-and-network-exposure.md),
[ADR-020](../adr/ADR-020-postgresql-persistence-foundation.md),
[provider route wiring](../../cmd/ai-hub/provider_admin.go),
[gateway route wiring](../../cmd/ai-hub/gateway_routes.go), and
[access policy](../../internal/access/authorize.go).

## Existing HTTP contracts

All routes are served by the same backend router. Unless stated otherwise,
protected routes require exactly one Authorization: Bearer header; missing or
invalid credentials return 401, and rejected Host/Origin values return 403.
Common local errors use a JSON error object with type and message fields.
E09 admin handlers set Cache-Control: no-store. The tables identify current
success shapes and material errors; detailed validation rules remain in the
linked handler/specification.

### Operational status

| Method and path | Authentication | Request and response | Relevant status |
| --- | --- | --- | --- |
| GET /healthz | None; exact health exemption | No body. JSON {status: up}. Liveness only. | 200 |
| GET /readyz | None; exact health exemption | No body. JSON status and per-dependency checks (ok/error); no internal error detail. | 200 when ready; 503 when a check fails |
| GET /diagnostics/providers | Gateway key | No request body. JSON providers array with runtime availability and health state. | 200 even when a provider is disabled or unhealthy |

### Models and catalog

| Method and path | Authentication | Request and response | Relevant status |
| --- | --- | --- | --- |
| GET /v1/models | Gateway key | Without `X-AIHub-Provider`, merges enabled providers' native `data[]` models. With the header, forwards that provider's native model-list response. | 200 merged or upstream status; 400 invalid selector; 404 unknown or unsupported provider; 503 no listable provider; 502 all upstream lists failed |
| GET /v1/catalog/models | Gateway key | Optional provider, capability, presence, and listing_state filters. Returns normalized catalog data and meta; entries include capabilities and their provenance, presence, freshness, runtime state, and execution-verification state. | 200; 400 invalid filter |
| GET /v1/catalog/models/{provider}/{model...} | Gateway key | No body. Returns one normalized model with provider metadata. | 200; 404 provider/model missing |
| GET /v1/catalog/providers | Gateway key | No body. Returns provider catalog sync state, freshness, and model counts. | 200 |
| POST /v1/catalog/sync | Gateway key | Optional JSON {provider: string}; an empty value requests all configured providers. Response reports per-provider outcome, model counts, and duration. The body is capped at 4 KiB. | 200 when at least one result is returned; 400 invalid body; 404 unknown provider; 409 disabled provider or sync already in progress; 500 persistence failure; 502 upstream failure |

The normalized catalog is not proof that a model can execute every feature.
execution_verification is currently unverified in E03. Catalog freshness,
provider state, and capability provenance must remain visible if the UI uses
these entries for eligibility or selection. See
[E03 specification](../specs/E03-model-catalog.md) and
[catalog handler](../../internal/gateway/catalog_handler.go).

### Provider administration

These routes are available in PostgreSQL mode and require the admin key.
Responses intentionally expose only whether a credential is configured, never
the credential or its ciphertext.

| Method and path | Request and response | Relevant status |
| --- | --- | --- |
| GET /api/providers | Response providers[]: id, name, type, endpoint, enabled, credential-configured flag, timestamps. | 200 |
| POST /api/providers | JSON fields id, name, type, endpoint, credential, optional enabled; returns a provider response. | 201; 400 invalid; 409 conflict |
| GET /api/providers/{id} | Returns one provider response. | 200; 404 missing |
| PATCH /api/providers/{id} | Optional name, endpoint, credential; returns the provider response. | 200; 400 invalid; 404 missing; 409 conflict |
| POST /api/providers/{id}/validate | Empty request. Probes the configured model-list endpoint; it does not run inference. Success returns {status: reachable}. | 200; 404 missing; 409 credential missing; 502 probe failure |
| POST /api/providers/{id}/enable and POST /api/providers/{id}/disable | Empty request; returns updated provider response. | 200; 404 missing |

Provider endpoints are constrained by the configured provider-host policy.
Provider configuration changes apply at runtime. A successful health probe
means the configured probe succeeded; it does not guarantee a particular model
can generate a response. See
[provider admin handler](../../internal/gateway/providers_admin_handler.go)
and [C2 compatibility](../specs/C2-openrouter-compatibility.md).

### Responses execution and records

| Method and path | Authentication | Request and response | Relevant status |
| --- | --- | --- | --- |
| POST /v1/responses | Gateway key | OpenAI Responses JSON request, forwarded unchanged. Optional X-AIHub-Provider selects provider; routing also respects explicit model selection. Streaming requests return progressive SSE; non-streaming requests return the upstream response. | Includes 400 invalid request, 401/403 access rejection, 429 resource limit, 502 upstream failure, and 503 provider/admission/concurrency unavailability. Streaming terminal errors follow the Responses event contract. |
| GET /v1/records/executions | Gateway key | Filters: limit, provider_id, model_id, outcome, started_after, started_before, cursor. Returns records, count, next_cursor, and degraded. | 200; 400 invalid/unknown query; 503 records unavailable |
| GET /v1/records/executions/{execution_id} | Gateway key | Returns a content-free record with execution/request IDs, provider/model, timing, outcome, phase, cause, upstream status, and response-committed state. Usage is null. | 200; 400 invalid id; 404 missing; 503 records unavailable |

There is no durable conversation or prompt history API. Execution records are
operational metadata and do not contain prompts or responses. E06 record usage
remains null; provider-reported usage is stored separately by E09.5. See
[E05](../specs/E05-execution.md), [E06](../specs/E06-execution-records.md),
[E08 routing](../specs/E08-intelligent-routing.md), and
[Responses handler](../../internal/gateway/responses_handler.go).

### Resource policies and concurrency

These PostgreSQL-backed routes require the admin key. Policy scopes are global,
provider, or provider/model.

| Method and path | Request and response | Relevant status |
| --- | --- | --- |
| GET /api/resource-policies | Returns policies[] with scope, limit, enabled state, and IDs. | 200 |
| PUT /api/resource-policies | JSON scope, optional provider/model IDs, daily_limit, optional enabled. Returns the policy. | 200 update / 201 create; 400 invalid scope/input; 409 conflict |
| GET /api/resource-usage | Optional provider_id, model_id, period_start; returns policy usage, admitted count, remaining requests, and reset time. | 200; 400 invalid query |
| POST /api/resource-policies/{id}/enable and POST /api/resource-policies/{id}/disable | Empty request; returns updated policy. | 200; 400 invalid id; 404 missing |
| GET /api/concurrency-policies | Returns policies with scope, concurrency limit, enabled state, and IDs. | 200 |
| PUT /api/concurrency-policies | JSON scope, optional provider/model IDs, concurrency_limit, optional enabled. Returns the policy. | 200 update / 201 create; 400 invalid scope/input; 409 conflict |
| GET /api/concurrency-usage | Optional provider_id and model_id; returns policy limit, active count, and available capacity. | 200; 400 invalid query |
| POST /api/concurrency-policies/{id}/enable and POST /api/concurrency-policies/{id}/disable | Empty request; returns updated policy. | 200; 400 invalid id; 404 missing |

See [E09 specification](../specs/E09-resource-management.md),
[resource admin handler](../../internal/gateway/resource_admin_handler.go),
and [concurrency admin handler](../../internal/gateway/concurrency_admin_handler.go).

### Usage analytics

All routes below are PostgreSQL-backed and require the admin key.

| Method and path | Request and response | Relevant status |
| --- | --- | --- |
| GET /api/resource-analytics/admissions | Required from/to; optional provider_id/model_id. Returns total admitted and a gap-filled UTC daily series. | 200; 400 invalid range/filter; 500 query failure |
| GET /api/resource-analytics/admissions/distribution | Same range and filters. Returns by_provider and by_model daily series. | 200; 400 invalid range/filter; 500 query failure |
| GET /api/resource-analytics/quota-utilization | Optional provider_id/model_id/period_start. Returns per-policy quota utilization and remaining requests. | 200; 400 invalid query; 500 query failure |
| GET /api/resource-analytics/concurrency | Optional provider_id/model_id. Returns active and available concurrency by policy. | 200; 400 invalid query; 500 query failure |
| GET /api/resource-analytics/provider-usage | Required from/to; optional provider_id/model_id. Returns token aggregates, execution status counts, and a gap-filled daily series. Unknown token metrics remain JSON null; explicit zero remains zero. | 200; 400 invalid range/filter; 500 query failure |
| GET /api/resource-analytics/provider-usage/distribution | Same range and filters. Returns observed by_provider and by_model groups; distribution days are not gap-filled. | 200; 400 invalid range/filter; 500 query failure |

Range values accept RFC3339 or YYYY-MM-DD, are inclusive UTC calendar days,
and span at most 366 days. Model filters require provider_id. Provider-reported
cost is persisted but is not included in these analytics responses. See
[E09.4 specification](../specs/E09.4-usage-analytics.md) and
[analytics handlers](../../internal/gateway/analytics_admin_handler.go).

## Functional navigation proposal

This is a prioritization proposal, not a committed screen list.

| Domain | Current backend support | Frontend without backend changes | Suggested priority |
| --- | --- | --- | --- |
| Overview / Status | Public liveness/readiness and authenticated provider diagnostics. | Health view is possible; diagnostics need the trusted proxy boundary. | First vertical increment |
| Providers | PostgreSQL admin CRUD, enable/disable, validation, diagnostics. | List state and later administration; never return or persist credentials in the browser. | First increment: read-only states; administration later |
| Models | Native model-list route and normalized catalog listing/detail/providers/sync. | Browse current catalog with provenance/freshness; sync is a privileged operation. | Next |
| Playground / Execution | Responses execution, deterministic routing, SSE, cancellation, resource controls. | A basic playground can consume the existing contract through a proxy; do not assume persisted chat history. | After secure connection |
| Usage / Analytics | Admission and provider-usage summaries/distributions. | Read-only views can consume existing endpoints through the admin proxy. | Later |
| Resources | Quota and concurrency policy CRUD, enable/disable, usage. | Existing API supports policy views and edits; server-side admin authorization is required. | Later |
| Settings | Provider settings only; no general application settings API. | No general settings screen can be backed by current routes. | Pending backend/product scope |

No frontend capability should be represented as implemented solely because a
route exists: access to protected APIs depends on an approved browser/server
integration boundary.

## Proposed frontend architecture

Keep the first application small and aligned with the existing workspace:

- Place the initial application in the expected workspace at
  apps/frontend, subject to confirming the stack and package requirements in
  the implementation SPEC.
- Separate route/page components from one API client. Keep transport,
  authentication forwarding, and error normalization out of UI components.
- Derive TypeScript types from the observed JSON contracts; do not create a
  second backend contract by guessing field names.
- Represent loading, empty, success, stale/unknown, and connection-error
  states explicitly.
- Treat JSON error envelopes and upstream status-preserving responses as
  different failure sources. Do not display credentials or raw secret-bearing
  upstream headers/bodies.
- For SSE, parse incrementally across arbitrary chunk boundaries, recognize
  terminal Responses events, and abort the request when the user cancels.
- Put only non-secret public configuration, such as a same-origin API base
  path, in browser build variables. Keep gateway/admin keys in the proxy/BFF
  environment.
- Add tests around the API client and first vertical flow once implementation
  is authorized. Do not add dependencies until the selected app stack and
  package needs are confirmed.
- No shared design system is present. Reuse existing browser/UI packages only
  if later repository inspection finds them; otherwise define a minimal,
  accessible baseline before expanding navigation.

## Pending decisions and risks

Resolve these before expanding beyond the first status view:

1. **Authentication:** Is this a single trusted local operator or a
   multi-user product? The backend has static gateway/admin keys, not users,
   roles, or sessions.
2. **BFF/proxy:** Where will the same-origin server-side proxy run, how will it
   hold each backend key, and which routes may it expose?
3. **Browser deployment and origin:** Decide local and production serving
   topology. The backend does not implement CORS or preflight handling.
4. **Provider administration:** Determine how operators enter/rotate provider
   credentials without sending them to browser storage or exposing them in
   responses.
5. **Playground:** Decide whether the initial playground is one request at a
   time, how it selects explicit provider/model, and how much streamed output
   is displayed.
6. **Preferences and history:** No interface preference persistence or
   conversation-history API exists. Decide whether either is required before
   adding storage.
7. **Design and responsiveness:** Select the design system, visual language,
   accessibility baseline, and responsive behavior.

## Future scope (post-MVP)

- **Internationalization (i18n):** Operator-facing UI in **Portuguese (pt-BR)** and
  **English (en)** — locale selection, translated strings, and locale-aware
  formatting. Requires a dedicated SPEC after MVP release (Setor 8); see
  `MVP-ROADMAP.md` §3.3 and gap **G9**. Repository docs and API contracts stay
  in English until i18n is implemented.

## Recommended first vertical increment

1. Confirm the browser-to-backend trust boundary and same-origin proxy/BFF.
2. Initialize the smallest workspace frontend using the approved framework and
   only justified dependencies.
3. Through that server-side boundary, read liveness/readiness and provider
   diagnostics without sending static keys to the browser.
4. Show actual system health and provider state with loading, empty, error, and
   success states.
5. Validate the single flow end to end before adding provider writes, model
   browsing, or execution.

This document prepares that next implementation task. It does not implement
the frontend or authorize backend/API changes.
