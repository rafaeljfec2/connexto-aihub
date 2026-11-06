# Frontend Technical & Functional Discovery — Connexto AI Hub

**Status:** Completed Discovery Baseline  
**Date:** 2026-11-03  
**Repository Snapshot:** `main` at `4a2ba70fb47125413802f68f001572d4d3545bd5`  
**Reference Documents:**  
- `docs/roadmap/AI-HUB-ROADMAP.md`
- `docs/frontend/FRONTEND-MVP.md`
- `docs/architecture.md`
- `docs/development-workspace.md`
- `docs/adr/ADR-019-hybrid-monorepo-workspace-tooling.md`
- `docs/adr/ADR-011-gateway-authentication-and-network-exposure.md`
- `docs/adr/ADR-020-postgresql-persistence-foundation.md`
- `docs/adr/ADR-021-provider-selection-and-openrouter-runtime.md`
- `docs/adr/ADR-022-deterministic-intelligent-routing.md`

---

## 1. Executive Summary

Connexto AI Hub has completed its foundational backend stages (E01–E07, Phase B PostgreSQL persistence, C1/C2 provider management, E08.1–E08.5 deterministic routing, and E09 resource management/analytics). While the backend is fully operational, verified with 982 automated tests, and capable of multi-provider orchestration, it currently lacks any graphical user interface.

This technical discovery establishes the product, architectural, operational, and UX/UI blueprints required to turn the Go backend into an intuitive, secure, and resilient application.

### Key Discoveries & Recommendations:
1. **Product Positioning:** AI Hub is a local-first Personal AI Orchestrator and multi-provider operational control center. For the MVP, it targets a **single trusted operator** managing AI infrastructure, eliminating multi-tenant authentication complexity while maintaining strict security boundaries.
2. **Backend Communication & Security Boundary:** The Go backend accepts two distinct static bearer tokens (`AIHUB_GATEWAY_API_KEY` and `AIHUB_ADMIN_API_KEY`) and enforces strict Host/Origin policies without CORS or preflight handling. Browsers cannot call the backend directly without leaking administrative credentials. **Recommendation:** Implement **Alternative A (Integrated Go BFF / Static Serving)** — embedding the built SPA directly into the Go binary (`go:embed`) or providing a same-origin loopback proxy during development, terminating browser sessions locally and injecting backend bearer tokens strictly server-side.
3. **Frontend Stack:** Modern React 19 + TypeScript + Vite within `apps/frontend`, integrated into the existing pnpm + Turborepo monorepo structure defined in ADR-019. State and server synchronization via TanStack Query v5; UI built with Tailwind CSS v4, Lucide icons, and accessible headless Radix UI primitives.
4. **Zero-Polling Mandate (Real-Time SSE):** Periodic data polling (`setInterval`, `refetchInterval`) is **strictly forbidden**. All backend notifications, system health updates, provider diagnostic changes, quota alerts, and model streaming must be pushed from the backend to the frontend in real time via Server-Sent Events (SSE).
5. **Vertical Slice FE-01:** Begin with a minimal, fully functional vertical slice: **FE-01 — BFF + System Health + Provider Overview**. This validates the trusted server-side transport, liveness/readiness reporting, and provider diagnostic states via real-time SSE before any write operations or playground streaming are introduced.

---

## 2. Current Context & Repository Baseline

### 2.1 Workspace & Git Inspection
- **Directory:** `/home/rafael/dev-rafael/connexto-aihub`
- **Branch:** `main`
- **HEAD Commit:** `4a2ba70` (`docs: consolidate roadmap and prepare frontend MVP`)
- **Remote Tracking:** In sync with `origin/main` (`Your branch is up to date with 'origin/main'`).
- **Working Tree:** Completely clean (`nothing to commit, working tree clean`).
- **Monorepo Structure:**
  - Root: Go modular monolith (`go.mod`, `cmd/ai-hub/`, `internal/`).
  - JS Tooling: Node.js 22+ (pinned via `.nvmrc = 24`), pnpm 12.16.1 (`pnpm-workspace.yaml`, `pnpm-lock.yaml`), Turborepo 2.11.10 (`turbo.json`).
  - Directories `apps/` and `packages/` are configured in `pnpm-workspace.yaml` but do not yet exist on disk. No frontend dependencies, React packages, or JavaScript workspace packages are installed.

### 2.2 Backend Baseline Capabilities
- **Operational Health:** Public `GET /healthz` (process liveness) and `GET /readyz` (dependency readiness). Authenticated `GET /diagnostics/providers` (runtime health and availability).
- **Model Catalog:** Aggregated provider models (`GET /v1/models`), optional per-provider native list via `X-AIHub-Provider`, normalized catalog with capability provenance (`GET /v1/catalog/models`), single model lookup (`GET /v1/catalog/models/{provider}/{model...}`), provider sync state (`GET /v1/catalog/providers`), and sync trigger (`POST /v1/catalog/sync`).
- **Provider Administration (PostgreSQL Mode):** Full CRUD, credential sealing, probe validation (`POST /api/providers/{id}/validate`), runtime enable/disable (`POST /api/providers/{id}/enable|disable`). Secrets are write-only and never returned in API responses.
- **Execution & Records:** Streaming responses (`POST /v1/responses`) using OpenAI Responses SSE protocol with deterministic routing (E08). Audit trail via content-free execution records (`GET /v1/records/executions` and `/executions/{id}`).
- **Resource Management (E09):** Daily quota policies (`/api/resource-policies`), concurrency limits (`/api/concurrency-policies`), real-time usage meters (`/api/resource-usage`, `/api/concurrency-usage`), and comprehensive analytics series (`/api/resource-analytics/*`).

