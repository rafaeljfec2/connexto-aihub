# ADR-004 — Explicit module lifecycle

- **Status:** accepted
- **Date:** 2026-10-30

## Decision

Components implement `Name`, `Start(ctx)` and `Stop(ctx)`. `App` starts modules
in declaration order, stops them in reverse order, rolls back started modules
when a later start fails and joins shutdown errors.

## Rationale

- Startup and shutdown order become explicit and testable.
- Rollback and reverse shutdown prevent partially initialised processes.
- No dependency injection container or reflection is needed.

## Consequences

- Every new infrastructure component must implement the interface.
- Modules must tolerate the same context being reused for rollback.
- Adding a component requires updating `main`, not the lifecycle engine.
