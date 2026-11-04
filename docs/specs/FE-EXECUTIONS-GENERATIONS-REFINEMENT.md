# SPEC — Generations Experience Refinement

**Status:** Approved — Product Lead approval received in conversation (2026-11-03)
**Date:** 2026-11-03
**Classification:** Frontend behavior and visual refinement with backend telemetry/API evolution
**Implementation status:** Phase 7 complete — implementation and verification complete; independent review approved.

## 1. Intent

Refine the existing `/executions` screen into a content-free, searchable history of gateway generations. Use the Connexto AI Hub visual system and the Doop Executions frame as the primary visual reference, with OpenRouter Generations as a functional organization reference only.

The current E06 execution-record contract remains a stable source for lifecycle metadata. Generation-only usage and timing data must be exposed only when it has a verified per-execution source. The UI must distinguish zero from unknown, partial, unavailable, and invalid measurements.

## 2. Scope and constraints

### In scope

- One `Generations` view at the existing `/executions` route; no tabs for sessions, batches, videos, or upstream requests.
- Search, outcome/provider/model/date filters, manual refresh, cursor pagination, and selected-generation details.
- An initially closed, right-side detail drawer that overlays the table without reducing its available width.
- A field-by-field backend audit and a proposed content-free API and telemetry design.
- Responsive, keyboard-operable, accessible table and drawer states.

### Out of scope

- Prompt or response content, gateway credentials, provider secrets, or raw upstream headers/bodies.
- Automatic polling, automatic provider retry, provider fallback, or new routing policy.
- Editing the source Doop frame. The approved on-demand drawer interaction is recorded here as a product decision.
- A chart or aggregate KPI based on illustrative values. A real generation-volume aggregate is not currently available.
- Column-preference persistence. It would add a new preference-storage requirement; the initial design uses a compact fixed column set with horizontal scrolling.

## 3. Reference evidence and limits

Reviewed inputs:

- Product execution brief supplied with this request.
- Doop canvas `Qactw8snfC`, frame `PgFh4Y5Uka` (`04 — Execuções`), 1280 × 800, including its HTML/CSS and rendered screenshot.
- The HTML attachment added with the approval message; byte-for-byte identical to the earlier pasted HTML.
- Doop style guide `ai-hub-visual-system`.
- Current frontend files under `apps/frontend/src/features/executions/`, shared UI primitives, app shell, and frontend design tokens.
- Current E06 records handler/model/storage, Responses routing and lifecycle, E09.5 metering/storage, E09.4 analytics contract, and related ADRs.

The source HTML marks its rows, chart, usage values, costs, app names, sessions, key labels, and provider-response timeline as illustrative. They are not evidence that those fields exist in production. The live Doop frame currently reports `approvalStatus: needs_review`; the separately stated approval for the on-demand drawer interaction is captured as a product requirement, not as approval of every visual detail in the frame.

The three OpenRouter screenshots named in the product brief were not present among the accessible attachments in this session. Functional requirements below use only the written brief and verified repository/API behavior. Pixel comparison with those screenshots remains pending their availability.

## 4. Visual and interaction analysis

### 4.1 Doop reference facts

- The artboard is 1280 × 800. It has a 224 px left navigation, a 62 px top bar, and a content area with roughly 25 px horizontal padding.
- The content hierarchy is an `OBSERVABILITY / GENERATIONS` eyebrow, editorial page title, short description, sample-data marker, one `Generations` tab, a compact activity chart, filters, a dense table, and a small API/source footnote.
- The table uses aligned, compact columns for date, model, provider, app, input/output, cost, usage type, speed, routing overhead, time to first token, finish reason, and API-key label. Its 1290 px minimum width is wider than the content region; horizontal scrolling is intentional.
- Hover and selected rows use restrained cobalt. Status semantics use text and muted semantic colors. The drawer is positioned over the right side and includes summary metrics, Overview, Request, Provider responses, and a footer action.
- The drawer is closed initially in the HTML. Row selection opens it, the close button/backdrop and Escape close it, and selecting another row updates the detail. The prototype clears row selection on close and does not implement dialog focus management; the product requirement takes precedence for production behavior.
- At widths below 900 px, the reference wraps filters, reduces chart height, and widens the drawer, but the 224 px sidebar remains in the grid. This is not a complete mobile layout and must not be copied as the mobile acceptance behavior.