---

## 3. Personas & User Journeys

### 3.1 Product Positioning
- **Primary Problem Solved:** Provides unified visibility, routing governance, and execution testing across multiple upstream AI providers (OpenCode Go, OpenRouter, and future models) without requiring custom code or manual CLI testing.
- **Primary Persona:** The **AI Platform Engineer / Technical Operator** who configures providers, ensures uptime, sets quotas, and monitors traffic.
- **Secondary Persona:** The **Application Developer / AI Engineer** who tests model outputs in a playground, evaluates latency and token usage, and diagnoses execution failures.
- **First Experience:** Landing on an operational **Overview Dashboard** displaying:
  1. Instant system health status (Core Gateway + PostgreSQL + Providers).
  2. Active vs. Degraded/Offline providers.
  3. Real-time concurrency saturation and daily quota consumption.
  4. Quick action buttons: "Launch Playground" and "Inspect Providers".

### 3.2 User Journeys

#### Journey 1: Operator — Routine Health & Availability Check
1. Operator navigates to AI Hub web URL.
2. System displays overall health badge ("HEALTHY" / "DEGRADED").
3. Provider card list displays OpenCode Go (green, reachable) and OpenRouter (green, reachable).
4. If a provider experiences an upstream outage, the diagnostic status turns red/yellow with the last probe timestamp and error cause.
5. Operator triggers a manual connectivity validation (`POST /api/providers/{id}/validate`) without leaving the page.

#### Journey 2: Developer — Model Exploration & Playground Testing
1. Developer enters the **Playground**.
2. Selects provider (`openrouter`) and model (`anthropic/claude-3.5-sonnet` or `google/gemini-pro`).
3. Enters system prompt and user message.
4. Clicks "Send Request".
5. Real-time tokens stream progressively via SSE directly into the UI.
6. Execution metrics (latency, TTFT, model identifier returned) appear in the execution summary drawer.
7. If the request fails (e.g. quota exceeded or upstream error), a structured error banner displays the exact normalized reason.

#### Journey 3: Operator — Quota & Concurrency Enforcement
1. Operator navigates to **Resources & Limits**.
2. Views daily token/request consumption per provider and global concurrency gauge.
3. Notices OpenRouter approaching 90% of its daily request limit.
4. Updates the daily policy (`PUT /api/resource-policies`) with immediate feedback.

---

## 4. Frontend Architecture & Technology Stack

### 4.1 Recommended Stack & Rationale

| Layer | Recommended Choice | Rationale |
|---|---|---|
| **Framework / Runtime** | **React 19 + TypeScript 5.7+** | High ecosystem maturity, concurrent streaming support, full compatibility with the modern React ecosystem and ADR-019 direction. |
| **Bundler & Dev Server** | **Vite 6** | Ultra-fast HMR, lightweight dev server with built-in proxy capabilities, standard ES modules output, seamless Turborepo integration. |
| **Workspace Location** | `apps/frontend` | Matches ADR-019 and `pnpm-workspace.yaml`. Kept completely decoupled from the root Go files. |
| **Server State & Caching**| **TanStack Query v5** (`@tanstack/react-query`) | Local cache and state store driven by **SSE real-time push** (`queryClient.setQueryData`). **Zero polling (`refetchInterval: false`)**. Manages optimistic UI mutations and snapshot queries. |
| **Routing** | **React Router v7** (in library mode) | Simple client-side SPA routing (`createBrowserRouter`), nested layouts (`AppShell`), minimal bundle overhead compared to full frameworks. |
| **Styling & Design System** | **Tailwind CSS v4 + Radix UI Primitives** | Zero runtime CSS overhead, accessible keyboard navigation, easy dark-mode customization, copy-paste component agility (shadcn pattern). |
| **Icons** | **Lucide React** | Clean, consistent, lightweight SVG icon library with wide coverage for developer/infrastructure tooling. |
| **Form Handling** | **React Hook Form + Zod** | Type-safe form validation for provider credentials, endpoint URLs, and policy numbers with minimal re-renders. |
| **SSE / Real-time Push** | **Native `EventSource` + `fetch` Streaming** | Central real-time transport for both system event notifications (health changes, provider status) and Playground inference token streaming. Zero external WebSocket dependencies. |
| **Unit & Component Testing**| **Vitest + Testing Library + MSW** | Shared Vite pipeline configuration, instant execution, realistic browser DOM emulation, and network-level contract mocking. |

