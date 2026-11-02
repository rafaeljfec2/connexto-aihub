# E01–E07 Delivery History, Validation, and Lessons

Status as of 2026-11-02.

This document is a cross-epic navigation and lessons record. It does not replace
the approved SPECs, ADRs, or dated validation reports. Historical verdicts remain
attached to the execution and scope that produced them. Claims below were
discovered through repository documentation, Git history, and historical
session notes, then cross-checked against current repository artifacts. Session
memory is a discovery aid, not the authority for current behavior.

## Executive status

- E01–E06 runtime capabilities are implemented in the current repository.
- E07 is `PASS/COMPLETE` for AC-1 through AC-13. AC-14 is formally
  `OUT OF SCOPE`; Codex CLI support is limited to the tested release and
  environment.
- E02's original report remains `PARTIALLY VALIDATED`. Later E04/E07 runs add
  evidence for the shared provider route but do not rewrite E02's original
  limitations.
- E08 routing, E09 policy/limits, E10 learning, and E11 dashboard work are not
  represented as implemented by this history.

## Epic delivery and evidence matrix

| Epic | Delivery | Execution / validation evidence | Primary references |
| --- | --- | --- | --- |
| E01 — Core Foundation | Implemented: Go bootstrap, configuration, modular lifecycle, HTTP server, health, logging, correlation, and graceful shutdown. There is no standalone E01 SPEC; the kickoff and implementation are the recorded baseline. | Automated foundation tests are part of the current Go suite. No separate E01 real-provider E2E report was located. | `d55ddd0`; [kickoff](../../AI-Hub-Product-Architecture-Kickoff.md); [architecture](../architecture.md); [ADR index](../adr/README.md) |
| E02 — Provider Management | Implemented: concrete OpenCode Go adapter, Responses byte-preserving passthrough, provider diagnostics, model endpoint passthrough, and streaming. | The original Codex/OpenCode Go report is explicitly `PARTIALLY VALIDATED`; its real-provider scenarios passed with limitations around Codex-hop rendering, live provider-session observation, and sandbox availability. Later E04/E07 evidence does not retroactively change that report's verdict. | `ad9ce7c`; [SPEC](../specs/E02-provider-management.md); [protocol spike](../research/opencode-go-protocol-spike.md); [E02 validation](E02-codex-opencode-go-e2e.md) |
| E03 — Model Catalog | Implemented through P1–P9: provider-scoped model identity, conservative capability claims, derived listing state, synchronization, versioned JSON snapshot, quarantine/recovery, and query API. SPEC revision 5 includes approved persistence amendment A1. | Automated mapping, sync, storage, and handler tests cover the contract. A real 28-model catalog was observed during E04 validation; no standalone E03 E2E report exists in `docs/validation/`. | `c9444ce`; [SPEC](../specs/E03-model-catalog.md); [design](../design/E03-model-catalog-design.md); ADR-009/ADR-010; [E04 validation](E04-gateway-hardening-access-control-e2e.md) |
| E04 — Gateway Hardening and Access Control | Implemented: distinct static gateway credential, fail-closed listener exposure, Host/Origin checks, authentication, bounded admission, and request/session identity boundaries. | Real Codex → AI Hub → OpenCode Go E2E report verdict: `VALIDATED` with the limitations recorded there; the listener was exercised locally, not exposed publicly. Review corrections and revalidation are preserved in the report. | `3b9015d`; [SPEC](../specs/E04-gateway-hardening-access-control.md); ADR-011–ADR-013; [E04 validation](E04-gateway-hardening-access-control-e2e.md) |
| E05 — Execution Lifecycle | Implemented: one logical execution per admitted Responses request, terminal arbitration, cancellation/timeouts, body-reader ownership, and coordinated shutdown. | SPEC records RED/GREEN corrections, race/quality gates, and a real Codex/OpenCode Go marker run. No separate `docs/validation/E05-*` report was located. | `b706aa8`; [SPEC and implementation evidence](../specs/E05-execution.md); ADR-014–ADR-016 |
| E06 — Execution Records | Implemented: content-free schema-v1 records, bounded asynchronous observation, append-only JSONL, retention/recovery, authenticated query routes, and a bounded shutdown drain. Schema v1 deliberately stores `usage: null`. | The final E07 real pass verified record listing, correlation, schema/outcome, seven by-ID lookups, and recovery of all seven records after restart. | `0d33ca2`; [Discovery](../discovery/E06-discovery.md); [SPEC](../specs/E06-execution-records.md); [design](../design/E06-execution-records-design.md); ADR-017/ADR-018 |
| E07 — First Harness Integration | Implemented as a validation-only Codex harness; no production runtime behavior or durable AI Hub conversation state was added. | Final full E2E: `PASS/COMPLETE` for AC-1–AC-13, with nine provider calls (the hard maximum), zero retries, and the tested Codex release on Linux amd64. AC-14 remains `OUT OF SCOPE`. | `3cabf79`; [SPEC](../specs/E07-codex-first-harness-integration.md); [design](../design/E07-codex-first-harness-integration-design.md); [final plan](E07-final-e2e-execution-plan.md); [final report](E07-codex-first-harness-integration.md) |