### 4.2 Design system

Use existing frontend tokens, which match the Doop guide:

| Role | Token / value |
| --- | --- |
| Graphite background | `#10141B` |
| Sidebar | `#131A24` |
| Surface | `#191F2A` |
| Raised surface | `#202A38` |
| Border | `rgba(180,194,218,.14)` |
| Primary text | `#EEF2F8` |
| Secondary text | `#A9B4C4` |
| Muted metadata | `#7D8A9D` |
| Cobalt | `#7896E8` |
| Focus accent | `#AFC2F2` |
| Healthy / warning / failure | `#88B995` / `#D6AD72` / `#D88186` |

Use Newsreader for the page title, DM Sans for interface text, and IBM Plex Mono for identifiers, numeric telemetry, and timestamps. Do not use OpenRouter's lime-green identity.

### 4.3 Reference-to-current comparison

| Area | Current frontend | Requested experience |
| --- | --- | --- |
| Page identity | “Execuções” lifecycle page | One Generations view, without sibling tabs |
| Summary area | Four cards calculated from the currently loaded page | No misleading page-local global KPIs; show only metrics backed by a defined aggregate source |
| Table | Time, model/provider, operation, status, duration | Compact generation fields with explicit unknown states and horizontal scrolling |
| Filters | Loaded-row search; outcome and provider API filters; model filter component exists but is not wired on this page | Search plus outcome, provider, model, and period controls; manual refresh; no polling |
| Pagination | API requests 50 records, then local page slicing; `next_cursor` is ignored | Use the E06 opaque cursor to load subsequent records; do not imply `count` is a global total |
| Selection | First record is selected automatically and detail is always visible in a 300 px master/detail column | No selection and no drawer on first render; drawer opens only after explicit row selection and overlays the table |
| Detail | E06 fields and a sanitized summary JSON block | Organized generation summary, Overview, Request, and actual observed provider events |
| Empty / error | Table and detail states exist | Preserve explicit list/detail loading, empty, error, degraded, and partial-telemetry states |

The existing `ExecutionsMetrics` totals and average duration cover only the loaded page, not the selected period. Remove or replace these cards unless a separately defined aggregate API supplies truthful totals. The reference's 30-day bars are illustrative; omit them until an authoritative all-generation time-series endpoint is approved and available.

## 5. Functional requirements

1. The page presents one Generations view at `/executions` and retains the AppShell, navigation, top bar, and Connexto identity.
2. On initial load, no generation is selected and the drawer is closed. The table uses all available content width.
3. Selecting a row opens the right drawer and identifies that row visually. Selecting another row updates the same drawer. Closing it with the close control or Escape preserves the selected row highlight until the selection is changed or filtered out.
4. On desktop, the drawer overlays the page and does not shrink the table. On small screens, it becomes a full-width or near-full-width overlay; the table remains horizontally scrollable behind it and is not forced into unreadable columns.
5. Search operates on loaded rows and covers execution ID, request ID, model, provider, and available cause/outcome text. Its scope is labelled as loaded results unless a server-side search contract is separately approved.
6. Outcome filters cover `completed`, `failed`, `canceled`, and `timed_out`; provider, model, and period filters map only to supported API query parameters. Date controls send RFC3339 `started_after` and `started_before` values.
7. Manual refresh is explicit. React Query polling remains disabled (`refetchInterval: false`). Pagination follows the returned `next_cursor`; page-local count is not presented as a global total.
8. Detail sections are Generation summary, Overview, Request, and Provider responses. Each absent field has a neutral unavailable/not-reported state; null is never rendered as zero.
9. The Provider responses timeline contains only events actually observed and durably recorded. The UI must not infer route, provider response, fallback, or finish events from the final status alone.
10. The table can horizontally scroll with stable column alignment, status text, hover and selected feedback. A column chooser is deferred.

## 6. Backend field audit

