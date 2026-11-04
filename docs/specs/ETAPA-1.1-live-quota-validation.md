# Stage 1.1 — Live Quota Data Validation and Operational Acceptance

**Date:** 2026-11-04  
**Status:** Approved execution scope from the project lead's request  
**Related delivery:** Unified Quota Management MVP (`fddc44c`)

## Intent

Prove that official OpenRouter quota values for each configured account reach
the Overview without losing values, units, scopes, or provider timestamps.
Correct any contract or presentation gaps found while establishing that proof.

## Requirements and constraints

- `GET /api/v1/key` uses only the account's inference API key and reports
  `api_key` scope.
- `GET /api/v1/credits` uses only a separately stored, encrypted Management API
  Key and reports `account` scope in a separate snapshot.
- Relevant provider quota fields from `/api/v1/key` must survive parsing and
  the API response, including budget meters with their own scope and reset
  timestamp and the daily free-model request meter. In the Overview, render
  key-level fields only when that account has no `account` snapshot.
- Missing provider fields remain absent; explicit zero remains zero.
- The Overview may show available account balance as
  `total_credits - total_usage` only when both provider values are present in
  the same `account` snapshot and both use USD. It must not derive other
  metrics or combine scopes or accounts.
- Configured accounts remain isolated through persistence, queries, and UI.
- The Overview renders one compact card per configured account. It shows a
  small set of provider-reported summary fields. When an account has an
  `account` snapshot, the Overview hides that account's `api_key` section while
  preserving both snapshots in the API response. Providers with only an
  `api_key` snapshot retain their key-level summary. An available account
  balance is shown only when both compatible account fields are present.
- Quota reads occur on page query and explicit refresh only; no periodic polling.
- Logs and frontend responses never contain credentials or raw provider bodies.
- Unsupported, unconfigured, unauthorized, and failed reads remain explicit.
- Intelligence & Automatic Routing is outside this scope. No commit or push.

## Acceptance criteria

1. Key-level and account-level OpenRouter metrics have separate credentials,
   snapshots, sources, and scopes.
2. The parser and DTO preserve all reported numeric values, units, scopes, and
   timestamps; omitted fields are omitted and explicitly returned zero remains
   zero. The Overview shows one compact card per account. When an `account`
   snapshot exists, its raw summary is shown and the matching `api_key` section
   is omitted from that card; an `api_key`-only account keeps its summary. The
   Overview may calculate available balance only from
   `total_credits - total_usage` in that same account snapshot, when both
   values are present and use USD. Missing fields do not produce a balance,
   zero remains a valid input, and values are never combined across accounts
   or scopes.
3. At least two configured OpenRouter accounts can be persisted and queried
   independently, including independent error states.
4. Page load and manual refresh query quotas; no timer-based refetch is enabled.
5. Automated tests establish credential isolation, scope separation, omission
   versus zero, UI fidelity, and secret non-disclosure.
6. Authorized live reads and browser screenshots are captured when credentials
   are available. Missing credentials are recorded as a limitation, not a pass.
7. Sanitized evidence is written to AI Memory and read back to verify
   persistence. Product visual acceptance remains pending until the product
   owner confirms the screenshots.

## Design boundary

Persist the optional Management API Key as a distinct AES-GCM protected field
on each provider account. Keep `/api/v1/key` and `/api/v1/credits` as independent
provider calls and return one snapshot per scope. Reuse the existing provider
account administration flow, quota query service, HTTP timeout, host allowlist,
and React Query manual refetch behavior. On the key snapshot, keep the
authentication scope (`api_key`) at snapshot level and preserve the provider's
budget scope (for example `member`) on the corresponding metric. Exclude
provider identifiers such as key labels, workspace IDs, user IDs, deprecated
rate-limit data, and duplicate `effective_budget` data from frontend responses.
Keep every supported raw field in the quota response. The Overview groups
snapshots by account and displays the account-scope summary when an account
snapshot exists, hiding that card's `api_key` section. It displays key-level
summary fields when the account has no account-scope snapshot. Never mix values
between scopes or accounts.

## Evidence

