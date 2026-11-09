# LLM Smoke Test — Phase 4 Preflight Evidence

- Date: 2026-11-09
- Gateway: started once via `go run ./cmd/ai-hub` from the repository root (dotenv auto-export; no file edited). Process kept running in the background.
- Authorization scope honored: no code/config/`.env`/catalog change; no `POST /v1/catalog/sync`; no model call.

## Checks

| Check | Result |
|---|---|
| Gateway binds `127.0.0.1:8080` only | PASS — `ss -ltn` shows single listener `127.0.0.1:8080`; curl to `10.255.255.254:8080` refused |
| `GET /healthz` | PASS — HTTP 200 |
| `GET /readyz` | PASS — HTTP 200 |
| `GET /v1/models` (Bearer gateway key) | PASS — HTTP 200, 466 aggregated model ids |
| 10 confirmed slugs served | PASS — all present, none missing |
| `GET /v1/catalog/models` (read-only) | PASS — HTTP 200, provider attribution resolved per model |
| `GET /v1/records/executions?limit=5` | PASS — HTTP 200, records carry `provider_id`, `model_id`, `outcome`, `started_at`, `completed_at`, `duration_ns`, `upstream_status`, `usage` |
| Model calls made | 0 (healthz/readyz/catalog/records do not execute inference) |

## Provider attribution (from gateway catalog, provider field)

| slug | provider | limits (context/output) | freshness |
|---|---|---|---|
| anthropic/claude-haiku-4.5 | openrouter | 200000 / 64000 | fresh |
| glm-5.3-flash | opencode-go | 1000000 / 131072 | fresh |
| deepseek-v4.1-flash | opencode-go | 1000000 / 384000 | fresh |
| claude-haiku-5-5 | opencode-go | 1000000 / 128000 | fresh |
| gpt-6-luna | opencode-go | 1050000 / 128000 | fresh |
| composer-2.5 | cursor | 200000 / 16384 | fresh |
| antigravity:claude-opus-4-6-thinking | antigravity | 200000 / 64000 | fresh |
| antigravity:claude-sonnet-4-6 | antigravity | 200000 / 64000 | fresh |
| antigravity:gemini-3.8-flash | antigravity | 1048576 / 65536 | fresh |
| antigravity:gpt-oss-120b-medium | antigravity | 128000 / 16384 | fresh |

Observations:

- Catalog `listing_state` is `stale` for all 10 while `presence: present` and `freshness: fresh` — the listing snapshot is older than the provider polls, but every slug is served by `GET /v1/models`. Per constraints, no catalog sync was triggered. This does not block the battery.
- `provider_state: unknown` per model is the readiness verdict for enabled providers before first traffic; will be confirmed by the first real call.

## Planned request count for the battery (pre-execution statement)

- TEST-03 is **one single call per model** (a single request carrying the full question "3 workers × 4 tasks"; expected answer 12). It is not 12 individual calls.
- Total planned: **10 models × 5 tests = 50 independent requests** (49 non-streaming + 10 streaming — precisely 40 with `stream:false`, 10 with `stream:true`).
- Plus, after the run, audit-only `GET /v1/records/executions?limit=100` (no inference).
- No retries, no fallback; each response is recorded once, pass or fail.

## Raw evidence files (this directory)

- `healthz.json`, `readyz.json` — gateway self-checks.
- `models.json` — full aggregated `/v1/models` response.
- `catalog-models.json` — full `/v1/catalog/models` response (provider/limits/capabilities).
- `diagnostics-providers.json` — enabled providers (antigravity, cursor, opencode-go, openrouter).
- `records-executions.json` — audit-capability proof sample (no secrets; sanitized because it contains no credentials by design).
