# ADR-007 — Provisional module path

- **Status:** provisional
- **Date:** 2026-10-30

## Decision

The Go module is named `aihub` because the repository has no remote yet.

## Rationale

- Inventing a hosting URL (`github.com/<org>/...`) would encode an unverified
  ownership decision into every import.

## Consequences

- Publishing the module or adding a remote requires renaming the module and all
  internal imports; the codebase is small enough for a mechanical change.
- No external consumer can import the module until the path is final.
