# Development Workspace — pnpm + Turborepo (Phase A)

This document describes the hybrid monorepo workspace set up in the AI Hub
Management Foundation Phase A and how the prioritized Frontend MVP can join it.
No frontend application exists yet. Current frontend preparation and API
security constraints are documented in the
[Frontend MVP guide](frontend/FRONTEND-MVP.md).
The tooling decision is recorded in
[ADR-019](adr/ADR-019-hybrid-monorepo-workspace-tooling.md).

## Layout

```text
/
├── apps/
│   ├── backend/      # Go backend module `aihub` (`go.mod`, `cmd/`, `internal/`)
│   └── frontend/     # React + Vite dashboard (`@connexto/frontend`)
├── packages/ui/      # Future shared UI kit (not created yet)
├── scripts/          # Validation, sync, and forensic scripts (Node/Python/Bash)
├── docs/             # Architecture, specs, ADRs, validation docs
├── bin/              # Release / smoke / E2E binaries (gitignored)
├── package.json      # Root scripts + pnpm/Turbo toolchain pins
├── pnpm-workspace.yaml
├── turbo.json
├── pnpm-lock.yaml
└── .nvmrc
```

The Go backend module (`aihub`) lives in `apps/backend/` as a pure Go module
without a `package.json`, keeping the Go and Node ecosystems independent while
sharing the monorepo `apps/*` directory structure.

## Tool responsibilities

| Tool       | Responsibility                                     |
| ---------- | -------------------------------------------------- |
| Go Modules | Backend dependencies and build                     |
| pnpm       | JavaScript/TypeScript dependencies and workspace   |
| Turborepo  | Workspace (`apps/*`, `packages/*`) task orchestration |
| PostgreSQL | Persistence foundation (Phase B)                   |
| React + Vite | Planned frontend direction; no app exists yet    |

The ecosystems are independent: Go never sees `node_modules`, and no Go file
is exposed as a pnpm package. Turborepo coordinates only the JS/TS packages.

## Requirements

- Go 1.23+ (backend only).
- Node.js 22+ and pnpm 12 — required only to run workspace tooling
  (`pnpm`, `turbo`), never to build or test the backend alone.
- The root `package.json` pins `packageManager: pnpm@12.16.1` and
  `engines.node >= 22`; `.nvmrc` pins major Node 24. With nvm installed,
  entering the repository with the provided shell hooks activates Node 24
  automatically.

## Commands

Run from the repository root:

| Command        | What it runs                                  | Implemented |
| -------------- | --------------------------------------------- | ----------- |
| `pnpm install` | Installs JS/TS workspace dependencies         | yes         |
| `pnpm build`   | `go -C apps/backend build ./...`              | yes         |
| `pnpm dev`     | `turbo run dev //#dev:backend`                | yes         |
| `pnpm lint`    | `go -C apps/backend vet ./...`                | yes         |
| `pnpm test`    | `go -C apps/backend test ./... -count=1 -race`| yes         |
| `pnpm check`   | `go -C apps/backend vet ./...` + `go -C apps/backend test ./...` + frontend check | yes |
| `pnpm typecheck` | — provided by frontend/TS packages          | frontend    |

Every backend command works without pnpm as well
(`go -C apps/backend test ./... -count=1 -race`, or `go test ./... -count=1 -race` from `apps/backend/`).

`golangci-lint run ./...` remains the documented deeper manual lint. It is not
part of the default gate; it currently reports pre-existing staticcheck
findings in the E07 platform-support files, which are out of Phase A scope.

## Turborepo pipeline

`turbo.json` declares the tasks `build`, `dev`, `lint`, `typecheck`, `test`
and `check` for workspace packages (`apps/*`, `packages/*`):

- `build` caches `dist/**` and `bin/**` outputs and depends on `^build`.
- `dev` is persistent and uncached, and depends on `^build`.
- `check` depends on `^check`.
- `lint`, `typecheck` and `test` run on every matched package.

while the workspace has zero JS/TS packages, `turbo run <task>` executes zero
tasks (verified with turbo 2.11.10; repository root scripts are outside the
package scope by design, so the root scripts are never faked as a pnpm
package). That is why today's Go gate lives in root scripts (`pnpm lint`,
`pnpm test`, `pnpm check`) and `typecheck` has no implementation until the
first TypeScript package exists.

## Adding the Frontend MVP

Phase A deliberately does not create an empty frontend application. When the
Frontend MVP implementation starts, follow this recipe; no structural
reorganization of the Go tree is required:

1. Scaffold the application:
   ```bash
   pnpm create vite@latest apps/frontend -- --template react-ts
   ```
2. Keep Vite's `dist` output under `apps/frontend/dist` so the `build` task's
   declared `outputs: ["dist/**", "bin/**"]` caches it.
3. Ensure `apps/frontend/package.json` defines the scripts the pipeline
   expects, at minimum `build`. Prefer also `dev`, `lint`, `typecheck`,
   `test` and `check`; `pnpm-workspace.yaml` already includes `apps/*`, so
   Turborepo picks the package up with no config change.
4. When the first package with a `typecheck` script lands, `turbo run
   typecheck` starts executing it; the declared-but-unimplemented task stops
   being a no-op.
5. Extend the root gate to include workspace packages once they exist, for
   example:
   ```json
   "check": "go vet ./... && go test ./... -count=1 -race && turbo run check"
   ```
6. Shared React components follow the same pattern under `packages/ui`
   (`packages/*` is already a workspace glob), consumed by ordinary pnpm
   workspace dependencies.

The earlier dashboard proposal listed React 19, TypeScript, Vite, TanStack
Query, Tailwind CSS, and shadcn/ui. Those packages are not installed in the
current checkout. Select the frontend stack and justify its dependencies in
the implementation SPEC; see the [Frontend MVP guide](frontend/FRONTEND-MVP.md).

## Known limitations

- No CI pipeline yet; `pnpm check` is the local gate.
- `turbo` is installed as a root devDependency and is inert until the first
  workspace package exists.
- One process owns `AIHUB_DATA_DIR`; workspace tooling adds no runtime
  services to the backend.
- PostgreSQL is optional for local development. When using Phase B persistence,
  reuse the existing `local-postgres` container (published on port `5432`)
  instead of creating another PostgreSQL container. For example:
  ```bash
  AIHUB_DATABASE_URL='postgres://dev:dev@127.0.0.1:5432/ai_hub' \
    AIHUB_CATALOG_PERSISTENCE=postgres \
    AIHUB_GATEWAY_API_KEY='<32+ character local key>' \
    go -C apps/backend run ./cmd/ai-hub
  ```
  The application creates and migrates its schema on startup; it does not
  create databases or containers. Integration tests opt in through
  `AIHUB_TEST_DATABASE_URL` and use disposable databases on that same server.
