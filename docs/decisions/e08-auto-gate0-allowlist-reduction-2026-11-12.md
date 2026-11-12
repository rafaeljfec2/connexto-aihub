# E08 Auto — Gate 0 allowlist reduction (7 models)

**Date:** 2026-11-12  
**Status:** Accepted  
**Related:** [AUTO-SPEC v1.1.0](../specs/AUTO-SPEC.md), [Gate 0 battery report](../../reports/llm-homologation/gate0-battery-2026-11-12.md)

## Context

The original Auto MVP allowlist contained ten models, including three Antigravity slugs that failed homologation on 2026-11-10 with upstream quota notices. A live retest on 2026-11-12 still returns HTTP 429/502 for those three ids.

Seven models passed a full Gate 0 battery (text, stream, strict `json_schema` non-stream).

## Decision

1. **`AutoMVPAllowlist` is reduced to seven models**, removing:
   - `antigravity:claude-opus-4-6-thinking`
   - `antigravity:claude-sonnet-4-6`
   - `antigravity:gpt-oss-120b-medium`
2. **`AutoGate0Satisfied()` applies only to the remaining allowlist.** All seven have `Gate0Approved: true` backed by the 2026-11-12 report.
3. Removed models stay in the general catalog and in `autoModelEvidenceTable` for audit, with `removed_from_auto_allowlist_2026-11-12` notes.
4. They may re-enter the allowlist only after a new homologation cycle and an explicit amendment.

## Consequences

- AUTO-SPEC Gate 0 language references “every allowlist model” rather than a fixed count of ten.
- ADR-023’s “ten catalog models” phrasing is superseded for Auto routing scope by this decision and AUTO-SPEC v1.1.0.
