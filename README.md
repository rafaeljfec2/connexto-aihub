# AI Hub

Local-first Personal AI Orchestrator. The product and architecture baseline is
documented in [`AI-Hub-Product-Architecture-Kickoff.md`](AI-Hub-Product-Architecture-Kickoff.md).

**Current status:** E01 (Core Foundation), E02 (OpenCode Go provider
integration, Responses pass-through) and E03 (Model Catalog) implemented.
Routing, learning, quotas and multi-provider orchestration are not implemented yet.

## Scope implemented

- Application bootstrap with explicit module lifecycle.
- Configuration loading and validation (defaults, file, environment).
- Local HTTP server with timeouts and bounded headers.
- `GET /healthz` (liveness) and `GET /readyz` (readiness).
- Structured JSON logging through `log/slog`.
- Request correlation ID (`X-Request-Id`) with validation and propagation.
- Panic recovery and access logging without headers or query strings.
- Graceful shutdown on `SIGINT` / `SIGTERM`.
- Automated tests for configuration, health, middleware, server and lifecycle.
- E03 model catalog: normalized, provider-scoped model entries with capability
  provenance, presence/freshness, startup and manual synchronization, and a
  versioned JSON snapshot with quarantine recovery.

## Requirements

- Go 1.23 or newer.
- No external Go dependencies.

## Run locally

```bash
cp .env.example .env   # then adjust values; never commit .env
go run ./cmd/ai-hub
```

With an explicit env file:

```bash
go run ./cmd/ai-hub -env-file /path/to/.env
```

The server binds to `127.0.0.1:8080` by default.

```bash
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/readyz
```

## Endpoints

| Method | Path                    | Purpose                                                                  |
| ------ | ----------------------- | ------------------------------------------------------------------------ |
| GET    | `/healthz`              | Liveness. Process is running; no dependency is evaluated.                  |
| GET    | `/readyz`               | Readiness. Runs registered checks; `503` when any fails.                   |
| POST   | `/v1/responses`         | Responses pass-through to OpenCode Go (E02; `503` when provider disabled). |
| GET    | `/v1/models`            | Provider model catalog pass-through (E02).                                 |
| GET    | `/diagnostics/providers`| Provider configuration and availability state, no secrets (E02).           |
| GET    | `/v1/catalog/models`    | Normalized catalog with filtered presence, listing state and capability (E03). |
| GET    | `/v1/catalog/models/{provider}/{model...}` | One model with provenance metadata and capabilities (E03).  |
| GET    | `/v1/catalog/providers` | Provider sync state, freshness and catalog counts (E03).                   |
| POST   | `/v1/catalog/sync`      | Trigger a bounded catalog synchronization (E03; state change).             |

Both return `application/json` with `Cache-Control: no-store`. A request may
send `X-Request-Id` (8–128 characters of `A-Z a-z 0-9 . _ : -`); a valid value is
reused and echoed, otherwise a new identifier is generated.

## Configuration

Precedence: built-in defaults, then environment variables (a dotenv file is
exported first; real environment variables always win). Unknown `AIHUB_*`
variables are ignored with a startup warning; invalid values abort startup.

| Environment variable                   | Default                              |
| -------------------------------------- | ------------------------------------ |
| `AIHUB_SERVER_HOST`                    | `127.0.0.1`                          |
| `AIHUB_SERVER_PORT`                    | `8080`                               |
| `AIHUB_SERVER_READ_HEADER_TIMEOUT`     | `5s`                                 |
| `AIHUB_SERVER_READ_TIMEOUT`            | `15s`                                |
| `AIHUB_SERVER_WRITE_TIMEOUT`           | `30s`                                |
| `AIHUB_SERVER_IDLE_TIMEOUT`            | `60s`                                |
| `AIHUB_SERVER_SHUTDOWN_TIMEOUT`        | `10s`                                |
| `AIHUB_LOG_LEVEL`                      | `info`                               |
| `AIHUB_LOG_FORMAT`                     | `json`                               |
| `AIHUB_PROVIDER_OPENCODEGO_BASE_URL`   | `https://opencode.ai/inference/go`   |
| `AIHUB_PROVIDER_OPENCODEGO_API_KEY`    | *(empty — provider disabled)*        |
| `AIHUB_DATA_DIR`                       | `${XDG_DATA_HOME:-$HOME/.local/share}/ai-hub` |
| `AIHUB_CATALOG_STALE_AFTER`            | `24h`                                |
| `AIHUB_CATALOG_SYNC_TIMEOUT`           | `10s`                                |

