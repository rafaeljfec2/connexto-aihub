# Product Decision: Unified Quota Management

**Date:** 2026-11-04  
**Status:** Accepted  
**Scope:** Connexto AI Hub — Quota Management & Multi-Account Provider Integration  
**Related Specs:** [`docs/specs/FE-PROVIDER-QUOTA-CONSUMPTION.md`](../specs/FE-PROVIDER-QUOTA-CONSUMPTION.md)  
**Related Decisions:** [`docs/decisions/provider-reported-quota-only.md`](provider-reported-quota-only.md)  

---

## 1. Context and Premise

The Connexto AI Hub centralizes the visibility of allowances, quotas, and usage across $N$ AI accounts distributed among multiple providers. The platform's role is to present what each provider's official APIs report.

The AI Hub is **not responsible for calculating balances, usage, limits, or percentages**. The source of truth is always the provider itself.

---

## 2. Product Roadmap: Two Discrete Stages

The product capability is divided into two distinct, sequential stages:

### STAGE 1 — MVP: Unified Quota Management (Current Scope)
- **Objective**: Allow the operator to track all their accounts and respective quotas in a single, reliable platform.
- **Requirements**:
  1. Dynamically discover all registered providers and accounts.
  2. Support multiple accounts for the same provider.
  3. Query each account independently and concurrently.
  4. Use official APIs available for each integration.
  5. Preserve the information scope (`api_key`, `account`, `workspace`, `organization`).
  6. Display only metrics actually provided by the provider.
  7. Preserve units, cycles, and timestamps.
  8. Transparently identify accounts without quota query support (`No reading`).
  9. Isolate failures between accounts and providers.
  10. Allow on-demand manual refresh (no periodic frontend polling).
  11. Do not calculate or automatically aggregate quotas.
  12. Do not implement intelligence or automatic routing in this stage.

### STAGE 2 — Intelligence & Automatic Routing (Future Stage)
- **Objective**: Use the centralized information from Stage 1 as signals for intelligent routing decisions.
- **Future Scope**:
  - Account availability interpretation.
  - Automatic account and provider selection.
  - Priority and load distribution rules.
  - Fallback mechanisms.
  - Decisions based on official limits and availability.
  - Explainability of routing decisions.
- **Boundary**: Stage 2 will have independent specifications, acceptance criteria, and validation. No intelligence or automatic routing logic will be anticipated during the Stage 1 MVP.

---

## 3. Core Conceptual Model

The architecture explicitly distinguishes:
- **Provider**: The AI vendor (e.g., OpenRouter, OpenAI, Anthropic).
- **Provider Account**: A specific account registered by the operator in the Hub.
- **Provider Credential**: Protected, encrypted credential associated with the account.
- **Provider Quota Snapshot**: Official information returned for that account at a point in time.
- **Provider Quota Metric**: An individual metric preserving its nature (`quota`, `usage`, `cost`, `rate_limit`), unit, and scope.

---

## 4. Fundamental Directive

> **First, we centralize and make the information of the $N$ accounts reliable. Then, we use this information for intelligence and automatic routing. Never mix the two responsibilities or create artificial calculations to anticipate the second stage.**

