# Repository Analysis — connexto-aihub

**Generated:** 2026-11-07 (America/Sao_Paulo)  
**Git remote:** `https://github.com/rafaeljfec2/connexto-aihub.git`  
**HEAD:** `44d2d9eded0e1876fe3cab9b51e4da370dbd1133` — fix: handle window bucketing and metrics for generative model pools in antigravity categories  
**AI Memory project:** `default/rafaeljfec2-connexto-aihub` (`connexto.project.json`)

## Executive summary

**AI Hub** is a local-first Personal AI Orchestrator: a Go modular monolith that exposes an OpenAI-compatible gateway (`/v1/models`, `/v1/responses`), multi-provider administration, deterministic routing (E08), resource and concurrency governance (E09), and optional PostgreSQL persistence. A **React 19 + Vite 6** frontend (`apps/frontend`, `@connexto/frontend`) provides operator UI for overview, providers, catalog, playground, executions, and resources, with cookie-based operator sessions (`internal/ui`) and same-origin API access.

The backend roadmap documents E01–E09 as delivered or closed; E08.6–E08.8 remain deferred; E10 and E11 are future work. The frontend has evolved substantially since the 2026-11-03 FRONTEND-MVP preparation document (which still describes a pre-implementation state).

## Verified scale (repository scan 2026-11-07)

| Layer | Count | Notes |
| --- | ---: | --- |
| Go source files | 462 | Excludes `apps/` and `node_modules` |
| Go physical LOC | 59,458 | `wc -l` aggregate |
| Frontend `src` TS/TSX files | 235 | Under `apps/frontend/src` |
| Frontend physical LOC | 17,708 | `wc -l` aggregate |
| Markdown under `docs/` | 92 | ADRs, specs, validation, roadmap |
| Go `*_test.go` files | 220 | Includes E07 harness and PG integration |
| ADR documents | 24 | `docs/adr/*.md` |
| Formal specs (`docs/specs/`) | 21 | Backend epics + FE increments |
| DB migration versions | 9 | `0001`–`0009` embedded pairs in `internal/db/migrations` |

**Monorepo tooling:** pnpm 12.16.1, Turborepo 2.11.10, Node.js ≥ 22. Root `package.json` scripts orchestrate Go build, vet, test, and frontend `check` (`pnpm check`). There is no `packages/` workspace yet; the hybrid monorepo is `cmd/` + `internal/` (Go) and `apps/frontend` (SPA).

**Top-level structure:**

- `cmd/ai-hub/` — process entry, route wiring (`gateway_routes.go`, `provider_admin.go`, `ui_routes.go`, analytics, concurrency, resource admin)
- `internal/` — domain packages (see inventory below)
- `apps/frontend/` — operator SPA
- `docs/` — product, specification, and ADR corpus
- `e2e/` — Playwright gate-zero visual captures
- `scripts/` — `sync-frontend-dist.mjs`, `sync-ai-icons.mjs`, etc.
- `.connexto/` — Connexto workflow run artifacts (`FEAT-007`, `BUG-001`)
- `reports/` — delivery and audit reports (E09, frontend QA, this analysis)

## Recent development trajectory

Commits after the roadmap snapshot (`9338c538`, E09 closure) focus on **Connexto Proxy integrations** and **operator-facing quota UX**, not new backend epics:

| Theme | Representative commits |
| --- | --- |
| Antigravity provider | OAuth flow, quota tracking, overview drawer sections, generative pool window bucketing in `antigravityCategories.ts` |
| Cursor provider | Proxy usage collect endpoint, session retrieval, gateway auth in providers UI, collect readiness and error messaging |
| Frontend polish | Provider drawer layout, quota presentation rounding, login error copy |

Product authorization in `docs/roadmap/AI-HUB-ROADMAP.md` still lists **Frontend MVP** as the next front; in practice the repository already ships a multi-route operator SPA plus proxy quota surfaces—roadmap and README should be reconciled.

## Backend architecture

**Pattern:** Single binary, explicit `internal/app` module lifecycle (ordered start, reverse-order stop, rollback on failure). HTTP serving uses Fiber v2 with a net/http-compatible routing tree (`internal/server`).

**Persistence modes:**

- **File-first:** catalog snapshots and E06 JSONL execution records under `AIHUB_DATA_DIR`
- **PostgreSQL** (`AIHUB_DATABASE_URL`): embedded migrations at startup; providers, catalog, E09 policies, metering, generations, provider quota accounts

**PostgreSQL migration timeline:**