The current list and detail routes are `GET /v1/records/executions` and `GET /v1/records/executions/{execution_id}`. Both use gateway authentication and `Cache-Control: no-store`. The list supports `limit` (1–200), `provider_id`, `model_id`, `outcome`, `started_after`, `started_before`, and `cursor`; it returns `records`, page `count`, `next_cursor`, and `degraded`. Detail returns the same immutable E06 schema-v1 record. `usage` is always JSON `null`.

| Requested field | Exists now | Source / persistence | Current API | Gap and display rule |
| --- | --- | --- | --- | --- |
| Date | Yes | E06 `started_at` / `completed_at`; JSONL record | Yes | Use `started_at` as the table date; format in the selected locale/time zone without changing the stored UTC value. |
| Model | Yes | Selected `execution.Target.ModelID` in E06 record | Yes, `model_id` | Shows the model selected for the execution. Historical canonical ID is not stored; do not infer it from a later catalog snapshot. |
| Provider | Yes | Selected `execution.Target.ProviderID` in E06 record | Yes, `provider_id` | Identifies the selected provider, not a separate provider-attempt history. |
| App | No | No client-app field is captured or persisted | No | Show “Not reported”. Do not infer an app from a User-Agent. A self-reported client field requires a product/API decision. |
| Input tokens | Partially | E09.5 captures terminal provider usage; stored per execution in PostgreSQL `provider_execution_usage` | No per-generation API; aggregate analytics only | Preserve integer `0` and `null` distinctly; partial/unavailable/invalid usage remains labelled. Not available in file-only mode today. |
| Output tokens | Partially | Same as input tokens | No per-generation API | Same handling as input tokens. |
| Cost | Partially | Provider-reported cost and currency in the PostgreSQL usage row | No per-generation API; E09.4 excludes cost from aggregate responses | Show only provider-reported cost with currency and provenance; never estimate from a model price list or label it as Hub-billed cost. |
| Usage type | Partially | Usage row has `usage_status` (`reported`, `partial`, `unavailable`, `invalid`) and `usage_source` | No per-generation API | “Credits” is not a supported generic type. Prefer a clearly named usage status/source. |
| Speed | No | No per-execution token timing | No | Requires output token count plus a defined, observed output-stream interval. Do not divide by total execution duration and label it provider throughput. |
| Routing overhead | No | E08 records content-free decision logs, not a durable per-execution duration | No | Instrument route decision start/end and persist the measured duration. |
| Time to first token | No | No timestamp for first output delta | No | Instrument the first observed output event while forwarding the stream; store a duration only, never event content. |
| Finish reason | No | E06 stores outcome/phase/cause, not model finish reason | No | Capture only a normalized upstream finish reason when one is actually returned; otherwise null. |
| API key | No | The gateway has a static bearer key and provider credentials, not per-client key identities or aliases | No | Never expose a credential. Omit the field until a non-secret identity contract exists. |
| Generation ID | Yes, with different shape | E06 `execution_id`, 32 lowercase hexadecimal characters | Yes | Use `execution_id`; do not invent `gen_…` identifiers. |
| Request ID | Yes | Existing correlation ID in E06 | Yes | Safe operational identifier; retain current correlation semantics. |
| Session ID | No in E06 | `X-Opencode-Session` is supplied or generated per request and sent upstream; not persisted in E06 | No | The current ID is provider-session metadata, not durable conversation history. Persist/display only after the Product Lead approves scope and retention. |
| Streaming | No per-request field | Stream tap can identify SSE during usage capture, but the value is not in E06/API | No | Do not infer it from `response_committed`; add an explicit observed boolean if required. |
| Provider latency | No distinct value | E06 `duration_ns` measures the whole execution-manager lifecycle, including upstream open and body forwarding | Only total `duration_ns` | Do not relabel total duration as provider latency. Add an upstream-open timing boundary. |
| Fallback count / latency | No | E08 explicitly has no retry or provider fallback after selection | No | Show no fallback metric or “Not applicable” for the current one-attempt contract; do not fabricate a fallback timeline. |
| Provider responses | No durable timeline | Routing decisions are logged; E06 stores only one terminal outcome/phase/cause/status | No | Requires content-free, per-execution event capture and persistence. |
| Canonical ID / data policy | No in the execution record | Catalog may have current model metadata, but no historical execution snapshot or data-policy field is recorded | No | Do not populate from a mutable current catalog or invent a provider policy. |

