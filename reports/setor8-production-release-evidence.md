# Setor 8 — Production release evidence (2026-11-04)

## Session objective

Close MVP Setor 8: embedded SPA, operator session boundary, E2E scaffold, validate Setor 7 integrated slice.

## Git baseline (start)

- Branch `main`, HEAD `cf7734f` (FE-05 executions), ahead of `origin/main` by 1; FE-06 Resources uncommitted under `apps/frontend/src/features/resources/`.

## Git baseline (end)

- Setor 8 Go/docs/scripts/e2e changes unstaged; FE-06 tree preserved; no commit/push performed.

## Artifacts

| Artifact | Path |
| --- | --- |
| SPEC | `docs/specs/FE-08-production-release.md` |
| Session decision | `docs/decisions/frontend-operator-session-2026-11-04.md` |
| Sync script | `scripts/sync-frontend-dist.mjs` |
| E2E hub | `scripts/e2e-hub.sh`, `e2e/` |

## Architecture

- `internal/static`: `go:embed dist`, SPA fallback, `static.Wrap` before API mux.
- `internal/ui`: operator gate (loopback trust + signed cookie session).
- `internal/access`: `OperatorGate` in policy; `/api/ui/login` exempt from auth.
- Routes: `POST /api/ui/login`, `POST /api/ui/logout`, `GET /api/ui/session`.
- Enable embed: `AIHUB_SERVE_EMBEDDED_UI=true` + build sync dist.

## Commands and results

| Command | Result |
| --- | --- |
| `pnpm --filter @connexto/frontend check` | PASS (56 tests) |
| `pnpm --filter @connexto/frontend build` + sync | PASS |
| `go test ./... -count=1` | PASS (all packages) |
| `go vet ./...` | PASS |
| `go build ./cmd/ai-hub/` | PASS |
| `git diff --check` | PASS |
| Embedded smoke `:18080` | health 200, index 200, session loopback_trust, records 200, policies 200 |
| `pnpm e2e` | FAIL — Playwright cannot install chromium on `ubuntu20.04-x64` in this environment |

## Setor 7 gate

- Functional: Vite `:5173` + backend `:8080`; overview screenshot; executions/resources routes wired.
- Doop canvas `Qactw8snfC` consulted; frames Overview–Executions `ready_for_dev`; Resources frame draft.
- Screenshot: `reports/setor8-gate-zero-overview-1280.png`

## Security / credentials

- Grep `internal/static/dist`: no `AIHUB_` or bearer strings in bundle.
- Keys remain server-only; loopback trust documented in decision M1.

## Code review

Subagent `senior-code-review-judge` unavailable in Task registry; author self-review against judge rubric:

- **Verdict:** APPROVED_WITH_NOTES
- **Notes:** (1) Passphrase mode lacks dedicated login page UI — API only until FE follow-up. (2) Playwright E2E blocked on host platform — Go smoke + manual browser used instead. (3) `internal/static/dist` is build output — add to `.gitignore`.

## Pending

- Lead approval recorded in decision doc for operator session (aligned with discovery §6.3).
- Commit/push Setor 7 + Setor 8 when authorized.
- Run Playwright E2E on CI/agent with supported browser platform.
- Optional: login screen when `AIHUB_OPERATOR_PASSPHRASE` set.

## Risks

- Loopback trust assumes trusted local machine; passphrase required for non-loopback embed (validated in config).