| Claim | Success type | Evidence path | Budget |
| --- | --- | --- | --- |
| Each provider route uses only its own credential and returns its own scope | test, oracle | Adapter and quota-service tests; authorized OpenRouter reads, if credentials are present | All quota fields, status classes, omitted and zero fields, and two accounts |
| The UI displays provider values without changing raw values and derives only the approved account balance | test, manual | Component tests and browser rendering of live `/api/ui/providers/quotas` data | Zero, high-precision amount, account total credits and usage, balance, budget scope and reset timestamp |
| Secrets and payloads do not leak | test, manual | API/admin response assertions, log capture, frontend network inspection | Both credentials and provider error bodies |
| Refresh is explicit and non-periodic | test, manual | Hook inspection/tests and browser network timeline | Initial query plus one manual refresh |
| Evidence is durable | command | AI Memory write followed by exact-page read | Session, sanitized data, tests, accounts, screenshots, limits |

**Affordance:** The existing provider admin form can accept a write-only
Management API Key for an account; no new provider or routing capability is
introduced.

## Validation record

**Validation date:** 2026-11-04  
**Operational acceptance:** pending the account Management API Key and product
owner visual confirmation.

### Live OpenRouter key response

An authorized read of `GET /api/v1/key` was made with the configured inference
credential. The audit retained only fields understood by the adapter and
excluded the provider key label, user/workspace/organization identifiers,
effective-budget duplicate, deprecated rate-limit data, and unrelated key
flags. The sanitized payload and field inventory are recorded in AI Memory at

`validation/etapa-1.1-live-quota-2026-11-04.md`.

Observed provider fields and values:

- `usage`, `usage_daily`, `usage_weekly`, `usage_monthly`, and all four
  `byok_usage*` fields were explicitly `0`.
- `limit`, `limit_remaining`, `limit_reset`, and `expires_at` were `null`.
- `budgets` was an empty array.
- `free_model_daily_requests` returned `used: 0`, `limit: 1000`, and
  `remaining: 1000`.
- The live `/api/ui/providers/quotas` response kept these metrics in an
  `api_key` snapshot. Null values were omitted; explicit numeric zero remained
  zero. No account-credit fields appeared in that snapshot.
- The same scope separation was rechecked through the running Vite proxy at
  `2026-11-04T14:28:37Z`; the API-key and account snapshots remained separate.

The independent account snapshot reports
`unconfigured_credential` because this provider record has no Management API
Key. The `GET /api/v1/credits` values supplied in the project brief
(`total_credits: 10`, `total_usage: 4.84186443`) were not revalidated through
the new account-scoped integration and are not shown as current application
data. The earlier local process returned those values mixed into its
`api_key` snapshot; that response is not accepted as scope-correct evidence.

### Persistence, UI, and security evidence

- A PostgreSQL integration test created two OpenRouter accounts in an isolated
  disposable database, persisted separate inference and Management API
  credentials, queried both accounts, and verified independent key/account
  values and encryption contexts.
- A live browser smoke test rendered the current `/api/v1/key` reading at
  desktop and mobile sizes. It verified explicit zero, omitted `null` values,
  account-scope missing-credential feedback, a successful manual refresh, and
  no periodic query during a 3.5-second observation window.
- Request counts render as integers while USD readings retain provider-reported
  decimal precision.
- The browser test inspected the returned quota DTO for credential, token,
  authorization, and raw-provider-payload fields; none were present. Backend
  tests also assert that provider error bodies and stored credentials do not
  reach API responses.
- `refetchInterval` is disabled for the quota query; focus and reconnect do not
  trigger an automatic quota refresh.

Screenshots are in `docs/validation/`:

- `etapa-1.1-live-quota-desktop.png`
- `etapa-1.1-live-quota-desktop-panel.png`
- `etapa-1.1-live-quota-account-scope-desktop.png`
- `etapa-1.1-live-quota-mobile.png`
- `etapa-1.1-live-quota-mobile-panel.png`
- `etapa-1.1-live-quota-account-scope-mobile.png`

### Overview summary correction — 2026-11-04

At this point in the validation, the Overview grouped scope snapshots into one
card per provider account and rendered key-level and account-level values in
separately labeled sections. It removed period and BYOK detail and did not
derive percentages or cross-account totals. An HTTP-level quota query failure
was shown as an error instead of the empty-accounts state. The subsequent
`API-key block removal` section records the current display behavior.

