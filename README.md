# AI Hub

Local-first Personal AI Orchestrator. The product and architecture baseline is
documented in [`AI-Hub-Product-Architecture-Kickoff.md`](AI-Hub-Product-Architecture-Kickoff.md).

**Current status:** E01 (Core Foundation), E02 (OpenCode Go provider
integration, Responses pass-through), E03 (Model Catalog), E04 (Gateway
hardening and access control), E05 (execution lifecycle) and E06 (execution
records) implemented. Routing, learning, quotas and multi-provider orchestration
are not implemented yet.

## Scope implemented

- Application bootstrap with explicit module lifecycle.
- Configuration loading and validation (defaults, environment and dotenv).
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
- E04 gateway access control: mandatory static bearer key, exact Host/Origin
  validation, fail-closed listener exposure, bounded request bodies and
  per-request provider sessions.
- E05 execution lifecycle: coordinated timeout, cancellation and shutdown with a
  single terminal result per registered execution.
- E06 execution records: non-blocking observation of finished executions,
  append-only JSONL storage under `AIHUB_DATA_DIR`, bounded retention and
  cursor-based query endpoints.

## Requirements

- Go 1.23 or newer.
- No external Go dependencies.

## Run locally

```bash
cp .env.example .env   # then adjust values; never commit .env
# AIHUB_GATEWAY_API_KEY is mandatory and must differ from the provider key.
# Generate a private value with `openssl rand -base64 48`; .env.example leaves
# it empty intentionally so startup fails closed until the operator sets it.
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

curl -i -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  http://127.0.0.1:8080/v1/models
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
| GET    | `/v1/records/executions` | Filtered page of retained execution records (E06).                        |
| GET    | `/v1/records/executions/{execution_id}` | One retained execution record (E06).                         |

Health endpoints return `application/json` with `Cache-Control: no-store`.
Every other route requires `Authorization: Bearer <AIHUB_GATEWAY_API_KEY>` and
an allowed `Host`; `HEAD /healthz` and `HEAD /readyz` are not authentication
exempt. A request may send `X-Request-Id` (8–128 characters of `A-Z a-z 0-9 . _
: -`); a valid value is reused and echoed, otherwise a new identifier is
generated. `X-Request-Id` is correlation only and is never used as a provider
session.

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
| `AIHUB_RECORDS_MAX_AGE`                | `720h` (30 days)                     |
| `AIHUB_RECORDS_MAX_COUNT`              | `10000`                              |
| `AIHUB_RECORDS_MAX_STORAGE_BYTES`      | `134217728` (128 MiB)                |
| `AIHUB_RECORDS_SEGMENT_BYTES`          | `8388608` (8 MiB)                    |
| `AIHUB_RECORDS_RETENTION_INTERVAL`     | `1h`                                 |
| `AIHUB_GATEWAY_API_KEY`                | *(required, no default)*             |
| `AIHUB_GATEWAY_ALLOWED_HOSTS`          | loopback hosts when unset            |
| `AIHUB_GATEWAY_ALLOWED_ORIGINS`        | *(empty — no browser Origin allowed)*|
| `AIHUB_GATEWAY_MAX_BODY_BYTES`         | `33554432` (32 MiB)                  |

Durations use Go syntax (`250ms`, `15s`, `1m30s`). See [.env.example](.env.example).

Gateway settings (E04):

- `AIHUB_GATEWAY_API_KEY` is mandatory, must contain 32–512 ASCII characters
  from the RFC 6750 `b64token` set and must differ from the provider key.
- Allowed hosts are exact lowercase DNS names or IP literals without ports,
  schemes, paths or wildcards. When unset on a loopback bind, the defaults are
  `localhost`, `127.0.0.1` and `::1`. An explicitly empty value is invalid.
- A non-loopback `AIHUB_SERVER_HOST` requires an explicit non-empty
  `AIHUB_GATEWAY_ALLOWED_HOSTS`; the actual bound listener is checked after
  `net.Listen` and startup fails closed otherwise.
- Allowed origins are exact `http`/`https` origins without path, query,
  fragment or user information. An empty origin list permits only requests
  without an `Origin` header.
- `AIHUB_GATEWAY_MAX_BODY_BYTES` accepts 1,024–134,217,728 bytes. Catalog sync
  keeps its own 4 KiB cap.

### Gateway access control (E04)

The gateway validates `Host` and `Origin` before authentication, then requires
one well-formed bearer token on every route except exactly `GET /healthz` and
`GET /readyz`. Local rejections use the stable envelope
`{"error":{"type":"...","message":"..."}}` with `Cache-Control: no-store`;
`401` adds `WWW-Authenticate: Bearer realm="ai-hub"`. The gateway key is never
forwarded upstream: the provider adapter continues to set the provider
credential.

`POST /v1/responses` requires `application/json` (optionally
`charset=utf-8`), rejects non-identity `Content-Encoding`, and buffers at most
the configured limit plus one sentinel byte before forwarding the exact accepted
body. SSE responses remain unbuffered. A missing or empty
`x-opencode-session` is replaced with `aihub-ses-<32 hex>`; a valid value of
1–128 characters from `A-Z a-z 0-9 . _ : -` is preserved; malformed or duplicate
values are rejected with `400`. No durable AI Hub session is created.

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
curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  http://127.0.0.1:8080/v1/catalog/models
curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  http://127.0.0.1:8080/v1/catalog/providers
curl -s -X POST -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  http://127.0.0.1:8080/v1/catalog/sync
```

