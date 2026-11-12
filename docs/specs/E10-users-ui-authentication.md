# SPEC E10 — Users UI Authentication

- **Status:** approved — implementation authorized under the SDD workflow.
- **Workflow:** SDD.
- **Date:** 2026-11-12.
- **Baseline:** AI Hub operator UI session (`internal/ui`) and gateway access policy (`internal/access`).

## 1. Objective

Add an optional `users` authentication mode for the embedded UI: email and password login, short-lived access JWT in an HttpOnly cookie, opaque refresh token with rotation, and password recovery. The existing `operator` mode (passphrase, HMAC session cookie, loopback trust) remains the default and unchanged in behavior.

## 2. Authentication modes

No `AIHUB_AUTH_MODE` flag. Behavior follows persistence:

| Condition | Behavior |
| --- | --- |
| `AIHUB_DATABASE_URL` set (postgres catalog persistence) and `AIHUB_UI_SESSION_SECRET` ≥ 32 bytes | Email/password UI, JWT + refresh cookies, no loopback trust. |
| Otherwise | Operator gate: optional `AIHUB_OPERATOR_PASSPHRASE`, `aihub_ui_session` cookie, loopback trust when passphrase unset. |

Static gateway keys `AIHUB_GATEWAY_API_KEY` and `AIHUB_ADMIN_API_KEY` remain for CLI. Bearer auth continues to take precedence over cookies in `access.Policy`.

## 3. Bootstrap

When postgres UI auth is active and the `users` table is empty, startup may create one admin:

- Default email `rafaeljfec2@gmail.com` (override `AIHUB_BOOTSTRAP_ADMIN_EMAIL`)
- Password from `AIHUB_BOOTSTRAP_ADMIN_PASSWORD`, else `AIHUB_OPERATOR_PASSPHRASE`

The account is stored with `owns_hub_data=true` (primary owner of hub data already in the database). If the email already exists, startup sets `owns_hub_data` only; it does not change the password. Bootstrap credentials must never appear in logs.

## 4. HTTP API contracts

All routes require valid Host and Origin per E04. Tokens never appear in JSON bodies, query strings, or client storage.

### `POST /api/ui/login`

**Operator mode** — unchanged: `{ "passphrase": string }` → `{ "ok": true }` and `Set-Cookie: aihub_ui_session`.

**Users mode** — `{ "email": string, "password": string }` → `{ "user": { "id", "email", "role" } }` and two cookies:

- Access JWT: `Path=/`, Max-Age 900 (15 minutes), HttpOnly, SameSite=Strict, `Secure` when `AIHUB_UI_SECURE_COOKIES=true`.
- Refresh opaque token: `Path=/api/ui`, Max-Age 604800 (7 days), same flags.

On failure (users): `401` with message `invalid credentials` (uniform for unknown email or bad password).

Rate limit: 5 attempts per client IP per minute (in-process).

### `POST /api/ui/refresh`

Users mode only. Reads refresh cookie only (no body). Rotates refresh (single-use); reusing a revoked refresh revokes all sessions for that user. Sets new access and refresh cookies. `401` when refresh invalid.

Exempt from access policy (no bearer/cookie access required beyond the refresh cookie).

### `POST /api/ui/logout`

Users mode: revokes all refresh sessions for the authenticated user and clears both cookies.

Operator mode: unchanged (requires operator session).

### `GET /api/ui/session`

**Operator mode** — unchanged: `{ "authenticated", "mode": "passphrase" | "loopback_trust" | "none" }`.

**Users mode** — when access cookie valid: `{ "authenticated": true, "mode": "users", "user": { "id", "email", "role" } }`; otherwise `{ "authenticated": false, "mode": "none" }`.

### `POST /api/ui/forgot-password`

Users mode only. Body `{ "email": string }`. Always `200` with `{ "message": "If an account exists for that email, password reset instructions were sent." }` whether or not the email exists.

Rate limit: 3 per IP per hour. When SMTP is not configured, same response; server logs that delivery is disabled (no token in logs).

### `POST /api/ui/reset-password`

Users mode only. Body `{ "token": string, "password": string }`. On success: updates password, marks token used, revokes all user sessions. Rate limit: 3 per IP per hour.

### Public paths (`uiAuthExempt`)

`GET /api/ui/session`, `POST /api/ui/login`, `POST /api/ui/refresh`, `POST /api/ui/forgot-password`, `POST /api/ui/reset-password`.

## 5. Tokens and storage

- Access: JWT HS256, claims `sub`, `email`, `role`, `sid` (session id), `exp` (15 minutes), signed with `AIHUB_UI_SESSION_SECRET`.
- Refresh: random opaque value; only SHA-256 hash stored in `user_sessions`.
- Password: Argon2id hash in `users.password_hash`.
- Password reset token: random opaque; SHA-256 hash in `password_reset_tokens`; TTL 30 minutes, single use.

## 6. Database schema (migration `0011_users_auth`)

- `users`: id (uuid), unique email, password_hash, role (`admin` only in this cut), status, timestamps.
- `user_sessions`: refresh_token_hash, user_id, expires_at, revoked_at, replaced_by.
- `password_reset_tokens`: token_hash, user_id, expires_at, used_at.

## 7. SMTP (minimal)

Environment: `AIHUB_SMTP_HOST`, `AIHUB_SMTP_PORT`, `AIHUB_SMTP_USER`, `AIHUB_SMTP_PASSWORD`, `AIHUB_SMTP_FROM`, `AIHUB_PUBLIC_APP_URL`. Reset link: `{AIHUB_PUBLIC_APP_URL}/reset-password?token=...`.

## 8. Security

- No passwords, refresh values, or reset tokens in logs, generation records, or model payloads.
- Login and recovery use constant-time comparisons where applicable.
- Parameterized SQL only (pgx placeholders).

## 9. Acceptance criteria

1. Without `AIHUB_DATABASE_URL`, operator/loopback tests pass unchanged.
2. With postgres configured, loopback requests without cookies receive `401` on `/api/*` and `/v1/*`.
3. Login sets cookies; `GET /api/ui/session` returns the user; gateway routes accept the access cookie like today’s operator session.
4. Expired access yields `401`; `POST /api/ui/refresh` with valid refresh restores access; reused revoked refresh revokes the user’s sessions.
5. Forgot-password always returns the generic message; reset with valid token changes password and invalidates sessions.
6. Bootstrap creates admin only when `users` is empty.
7. Rate limits return `429` when exceeded.
8. Migration `0011` applies cleanly via embedded migrate.

## 10. Out of scope

Public signup, Google OAuth, email confirmation, multi-role RBAC, per-user API keys, per-user generation isolation.