At the time of the local application response recorded here, OpenRouter key
usage was explicit zero and the free-model daily request meter was `used: 0`,
`limit: 1000`, and `remaining: 1000`. A sanitized local PostgreSQL check then
found the inference credential ciphertext configured and the separate
Management API credential column empty. The later live account-scope result is
recorded in the follow-up section below.

### Test results and remaining gates

- `go vet ./...`: passed.
- Provider and PostgreSQL provider tests: passed; the dedicated two-account
  persistence integration test ran against a throwaway database and passed.
- Gateway quota and Management API credential tests: passed.
- Frontend tests: passed, including value-fidelity and write-only credential
  form coverage.
- Production frontend build: passed. Existing dependency-comment and bundle
  size warnings were emitted by Vite.
- Live Playwright smoke test: passed with the real OpenRouter key snapshot.
- The full `go test ./... -count=1 -race` run had one failure in the unrelated
  E09.5 provider-usage integration test: its fixed query date is 2026-11-03,
  while the usage row is recorded on the current date, 2026-11-04. All other
  packages reported by that run passed.
- Snyk SAST could not run because the connected Snyk tool is unauthenticated.
- Product owner visual confirmation is still required. Real `/credits` values
  also require an operator to enter the separate Management API Key through
  provider administration before a live account-scope read can be accepted.

Do not mark Stage 1.1 operationally accepted until both remaining live-data and
product-owner gates are cleared.

### Product clarification — 2026-11-04

The product owner explicitly approved displaying available balance in response
to a clarification about the earlier no-derived-values constraint. For an
available account snapshot, calculate only `total_credits - total_usage` when
both values are explicitly present and both units are USD. The API-key `usage`
value remains a separate key-scope reading and is never used in this
calculation. Preserve both raw account values beside the calculated balance,
render the balance with two decimal places, and omit it when either value is
absent or the units differ. An explicit zero is present and remains valid.
Never combine values across snapshots, scopes, or accounts.

### Overview display refinement — 2026-11-04

The product owner requested removal of the `Chave API` section from the
provider card when an `account` snapshot is present. Keep the API-key snapshot
and its raw data in the response for fidelity, but do not render its section in
that card. Continue rendering API-key summaries for accounts that do not have
an account-scope snapshot. The account section continues to show raw purchased
credits and total usage, plus the approved available-balance calculation.

Follow-up validation completed on 2026-11-04:

- Component tests verify the USD 10.00 purchased-credit value, USD 4.84186443
  account usage, USD 5.16 available balance, explicit zero, missing-field
  omission, unit mismatch omission, and independent calculations for two
  account snapshots.
- Frontend check passed: 41 test files and 108 tests. Production build passed;
  Vite emitted existing dependency-comment and bundle-size warnings.
- The independent senior code review returned `APPROVED_WITH_NOTES` with no
  critical or important findings. Its minor finding about sensitive field-name
  variants was fixed by normalizing names and explicitly testing
  `management_api_key` and `accessToken`; the reviewer confirmed the fix in a
  second round with no new findings.
- The authorized live Playwright smoke passed against the local application.
  It compared the account fields from `/api/ui/providers/quotas` with the raw
  UI values and checked the calculated balance against that same DTO snapshot.
  It also confirmed manual refresh, no new request during the 3.5-second
  observation window, and no credential or raw-provider-payload keys in the
  response.
- Updated desktop and mobile screenshots are in `docs/validation/`; the
  account-scope captures show purchased credits, account usage, and calculated
  available balance without the key-scope block.

### API-key block removal — 2026-11-04

The Overview now omits an account's `api_key` section whenever the same
account has an `account` snapshot. The key snapshot and its explicit zero
remain present in the quota API response. Providers with only an `api_key`
snapshot still render their key summary. The live smoke asserts both the
response and the rendered omission; refreshed desktop and mobile screenshots
contain only the OpenRouter account section.

Product visual acceptance remains pending until the product owner reviews the
updated screenshots. This implementation and evidence do not complete
operational acceptance by themselves.
