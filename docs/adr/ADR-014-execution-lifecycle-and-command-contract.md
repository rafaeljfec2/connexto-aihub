# ADR-014 — Execution lifecycle and command contract

- **Status:** accepted
- **Date:** 2026-10-31
- **Related:** SPEC E05, ADR-008, ADR-013

## Context

The Responses handler currently opens the concrete OpenCode Go client and owns
stream copying. That leaves no logical execution identity, atomic drain gate,
terminal result, or joined reader lifecycle. E05 requires these responsibilities
without adding routing policy or a speculative provider framework.

## Decision

1. Add `internal/execution` as the in-process owner of one execution, its
   registration, derived context, upstream response body, sole body reader,
   terminal arbitration, and immutable result. Keep HTTP response writing in
   `internal/gateway` through a narrow sink with metadata-commit and chunk-write
   operations.
2. Define `Target{ProviderID, ModelID}` and
   `Command{RequestID, Target, ProviderSessionID, Payload, ContentType}`.
   `ModelID` may be absent only on the E02 passthrough path; if supplied it must
   be non-empty. The legacy gateway sets `ProviderID` to `opencode-go`, leaves
   `ModelID` absent, and does not parse the payload to discover or validate a
   model.
3. `Manager.Execute(ctx, command, sink)` returns a terminal `Result` for an
   admitted execution. It returns a typed admission error without creating an
   execution when the command is invalid or the drain gate is closed. The
   result carries execution ID, terminal outcome, cancellation cause or timeout
   phase, upstream status when available, and elapsed duration; it carries no
   payload or provider error text. The sink exposes `Commit(status,
   contentType)` and `WriteChunk(bytes) error`; it never exposes
   `http.ResponseWriter` to the manager.
4. Compose the manager with the one configured OpenCode Go opener. It checks
   target structure and that the provider ID resolves to that configured
   adapter. It does not expose a provider registry or general provider
   interface. A narrow opener function is the test seam. The provider adapter
   receives a provider target and payload together, validates its provider
   identity constraints, and forwards the payload bytes unchanged. Target
   metadata never rewrites payload fields.
5. E04 remains responsible for bounded body admission and provider session
   metadata. The gateway reads the already admitted body into the command and
   copies no response body. E05 does not parse JSON or serialize the request.
6. Protect the accepting gate and active-execution map with one manager mutex.
   Registration and drain-gate closure use that mutex. Waiters use a manager
   change notification to wait for an empty active map without racing a later
   registration. Each execution has a separate mutex for terminal reservation.
   Closing
   the gate and waiting for active work therefore cannot race a later
   registration. Reject a closed gate before creating an execution.
7. After a successful open, one execution-owned reader sends chunks through a
   capacity-one channel to the coordinator. It has a `readerDone` signal and
   checks cancellation before initiating another read. The coordinator is the
   only body closer, stops forwarding after a terminal event, cancels the
   derived context, closes the body exactly once, joins the reader, then
   publishes the immutable result and removes the registration.
8. One synchronized arbiter reserves the first observed terminal event. The
   reserved event controls cancellation and cleanup; the terminal result is
   visible only after cleanup and reader join finish. Losing signals cannot
   mutate the result.
9. Emit structured lifecycle records with execution ID, request ID, known
   target identity, outcome, phase, upstream status when available, and
   duration. Never include payload bytes, generated content, credentials, or
   raw provider response bodies.

## Rationale

- A concrete opener seam fits the repository's single-provider architecture
  and preserves E02's byte-for-byte passthrough.
- A manager-owned reader gives cancellation, timeout, and body closure one
  coordination point while keeping `http.ResponseWriter` at the HTTP boundary.
- Joining before unregistering makes active-count zero meaningful evidence
  that execution-owned read work has ended.

## Consequences

- `internal/gateway` delegates lifecycle and no longer owns a reader goroutine.
- `internal/providers` gains a target-aware Responses entry point while its
  existing byte-preserving operation remains available to its current callers.
- Future model mapping must be an explicit, separately approved request
  preparation contract before an E08 caller changes payload bytes.
- Execution state remains in memory and disappears at process exit.
