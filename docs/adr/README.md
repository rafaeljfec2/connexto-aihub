# Architecture Decision Records

Decisions taken across E01–E06, Phase C2, accepted E08, and proposed E09 work.
Each record states context, decision, rationale and consequences. E07 was a
validation-only harness integration and added no runtime architecture decision
or ADR.

| ADR     | Decision                                                        |
| ------- | --------------------------------------------------------------- |
| ADR-001 | Fiber HTTP server with net/http-compatible handlers             |
| ADR-002 | JSON configuration with environment overrides                   |
| ADR-003 | Structured logging with slog and correlation IDs                |
| ADR-004 | Explicit module lifecycle                                       |
| ADR-005 | Loopback binding without gateway authentication in E01          |
| ADR-006 | Health endpoint semantics                                       |
| ADR-007 | Provisional module path                                         |
| ADR-008 | Streaming inactivity deadline                                   |
| ADR-009 | Provider-scoped model identity, capability claims and listing semantics |
| ADR-010 | Catalog persistence: versioned JSON snapshot with quarantine    |
| ADR-011 | Gateway API key authentication and network exposure policy      |
| ADR-012 | Gateway request admission limits and local error contract       |
| ADR-013 | Request and session identity                                    |
| ADR-014 | Execution lifecycle and command contract                        |
| ADR-015 | Coordinated execution shutdown                                  |
| ADR-016 | Execution timeout and HTTP error mapping                        |
| ADR-017 | Execution records storage: append-only JSONL segments            |
| ADR-018 | E05 terminal observation and records delivery                    |
| ADR-019 | Hybrid monorepo: independent Go Modules and pnpm ecosystems with Turborepo orchestration |
| ADR-020 | PostgreSQL persistence foundation: pgx, embedded migrations and catalog persistence modes |
| ADR-021 | Provider selection and OpenRouter runtime                        |
| ADR-022 | Deterministic intelligent routing (accepted E08.1 / E08.1.1, 2026-11-03)   |
| ADR-023 | Resource admission and quota accounting (accepted E09.1, 2026-11-03) |

New decisions get the next number. Superseded records stay in place and are
marked as such.
