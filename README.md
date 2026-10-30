# AI Hub

Local-first Personal AI Orchestrator. The product and architecture baseline is
documented in [`AI-Hub-Product-Architecture-Kickoff.md`](AI-Hub-Product-Architecture-Kickoff.md).

**Current status:** E01 (Core Foundation) implemented. Provider integration (E02)
and everything above it are not implemented yet.

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
go run ./cmd/ai-hub
```

With an explicit configuration file:

```bash
go run ./cmd/ai-hub -config configs/config.example.json
```

The server binds to `127.0.0.1:8080` by default.

```bash
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/readyz
```

## Endpoints

| Method | Path       | Purpose                                                    |
| ------ | ---------- | ---------------------------------------------------------- |
| GET    | `/healthz` | Liveness. Process is running; no dependency is evaluated.   |
| GET    | `/readyz`  | Readiness. Runs registered checks; `503` when any fails.    |

Both return `application/json` with `Cache-Control: no-store`. A request may
send `X-Request-Id` (8–128 characters of `A-Z a-z 0-9 . _ : -`); a valid value is
reused and echoed, otherwise a new identifier is generated.

## Configuration

Precedence: built-in defaults, then the JSON file, then environment variables.
Unknown JSON fields, trailing content and invalid values abort startup.

| JSON key                       | Environment variable               | Default     |
| ------------------------------ | ---------------------------------- | ----------- |
| `server.host`                  | `AIHUB_SERVER_HOST`                | `127.0.0.1` |
| `server.port`                  | `AIHUB_SERVER_PORT`                | `8080`      |
| `server.read_header_timeout`   | `AIHUB_SERVER_READ_HEADER_TIMEOUT` | `5s`        |
| `server.read_timeout`          | `AIHUB_SERVER_READ_TIMEOUT`        | `15s`       |
| `server.write_timeout`         | `AIHUB_SERVER_WRITE_TIMEOUT`       | `30s`       |
| `server.idle_timeout`          | `AIHUB_SERVER_IDLE_TIMEOUT`        | `60s`       |
| `server.shutdown_timeout`      | `AIHUB_SERVER_SHUTDOWN_TIMEOUT`    | `10s`       |
| `log.level`                    | `AIHUB_LOG_LEVEL`                  | `info`      |
| `log.format`                   | `AIHUB_LOG_FORMAT`                 | `json`      |

Durations use Go syntax (`250ms`, `15s`, `1m30s`). See
[`configs/config.example.json`](configs/config.example.json).

## Tests and validation

```bash
go test ./... -count=1 -race
go vet ./...
staticcheck ./...
golangci-lint run ./...
```

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — implemented architecture and flow.
- [`docs/adr/`](docs/adr/) — architecture decision records.

## Known limitations

- The gateway has no authentication. It is bound to loopback by default, and a
  non-loopback bind emits a startup warning; do not expose it to other hosts
  before authentication exists (E04).
- No provider, credential or model catalog support (E02 onward).
- No persistence; readiness has no dependency checks registered yet.
- No metrics or OpenTelemetry export (E06).
- `X-Request-Id` is the only correlation header accepted.
- No CI pipeline and no container image.