### 6.1 Verified usage pipeline

`internal/metering/capture` observes forwarded chunks and extracts usage from a terminal Responses event or a bounded non-streaming JSON tail. `internal/metering` normalizes OpenCode Go and OpenRouter usage, including input/output/total/cached-input tokens and optional provider-reported cost/currency. The row is keyed by `execution_id` and stored in PostgreSQL by `internal/metering/pgstore`; E09.4 exposes aggregates, not individual usage. The E06 JSONL record remains separate and content-free. Usage persistence is best-effort and the current analytics/API path is PostgreSQL-only.

E08 captures the selected provider/model pair and emits content-free routing decision logs, but it does not durably retain a per-request event trace. The execution manager makes one provider attempt; E08 prohibits retry/fallback after selection. The provider session ID is transient command metadata. The static gateway/admin keys are never returned by APIs.

## 7. Proposed API and persistence

Keep E06 schema v1 and its existing routes unchanged for current consumers. Add a read-only generation projection that combines retained E06 records with optional per-execution usage and telemetry:

- `GET /v1/generations?limit=&provider_id=&model_id=&outcome=&started_after=&started_before=&cursor=` returns `{ generations, count, next_cursor, degraded }`. The filter vocabulary and cursor semantics match E06. `count` is the current page size, not an overall total.
- `GET /v1/generations/{execution_id}` returns one generation by the existing 32-hex E06 execution ID, with its usage, timing metadata, and ordered observed events.
- Both routes use the existing gateway authorization boundary, reject invalid filters/IDs with the current JSON error style, and return `Cache-Control: no-store`. They never return credentials, prompts, response bodies, raw provider headers, or raw provider errors.

Proposed content-free DTO shape (field names are a proposal for Phase 4 design):

```json
{
  "execution_id": "32-lowercase-hex",
  "request_id": "correlation-id-or-null",
  "provider_id": "provider-id",
  "model_id": "provider-model-id-or-null",
  "started_at": "RFC3339Nano UTC",
  "completed_at": "RFC3339Nano UTC",
  "duration_ns": 0,
  "outcome": "completed|failed|canceled|timed_out",
  "phase": "opening|forwarding",
  "cause": "known-cause-or-null",
  "upstream_status": 200,
  "usage": {
    "status": "reported|partial|unavailable|invalid",
    "source": "provider-source",
    "input_tokens": 0,
    "output_tokens": null,
    "provider_reported_cost": "0.000000000000",
    "currency": "USD"
  },
  "telemetry": {
    "streaming": null,
    "provider_latency_ns": null,
    "routing_overhead_ns": null,
    "time_to_first_token_ns": null,
    "output_stream_duration_ns": null,
    "finish_reason": null
  },
  "provider_responses": []
}
```

The numeric values above are shape examples only and must not become fixtures or production defaults. Cost should be represented as a decimal string to preserve PostgreSQL `numeric` precision. Preserve null token values, explicit zero, and E09.5 usage status independently. A missing telemetry row means unknown/unavailable; it does not mean zero.

For `provider_responses`, persist ordered, bounded events such as routing selection, upstream request start, upstream response headers, first output event, and terminal outcome. Store sequence, observed timestamp, event type, safe provider/model identity, optional HTTP status, and measured duration only. Do not persist event payloads. The current one-attempt contract yields no fallback event.

Recommended persistence direction: keep E06 JSONL records and E09.5 provider usage storage intact, add a content-free per-execution telemetry/event store keyed by `execution_id`, and query it through an application service that joins against retained E06 records. Align its retention with E06 record retention and clean up orphaned telemetry. The simplest implementation reuses PostgreSQL in the existing PostgreSQL mode; in file-only mode these enrichment fields would remain unavailable unless a file-backed sidecar is separately approved. This storage-mode difference is a product decision, not an implicit assumption.

Instrumentation boundaries to design in Phase 4:

