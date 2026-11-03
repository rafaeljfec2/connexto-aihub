# Frontend MVP architecture — formal approval (D1, D2, D3)

**Date:** 2026-11-03

**Status:** accepted (implementation authorized for Frontend MVP FE-01+)

**Resolves roadmap gap:** G1 (formal sign-off for D1–D3)

## Approved decisions

### D1 — Operator model

- **Single trusted operator** for the Frontend MVP.
- No multi-tenant product auth, user accounts, or RBAC in this increment.
- Security relies on a trusted local or private-network perimeter (loopback or private VPN/Tailscale) plus server-side credential boundaries.

### D2 — BFF and serving topology

- **Development:** Vite dev server same-origin proxy; gateway and admin bearer keys injected **server-side** only (never in browser bundles).
- **Production:** Compiled SPA embedded in the Go binary (`go:embed apps/frontend/dist`); same-origin API and static assets from one process.
- No CORS shortcuts; no exposure of `AIHUB_GATEWAY_API_KEY`, `AIHUB_ADMIN_API_KEY`, or provider credentials to browser code.

### D3 — Frontend core stack

- React 19, TypeScript, Vite 6 in `apps/frontend`.
- TanStack Query v5 (zero operational polling; SSE-driven cache updates).
- Tailwind CSS v4, Radix UI primitives, Lucide icons, React Router v7.

## Related artifacts

- Discovery baseline: [FRONTEND-DISCOVERY.md](../frontend/FRONTEND-DISCOVERY.md) (§11 decision matrix)
- MVP roadmap: [MVP-ROADMAP.md](../frontend/MVP-ROADMAP.md)
- D7 (zero-polling operational SSE): approved by project lead in discovery; unchanged
- Implementation SPEC: [FE-01-bff-health-overview-sse.md](../specs/FE-01-bff-health-overview-sse.md)

## Authorized scope

Implement Frontend MVP increments per `MVP-ROADMAP.md`, starting with **FE-01** (development BFF, operational overview, operational Event Hub SSE).

## Explicitly not authorized by this decision alone

- Multi-user authentication product features
- E10/E11 scope
- Backend contract changes beyond FE-01 operational Event Hub (documented in FE-01 SPEC)