`GET /v1/models` remains the E02 byte-for-byte provider pass-through.

### Execution records (E06)

E05 offers every finished execution to the records service after the response
body is closed and before the execution is deregistered. The offer is
non-blocking: records are queued in memory (256 entries) and written by a single
writer, so queue overflow, storage failure or slow disk can never change the
execution result, the client response or the provider-release guard. Overflow
drops the newest event and marks the process degraded.

Records are immutable schema-v1 JSONL lines written to
`${AIHUB_DATA_DIR}/records` as rotating segments. Only operational metadata is
persisted: execution/request IDs, provider and model IDs (null when unknown),
UTC timestamps, duration, outcome, phase, cause, upstream status and whether the
response was committed. Payloads, generated content, headers, provider session
ids and credentials are never stored; `usage` is always `null` in schema v1.

Retention trims the oldest eligible records by age, count and managed storage,
and startup recovery keeps the valid prefix of a corrupted segment while
isolating the invalid suffix. A `404` means the record is absent from the
currently retained set, not that the execution never happened; `503` is returned
only when the query index is unavailable.

```bash
curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  "http://127.0.0.1:8080/v1/records/executions?limit=20&outcome=completed"
curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  "http://127.0.0.1:8080/v1/records/executions?provider_id=opencode-go&started_after=2026-10-01T00:00:00Z"
curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  "http://127.0.0.1:8080/v1/records/executions/<32-hex-execution-id>"
```

List filters are `limit` (1–200, default 50), `provider_id`, `model_id`,
`outcome` (`completed`, `failed`, `canceled`, `timed_out`), `started_after`,
`started_before` (RFC3339) and `cursor`. Pages are ordered by
`completed_at DESC, execution_id DESC`; pass `next_cursor` back to continue.
An `execution_id` is exactly 32 lowercase hexadecimal characters.

Retention settings accept `AIHUB_RECORDS_MAX_AGE` 1h–8760h,
`AIHUB_RECORDS_MAX_COUNT` 1–100000, `AIHUB_RECORDS_MAX_STORAGE_BYTES`
16 MiB–1 GiB, `AIHUB_RECORDS_SEGMENT_BYTES` 64 KiB–32 MiB and
`AIHUB_RECORDS_RETENTION_INTERVAL` 1m–24h. The segment size must be at least
16 KiB smaller than the storage cap and no greater than half of it.