### 4.2 Workspace Integration Plan
- In `apps/frontend/package.json`:
  ```json
  {
    "name": "@connexto/frontend",
    "private": true,
    "type": "module",
    "scripts": {
      "dev": "vite",
      "build": "tsc -b && vite build",
      "lint": "eslint .",
      "typecheck": "tsc --noEmit",
      "test": "vitest run",
      "check": "tsc --noEmit && vitest run"
    }
  }
  ```
- Turborepo orchestration: Root `pnpm build`, `pnpm check`, and `pnpm dev` will automatically delegate to `apps/frontend` without breaking the root Go commands.

### 4.3 State & Context Management Architecture

To ensure high performance, maintainability, and clean separation of concerns, the frontend distinguishes between two distinct context categories: **Application State** and **AI Conversation / Execution Context**.

#### 4.3.1 Application State Hierarchy: Server State vs. Client State

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        APPLICATION STATE HIERARCHY                     │
├───────────────────────────────────┬────────────────────────────────────┤
│           SERVER STATE            │            CLIENT STATE            │
│  (TanStack Query v5 + Realtime SSE│        (React Context API)         │
├───────────────────────────────────┼────────────────────────────────────┤
│ • System Readiness (/readyz)      │ • UI Preferences (sidebar state,   │
│ • Provider Diagnostics            │   table density, active theme)     │
│ • Model Catalog & Capabilities    │ • Local Operator Session / Token   │
│ • Resource Policies & Quotas (E09)│ • Ephemeral filters & search terms │
│ • Execution Records (E06)         │ • Global Toast / Alert Banner state│
│                                   │                                    │
│ Invariant: Zero Polling           │ Invariant: Lightweight & Zero Extra│
│ Driven by SSE push notifications  │ Dependencies (native React Context)│
└───────────────────────────────────┴────────────────────────────────────┘
```

1. **Server State (TanStack Query v5):**
   - **Zero Polling Mandate:** `refetchInterval` is strictly disabled (`false`). The client does not issue periodic background GET requests.
   - **Push-Driven Cache Invalidation:** Real-time updates delivered over the SSE stream invoke `queryClient.setQueryData()` or `queryClient.invalidateQueries()`, ensuring instant UI updates with zero network polling waste.
   - **Optimistic Updates:** Administrative operations (e.g. toggling provider enable/disable) immediately mutate local cache and reconcile upon server acknowledgment.

2. **Client State (Native React Context):**
   - Avoids heavyweight state stores (Redux, Zustand) in favor of scoped native React contexts:
     - `UiContext`: Controls sidebar collapse/expansion, table row density, and global error modals.
     - `SessionContext`: Holds operator connection status with the local BFF/proxy.

#### 4.3.2 AI Conversation & Prompt Context Management (Playground)

AI Hub's backend is **strictly content-free** by design (E06 records store operational execution metadata only; prompts and completions are never persisted in PostgreSQL or logs). Consequently, **conversation context assembly and retention reside entirely within the frontend client**.

1. **Data Model:**
   ```typescript
   export interface ChatMessage {
     id: string;
     role: 'system' | 'user' | 'assistant';
     content: string;
     timestamp: number;
     metrics?: {
       ttftMs?: number;       // Time to first token
       totalDurationMs?: number;
       tokenCount?: number;
     };
   }

   export interface PlaygroundContextState {
     selectedProvider: string; // e.g., "openrouter" or "opencode_go"
     selectedModel: string;    // e.g., "anthropic/claude-3.5-sonnet"
     temperature: number;
     maxTokens?: number;
     systemPrompt: string;
     messages: ChatMessage[];
     isStreaming: boolean;
     abortController: AbortController | null;
   }
   ```

2. **Context Lifecycle & Execution Pipeline:**
   - **Local Reducer (`usePlayground`):** Manages message history, dynamic prompt compilation, and chunk accumulation during streaming.
   - **OpenAI Responses Compilation:** When "Send" is triggered, the context compiles messages into the standard payload:
     ```json
     {
       "model": "anthropic/claude-3.5-sonnet",
       "messages": [
         { "role": "system", "content": "You are a code assistant..." },
         { "role": "user", "content": "Refactor this function..." }
       ],
       "stream": true
     }
     ```
   - **Streaming & Chunk Ingestion:** Tokens arriving via SSE are appended to the active `assistant` message progressively without full-state re-renders.
   - **Cancellation:** The active `AbortController` is triggered upon clicking "Stop Generation", immediately aborting the HTTP request and signaling the gateway to cancel upstream execution.
   - **Context Reset:** A dedicated "Clear Context" action resets the in-memory array.

3. **Storage & Persistence Boundary:**
   - **MVP (In-Memory Ephemeral):** Context exists in React state for the session duration. Reopening the tab or refreshing (F5) clears the conversation, reinforcing the privacy-first, content-free nature of the tool.
   - **Optional Enhancement:** Synchronization with `sessionStorage` (tab-scoped) to survive accidental page reloads, without ever writing prompt data to persistent backend storage.

---

## 5. Security Boundary & Authentication Model

### 5.1 The Security Challenge
The current Go backend relies on:
- `AIHUB_GATEWAY_API_KEY`: Protects `/v1/*` routes.
- `AIHUB_ADMIN_API_KEY`: Protects `/api/*` routes.
- Strict Host and Origin validation (rejection with 403 Forbidden).
- **No CORS headers and no OPTIONS preflight handling.**
- **No user authentication, sessions, or browser cookies.**

> [!CAUTION]
> Static admin or gateway API keys must **NEVER** enter client-side browser bundles, local storage, session storage, or client-accessible cookies. Any architecture that exposes these keys directly in the browser represents an unacceptable security vulnerability.

### 5.2 Operator Model Decision: Single Operator MVP
- **Decision:** For the MVP, Connexto AI Hub operates in **Single Operator Mode** within a trusted local or private network perimeter (loopback or private VPN/Tailscale).
- **Justification:** Adding multi-tenant user management, password hashing, JWT refresh flows, and RBAC to the Go backend before the product has a working UI would massively delay the roadmap and contradict the MVP principles.
- **Production Guardrail:** When AI Hub is deployed beyond loopback, it must either run behind an authenticating reverse proxy (e.g. Cloudflare Access, Authelia, or HTTP Basic Auth) or leverage an integrated local session secret.

---

## 6. Backend Communication & BFF Evaluation

We evaluated three potential communication patterns between the browser and the Go backend:

| Evaluation Dimension | Alternativa A: BFF / Static Serving no Go Backend | Alternativa B: BFF Separado (Node.js/Fastify) | Alternativa C: Proxy de Hospedagem / Dev Server Only |
|---|---|---|---|
| **Architecture** | Go binary serves both API and embedded SPA static files (`go:embed apps/frontend/dist`), with optional internal session/proxy handler. | Standalone Node.js/Go service running on another port, holding credentials and speaking to backend. | Standalone SPA served by Nginx/Vercel/Vite, relying on external reverse proxy config. |
| **Security & Secrets** | **Excellent:** Static keys remain in Go server memory. Browser communicates strictly Same-Origin (`/`). | **Good:** Secrets live in BFF; requires secure inter-service network communication. | **High Risk:** Easy for developers to accidentally leak keys via client env vars or misconfigured CORS. |
| **CORS / Preflight** | **Zero CORS issues:** Both frontend and APIs share the exact same scheme, host, and port. | Requires CORS configuration between browser and BFF, plus Host/Origin handling. | Depends on external proxy setup; fragile across environments. |
| **Operational Complexity**| **Lowest:** Exactly **one process** to run, configure, and monitor (`./cmd/ai-hub`). Single binary deployment. | **High:** Requires running and orchestrating two distinct runtimes (Go + Node.js). | **Medium:** Requires configuring external web servers (Nginx/Caddy) even for simple deployments. |
| **Development Experience** | Vite dev server proxies `/api` and `/v1` to `localhost:8080` with server-side injected headers. | Must start Go backend, Node BFF, and Vite frontend concurrently. | Vite proxy in dev, but diverged setup in production. |
| **MVP Fit** | **Optimal:** Aligns 100% with AI Hub's philosophy as a local-first single-binary tool. | Over-engineered for a single-operator MVP. | Insufficient security for production. |

### Technical Recommendation: Alternativa A (Embedded Same-Origin BFF)
1. **In Development:** Vite runs on `http://localhost:5173`. Its `vite.config.ts` proxy intercepts `/v1` and `/api` requests, forwarding them to `http://127.0.0.1:8080` and injecting the configured `AIHUB_GATEWAY_API_KEY` and `AIHUB_ADMIN_API_KEY` in the local development proxy layer. The browser never sees or stores these keys.
2. **In Production / Release:** The compiled frontend assets (`apps/frontend/dist`) are embedded directly into the Go binary using `go:embed`. The Go HTTP router serves these static files on `/` and `/assets/*` while routing `/v1/*` and `/api/*` internally. 
3. **Session Layer (Production):** The Go server sets a signed, `HttpOnly`, `SameSite=Strict` session cookie upon entering a single local access passphrase (or bypasses when bound strictly to `127.0.0.1`), granting browser access to the existing endpoints without ever exposing the raw bearer keys to browser scripts.
4. **Real-Time Notification Hub (SSE Push):** To satisfy the **Zero-Polling Mandate**, the BFF/backend exposes a long-lived SSE stream (`GET /api/events` or `/v1/events`). The Go backend pushes events directly to the browser (e.g. `health_status`, `provider_probe_result`, `provider_toggled`, `quota_alert`). The frontend subscribes to this stream via `EventSource`, eliminating periodic GET polling completely.

---

## 7. Deployment & Serving Topology

```
+-----------------------------------------------------------------------------------+
| DEVELOPMENT ENVIRONMENT                                                           |
|                                                                                   |
|  [Browser]                                                                        |
|      |                                                                            |
|      | http://localhost:5173 (Same-Origin to Vite)                                |
|      v                                                                            |
|  [Vite Dev Server :5173] ---------------------------------------+                 |
|      | (Serves React HMR / TS)                                  |                 |
|      |                                                          |                 |
|      +-- /api/*  --(Injects AIHUB_ADMIN_API_KEY)--------------+ |                 |
|      +-- /v1/*   --(Injects AIHUB_GATEWAY_API_KEY)------------+ |                 |
|                                                               v v                 |
|                                                    [Go AI Hub Backend :8080]      |
|                                                    - Loopback listener            |
|                                                    - Validates Origin & Bearer    |
|                                                    - PostgreSQL Pool              |
+-----------------------------------------------------------------------------------+

+-----------------------------------------------------------------------------------+
| PRODUCTION / LOCAL-FIRST RUNTIME                                                  |
|                                                                                   |
|  [Browser]                                                                        |
|      |                                                                            |
|      | http://127.0.0.1:8080 (Strict Same-Origin)                                 |
|      v                                                                            |
|  [AI Hub Go Single Binary :8080]                                                  |
|      +-- GET /              --> Embedded SPA index.html (go:embed)                |
|      +-- GET /assets/*      --> Embedded JS/CSS bundles                           |
|      +-- GET /healthz       --> Core liveness                                     |
|      +-- GET /readyz        --> Core readiness                                    |
|      +-- /api/*             --> Internal admin router (Authenticated via Cookie)  |
|      +-- /v1/*              --> Internal gateway router (Authenticated via Cookie)|
|      |                                                                            |
|      +-- PostgreSQL Pool / Providers (OpenCode Go, OpenRouter)                    |
+-----------------------------------------------------------------------------------+
```

---

## 8. UX/UI Direction & Design System

### 8.1 Visual Philosophy: "Professional AI Infrastructure"
The UI must project reliability, high information density, and developer ergonomics. It should feel like high-end developer tools (such as Cloudflare Dashboard, Linear, Vercel, or Datadog) rather than a generic CMS or consumer chat app.

### 8.2 Foundations & Tokens
- **Theme:** Dark mode first (high-contrast neutral grays `slate-900` / `zinc-950` with crisp borders `zinc-800`). Optional light mode in complementary MVP.
- **Typography:** 
  - Sans: `Inter` or system modern sans-serif for UI labels and metrics.
  - Mono: `JetBrains Mono` or `Fira Code` for model names, tokens, latency, JSON bodies, and error codes.
- **Semantic Status Palette:**
  - **Healthy / Operational / Active:** Emerald green (`emerald-500`, bg `emerald-950/40`, border `emerald-800`).
  - **Degraded / Stale / Syncing:** Amber / Yellow (`amber-500`, bg `amber-950/40`, border `amber-800`).
  - **Unhealthy / Down / Error:** Rose / Crimson (`rose-500`, bg `rose-950/40`, border `rose-800`).
  - **Inference / Streaming / Active:** Violet / Indigo (`indigo-400`, pulse animation).
  - **Disabled / Idle / Unconfigured:** Muted gray (`zinc-500`, bg `zinc-900`).

### 8.3 Layout Shell Structure
```text
+-----------------------------------------------------------------------------------+
| Header: [Logo: AI Hub] [Env: Local/Prod] [Global Health Badge] [Active Models: 42] |
+------------------+----------------------------------------------------------------+
| Sidebar (Left)   | Main Content Area                                              |
| - Overview       | [Page Title & Breadcrumb]                 [Action Buttons]     |
| - Providers      | +------------------------------------------------------------+ |
| - Models         | | Metric Cards Grid (Total Providers, Active Models, Quotas) | |
| - Playground     | +------------------------------------------------------------+ |
| - Executions     | | Data Table / Streaming View / Diagnostic Inspector         | |
| - Resources      | | - High density, tabular numbers, clear status badges        | |
| - Analytics      | +------------------------------------------------------------+ |
|                  |                                                                |
| [v0.1.0-dev]     | Status Bar: [Postgres: Connected] [Gateway: 127.0.0.1:8080]    |
+------------------+----------------------------------------------------------------+
```

---

## 9. Screen Mapping & Backend Contract Inventory

### 9.1 Overview / System Status
- **Goal:** Single-pane-of-glass operational status of AI Hub core, database, and all configured providers.
- **Persona:** Operator / Developer.
- **Displayed Data:** Liveness status, database readiness check, active provider count, degraded provider alert banner, catalog freshness timestamp.
- **Allowed Actions:** Refresh status, trigger manual provider diagnostics probe, navigate to failing provider details.
- **Endpoints:**
  - `GET /healthz` (200 OK `{status: "up"}`)
  - `GET /readyz` (200 OK / 503 Service Unavailable `{status: "ok|error", checks: {database: "ok|error"}}`)
  - `GET /diagnostics/providers` (200 OK `{providers: [{id, name, type, status, last_probe_at, error}]}`)
- **UI States:** Loading skeleton, healthy operational state, degraded state with issue breakdown, backend unreachable state.
- **MVP Priority:** **Essential (P0 — Included in FE-01)**.

### 9.2 Providers Management
- **Goal:** Inspect, configure, validate, and toggle AI providers (OpenCode Go, OpenRouter).
- **Persona:** Operator.
- **Displayed Data:** List of providers, base URLs, credentials configured (`true`/`false` — never values), operational status, last probe latency and timestamp.
- **Allowed Actions:**
  - Enable / Disable provider (`POST /api/providers/{id}/enable|disable`).
  - Run connectivity probe (`POST /api/providers/{id}/validate`).
  - Edit endpoint or update sealed API key (`PATCH /api/providers/{id}`).
- **Endpoints:**
  - `GET /api/providers` (List all configured providers)
  - `GET /api/providers/{id}` (Provider details)
  - `POST /api/providers/{id}/validate` (Connectivity probe against upstream models API)
  - `POST /api/providers/{id}/enable` & `disable`
- **UI States:** Empty list, loading, modal for credential entry, probing spinner, inline badge updating.
- **MVP Priority:** **Essential (P0 — Read-only in FE-01; Write/Validate in FE-02)**.

### 9.3 Model Catalog
- **Goal:** Explore normalized models available across all active providers, check capabilities, and trigger sync.
- **Persona:** Developer / Operator.
- **Displayed Data:** Model ID, provider, capabilities tags (`streaming`, `tools`, `vision`), context window size, freshness state (`fresh`, `stale`, `quarantined`), execution verification flag.
- **Allowed Actions:** Search by model ID, filter by provider or capability, trigger manual catalog sync (`POST /v1/catalog/sync`).
- **Endpoints:**
  - `GET /v1/catalog/models?provider=&capability=&presence=`
  - `GET /v1/catalog/models/{provider}/{model...}`
  - `GET /v1/catalog/providers`
  - `POST /v1/catalog/sync`
- **UI States:** Grid/Table view, search filter debounce, syncing indicator, model details drawer.
- **MVP Priority:** **Essential (P1 — FE-03)**.

### 9.4 Playground (Execution & Streaming)
- **Goal:** Test prompt completion and chat interactions with live streaming output across any enabled provider/model.
- **Persona:** Developer.
- **Displayed Data:** Provider selector, model selector, parameter controls (temperature, max_tokens), progressive Markdown/text streaming output, latency metrics (TTFT, total elapsed time, tokens/sec if available).
- **Allowed Actions:** Enter system/user messages, send streaming request, cancel ongoing generation via `AbortController`, copy output.
- **Endpoints:**
  - `POST /v1/responses` (OpenAI Responses SSE payload, header `X-AIHub-Provider` or body model routing).
- **UI States:** Ready/idle, streaming active (blinking cursor, auto-scroll), cancelled by user, upstream error banner with error category (`invalid_request`, `upstream_failure`, `resource_limit`).
- **Note:** Content-free backend: AI Hub does not store prompt text or conversation history in PostgreSQL (E06 records are operational metadata only). The conversation context is managed entirely in-memory on the client via `usePlayground` (see Section 4.3.2 for the context lifecycle, message model, and cancellation pipeline).
- **MVP Priority:** **Essential (P1 — FE-04)**.

### 9.5 Executions & Audit Log
- **Goal:** Audit recent execution events, verify routing decisions, and troubleshoot failed requests.
- **Persona:** Operator / Developer.
- **Displayed Data:** Execution ID, request ID, provider ID, model ID, timestamp, outcome (`succeeded`, `failed`, `cancelled`), duration ms, phase, upstream HTTP status, error cause. Content is strictly metadata (content-free as per E06).
- **Allowed Actions:** Cursor-based pagination, filter by outcome or provider/model, inspect execution JSON detail drawer.
- **Endpoints:**
  - `GET /v1/records/executions?limit=25&provider_id=&outcome=&cursor=`
  - `GET /v1/records/executions/{execution_id}`
- **UI States:** Virtualized/paginated table, filter controls, detail slide-over.
- **MVP Priority:** **Complementary (P2 — FE-05)**.

### 9.6 Resource Policies & Analytics
- **Goal:** Monitor token usage, daily quotas, concurrency limits, and admission trends.
- **Persona:** Operator.
- **Displayed Data:** Daily admissions time series, provider token distribution charts, concurrency utilization vs. policy limit, quota remaining.
- **Allowed Actions:** Update quota or concurrency policies (`PUT /api/resource-policies`, `PUT /api/concurrency-policies`), enable/disable policy.
- **Endpoints:**
  - `GET /api/resource-policies`, `PUT /api/resource-policies`
  - `GET /api/concurrency-policies`, `PUT /api/concurrency-policies`
  - `GET /api/resource-analytics/admissions`, `/provider-usage`, `/quota-utilization`
- **UI States:** KPI summary cards, recharts/SVG time-series charts, policy edit modal.
- **MVP Priority:** **Complementary (P2 — FE-06)**.

---

## 10. MVP Scope Breakdown

```text
+-------------------------------------------------------------------+
| ESSENTIAL MVP (Core Usability & Operational Control)              |
|                                                                   |
| [FE-01] BFF Boundary + System Health + Provider Status Overview   |
| [FE-02] Provider Administration (Enable/Disable, Validate, Keys)   |
| [FE-03] Normalized Model Catalog Explorer & Manual Sync           |
| [FE-04] Interactive Playground with Real-time SSE Streaming       |
+-------------------------------------------------------------------+
                                 |
                                 v
+-------------------------------------------------------------------+
| COMPLEMENTARY MVP (Observability & Governance)                    |
|                                                                   |
| [FE-05] Content-Free Execution Records & Audit Log (E06)          |
| [FE-06] Resource Policies & Usage Analytics Dashboard (E09)       |
| [FE-07] Local Session Persistence & UI Operator Preferences       |
+-------------------------------------------------------------------+
                                 |
                                 v
+-------------------------------------------------------------------+
| FUTURE EVOLUTION (Post-MVP)                                       |
|                                                                   |
| - E10 Automated Learning Engine Visualization                     |
| - E11 Full Multi-Tenant Enterprise Dashboard                      |
| - Multi-User RBAC & External Identity Provider (OIDC / OAuth2)    |
| - Persistent Server-Side Chat Conversation Storage & Prompt Books |
| - Non-deterministic LLM Routing Rules (E08.6+)                    |
+-------------------------------------------------------------------+
```

---

## 11. Pending Architectural & Product Decisions Matrix

| # | Decision Item | Identified Alternatives | Technical Recommendation | Architectural Impact | Requires Project Lead Approval? |
|---|---|---|---|---|---|
| **D1** | **Operator Model** | A) Single trusted local operator<br>B) Multi-tenant with user accounts/roles | **Alternative A (Single Operator)** for MVP | Zero backend schema changes; relies on network perimeter or local passphrase. | **Yes** (Formal confirmation of single operator boundary) |
| **D2** | **BFF & Serving Topology** | A) Go Embedded SPA (`go:embed`) + Same-Origin<br>B) Separate Node.js BFF<br>C) SPA on CDN + CORS in Go | **Alternative A (Go Embedded SPA)** | Single binary deployment, zero CORS, static API keys remain isolated in backend memory. | **Yes** (Architectural sign-off) |
| **D3** | **Frontend Core Stack** | A) React 19 + Vite + TypeScript<br>B) Next.js (SSR)<br>C) Svelte / Vue | **Alternative A (React 19 + Vite)** | Matches ADR-019, optimal SPA build for `go:embed`, lightweight dev proxy. | **Yes** (Confirm React/Vite baseline) |
| **D4** | **Component & Design System** | A) Tailwind CSS v4 + Radix UI (shadcn style)<br>B) Heavy component library (MUI / AntD)<br>C) Custom plain CSS | **Alternative A (Tailwind + Radix)** | High accessibility, zero CSS runtime, complete control over operational density. | **No** (Standard engineering decision) |
| **D5** | **Playground History Scope** | A) Ephemeral (in-memory React state)<br>B) Browser `localStorage`<br>C) Backend DB schema addition | **Alternative A (Ephemeral in-memory)** | Preserves backend content-free contract (E06); zero database debt. | **Yes** (Confirm chat history is not stored server-side) |
| **D6** | **Credential Entry Experience**| A) Write-only password inputs sent via sealed PATCH<br>B) File-based `.env` only | **Alternative A (Write-only web form)** | Uses existing C1 `PATCH /api/providers/{id}` contract safely; never echoes secrets back. | **No** (Complies with existing C1 contracts) |
| **D7** | **Real-Time Updates vs Polling** | A) Server-Sent Events (SSE) push only<br>B) Periodic HTTP polling (`refetchInterval`)<br>C) WebSockets | **Alternative A (Strict Zero-Polling via SSE)** | **Approved by Lead**: Zero polling in frontend. Backend/BFF emits real-time SSE event stream; frontend updates reactive state instantaneously. | **Approved by Lead** (Zero-polling mandate) |

---

## 12. First Vertical Increment Proposal (FE-01)

### Title: FE-01 — BFF Foundation, System Health & Provider Overview

#### Objective:
Deliver the first end-to-end operational vertical slice: scaffold the React 19 + TypeScript + Vite application in `apps/frontend`, configure the secure same-origin proxy boundary in development, connect to Go backend operational endpoints, and render real-time system health and provider diagnostics.

#### Scope of FE-01:
1. **Workspace Setup:**
   - Initialize `apps/frontend` using Vite + React 19 + TypeScript.
   - Configure `apps/frontend/vite.config.ts` proxy to route `/readyz`, `/healthz`, and `/diagnostics/providers` to `http://127.0.0.1:8080` while injecting `AIHUB_GATEWAY_API_KEY`.
   - Configure Tailwind CSS v4 and base design tokens.
   - Wire `apps/frontend` scripts into root `turbo.json`.
2. **API Client Layer & Real-Time Event Hub:**
   - Centralized typed HTTP fetch client handling JSON error envelopes `{type: string, message: string}`.
   - TanStack Query hooks configured with **strictly zero polling** (`refetchInterval: false`).
   - Real-time SSE subscriber (`useRealtimeEvents`) that connects to the backend/BFF event stream and pushes incoming status changes directly into the Query cache (`queryClient.setQueryData`).
3. **UI Components:**
   - `AppShell`: Header with AI Hub logo, environment tag, global health indicator, real-time connection badge ("Live SSE"), and navigation sidebar.
   - `OverviewPage`:
     - System Status Card: Liveness, PostgreSQL database readiness, active listener address.
     - Provider Status Grid: Cards for `opencode_go` and `openrouter` showing operational state (`healthy`, `degraded`, `unreachable`), last probe timestamp, and error details if failing.
     - Manual Refresh / Reconnect Button: Re-initializes connection if stream dropped.
4. **Resilience & State Handling:**
   - Loading skeletons during initial data fetch.
   - Connection failure banner if the Go backend is stopped or SSE stream disconnects (automatic reconnection with backoff).
   - Zero hardcoded mock data.

#### Out of Scope for FE-01:
- Provider credential editing or enable/disable mutations (reserved for FE-02).
- Model catalog browsing or sync (reserved for FE-03).
- Playground execution or streaming (reserved for FE-04).
- Production `go:embed` asset embedding (can follow after dev workflow is proven).

---

## 13. Acceptance Criteria & Quality Gates for FE-01

### Functional Acceptance Criteria:
- [ ] **AC-FE01-1 (App Boot):** Running `pnpm dev` starts the Vite dev server and displays the AI Hub AppShell on `http://localhost:5173`.
- [ ] **AC-FE01-2 (Zero Key Leakage):** Browser network inspection confirms that `AIHUB_GATEWAY_API_KEY` or `AIHUB_ADMIN_API_KEY` are **never** present in client source bundles, `localStorage`, or browser request headers (injected strictly by the proxy layer).
- [ ] **AC-FE01-3 (Real Health Display):** When the Go backend is running with PostgreSQL, the UI displays `System: Healthy` and `Database: Connected`.
- [ ] **AC-FE01-4 (Provider Diagnostics):** Providers configured in backend (`opencode_go`, `openrouter`) appear with their real runtime status matching `GET /diagnostics/providers`.
- [ ] **AC-FE01-5 (Error Handling):** When the backend is terminated, the UI cleanly transitions to an informative "Backend Disconnected / Retrying..." state without unhandled JavaScript exceptions or blank screens.
- [ ] **AC-FE01-6 (Empty/Degraded State):** If a provider is marked unhealthy, its card displays a warning badge with the reported error cause.
- [ ] **AC-FE01-7 (Zero Polling & Real-Time SSE Push):** Verify that the frontend performs **zero timer-based polling** (`refetchInterval: false`, no `setInterval`). Status updates are delivered exclusively via real-time SSE stream or explicit user actions.

### Technical & Engineering Gates:
- [ ] **Lint & Typecheck:** `pnpm --filter @connexto/frontend typecheck` passes with zero TypeScript errors (`noImplicitAny`, strict mode).
- [ ] **Unit Tests:** Vitest tests for the API client, error mapping, and health status badge pass.
- [ ] **Root Compatibility:** Running root `pnpm check` executes both the Go test suite (`go test ./... -count=1 -race`) and frontend checks without interference.

---

## 14. Risks, Assumptions & Dependencies

| Risk / Dependency | Impact | Mitigation Strategy |
|---|---|---|
| **R1: Accidental CORS / Header Leakage** | High | In dev, use Vite proxy exclusively; never configure client-side `fetch('http://localhost:8080')`. In production, serve strictly same-origin from the Go binary. |
| **R2: Long SSE Streaming Timeouts** | Medium | The backend enforces a 2-minute stream inactivity timeout (`StreamInactivityTimeout`). Frontend SSE consumer must handle connection dropouts and abort cleanly. |
| **R3: Node / pnpm Version Mismatch** | Low | Root `package.json` pins `pnpm@12.16.1` and `.nvmrc` pins Node 24. Documented in `docs/development-workspace.md`. |
| **R4: Content-Free Confusion** | Low | Developers might expect previous prompts to appear in the Playground. Clarify in the UI that AI Hub is an infrastructure proxy that does not store chat content. |
| **R5: PostgreSQL Optionality** | Medium | If AI Hub runs in File mode, `/api/*` provider routes return 404/503. Frontend must gracefully adapt by detecting persistence mode via readiness checks. |

---

## 15. Action Plan & Next Steps

1. **Project Lead Review:** Present this discovery document and the Pending Decisions Matrix (Section 11) to the project lead for approval.
2. **Phase 1 SPEC Creation:** Upon lead authorization of Decisions D1, D2, and D3, generate the formal implementation specification `docs/specs/FE-01-bff-and-health-overview.md`.
3. **Autonomous Implementation:** Execute FE-01 following the `/sdd-workflow` (TDD, implementation, component unit tests, and E2E verification).