| Version | Focus |
| ---: | --- |
| 0001 | Initial schema |
| 0002 | Provider management |
| 0003 | Resource admission (E09) |
| 0004 | Concurrency policies |
| 0005 | Provider execution usage |
| 0006 | Generation telemetry |
| 0007 | Provider management credential extensions |
| 0008 | Provider execution list cost |
| 0009 | Remove legacy transport provider |

**Security model:**

| Credential | Scope |
| --- | --- |
| `AIHUB_GATEWAY_API_KEY` | `/v1/*`, diagnostics, provider inference paths |
| `AIHUB_ADMIN_API_KEY` | `/api/*` administration (required in PostgreSQL mode) |
| Operator UI passphrase + `AIHUB_UI_SESSION_SECRET` | `/api/ui/login`, session cookie (`internal/ui`) |

Host and Origin validation use exact configured values; the server is fail-closed and does not implement CORS. Provider and admin credentials are sealed server-side and are not returned by list/get admin APIs.

**Provider templates** (`internal/providers/service.go` → `AvailableProviders`):

| ID | Connection type | Integration |
| --- | --- | --- |
| `cursor` | Connexto Proxy | Models + Responses; gateway management key; assisted proxy auth and usage collect |
| `antigravity` | Connexto Proxy | OAuth at proxy runtime; quota APIs |
| `openrouter` | Direct API | OpenRouter catalog sync and Responses |
| `opencode-go` | Direct API | OpenCode Go inference endpoint |

Default Connexto Proxy base URL: `http://127.0.0.1:9090/v1` (see `.env.example`).

**HTTP surface (representative):**

- **Public health:** `GET /healthz`, `GET /readyz`
- **Gateway:** `POST /v1/responses`, `GET /v1/models`, catalog query/sync, execution records, generations list/detail
- **Provider admin:** CRUD, validate, enable/disable, `sync-models`, proxy session clear, Cursor/Antigravity auth flows, `GET /api/ui/providers/quotas`
- **E09 admin:** resource and concurrency policies, usage endpoints, resource analytics (admissions, quota utilization, provider usage, distributions)
- **Runtime / telemetry:** `GET /api/runtime/meta`, `GET /api/telemetry/traffic-summary`
- **UI:** `POST /api/ui/login|logout`, `GET /api/ui/session`
- **Events:** `GET /api/events` (operational SSE)
- **Static UI:** `internal/static` serves embedded frontend when built via `pnpm build:release`

**`internal/` package inventory:**

| Package | Responsibility |
| --- | --- |
| `access` | Bearer auth, Host/Origin admission, operator vs gateway authorization |
| `analytics` | Resource analytics queries (PostgreSQL) |
| `app` | Module lifecycle orchestration |
| `catalog` | Model catalog domain, sync, capabilities, pricing |
| `concurrency` | Concurrency policies and usage (E09) |
| `config` | Typed configuration from environment |
| `db` | Connection pool, embedded migrations |
| `execution` | Per-request lifecycle, streaming, shutdown coordination |
| `gateway` | HTTP handlers for gateway, admin, UI auth, analytics |
| `generations` | Generation records query service |
| `health` | Liveness and readiness |
| `metering` | Provider usage capture |
| `observability` | Structured logging, correlation IDs |
| `operational` | Hub operational state and events |
| `providers` | Multi-provider adapters, quotas, proxy integration |
| `records` | Content-free execution records (JSONL) |
| `resources` | Resource admission policies (E09) |
| `routing` | Deterministic E08 model/provider selection |
| `server` | Listener, middleware, route registration |
| `static` | Embedded SPA asset serving |
| `ui` | Operator session gate (HMAC cookie) |

**E07 harness:** `cmd/ai-hub/e07/` contains extensive Codex CLI validation support (integrity mounts, live scenarios, records verification, codex environment policy). This is the bounded evidence path for harness integration—not general production runtime.

## Frontend architecture

**Stack:** React 19, React Router 7, TanStack Query 5, Tailwind CSS 4 (Vite plugin), Radix UI, react-hook-form, Zod 4, Recharts, Vitest, Testing Library.

**Routes** (`apps/frontend/src/app/router.tsx`):

| Path | Page |
| --- | --- |
| `/login` | Operator login |
| `/` | Overview |
| `/providers` | Provider administration UI |
| `/catalog` | Model catalog browser |
| `/playground` | Inference playground |
| `/executions` | Generations / activity |
| `/resources` | Resource and concurrency policies |

**Code organization:**

