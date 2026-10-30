# AI Hub

Local-first Personal AI Orchestrator. The product and architecture baseline is
documented in [`AI-Hub-Product-Architecture-Kickoff.md`](AI-Hub-Product-Architecture-Kickoff.md).

**Current status:** E01 (Core Foundation) and E02 (OpenCode Go provider
integration, Responses pass-through) implemented. Routing, learning, quotas
and multi-provider orchestration are not implemented yet.

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
- [`docs/research/opencode-go-protocol-spike.md`](docs/research/opencode-go-protocol-spike.md) — validated integration evidence.
- [`docs/adr/`](docs/adr/) — architecture decision records.

## Known limitations

- The gateway has no authentication. It is bound to loopback by default, and a
  non-loopback bind emits a startup warning; do not expose it to other hosts
  before authentication exists (E04).
- Provider sessions are per-request in E02; cross-turn continuity of generated
  `x-opencode-session` values depends on a future AI Hub context/session layer.
- No retries, fallback or quota enforcement (E05/E09).
- No persistence; readiness has no dependency checks registered yet.
- No metrics or OpenTelemetry export (E06).
- `X-Request-Id` is the only correlation header accepted.
- No CI pipeline and no container image.
