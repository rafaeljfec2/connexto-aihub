# ADR-002 — JSON configuration with environment overrides

- **Status:** accepted
- **Date:** 2026-10-30

## Decision

Configuration is defined by explicit Go structs, decoded from an optional JSON
file with `DisallowUnknownFields`, then overridden by `AIHUB_*` environment
variables and validated before use.

## Rationale

- `encoding/json` is stdlib; YAML or TOML would add a dependency for ergonomics
  only.
- Strict decoding turns typos into startup failures instead of silent defaults.
- A fixed environment variable list keeps precedence predictable and testable.

## Consequences

- No comments or trailing commas in the configuration file.
- Durations are strings (`"15s"`) rather than opaque nanosecond integers.
- Adding a setting requires updating the struct, the env mapping, the example
  file and the README table.