The E03 commit subject under-describes the delivery: E04 Discovery records that
`c9444ce` is the E03 closure point based on its full implementation and test
diff. Use the SPEC, design, and commit contents—not the commit subject alone—to
identify the delivered scope.

## Recorded corrections and error learnings

| Epic | Observed error or ambiguity | Correction or lesson | Evidence |
| --- | --- | --- | --- |
| E01 | No standalone E01 SPEC exists. ADR-002 describes optional JSON configuration while the implemented application uses defaults plus environment/dotenv. | Treat the kickoff and implementation as E01's historical baseline. E04 recorded the ADR/configuration divergence; do not silently rewrite the old decision or invent an E01 acceptance report. | E04 SPEC §3.3 DIV-01/DIV-02; README configuration; `docs/architecture.md` |
| E02 | An initial `401` came from a probe script preserving quotes around a dotenv value, not from the production dotenv loader. An object-shaped `function_call_output.output` produced an upstream `400`. | Use the production dotenv parser for probes; follow the Responses wire shape (`output` is a string). Keep probe defects distinct from gateway defects. | [E02 validation](E02-codex-opencode-go-e2e.md) §§2–3 |
| E02 / E04 | `codex exec resume --last` selected a different session in the tested CLI; cross-request provider session headers alone do not create durable AI Hub conversation state. | Capture the `thread.started` ID and resume that explicit thread. For translated providers, replay prior conversation input; do not send `previous_response_id`. | [E02 validation](E02-codex-opencode-go-e2e.md) §4; [E04 validation](E04-gateway-hardening-access-control-e2e.md) §5; [E07 report](E07-codex-first-harness-integration.md) AC-5 evidence |
| E03 | Provider capability metadata does not guarantee completeness. A missing modality cannot safely mean unsupported; an empty sync or failed persistence could otherwise erase accepted data. | Missing/undeclared capabilities remain `unknown`; reject empty catalogs; persist the complete candidate before publishing accepted state; distinguish atomic visibility from crash durability. Revisions 3–5 formalized these rules. | [E03 SPEC](../specs/E03-model-catalog.md) §§4–8, §17; ADR-009/ADR-010 |
| E04 | The gateway trust boundary was initially absent; a gateway key could be confused with provider credentials, and permissive listener/Host/Origin handling could expose the service. | Keep gateway/provider credentials distinct; authenticate before body/provider work; validate actual listener exposure and exact Host/Origin; reject oversize input before forwarding. E04 review also removed a public placeholder key and corrected error/session documentation. | [E04 SPEC](../specs/E04-gateway-hardening-access-control.md) DIV-07–DIV-10; [E04 validation](E04-gateway-hardening-access-control-e2e.md) §§2–3, §9 |
| E05 | RED evidence found a response-header timeout mapped to `502` instead of the approved `504`; tests also exposed ignored flush errors and a race between terminal reservation and response commitment. | Preserve `502` for ordinary transport failures, return `504` only for the recognized opening deadline, propagate sink flush failures, and synchronize metadata commitment with terminal arbitration. | [E05 SPEC](../specs/E05-execution.md) §14; ADR-014–ADR-016 |
| E06 | The E05 terminal result did not carry all record identity/timing fields, and usage existed only inside the opaque response stream. | Add a narrow, content-free terminal observer and schema-v1 records. Keep `usage: null`; bounded non-blocking enqueue, at-most-once loss semantics, and the two-second flush prevent record storage from changing execution behavior. | [E06 Discovery](../discovery/E06-discovery.md); [E06 SPEC](../specs/E06-execution-records.md) §§3–4, §§6–11; ADR-017/ADR-018 |
| E07 | In the failed real replay, `exec resume` omitted `--skip-git-repo-check` while using a temporary workspace outside a Git repository; the second invocation did not reach Hub ingress. The supervisor also contained host-level Codex snapshots inconsistent with AC-14's later `OUT OF SCOPE` disposition. | The local real-CLI regression demonstrated RED (resume exits before another synthetic request) and GREEN after adding the flag. The validation-only supervisor stopped fingerprinting host-global Codex/environment state while retaining private namespace isolation. The final real pass proved both replay turns and stayed within the nine-call ceiling. | [E07 report](E07-codex-first-harness-integration.md), sections “Local AC-5 correction” and “Final complete E2E homologation”; [plan](E07-final-e2e-execution-plan.md) §§2–4, §11 |

