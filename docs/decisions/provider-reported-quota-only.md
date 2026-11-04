# Provider-Reported Quota Only

**Date:** 2026-11-04

**Status:** accepted product decision; implementation not started

**Scope:** the provider quota and allowance view in the Connexto AI Hub

## Context

The provider administration screen presents quota information, but the Hub
currently has separate sources for configured admission policies, per-execution
usage, and provider-reported account limits. These sources have different
meanings. A value derived from Hub policy configuration or execution history
cannot be presented as a provider's current quota.

The pre-implementation capability audit is recorded in
[`../discovery/provider-reported-quota-audit-2026-11-04.md`](../discovery/provider-reported-quota-audit-2026-11-04.md).

## Decision

The provider quota view uses only quota or allowance data returned by the
official API of the corresponding configured provider. The Hub is an
integrator, parser, and presenter of those responses. It does not calculate
quota consumption, percentages, balances, limits, remaining amounts, or
renewal times.

Each displayed value must be traceable to the corresponding provider's API
response. Preserve its original value, unit, cycle, and provider timestamp.
Record the Hub query time separately when available. Never replace absent data
with zero.

## Product contract

- Discover configured providers from the platform's provider configuration
  store at runtime. Include configurations without a supported quota adapter
  and show why they have no reading.
- Use one quota-query adapter per provider type, existing backend-only stored
  credentials, and that provider's official endpoint.
- Run each provider query independently with a bounded timeout and isolated
  error handling. Normalize the result without changing its meaning or
  provenance.
- Show a percentage or a used, limit, or remaining value only if that exact
  value is returned by the provider. Preserve the returned unit and period.
- Distinguish provider-reported readings from providers with no reading.
  Explain whether a reading is unavailable because the API does not provide
  it, the provider is unsupported, credentials are missing or rejected, or
  the request failed. Do not expose credentials, tokens, or sensitive raw
  responses to the browser.
- Render all configured providers and the provider governance component
  referenced in Doop, preserving the Connexto visual identity. Render a
  progress bar only when an official percentage is present.
- Do not create demonstration data in the operational application.

## Scope boundary

This decision applies to provider-reported quotas and allowances shown in the
provider quota view. E09.2/E09.4 Hub admission policies and analytics remain
Hub-side controls and must not be treated as provider quota. E09.5 execution
metering remains per-execution usage and must not be accumulated to estimate a
provider balance or quota.

## Acceptance criterion

Every quota-related value rendered for a provider can be traced to a response
from that provider's official API. When no such value is available, the view
shows the reason for the missing reading and does not fabricate a substitute.

## Consequences

- Providers without a documented and credential-compatible official quota
  endpoint remain visible as unmonitored, with a reason.
- Some providers may expose only a subset of quota fields; the view must support
  partial readings without filling gaps through arithmetic.
- Official provider APIs may be experimental or change their response schema;
  adapter support must retain provider provenance and surface failures without
  falling back to Hub-side estimates.
