# SPEC FE-01 — BFF, system health, provider overview, operational SSE

**Status:** Approved — project lead authorized D1–D3 on 2026-11-03 ([decision](../decisions/frontend-mvp-architecture-approval-2026-11-03.md))

**Date:** 2026-11-03

**Scope:** MVP roadmap Setores 1–4 (development BFF + backend Event Hub + frontend SSE client + Overview UI)

**Resolves:** G2 (path/auth), G4 (event envelope), G5 (minimum event set for MVP)

## 1. Intent

Deliver the first operational vertical slice: secure dev proxy, real health/readiness and provider diagnostics on the Overview, and **zero frontend polling** via a dedicated operational SSE channel (D7).

## 2. Backend — operational Event Hub (Setor 2)

### 2.1 Route and authentication

- **Method/path:** `GET /api/events`
- **Auth:** administrative namespace (`AIHUB_ADMIN_API_KEY`), same as other `/api/*` routes (`internal/access/authorize.go`).
- **Rationale (G2):** operational feed includes provider runtime snapshots; admin namespace matches Vite proxy injection for `/api/*`. Browser uses `EventSource` on same-origin `/api/events`; dev proxy adds `Authorization`.

### 2.2 Event envelope (G4)

Every `data:` line is a JSON object:

```json
{ "schema_version": 1, ... }
```

`schema_version` is `1` for this increment.

### 2.3 Event catalog (G5 minimum)

| SSE `event` name | Payload fields (in addition to `schema_version`) | When emitted |
| --- | --- | --- |
| `readiness_changed` | `status` (`up`/`down`), optional `checks` map (`ok`/`error`) | On client connect; when readiness snapshot is published |
| `provider_status` | `providers` array (`id`, `enabled`, `state`) | On connect; on provider runtime state change; after admin validate/enable/disable |
| `heartbeat` | `{}` | Every 30s while stream open |

Not in FE-01 MVP: `quota_alert` (deferred; E09 hooks later).

### 2.4 Streaming behavior

- `Content-Type: text/event-stream`
- Flush after each event (progressive delivery)
- Client disconnect cancels handler context
- Distinct from `POST /v1/responses` inference SSE

## 3. Frontend — development BFF (Setor 1)

- Package `@connexto/frontend` at `apps/frontend`
- Vite proxy forwards `/healthz`, `/readyz`, `/diagnostics/providers`, `/v1/*`, `/api/*` to backend
- Proxy injects `Authorization: Bearer` from **server** env (`AIHUB_GATEWAY_API_KEY`, `AIHUB_ADMIN_API_KEY`) — not `VITE_*`
- Operators must set `AIHUB_GATEWAY_ALLOWED_ORIGINS` to include `http://localhost:5173` when using the Vite dev origin (ADR-011)

## 4. Frontend — operational client (Setor 3)

- `EventSource` to `/api/events`
- TanStack Query: `refetchInterval: false`; updates via `setQueryData` on `readiness_changed` and `provider_status`
- Reconnect with backoff; manual reconnect control
- No operational `setInterval` polling

## 5. Overview UI (Setor 4)

- Bootstrap fetch for initial `/healthz`, `/readyz`, `/diagnostics/providers`
- Cards for system liveness, database readiness, provider grid
- Degraded / backend-down states

## 6. Acceptance criteria

Maps to discovery §13:

- AC-FE01-1: `pnpm dev` serves UI on `localhost:5173`
- AC-FE01-2: No static keys in bundle or browser-set Authorization on protected routes
- AC-FE01-3: Healthy PostgreSQL mode shows system healthy + DB connected
- AC-FE01-4: Providers match diagnostics API
- AC-FE01-5: Backend stopped → clear error state
- AC-FE01-6: Unhealthy provider shows state badge
- AC-FE01-7: No operational polling; updates via `/api/events`

## 7. Out of scope (FE-01)

- `go:embed` production serving (Setor 8 / G3)
- Provider writes, catalog, playground
- Playground inference streaming

## 8. Evidence

| Claim | Success type | Evidence path |
| --- | --- | --- |
| Operational SSE route exists and auth enforced | Automated test | `internal/gateway/operational_events_handler_test.go` |
| Hub publishes on provider state change | Automated test | `internal/operational/hub_test.go` |
| Frontend typecheck and unit tests | Command | `pnpm --filter @connexto/frontend check` |
| Go regression | Command | `go test ./... -count=1 -race` |
