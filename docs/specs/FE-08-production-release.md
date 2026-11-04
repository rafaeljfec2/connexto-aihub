# SPEC FE-08 — Production release (Setor 8)

**Status:** Approved — Setor 8 execution brief (Lead, 2026-11-04)

**Date:** 2026-11-04

**Scope:** MVP roadmap Setor 8 — embedded SPA, operator session boundary, E2E smoke, release validation

**Resolves:** G3 (production session / embed), G7 (E2E tool choice)

**References:**

- [`MVP-ROADMAP.md`](../frontend/MVP-ROADMAP.md) § Setor 8
- [`FRONTEND-DISCOVERY.md`](../frontend/FRONTEND-DISCOVERY.md) §5–7
- [`frontend-mvp-architecture-approval-2026-11-03.md`](../decisions/frontend-mvp-architecture-approval-2026-11-03.md) (D1–D3)
- [`frontend-operator-session-2026-11-04.md`](../decisions/frontend-operator-session-2026-11-04.md)

## 1. Current state

- Development BFF: Vite proxy injects gateway/admin bearer keys server-side (`apps/frontend/vite.config.ts`).
- Production: Go serves API only; no embedded UI; browser cannot call protected routes without bearer headers.
- Operational SSE: `GET /api/events` (admin namespace); EventSource requires same-origin session or dev proxy.
- Setor 7: Executions and Resources implemented locally; FE-05 commit `cf7734f`; FE-06 uncommitted.

## 2. Production architecture

| Layer | Behavior |
| --- | --- |
| Static | `go:embed` of `apps/frontend/dist` (synced to `internal/static/dist` at build time) |
| SPA | `GET` non-API paths → `index.html` fallback; `/assets/*` immutable cache |
| API | Existing `/v1/*`, `/api/*`, health routes unchanged in semantics |
| Dev | Vite unchanged; no embed required for `pnpm dev` |

Single process: `cmd/ai-hub` on configured host/port (default loopback `:8080`).

## 3. Operator session (G3)

See decision doc for formal model. Summary:

| Mode | When | Browser access to `/v1` and `/api` |
| --- | --- | --- |
| Loopback trust | Server listens on loopback, `AIHUB_OPERATOR_PASSPHRASE` unset | Allowed from loopback client IP without cookie (D1 perimeter) |
| Passphrase session | `AIHUB_OPERATOR_PASSPHRASE` + `AIHUB_UI_SESSION_SECRET` set | `POST /api/ui/login` → HttpOnly session cookie; logout clears |
| Bearer (unchanged) | Automation / CLI | `Authorization: Bearer` with gateway or admin key |

Cookie flags: `HttpOnly`, `SameSite=Strict`, `Secure` when `AIHUB_UI_SECURE_COOKIES=true` or request TLS.

CSRF: same-origin + `SameSite=Strict`; no cross-site cookie use.

Restart: signed cookie sessions survive process restart if secret unchanged; in-memory revocation list not required for MVP.

Static assets and `/healthz` / `/readyz` remain unauthenticated.

## 4. Credential boundary

- `AIHUB_GATEWAY_API_KEY` and `AIHUB_ADMIN_API_KEY` never sent to browser.
- No `VITE_*` secrets.
- UI session proves operator access; server still injects internal authorization for upstream handlers via access metadata (no raw keys in responses).

## 5. UI auth routes

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| POST | `/api/ui/login` | Public (rate-limit deferred) | Passphrase → session cookie |
| POST | `/api/ui/logout` | Session or loopback trust | Clear session |
| GET | `/api/ui/session` | Session or loopback trust | `{ authenticated, mode }` |

## 6. E2E strategy (G7)

**Tool:** Playwright (`@playwright/test`), scoped to `e2e/` at repository root.

**Flow:**

1. Build frontend → sync dist → `go build ./cmd/ai-hub`
2. Start hub with test env (loopback, file persistence, test keys)
3. Browser tests against `http://127.0.0.1:<port>/`

Minimum scenarios (Setor 8 brief §4): app boot from Go binary, overview load, navigation, executions list, resources policies read, unauthorized handling, SSE reconnect smoke.

Out of scope for first slice: live provider playground stream, paid upstream calls.

## 7. Build and release commands

```bash
pnpm --filter @connexto/frontend build
node scripts/sync-frontend-dist.mjs
go build -o bin/ai-hub ./cmd/ai-hub/
./bin/ai-hub
```

Root `pnpm check` unchanged; optional `pnpm e2e` after hub build.

## 8. Acceptance criteria

| ID | Criterion |
| --- | --- |
| AC-S8-1 | SPA served from Go binary without separate Node process |
| AC-S8-2 | Frontend bundle contains no API keys (grep dist + sourcemaps policy) |
| AC-S8-3 | `go test ./... -count=1`, `go vet`, frontend `check` pass |
| AC-S8-4 | Playwright smoke passes against embedded origin |
| AC-S8-5 | Loopback trust and passphrase session documented and tested |
| AC-S8-6 | Setor 7 integrated validation recorded (functional + Doop alignment) |

## 9. Risks

| Risk | Mitigation |
| --- | --- |
| Loopback trust on shared machine | Document; require passphrase when exposing non-loopback listener |
| Embed stale vs UI | Release script always rebuilds frontend before Go compile |
| Session underspec | Decision doc + explicit env contract |

## 10. Out of scope

- FE-07 Settings, i18n (G9), multi-user RBAC, billing, new backend domains
