# Antigravity Storage Isolation Plan

## Problem Statement

When the Connexto Proxy runs on the host system or inside WSL where Antigravity IDE is also running, both environments previously defaulted to reading and writing credentials under `$HOME/.gemini` (e.g., `oauth_creds.json`, `jetski-standalone-oauth-token`, `google_accounts.json`).

This created severe operational conflicts:
1. **Accidental IDE Logout**: Clicking "Encerrar sessão" in Connexto AI Hub or calling `POST /v1/runtime/sessions/clear` / `POST /v1/providers/antigravity/auth/logout` purged `$HOME/.gemini`, causing the host Antigravity IDE to immediately lose its authentication session ("Authentication expired. Your session has expired. Please log back in to continue.").
2. **Account Clobbering**: Performing an OAuth login in the proxy overwritten the user's IDE accounts and active refresh tokens.
3. **Usage Conflicts**: Usage quota queries for the proxy read credentials from the IDE session instead of an independent proxy session.

## Architecture & Isolation Solution

The Connexto Proxy establishes dedicated storage isolation for all Antigravity operations:

### 1. Configurable Storage Root
The proxy introduces the `ANTIGRAVITY_STORAGE_PATH` environment variable:
- If specified, the proxy uses this absolute directory path.
- If unspecified, the proxy defaults to `path.join(os.homedir(), ".connexto-proxy", "antigravity")`.

### 2. Service-Level Propagation
In `apps/api/src/services/gateway.ts`, `createGatewayServices` receives `ANTIGRAVITY_STORAGE_PATH` and injects it as `homeDir` to:
- `AntigravityAuthManager({ homeDir })`: writes OAuth tokens, operation tokens, and account info into `<storagePath>/.gemini` and `<storagePath>/.config/antigravity`.
- `AntigravityTokenResolver({ homeDir })`: resolves and refreshes tokens exclusively inside `<storagePath>/.gemini`.
- `AntigravityUsageService({ homeDir })`: queries quotas and consumption using credentials resolved from `<storagePath>`.
- `AntigravityProvider({ homeDir })`: forwards an isolated process environment to the spawned `agy` ACP process (see **Single proxy session** below).

### 3. Single Proxy Session (Auth, Usage, and `agy`)

All Antigravity operations inside the proxy share **one** credential session under `ANTIGRAVITY_STORAGE_PATH` (default `~/.connexto-proxy/antigravity`):

| Consumer | Reads / writes |
| --- | --- |
| Providers OAuth (`AntigravityAuthManager`) | `<storage>/.gemini/*`, `<storage>/.config/antigravity/*` |
| Usage / quota (`AntigravityTokenResolver`) | Same token files under `<storage>/.gemini` |
| Codex turns (`agy acp` via `AntigravityProvider`) | Same tree via `HOME` + `ANTIGRAVITY_APP_DATA_DIR` |

OAuth persistence writes the **nested CLI token format** to both:

- `<storage>/.gemini/jetski-standalone-oauth-token`
- `<storage>/.gemini/antigravity-cli/antigravity-oauth-token`

so `agy` deserializes the same account the Providers UI logged in with.

The host Antigravity IDE (`$HOME/.gemini`) and the desktop **secret service / keyring** are **not** part of this session. When `homeDir` is set, the `agy` child process uses:

- `HOME` and `ANTIGRAVITY_APP_DATA_DIR` = absolute storage root
- `XDG_CONFIG_HOME` = `<storage>/.config`
- `XDG_DATA_HOME` = `<storage>/.local/share`
- No `DBUS_SESSION_BUS_ADDRESS` (avoids IDE keyring winning over on-disk tokens)

Interop variables (`PATH`, `DISPLAY`, `LANG`, WSL) are preserved so the CLI binary and browser OAuth still work when no token file exists yet.

### 4. Isolated Purge & Logout
When `antigravityAuthManager.clearCredentials()` is invoked during `POST /v1/runtime/sessions/clear` or `POST /v1/providers/antigravity/auth/logout`, it deletes only the credential files within `this.homeDir` (`<storagePath>/.gemini/*`).

The host IDE's files at `$HOME/.gemini` remain completely untouched, keeping the developer's IDE logged in.

### 5. Verification Checklist

After a real Providers login:

1. `getStatus()` account matches the email in both CLI token files.
2. An `agy acp` turn with isolated env reuses that account (no `authenticated via keyring` for a different IDE user).
3. `POST .../auth/logout` or runtime session clear removes only files under `<storage>`, not `$HOME/.gemini`.

