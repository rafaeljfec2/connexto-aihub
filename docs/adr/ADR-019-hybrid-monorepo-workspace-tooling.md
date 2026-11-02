# ADR-019 — Hybrid monorepo: independent Go Modules and pnpm ecosystems with Turborepo orchestration

- **Status:** accepted (Management Foundation Phase A, 2026-11-02)
- **Related:** `README.md`, `docs/development-workspace.md`, ADR-007, ADR-002

## Context

The AI Hub Management Foundation roadmap adds a future React + Vite dashboard,
shared JavaScript/TypeScript packages and PostgreSQL persistence, on top of a
backend that is a single Go module (`aihub`) with zero external dependencies.
The repository now needs JavaScript/TypeScript tooling (pnpm) and a task
orchestrator (Turborepo) without coupling the Go backend to the Node
ecosystem or restructuring existing Go code unnecessarily.

## Decision

1. **Keep the Go backend at the repository root** as the Go module `aihub`
   (`go.mod`, `internal/`, `cmd/`). Do not move it into
   `apps/backend`; the current layout already is the authority.
2. The **pnpm workspace covers only `apps/*` and `packages/*`** globs. The
   backend is not a workspace package.
3. **Ecosystem separation** (per product decision):

   | Tool            | Responsibility                                    |
   | --------------- | ------------------------------------------------- |
   | Go Modules      | Backend dependencies and build                    |
   | pnpm            | JavaScript/TypeScript dependencies and workspace  |
   | Turborepo       | Workspace task orchestration                      |
   | PostgreSQL      | Persistence (Phase B)                             |
   | React + Vite    | Future dashboard application                      |

4. The backend keeps **native Go commands**. Root `package.json` scripts call
   `go build` / `go run` / `go vet` / `go test` directly; none of them pretend
   the backend is a pnpm package, and no `node_modules` artifact participates
   in the Go build.
5. Root `pnpm check` runs the Go gate directly
   (`go vet ./... && go test ./... -count=1 -race`). Turborepo package scopes
   cover `apps/*`/`packages/*` only: while no JS/TS workspace package exists,
   `turbo run <task>` executes zero tasks (verified), which is why the Go
   commands live in root scripts instead of a fake root package. When the
   first workspace package lands, its own `build`/`dev`/`lint`/`typecheck`/
   `test`/`check` scripts join `turbo.json` automatically, and the guide's
   recipe shows how to extend the root `check`.
6. Toolchain is pinned: `packageManager: pnpm@12.16.1`,
   `engines.node >= 22`, and `.nvmrc = 24`.
7. `pnpm-lock.yaml` is committed for reproducible JavaScript installs.

## Rationale

- Independence of ecosystems avoids coupling: Go updates never require Node
  changes and vice versa; the backend keeps its zero-dependency posture.
- Turborepo is honest about ownership: it only coordinates the JS/TS
  workspace, which satisfies the "no artificial Go-as-pnpm tasks" constraint.
- Declaring `typecheck` in `turbo.json` ahead of the first TypeScript package
  prepares the pipeline; Turborepo exits successfully with zero matched tasks
  today, so no dead command fails the gate.

## Consequences

- Root scripts are the single truth for backend validation
  (`pnpm build/lint/test/check`); `go test ./...` must keep working without
  pnpm.
- `typecheck` has no implementation until the first TS package exists; `lint`
  maps to `go vet ./...`. `golangci-lint run ./...` remains the documented
  deeper manual lint and is not part of the default gate.
- The future frontend is expected at `apps/frontend` (with `packages/ui` for
  shared components) and must follow the documented workspace guide instead of
  restructuring Go code.
- Node.js tooling availability does not affect the ability to run or test the
  backend.