### Real integration smoke test

```bash
AIHUB_LIVE_SMOKE=1 AIHUB_PROVIDER_OPENCODEGO_API_KEY=oc_sk_... \
  go test ./internal/providers/ -run TestLiveOpenCodeGoSmoke -v
```

It is skipped automatically without both variables. The provider-only smoke
also honors `AIHUB_PROVIDER_OPENCODEGO_BASE_URL` when set and does not require
the gateway API key; mock-based green tests do not count as real integration
success.

See also the [Codex E07 report](docs/validation/E07-codex-first-harness-integration.md).

### Codex CLI validation (experimental; E07 incomplete)

E07 has not established Codex CLI support. The latest real-provider attempt
used `codex-cli 0.164.0-alpha.11.1`, Go `1.23.0`, Linux `amd64`, and model
`deepseek-v4.1-flash`; see the [E07 validation report](docs/validation/E07-codex-first-harness-integration.md)
for the scenario results and limitations. Do not treat this procedure as a
support claim.

Prerequisites are Bash, Go 1.23, the recorded Codex CLI release, an available
OpenCode Go provider credential, and the selected model. This manual smoke
uses two separate terminal sessions. It does not edit `~/.codex`, project
`.env` files, or tracked Codex configuration.

In Terminal A, make the provider key available only to the AI Hub process,
enter a distinct gateway key without echo, create dedicated data storage, and
start the Hub on its loopback default:

```bash
export AIHUB_PROVIDER_OPENCODEGO_API_KEY="$OPENCODE_API_KEY"
export AIHUB_DATA_DIR="$(mktemp -d)"
read -r -s -p 'AI Hub gateway key: ' AIHUB_GATEWAY_API_KEY
printf '\n'
export AIHUB_GATEWAY_API_KEY
test -n "${AIHUB_PROVIDER_OPENCODEGO_API_KEY:-}" && test -n "${AIHUB_GATEWAY_API_KEY:-}"
go run ./cmd/ai-hub -env-file /dev/null
```

The gateway key must be 32–512 characters in E04's `b64token` grammar and
must differ from the provider key. Do not place either value in a command,
Codex TOML, or repository file.

In Terminal B, create `CODEX_HOME` before writing its configuration. The
provider uses the local Responses endpoint and reads the gateway key from the
environment variable named by `env_key`:

```bash
export CODEX_HOME="$(mktemp -d)"
mkdir -m 700 -p "$CODEX_HOME"
cat > "$CODEX_HOME/config.toml" <<'EOF'
model = "deepseek-v4.1-flash"
model_provider = "aihub"
sandbox_mode = "workspace-write"

[model_providers.aihub]
name = "AI Hub OpenCode Go"
base_url = "http://127.0.0.1:8080/v1"
wire_api = "responses"
env_key = "AIHUB_CLIENT_KEY"
request_max_retries = 0
stream_max_retries = 0
EOF
read -r -s -p 'AI Hub gateway key: ' AIHUB_CLIENT_KEY
printf '\n'
export AIHUB_CLIENT_KEY
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY OPENCODE_API_KEY
test -n "${AIHUB_CLIENT_KEY:-}"
WORKDIR="$(mktemp -d)"
printf 'Reply with exactly E07_BASIC_OK and no other text.\n' |
  (cd "$WORKDIR" && HOME="$CODEX_HOME" codex exec --sandbox workspace-write --skip-git-repo-check -)
unset AIHUB_CLIENT_KEY
rm -r -- "$CODEX_HOME" "$WORKDIR"
```

The Codex process receives only the gateway credential; it does not inherit
the provider credential or the Hub's gateway variable. Keep sandbox and
approval behavior enabled. After the manual smoke, stop the Hub gracefully
with Ctrl-C and remove only the temporary `CODEX_HOME`, `WORKDIR`, and data
directories created for this run:

```bash
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY
rm -r -- "$AIHUB_DATA_DIR"
```

The opt-in E07 validator exercises records, probes, streaming, restart, and
before/after integrity checks in one bounded pass. Supply the provider
credential through `AIHUB_PROVIDER_OPENCODEGO_API_KEY` or the existing
`OPENCODE_API_KEY` environment variable. If `AIHUB_GATEWAY_API_KEY` is unset,
the validator generates a distinct temporary gateway key in memory and passes
it to the Hub and Codex child without printing or persisting it. Then run:

```bash
AIHUB_E07_LIVE=1 go test ./cmd/ai-hub -run '^TestE07LiveCodexOpenCodeGo$' -count=1 -v
```

The validator creates and removes isolated Codex, data, and tool-workspace
directories. Its report emits scenario statuses and aggregate measurements
only. A skipped or failed required scenario leaves E07 incomplete; do not add
a Codex support claim until every required scenario passes.

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
- [`docs/specs/E04-gateway-hardening-access-control.md`](docs/specs/E04-gateway-hardening-access-control.md) — E04 spec and decisions.
- [`docs/design/E04-gateway-hardening-access-control-design.md`](docs/design/E04-gateway-hardening-access-control-design.md) — E04 technical design.
- [`docs/specs/E05-execution.md`](docs/specs/E05-execution.md) — E05 execution lifecycle spec.
- [`docs/specs/E06-execution-records.md`](docs/specs/E06-execution-records.md) — E06 spec and decisions.
- [`docs/design/E06-execution-records-design.md`](docs/design/E06-execution-records-design.md) — E06 technical design.
- [`docs/specs/E07-codex-first-harness-integration.md`](docs/specs/E07-codex-first-harness-integration.md) — approved E07 scope and acceptance criteria.
- [`docs/design/E07-codex-first-harness-integration-design.md`](docs/design/E07-codex-first-harness-integration-design.md) — E07 validation design.
- [`docs/validation/E07-codex-first-harness-integration.md`](docs/validation/E07-codex-first-harness-integration.md) — incomplete E07 scenario evidence.
- [`docs/discovery/E06-discovery.md`](docs/discovery/E06-discovery.md) — E06 discovery evidence.
- [`docs/research/opencode-go-protocol-spike.md`](docs/research/opencode-go-protocol-spike.md) — validated integration evidence.
- [`docs/adr/`](docs/adr/) — architecture decision records (including ADR-009 through ADR-013).

## Known limitations

- The gateway uses one static client key. Key rotation, multiple concurrent
  keys and a secret manager are not implemented (E04 D15).
- TLS termination, rate limiting and CORS preflight are out of E04 scope. A
  non-loopback listener requires explicit configuration and must run behind a
  trusted network, VPN or TLS-terminating proxy.
- Provider sessions are per-request in E02 and are not stored in E06 records;
  cross-turn continuity of generated `x-opencode-session` values depends on a
  future AI Hub context/session layer.
- No retries, fallback or quota enforcement (E05/E09).
- Execution records are at-most-once and best-effort: queue overflow, storage
  failure, crash or flush expiry loses history, and `usage` stays `null` in
  schema v1.
- No persistence beyond the catalog snapshot and execution-record segments;
  readiness has no dependency checks registered yet.
- The catalog is derived data: it reflects the last accepted sync, not a live
  execution probe. Synchronization runs at startup (healthy provider only) and
  on demand; there is no periodic refresh. One process owns a data directory.
- A successful catalog sync records `upstream_status` as 200; non-200 2xx
  variants from the models endpoint are not propagated in E03.
- No metrics or OpenTelemetry export (deferred beyond E06; E06 exposes only
  process-local degradation counters).
- `X-Request-Id` is the only correlation header accepted.
- No CI pipeline and no container image.
