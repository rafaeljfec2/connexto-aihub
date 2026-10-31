# Architecture Decision Records

Decisions taken during E01–E05. Each record states context, decision, rationale
and consequences.

| ADR     | Decision                                                        |
| ------- | --------------------------------------------------------------- |
| ADR-001 | Standard library HTTP server                                    |
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

New decisions get the next number. Superseded records stay in place and are
marked as such.