- `features/*` — domain UI (auth, overview, providers, catalog, playground, executions, resources)
- `shared/api` — `fetchJson` client to same-origin `/api` and proxied routes
- `shared/ui` — design-system primitives (dialog, buttons, copy helpers)
- `shared/branding` — AI provider icon catalog (hundreds of SVGs, `sync:ai-icons` script)

**Build path:** `pnpm build:release` → frontend `vite build` → `scripts/sync-frontend-dist.mjs` → `go build -o bin/ai-hub`.

## Documentation and product state

- **24 ADRs** (`docs/adr/`) covering HTTP stack, configuration, access control, catalog, execution, records, hybrid monorepo, PostgreSQL, multi-provider runtime, deterministic routing, resource admission
- **21 specifications** under `docs/specs/` (E08, E09, C2 compatibility, FE-01–FE-09 frontend increments, quota validation)
- **Roadmap** (`docs/roadmap/AI-HUB-ROADMAP.md`, status date 2026-11-03) — cites E09 closure at `9338c538`; repository HEAD is newer
- **Architecture baseline** (`docs/architecture.md`) — E01–E06 diagram; points to roadmap for later increments
- **Kickoff** (`AI-Hub-Product-Architecture-Kickoff.md`) — long-term product vision

## Testing and quality gates

| Command | Purpose |
| --- | --- |
| `go test ./... -count=1 -race` | Backend unit and integration (Connexto default in `connexto.project.json`) |
| `go vet ./...` | Static analysis |
| `pnpm --filter @connexto/frontend check` | Typecheck + Vitest |
| `pnpm e2e` | Playwright (Chromium) visual gate-zero |
| `pnpm check` | Combined Go vet/test + frontend check |

E09 closure documentation references ~982 passing Go tests with race detector. The 2026-11-07 analysis container had **no `go` binary on PATH**, so live test counts were not re-run during this scan.

## Dependencies

**Go (`go.mod`):** module `aihub`, Go 1.25; direct dependencies `github.com/gofiber/fiber/v2`, `github.com/jackc/pgx/v5`.

**Note:** Root `README.md` still states "No external Go dependencies" under Requirements; this is **outdated** relative to `go.mod`.

## Configuration highlights (`.env.example`)

- Server binds `127.0.0.1:8080` by default with bounded timeouts
- Mandatory `AIHUB_GATEWAY_API_KEY` for all startups (including loopback)
- Optional PostgreSQL DSN enables full admin and persistence features
- Connexto Proxy URLs and gateway keys for Cursor/Antigravity
- Catalog staleness, records retention, execution timeouts documented inline

## Engineering governance

- **Root `AGENTS.md`:** mandates `/sdd-workflow` for approved SPECs; autonomous phase execution within SPEC scope; halt only on ambiguity, scope expansion, or material risk.
- **`connexto.project.json`:** Go tests with race detector required; lint via `go vet`; AI Memory scope `default/rafaeljfec2-connexto-aihub` with `memory.enabled: true` but no automated `memory_write_command` configured in-repo.

## Risks, documentation drift, and constraints

1. **README** still claims no frontend application exists; `apps/frontend` is a full MVP-scale UI.
2. **`docs/frontend/FRONTEND-MVP.md`** and **roadmap "next front"** describe pre-implementation state from 2026-11-03; should be updated or superseded by a shipped-frontend status doc.
3. **Operational complexity:** three secret classes (gateway, admin, UI session) plus per-provider credentials and proxy gateway keys.
4. **Hard dependency on Connexto Proxy** for Cursor and Antigravity in default configuration.
5. **Agent file-length rule (300 lines):** some existing Go test files exceed 250 lines; new edits should split before adding features.
6. **Deferred scope:** E08.6–E08.8, E10 Learning Engine, E11 Dashboard — do not implement without new product authorization.

## Contributor entry points

| Concern | Location |
| --- | --- |
| Bootstrap | `cmd/ai-hub/main.go`, `cmd/ai-hub/lifecycle/` |
| Routes | `cmd/ai-hub/gateway_routes.go`, `provider_admin.go`, `ui_routes.go` |
| Configuration | `internal/config/`, `.env.example` |
| Inference path | `internal/gateway/responses_*.go`, `internal/routing/` |
| Provider adapters | `internal/providers/` |
| Frontend shell | `apps/frontend/src/app/router.tsx`, `RootLayout` |
| Product status | `docs/roadmap/AI-HUB-ROADMAP.md` |

---

*Durable project context for AI Memory. Canonical repo copy: `reports/repository-analysis-2026-11-07.md`; working mirror: `docs/research/repository-analysis-2026-11-07.md`.*
