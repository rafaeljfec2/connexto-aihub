# Frontend operator session — production boundary (G3)

**Date:** 2026-11-04

**Status:** accepted (Setor 8 brief + discovery §6.3 alignment)

**Scope:** Single-operator MVP (D1); embedded same-origin UI (D2)

## Context

Browsers cannot attach `Authorization` to `EventSource`. Dev Vite proxy injects bearer tokens server-side. Production embed requires an equivalent server-side trust boundary without exposing `AIHUB_GATEWAY_API_KEY` or `AIHUB_ADMIN_API_KEY` to JavaScript.

E04 intentionally excludes cookie login; this decision adds an **operator UI session** layer compatible with D1, not multi-user product auth.

## Decision

### M1 — Loopback trust (default local MVP)

When **all** hold:

- `AIHUB_SERVER_HOST` resolves to loopback (default `127.0.0.1`)
- `AIHUB_OPERATOR_PASSPHRASE` is **unset**
- Client TCP address is loopback

Then the access policy treats the request as an authenticated operator for both gateway and admin path namespaces (metadata principal `ui-operator`). No cookie required.

Rationale: matches discovery production guardrail for strict loopback binding (FRONTEND-DISCOVERY §5.2, §6.3).

### M2 — Passphrase session (non-loopback or explicit lock)

When `AIHUB_OPERATOR_PASSPHRASE` is set:

- `AIHUB_UI_SESSION_SECRET` (≥ 32 bytes) is **required**
- Operator logs in via `POST /api/ui/login`
- Server sets signed HttpOnly cookie (`aihub_ui_session`), `SameSite=Strict`, `Secure` when `AIHUB_UI_SECURE_COOKIES=true`
- Logout via `POST /api/ui/logout`
- Invalid or expired cookie → `401` on protected routes

Passphrase verified with constant-time compare; never logged or returned in API bodies.

### M3 — Static automation path unchanged

CLI and integrations continue using bearer tokens on `/v1/*` and `/api/*`. UI session and bearer are independent; bearer takes precedence when present.

### M4 — Restart and rotation

- Session cookies are HMAC-signed (server secret); valid until expiry (default 24h, `AIHUB_UI_SESSION_TTL`)
- Changing `AIHUB_UI_SESSION_SECRET` invalidates all cookies
- No PostgreSQL session store for MVP

## Security notes

- Loopback trust MUST NOT apply when listening on non-loopback without passphrase (config validation already requires allowed hosts).
- Provider credentials remain server-only; UI session does not expose them.
- Reverse proxy forwarded headers are not trusted (ADR-011 / E04 unchanged).

## Alternatives rejected

| Alternative | Reason |
| --- | --- |
| Bearer in localStorage | Violates AC-FE01-2 / discovery caution |
| Open `/api` on embed without auth | Unacceptable exposure of admin surface |
| Full user accounts / JWT refresh | Out of D1 scope |

## Related

- [`FE-08-production-release.md`](../specs/FE-08-production-release.md)
- [`FRONTEND-DISCOVERY.md`](../frontend/FRONTEND-DISCOVERY.md) §6.3 item 3