1. Measure route selection from entry to selected/rejected decision; persist only successful generation association.
2. Measure provider-open latency from upstream dispatch to response headers.
3. Observe the first output event and terminal event while preserving byte-for-byte response forwarding; persist timestamps/durations, not streamed content.
4. Capture a normalized finish reason only where the provider response contract supplies one. Record the explicit streaming flag at request/response negotiation.
5. Keep fallback metrics absent under the current E08 contract. If retries/fallback are later approved, model each attempt as a distinct observed event.

Throughput is not yet finalized. Recommended definition: output tokens divided by the measured output-stream interval after the first output event, when both values exist and the interval is positive. Return null otherwise. Do not derive it from total execution duration.

## 8. Frontend architecture and behavior

Keep the Feature-First boundary in `apps/frontend/src/features/executions/`. Reuse the existing `fetchJson`, React Query, `ExecutionOutcomeBadge` visual semantics, shared `Button`, `FilterChips`, `GridPagination` where compatible, design tokens, AppShell, and the installed Radix Dialog primitive for focus and Escape behavior. Do not add a dependency.

Proposed feature responsibilities:

- `api/` and `types.ts`: generation list/detail DTOs and exact query parameter mapping.
- `hooks/`: list/detail queries, cursor accumulation, manual refresh, and filter state. Detail queries are disabled until the user selects a row; polling stays off.
- `components/GenerationsTable.tsx` and focused cell components: semantic table, aligned widths, selected row, status, compact values, and horizontal scrolling.
- `components/GenerationDetailsDrawer.tsx` with summary, Overview, Request, and Provider responses sections; explicit loading, error, and missing-data states.
- `pages/ExecutionsPage.tsx`: compose the existing AppShell and feature components; no business logic or transport parsing in table cells.

Use the existing cursor API for subsequent records. Reset the cursor and detail selection when filters change. If a selected record is filtered out, close the drawer and clear selection. Keep selected row styling while the drawer is closed after an explicit close. Restore keyboard focus to the selected row when the drawer closes. On mobile, use a full-width drawer overlay and preserve the AppShell's existing responsive navigation behavior.

## 9. Non-functional requirements

- Privacy: content-free metadata only; no prompt/response text, secrets, gateway/admin/provider keys, raw headers, or raw upstream errors.
- Provenance: label provider-reported values and currency; never conflate provider-reported cost, gateway-measured usage, or operator-configured budgets.
- Correctness: unknown, zero, partial, unavailable, and invalid are distinct states. No current-catalog backfill for historical canonical IDs or policies.
- Reliability: list records remain usable when enrichment is absent; expose a degraded/partial telemetry indicator rather than hiding E06 rows or substituting zero.
- Performance: bounded cursor pages and bounded event timelines; batch usage/telemetry lookup for list rows rather than one query per row.
- Accessibility: semantic table headers, keyboard-operable row selection, dialog semantics, visible focus, focus restoration, labelled controls, status text in addition to color, reduced-motion support, and WCAG AA contrast.
- Responsiveness: validate 375 × 812, 768 × 1024, 1280 × 800, 1440 × 900, and 1920 × 1080. Keep horizontal overflow inside the table container and avoid compressing the AppShell.
- Observability: preserve zero polling and explicit manual refresh. No new background refresh or external telemetry export.

## 10. Acceptance criteria

- **AC-GEN-01:** `/executions` shows one Generations experience and no additional data-source tabs.
- **AC-GEN-02:** Initial state has no selected row and no open drawer; the table uses the available width.
- **AC-GEN-03:** Selecting and switching rows updates the right drawer; close button and Escape close it; the selected row remains identifiable and focus returns to it.
- **AC-GEN-04:** Desktop drawer overlays without narrowing the table; mobile drawer adapts without making the table unreadable or obscuring navigation state.
- **AC-GEN-05:** Search, all four outcomes, provider/model/date filters, manual refresh, and cursor pagination map to the documented API behavior. No polling occurs.
- **AC-GEN-06:** List and detail use actual persisted data. The UI does not ship the Doop's illustrative chart, sample rows, costs, app names, session IDs, or provider timeline as operational data.
- **AC-GEN-07:** Tokens and cost appear only from a matching E09.5 per-execution record; cost includes provider provenance and currency. Null and zero remain distinct.
- **AC-GEN-08:** Speed, routing overhead, first-token time, finish reason, streaming, and provider-response events appear only after their specified instrumentation is captured; absent values remain unavailable.
- **AC-GEN-09:** No API key or secret is exposed. App, session ID, canonical ID, and data policy follow the explicit availability and approval rules in Sections 6–7.
- **AC-GEN-10:** Loading, empty, error, degraded, no-usage, partial-usage, and invalid-usage states are explicit and accessible.
- **AC-GEN-11:** The visual result is checked against the Doop screenshot at 1280 × 800 and the three OpenRouter screenshots when supplied; the Doop frame itself is not edited.
- **AC-GEN-12:** Keyboard, focus, reduced motion, contrast, and responsive behavior pass the test strategy below.

