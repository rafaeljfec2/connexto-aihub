# E07 — Codex First Harness Integration — Design

Status: accepted under the approved E07 SPEC (2026-10-31).
Related: `docs/specs/E07-codex-first-harness-integration.md`.

## 1. Design boundary

E07 uses the existing Codex Responses configuration path and existing AI Hub
E04–E06 runtime contracts. It adds no production behavior, endpoint, runtime
dependency, or provider abstraction. The only executable addition is an
opt-in validation test helper, kept in test files and outside production
composition.

The helper composes the existing provider client, gateway policy and handlers,
execution manager, records service, HTTP server, and coordinated lifecycle.
It binds to loopback and uses a dedicated temporary data directory. Its
provider client receives a validation `http.Client` through the existing
`providers.NewClient` injection point.

## 2. Validation instrumentation

The injected `RoundTripper` has two bounded responsibilities:

1. Count provider-boundary transport attempts with an aggregate atomic counter.
   The counter does not inspect or emit URLs, headers, credentials, response
   bodies, or generated identifiers.
2. During the second synthetic Codex turn only, inspect the in-memory request
   body for the synthetic marker and the absence of `previous_response_id`.
   It restores the exact bytes before forwarding and reports only the two
   structural booleans. It retains no payload.

An outer test-only response observer captures `X-Request-Id` in memory so
records can be correlated with scenario requests. The observer publishes no
identifier. Codex stdout and stderr are also held in memory; assertions inspect
only the synthetic result and JSON event types, and diagnostics never include
captured content.

## 3. Process and secret isolation

The test reads the provider credential from `AIHUB_PROVIDER_OPENCODEGO_API_KEY`,
the existing `OPENCODE_API_KEY` environment variable, or the existing `.env`
without modifying that file. It uses a supplied gateway key or generates a
temporary one in memory. Presence checks report booleans only. The Codex child
receives an explicit environment allowlist containing the gateway key under
the variable named by `env_key`; both `HOME` and `CODEX_HOME` point to the
temporary Codex directory, and the child excludes the provider key and the
Hub-only gateway and data-directory variables. The version probe uses the same
isolated environment.

Before invoking Codex, the helper creates a unique temporary `CODEX_HOME`,
creates its `config.toml`, and uses a separate temporary tool workspace. The
Codex configuration contains the selected model, loopback Hub `/v1` base URL,
`wire_api = "responses"`, and the gateway environment-variable name only. The
tool scenario uses `--sandbox workspace-write` with normal approval behavior.
The helper skips only the Git-repository precondition for its temporary
workspace; it never selects a sandbox or approval bypass option.

## 4. Scenario sequence

1. Record exact Go, Codex CLI, operating-system, model, and AI Hub revision
   information; verify credential presence without reading values into output.
2. Start the loopback Hub composition with the validation transport and
   temporary data directory.
3. Run basic Codex generation, two-turn replay, and a bounded workspace tool
   task. Capture only scenario outcomes and the structural replay result.
4. Compare the provider counter around an unauthenticated request and require
   the existing `401` response with no counter increase.
5. Compare a direct provider unknown-model response with the same authenticated
   gateway probe, retaining only status and generic-body classification.
6. Measure SSE arrivals at the Hub client connection. Record Codex CLI
   progressive rendering independently from gateway-hop streaming.
7. Query records for the in-memory correlated request IDs, verify schema-v1
   null semantics, stop the Hub gracefully, restart it on the same data
   directory, and query the retained records again.
8. Compare before/after integrity fingerprints in memory, then remove all
   temporary Codex state, data, workspace files, and instrumentation state.

The report contains only aggregate counts, status codes, scenario outcomes,
versions, environment class, and limitations. It excludes prompt/response
content, credentials, session IDs, request IDs, execution IDs, paths, and
integrity hashes.

## 5. Delivery sequence

1. Add deterministic tests for counter-only observation, synthetic replay
   inspection, environment filtering, and redacted result formatting; observe
   their initial failure before implementing the test helper.
2. Implement and test the opt-in helper without changing production files.
3. Run one bounded real-provider scenario pass using Codex CLI
   `0.164.0-alpha.11.1` and `deepseek-v4.1-flash`, subject to confirming those
   values at execution time.
4. Publish the scenario report. Add the README support claim only if every
   required scenario passes; otherwise report the limitation and leave the
   support claim absent.
5. Run the full Go quality gates, formatting and diff checks, perform the
   security scan, then complete the independent Phase 7 review.

No ADR is required because the integration adds no architectural decision or
change to an existing contract.

## 6. Supplemental validation authorization

On 2026-10-31, the project lead authorized exactly one additional real E2E
pass beyond the original one-pass budget. The authorization is limited to
validation and evidence documentation; it changes no functional requirement
or implementation scope. The pass requires `bwrap` in `PATH`, isolated
`HOME` and `CODEX_HOME`, and the existing sandbox and approval controls.

An earlier preflight on 2026-10-31 found `bwrap` unavailable; no scenario was
started during that preflight. After the environment was prepared, a later
preflight passed and the one authorized additional pass was executed once.
The pass is consumed. Its scenario and integrity evidence is recorded in
`docs/validation/E07-codex-first-harness-integration.md`.
