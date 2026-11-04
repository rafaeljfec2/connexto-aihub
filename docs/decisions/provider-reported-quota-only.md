# Provider-Reported Quota Only

**Date:** 2026-11-04

**Status:** accepted product decision; implemented for OpenRouter and OpenCode Go quota views

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
integrator, parser, and presenter of those responses. It must not invent
missing fields from Hub admission policy, execution history, or synthetic
thresholds.

Each displayed scalar must be traceable to the corresponding provider's API
response (or to a presentation derived only from two such official scalars, as
defined below). Preserve original units, cycles, and provider timestamps.
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
- Show a percentage only when the provider returns that exact percentage field.
  Show used, limit, or remaining only when those exact fields are returned, or
  when remaining is presented as `limit − used` and both operands are official
  fields from the same snapshot (provider did not return remaining).
- Distinguish provider-reported readings from providers with no reading.
  Explain whether a reading is unavailable because the API does not provide
  it, the provider is unsupported, credentials are missing or rejected, or
  the request failed. Do not expose credentials, tokens, or sensitive raw
  responses to the browser.
- Render all configured providers and the provider governance component
  referenced in Doop, preserving the Connexto visual identity.
- Progress bars are allowed when:
  1. an official percentage is present, or
  2. both official `used` and `limit` are present for the same reading.
  In case (2), the bar is a visual fill of those two fields only. Do not label
  a Hub-invented percentage for providers that did not return one.
- Do not create demonstration data in the operational application.
- Do not raise attention states from Hub-side thresholds (for example 80%).
  Amber/attention chips require an explicit provider-reported signal.

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
  partial readings. Allowed presentation arithmetic is limited to progress-bar
  fill and `limit − used` remaining when both official operands exist.
- Official provider APIs may be experimental or change their response schema;
  adapter support must retain provider provenance and surface failures without
  falling back to Hub-side estimates.
