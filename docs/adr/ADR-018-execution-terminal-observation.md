# ADR-018 — E05 terminal observation and records delivery

- **Status:** accepted (E06 design, 2026-10-31)
- **Date:** 2026-10-31
- **Related:** SPEC E06, ADR-014, ADR-015

## Context

E05 publishes one immutable terminal `Result` per execution. Its only consumer
is the Gateway, which discards it after mapping opening failures. E06 needs a
delivery point that receives finished executions without importing storage into
`internal/execution` and without letting disk latency or storage failures change
E05 results, HTTP responses or the provider-release guard.

## Decision

1. `internal/execution` gains an additive immutable `FinishedExecution` value
   containing the terminal `Result`, `RequestID`, `Target{ProviderID, ModelID}`
   and the UTC `StartedAt` copied from the registered execution. It omits the
   provider session, payload, content type, response body, credentials and HTTP
   headers.
2. `internal/execution` defines `ResultObserver` with
   `OnExecutionFinished(FinishedExecution)`. Records implements it; `cmd`
   wires the implementation through `execution.WithObserver`. The execution
   package never imports `internal/records`.
3. The manager invokes the observer exactly once per registered execution, after
   the upstream body is closed, the reader is joined and the terminal result is
   fixed, but before the execution is removed from the active registry and
   before `Manager.Execute` returns. A successful `Manager.Wait` therefore
   proves that every completed callback returned.
4. The callback performs a bounded non-blocking enqueue only. It performs no
   file I/O and never waits for queue capacity. Capacity is 256 records; on
   overflow the newest notification is dropped, a counter increments and
   history is marked degraded. A duplicate `execution_id` already pending or
   retained keeps the first value and increments `duplicate_observation`.
5. A single writer goroutine appends accepted records in queue admission order.
   The queue mutex is released before any filesystem operation, so producer
   progress never depends on disk progress.
6. `Manager` recovers an observer panic at the callback boundary, logs only a
   stable failure class and the execution ID, and continues result publication
   and registry removal.
7. After E05 confirms an empty registry, the lifecycle closes observer admission
   and drains the queue for at most two seconds, then syncs and closes the active
   segment. A timeout is logged as degradation and never changes the E05
   cleanup verdict or the provider-release guard. If E05 cleanup cannot confirm
   an empty registry, the recorder is not closed and no clean flush is claimed.

## Rationale

- The interface lives where the event is produced, keeping the dependency
  direction clean and testable.
- Asynchronous single-writer delivery keeps disk latency out of the execution
  cleanup path while giving the history a deterministic append order.
- The honest backpressure policy (drop newest, no producer blocking, no retry)
  bounds memory and avoids turning a storage problem into an execution problem.

## Consequences

- A notification accepted by the queue can still be lost on crash, overflow,
  append failure or flush expiry. Delivery is at-most-once, never
  exactly-once.
- Degradation counters and the API `degraded` flag are process-local and do not
  certify completeness across restarts.
- E05 shutdown ordering must run the records drain after `Manager.Wait` and
  before provider release; this is a fixed two-second addition at most.