## 11. Test and visual validation strategy

Phase 4 was analysis-only and added or ran no tests. Phase 5 ran behavior tests before production changes; the missing generation route and the existing executions-record endpoint were observed as the initial Red state. Phase 6 adds claim-scoped backend, frontend, and PostgreSQL persistence coverage. Results are recorded in [the implementation validation report](../validation/FE-EXECUTIONS-GENERATIONS-REFINEMENT.md).

Planned implementation evidence:

- Go unit tests for instrumentation boundaries, event ordering, usage enrichment, nullable values, throughput definition, bounds, and retention.
- Go handler/integration tests for list/detail joins, cursor/filter behavior, unknown execution IDs, absent or unavailable enrichment, PostgreSQL degradation, no-store, gateway authentication, and absence of secret/content fields.
- React tests for initial closed state, no default selection, row selection/switching, Escape/close/focus restoration, each outcome and unknown state, cursor loading, filters, manual refresh, zero polling, and safe null/zero rendering.
- End-to-end checks for the complete list-to-detail flow using deterministic test fixtures only, with no sample fixture data in operational responses.
- Screenshot review at the five viewports in AC-GEN-11/12. Compare 1280 × 800 against the captured Doop frame. Compare OpenRouter organization only against the product-provided screenshots; those images are currently missing.
- Relevant quality gates after implementation: frontend typecheck, lint, tests and build; focused Go package tests, full Go checks and build; visual and accessibility inspection. Scope-proportional gates must be confirmed in the approved delivery plan.

## 12. Incremental implementation plan

The Product Lead approved this SPEC in conversation on 2026-11-03. Repository governance authorizes the complete delivery cycle without additional phase-by-phase approval.

### 12.1 Phase 4 design

**Architecture and reuse**

- Keep E06 JSONL as the lifecycle record source and keep E09.5 as the provider-usage source. Add an `internal/generations` application service that composes the existing `records.Service` with an optional enrichment repository. This preserves existing E06 consumers and keeps list enrichment batched.
- Add a PostgreSQL implementation under `internal/generations/pgstore` for bounded telemetry/event writes, batch reads by execution ID, detail reads, and retention cleanup. The new migration is additive. A nil enrichment repository represents file-only mode; list/detail still return E06 records and mark enrichment unavailable.
- The application service owns enrichment semantics: missing rows are unknown, usage statuses retain E09.5 values, cost remains provider-reported decimal plus currency, and throughput is computed only from valid output tokens and a positive observed interval. PostgreSQL read/write failures degrade enrichment without suppressing retained E06 records.
- Add `/v1/generations` handlers alongside the existing record handlers. Reuse `parseRecordFilter`, `records.Filter`, cursor semantics, gateway access policy, and existing JSON errors. Keep E06 routes and DTOs unchanged. Batch-load usage and telemetry for a list page; detail loads one execution and its bounded event sequence.
- Extend the existing execution result with content-free telemetry separate from E06 `FinishedExecution`. Carry the validated provider session only from the HTTP handler to the telemetry writer; do not add it to logs or E06. Measure provider open latency, first forwarded output, output interval, routing duration, and terminal outcome at the points that already own those lifecycle transitions. A forwarded chunk is not proof of a token event: expose its observed timeline duration, but keep time-to-first-token null until a provider stream parser can identify a token event. Record normalized finish reason only when the existing provider stream capture can establish one without retaining content; otherwise return null. Preserve response bytes and E08 single-attempt routing.
- Capture only bounded event metadata: sequence, observed time, normalized type, selected provider/model, optional upstream status, and duration. The event store must reject oversized event batches and avoid payloads, headers, or provider error text. Retention uses the configured E06 maximum age, with cleanup of telemetry and usage rows whose execution IDs are no longer retained.