## Cross-epic lessons

1. **Separate implementation, automated tests, and real E2E evidence.** A green
   fake-upstream suite does not establish a provider integration; a real E2E
   limitation must remain attached to the scenario that observed it.
2. **Classify failures before changing product code.** The E02 quoted-value
   `401` was a probe error; the E07 missing-resume-flag failure was a harness
   error. Neither justified an AI Hub runtime change.
3. **Observe each boundary.** Process start, emitted/received request, provider
   forwarding, response receipt, and functional result are separate evidence.
   E07's final replay passed only after both Hub requests, both provider
   forwards, context, and marker recall were observed.
4. **Preserve contracts across epics.** E02 passthrough stays opaque; E04 owns
   authentication/admission; E05 owns execution lifecycle; E06 observes
   terminal metadata without affecting E05; E07 validates the harness without
   adding durable AI Hub sessions.
5. **Be explicit about uncertain state.** E03 uses `unknown` rather than
   inferring unsupported capabilities; E06 uses `usage: null` and best-effort
   records rather than fabricating completeness or billing facts.
6. **Treat historical reports as dated evidence.** Later validation may add
   evidence but does not rewrite an earlier verdict. AC-14 remains `OUT OF
   SCOPE`, and Codex stdout progressive rendering remains `NOT VERIFIED` where
   the CLI does not expose it.
7. **Keep operational limits measurable.** Isolate `HOME`/`CODEX_HOME`, keep
   secrets out of output and files, use zero retries, and count requests at the
   provider boundary. The final E07 pass used the full nine-call budget without
   exceeding it.

## Evidence index

- [E02 protocol spike](../research/opencode-go-protocol-spike.md)
- [E02 real-provider validation](E02-codex-opencode-go-e2e.md)
- [E03 SPEC and design](../specs/E03-model-catalog.md), [design](../design/E03-model-catalog-design.md)
- [E04 SPEC, ADRs, and real E2E](../specs/E04-gateway-hardening-access-control.md), [validation](E04-gateway-hardening-access-control-e2e.md)
- [E05 SPEC and implementation evidence](../specs/E05-execution.md), [ADR-014](../adr/ADR-014-execution-lifecycle-and-command-contract.md), [ADR-015](../adr/ADR-015-coordinated-execution-shutdown.md), [ADR-016](../adr/ADR-016-execution-timeout-and-error-mapping.md)
- [E06 Discovery, SPEC, design, ADR-017, and ADR-018](../discovery/E06-discovery.md), [SPEC](../specs/E06-execution-records.md), [design](../design/E06-execution-records-design.md)
- [E07 SPEC, design, final E2E plan, and validation report](../specs/E07-codex-first-harness-integration.md), [design](../design/E07-codex-first-harness-integration-design.md), [plan](E07-final-e2e-execution-plan.md), [report](E07-codex-first-harness-integration.md)