Durations use Go syntax (`250ms`, `15s`, `1m30s`). See [.env.example](.env.example).

### OpenCode Go provider (E02)

With `AIHUB_PROVIDER_OPENCODEGO_API_KEY` set, AI Hub forwards OpenAI Responses
requests byte-for-byte to the OpenCode Go upstream, injects a stable
`x-opencode-session` header (an incoming value is preserved), streams the
reply with inactivity-bounded deadlines and preserves upstream error status
codes. The API key never appears in logs, errors or responses. Provider state
is visible at `GET /diagnostics/providers`; a missing key disables the
provider without affecting readiness. `previous_response_id` is never emitted
(translated models such as `deepseek-v4.1-flash` require full conversation
replay — see `docs/research/opencode-go-protocol-spike.md`).

### Model catalog (E03)

The catalog stores a provider-scoped entry per model (`provider_id` +
`model_id`) with the closed nine-capability vocabulary, provenance/confidence
and the full upstream metadata. Capabilities are conservative: a modality is
`supported` only when the provider declares it, and absence is `unknown`, never
`unsupported`. Listing state combines presence, freshness and provider health;
it never implies that a model executed successfully (`execution_verification`
is `unverified`).

Synchronization runs asynchronously at startup (only when the provider is
healthy) and on demand through `POST /v1/catalog/sync`. A successful snapshot
is written atomically under `AIHUB_DATA_DIR`; a corrupt snapshot is quarantined
before the catalog starts empty, and a failed write never replaces the previous
valid snapshot. An empty upstream payload is rejected as `rejected_empty`
without deleting known models.

```bash
curl -s http://127.0.0.1:8080/v1/catalog/models
curl -s http://127.0.0.1:8080/v1/catalog/providers
curl -s -X POST http://127.0.0.1:8080/v1/catalog/sync
```

`GET /v1/models` remains the E02 byte-for-byte provider pass-through.

### Real integration smoke test

```bash
AIHUB_LIVE_SMOKE=1 AIHUB_PROVIDER_OPENCODEGO_API_KEY=oc_sk_... \
  go test ./internal/providers/ -run TestLiveOpenCodeGoSmoke -v
```

It is skipped automatically without both variables; mock-based green tests do
not count as real integration success.

## Tests and validation

```bash
go test ./... -count=1 -race
go vet ./...
staticcheck ./...
golangci-lint run ./...
```

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — implemented architecture and flow.
- [`docs/specs/E02-provider-management.md`](docs/specs/E02-provider-management.md) — E02 spec and decisions.
- [`docs/specs/E03-model-catalog.md`](docs/specs/E03-model-catalog.md) — E03 spec and decisions.
- [`docs/design/E03-model-catalog-design.md`](docs/design/E03-model-catalog-design.md) — E03 technical design.
- [`docs/research/opencode-go-protocol-spike.md`](docs/research/opencode-go-protocol-spike.md) — validated integration evidence.
- [`docs/adr/`](docs/adr/) — architecture decision records (including ADR-009 and ADR-010).

## Known limitations

- The gateway has no authentication. It is bound to loopback by default, and a
  non-loopback bind emits a startup warning; do not expose it to other hosts
  before authentication exists (E04).
- Provider sessions are per-request in E02; cross-turn continuity of generated
  `x-opencode-session` values depends on a future AI Hub context/session layer.
- No retries, fallback or quota enforcement (E05/E09).
- No persistence beyond the catalog snapshot; readiness has no dependency checks registered yet.
- The catalog is derived data: it reflects the last accepted sync, not a live
  execution probe. Synchronization runs at startup (healthy provider only) and
  on demand; there is no periodic refresh. One process owns a data directory.
- A successful catalog sync records `upstream_status` as 200; non-200 2xx
  variants from the models endpoint are not propagated in E03.
- No metrics or OpenTelemetry export (E06).
- `X-Request-Id` is the only correlation header accepted.
- No CI pipeline and no container image.
