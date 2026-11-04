# Provider-Reported Quota Capability Audit

**Date:** 2026-11-04

**Status:** pre-implementation audit

**Decision:** [`../decisions/provider-reported-quota-only.md`](../decisions/provider-reported-quota-only.md)

## Summary

The checkout contains execution and model-list adapters for OpenCode Go and
OpenRouter. It does not contain a quota-query adapter interface or a quota
endpoint. The configured-provider API and frontend list are dynamic, while the
runtime adapter registry is bound to the two fixed provider IDs at startup.

OpenRouter documents an official current-key endpoint that returns key-level
usage, limit, remaining amount, and reset interval. Its existing bearer API key
auth model is compatible with the Hub's stored OpenRouter credential. Quota
support is therefore feasible, but has not been implemented or validated
against a live account.

OpenCode documents an experimental usage-limits endpoint that returns Go
rolling, weekly, and monthly meters and Console budgets. The documentation says
the endpoint reports the workspace that is signed in for the requested
location, or `null` when it is not signed in. The checkout's OpenCode Go
adapter stores an inference API key and calls inference/model endpoints; the
official documentation does not establish that this key can authenticate the
usage-limits route. Compatibility and endpoint reachability are unresolved.
OpenCode inference responses also report per-request allowance, but that is
not a current quota snapshot and must not be aggregated from execution records.

No other quota adapters are present in the checkout. Arbitrary provider records
may be stored, but the runtime registry currently binds only OpenCode Go and
OpenRouter. Other configured records must remain visible with an unsupported
adapter/no-reading reason until a provider-specific capability is established.

## Repository evidence

| Area | Evidence | Finding |
| --- | --- | --- |
| Configured-provider discovery | `internal/providers/service.go` (`Service.List`); `internal/providers/pgstore/store.go` (`selectAllManaged`, `Store.List`) | The backend can list persisted provider configurations. |
| Runtime adapter set | `cmd/ai-hub/main.go` (`providerRegistry.Bind`); `internal/providers/registry.go` (`Registry.ProviderIDs`) | Startup binds fixed IDs `opencode-go` and `openrouter`; registry IDs represent bound runtimes, not every configured record. |
| Existing adapter contracts | `internal/providers/registry.go` (`Adapter`, `ResponsesAdapter`, `ModelListAdapter`) | Contracts cover execution and model listing; no quota capability exists. |
| Credential boundary | `internal/providers/credentials.go` (`CredentialCipher`); `internal/providers/service_credentials.go` (`credentialOf`) | Provider credentials are sealed with AES-GCM and loaded by backend services. Provider administration responses expose only whether a credential is configured. |
| OpenRouter adapter | `internal/providers/openrouter.go` (`OpenRouterClient`, `ListModels`, `OpenTargetResponses`) | Uses a provider base URL and bearer key for model listing and inference; no quota query exists. |
| OpenCode Go adapter | `internal/providers/client.go` (`Client`, `ListModels`, `OpenTargetResponses`) | Uses the configured inference endpoint and bearer key; no quota query exists. |
| Execution metering | `internal/metering/normalize.go` (`NormalizeExecutionUsage`) | Normalizes per-execution token/cost response fields for OpenCode Go and OpenRouter. It is not a quota snapshot. |
| Existing frontend discovery | `apps/frontend/src/features/providers/api/queries.ts` (`listProviders`); `ProvidersPage.tsx` | The screen already loads the configured provider list dynamically. |
| Existing frontend placeholders | `apps/frontend/src/features/providers/components/ProviderQuotaTable.tsx`; `ProviderGovernanceMetrics.tsx`; `lib/providerPresentation.ts` | Rows hardcode no reading and a zero-valued `ProgressBar`; governance metrics include fixed `00` values, and current helper marks every provider as without a reading. These are not provider data. |
| Hub-side quota semantics | `docs/specs/E09.4-usage-analytics.md` | E09.4 computes Hub policy utilization from admission counters. It is not provider-reported quota and must not feed this view. |

## Provider capability matrix

| Configured provider type | Official capability found | Existing credential compatibility | Audit result |
| --- | --- | --- | --- |
| OpenRouter | `GET https://openrouter.ai/api/v1/key` returns current API-key information. The published response includes `usage`, `limit`, `limit_remaining`, `limit_reset`, and may include budget fields with `limit_usd`, `spend_usd`, `remaining_usd`, `reset_interval`, and `resets_at`. These are provider-returned fields. | The endpoint documents the API key as a bearer token, matching the existing OpenRouter credential model. | Official quota values are available in documentation; implementation and authenticated live validation remain outstanding. Only returned percentage values may be displayed; the documented fields do not authorize deriving one. |
| OpenCode Go | The experimental `GET /api/experimental/usage/limits` route documents `go.rolling`, `go.weekly`, `go.monthly` meters (`percent`, `limited`, `resetsAt`) and budget entries (`limit`, `spent`, scope and reset time). It may return `null`. | The route describes the Console workspace signed in at a location. Existing Hub configuration stores a Go inference API key. Official docs reviewed here do not confirm that this key authorizes the route. | Provider capability is documented, but a provider-only integration using the existing stored credential is not proven. Resolve authentication/endpoint compatibility before claiming this provider as monitorable. Do not infer remaining values from `limit` and `spent`. |
| Any other provider record/type | No other quota provider adapter is bound in the checkout. | Depends on future provider-specific credential and API contracts. | Display dynamically as unsupported/no reading until official capability and an adapter are available. |

## Gaps to resolve before implementation

1. Define a quota adapter contract carrying configured provider identity,
   reading state/reason, original provider fields, units, cycles, provider
   timestamps, query timestamp, and provenance. Keep nullable values distinct
   from zero.
2. Drive the provider result set from persisted configuration records, not
   only `Registry.ProviderIDs`; preserve a no-adapter result for configured
   records that cannot be queried.
3. Implement a backend-only OpenRouter quota query against its official key
   endpoint, with its own timeout/error boundary and safe response parsing.
   Validate behavior with provider fixtures and, when authorized credentials
   are available, a live read-only call.
4. Resolve OpenCode Go's official experimental usage endpoint authentication,
   base URL, and workspace/location semantics. If the existing stored Go key
   cannot access it, report the capability as unavailable rather than adding a
   manually configured substitute or reusing execution history.
5. Replace all current frontend placeholder values with nullable provider data,
   explicit monitored/no-reading states and provider-specific absence reasons.
   Show no progress bar without an official percentage and no `0` for missing
   fields.
6. Keep E09.4 admission-policy utilization and E09.5 per-execution metering out
   of the provider quota view. They may remain available in their existing
   product areas with their existing meanings.
7. Ensure API responses sent to the frontend contain only normalized quota
   fields and provenance, never credentials, tokens, or sensitive provider
   response payloads.

## External evidence

Official documentation reviewed on 2026-11-04:

- OpenRouter, [Get current API key](https://openrouter.ai/docs/api/api-reference/api-keys/get-current-api-key): bearer authentication and key/budget response fields.
- OpenCode, [HTTP API reference](https://dev.opencode.ai/v2/docs/api): the experimental usage limits route and its workspace authentication semantics and response schema.
- OpenCode, [Console inference API](https://opencode.ai/v2/docs/console/api/inference): inference usage can include `opencode.allowance` for a request. This is per-request data, not an account quota snapshot.
- OpenCode, [Go usage limits](https://opencode.ai/v2/docs/console/go/): documents Go plan usage limits; published limit tables are not current account usage readings.

This audit inspected repository code and public provider documentation. It did
not call a provider account API, use stored credentials, or validate a live
account response. No application code or tests were changed as part of this
audit.