**Implementation sequence and boundaries**

1. **Backend persistence and observation:** add migration, telemetry types/store, bounded retention, execution timing observations, and batched usage enrichment access. Validate the new behavior at package and PostgreSQL integration boundaries.
2. **Backend API:** add the generation application service, authenticated list/detail routes, filters/cursors, safe DTO mapping, degraded states, and handler tests. Keep existing E06 handlers and schemas untouched.
3. **Frontend data layer:** add generation DTOs/API, list and selected-detail queries, cursor accumulation, filters, loaded-results search, and explicit manual refresh. Disable polling and clear pagination/selection when API filters change.
4. **HTML-led experience:** replace the page-local metrics and always-visible master/detail layout with the approved HTML hierarchy, a single Generations view, dense horizontally scrollable table, and initially closed right overlay drawer. Keep the chart omitted because its HTML values are explicitly illustrative. Use existing design tokens, AppShell, shared primitives, and Radix Dialog focus behavior.
5. **Converge:** run Go and frontend claim-scoped tests, package checks, full required quality gates, visual checks at the five specified viewports, security review, author review, and the mandatory Phase 7 readonly independent review. Do not commit, push, or create a PR.

**Delivery boundary gate (DEEPWORK=yes)**

The change crosses execution capture, persistence, authenticated API, and frontend, and has two independently verifiable boundaries. Complete and validate the backend contract before binding the frontend to it. The final Phase 7 independent review runs once across the complete diff.

## 13. Decisions accepted for this delivery

1. **OpenRouter visual references:** the three screenshots were unavailable at implementation start. Continue from the supplied HTML, Doop frame, and visual guide; report screenshot comparison as an evidence limitation.
2. **Persistence modes:** persist usage/telemetry enrichment in PostgreSQL mode. In file-only mode, keep E06 generations available and return enrichment as unknown/unavailable; do not add a second file-backed telemetry store.
3. **Session ID:** persist the already validated `X-Opencode-Session` value with generation telemetry, return it only from the authenticated detail endpoint, never log it, and delete it using the configured E06 record age/retention interval. This includes both supplied and generated IDs.
4. **Application identity:** add no client header or heuristic. Render “Not reported” until a separate trusted identity contract exists.
5. **Throughput:** calculate output tokens divided by the measured interval from first observed output event to terminal event, only when both values exist and the interval is positive. Otherwise return null.
6. **Chart:** omit the illustrative chart and page-local summary cards. No generation aggregate endpoint is in this delivery.
7. **API key identity:** omit the API-key column. The static gateway credential has no per-client alias and must never be displayed.

## 14. Evidence

- **Claim:** This SPEC maps every requested generation field to a verified source or an explicit backend gap and defines observable UI/API acceptance criteria without weakening privacy or existing E06/E08 contracts.
- **Success type:** `manual`.
- **Evidence path:** Review this document against the approved product brief and supplied HTML attachment, Doop frame and style guide, `internal/records/`, `internal/execution/`, `internal/gateway/`, `internal/metering/`, the database migrations, related E06/E08/E09 specifications, and the current `apps/frontend/src/features/executions/` implementation. See [the implementation validation report](../validation/FE-EXECUTIONS-GENERATIONS-REFINEMENT.md) for test, visual, security, and independent-review evidence.
- **Budget:** One source audit for each field in Section 6; one review of the supplied HTML and Doop visual/interaction requirements in Sections 3–5; one completeness review of API, architecture, acceptance, privacy, tests, and decisions in Sections 7–13.
- **Affordance:** The generation projection and PostgreSQL content-free per-execution telemetry/event store are approved affordances. File-only records retain E06 fields and expose enrichment as unavailable.

Phase 4 design, Phase 5 Red tests, and Phase 6 implementation are complete under the approved SPEC and repository autonomy rules. Phase 7 independent review returned `APPROVED` after the provider-filter and navigation findings were corrected. The final frontend and focused E2E evidence is recorded in the validation report.
