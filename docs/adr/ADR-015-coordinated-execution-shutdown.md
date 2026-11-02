# ADR-015 — Coordinated execution shutdown

- **Status:** accepted
- **Date:** 2026-10-31
- **Related:** SPEC E05, ADR-004, ADR-014

## Context

`App.Stop` stops modules sequentially in reverse declaration order and continues
after a module error. Fiber graceful shutdown can use the full shared deadline
while waiting for active handlers. E05 must close execution admission before
that wait, cancel remaining work at the grace deadline, join execution-owned
readers, and keep provider resources open if cleanup cannot be confirmed.

## Decision

1. Compose the HTTP server and execution manager in one application lifecycle
   module in `cmd/ai-hub`. Start delegates to the server. Stop closes the
    manager's registration gate before calling Fiber shutdown with the
   configured shutdown context.
2. Let registered HTTP executions finish while Fiber performs graceful
   shutdown. If graceful shutdown returns an error or reaches its deadline, ask
   the manager to reserve `canceled(shutdown)`, cancel each upstream context,
   and close each owned response body before force-closing active HTTP
   connections. This order keeps shutdown cancellation distinct from the
   request-context cancellation caused by forced connection closure.
3. After forced closure, wait for execution handlers, readers, body cleanup, and
   active registrations to finish using a separate cleanup context whose
   timeout equals the configured shutdown duration. The configured duration
   remains the graceful drain period; cleanup may extend total shutdown by at
   most one additional configured interval. The shutdown module reports an
   error if this cleanup deadline expires or active work remains. It never
   reports a clean stop in that condition.
4. Declare application modules in startup order `provider`, `catalog`, then the
   combined HTTP/execution lifecycle module. Reverse shutdown therefore
   completes HTTP/execution cleanup before catalog and provider shutdown.
5. Because `App.Stop` continues after errors, wrap provider shutdown at the
   composition boundary. It closes provider idle connections only after the
   execution manager confirms its active registry is empty and all execution
   readers have joined. Otherwise it returns an error and leaves the provider
   client open.
6. Requests that reached the HTTP server but attempt registration after the
   gate closes receive the local `503 execution_unavailable` response. The
   closed gate creates no execution and makes no provider call.

## Rationale

- Closing the gate before listener shutdown removes the registration race.
- HTTP shutdown waits for active request handlers within the configured
  graceful period; manager admission remains closed for the rest of shutdown.
  Cancellation happens only if that period expires or graceful shutdown fails.
- Forced connection closure unblocks a Gateway write after its response has
  already been committed, while the earlier shutdown reservation preserves the
  correct terminal cause.
- A provider-stop guard is required because the existing application runner
  deliberately attempts every module stop even when one fails.

## Consequences

- `cmd/ai-hub` owns the lifecycle composition and provider-stop guard; the
  generic application runner's existing error-joining behavior is preserved.
- Forced shutdown can terminate a partially delivered response, as required
  when the configured graceful period expires.
- If cleanup cannot be confirmed, shutdown returns an error and provider
  resources are not released as though the process stopped cleanly.
