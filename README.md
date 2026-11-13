# AI Hub

Local-first Personal AI Orchestrator. The product and architecture baseline is
documented in [`AI-Hub-Product-Architecture-Kickoff.md`](AI-Hub-Product-Architecture-Kickoff.md).

**Current status:** E01–E07, the PostgreSQL foundation, C1 provider
administration, C2 multi-provider runtime, E08.1–E08.5 deterministic routing,
and E09 resource management are implemented and accepted within their recorded
scopes. E07 is validated for Codex CLI `0.164.0-alpha.11.1` on Linux
`amd64`; AC-14 is `OUT OF SCOPE`. The next authorized development
front is the Frontend MVP; no frontend app exists yet. E08.6–E08.8 remain
deferred, while E10 Learning Engine and E11 Dashboard remain future work. See
the [consolidated roadmap](docs/roadmap/AI-HUB-ROADMAP.md) and
[Frontend MVP preparation](docs/frontend/FRONTEND-MVP.md).

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
- Management Foundation Phase A (workspace foundation): pnpm 12 + Turborepo 2
  workspace prepared for the Frontend MVP, with the Go backend unchanged at the
  root and validated through native Go commands. No frontend app exists yet.

## Requirements

- Go 1.23 or newer.
- No external Go dependencies.
- Node.js 22+ with pnpm 12 is required only for the workspace tooling
  (Turborepo and planned frontend); it is not needed to build or test the
  backend.

## Workspace (pnpm + Turborepo)

The repository is a hybrid monorepo since the Management Foundation Phase A
(see [ADR-019](docs/adr/ADR-019-hybrid-monorepo-workspace-tooling.md) and the
[workspace guide](docs/development-workspace.md)). The Go backend module
(`aihub`) lives under `apps/backend` and keeps native Go commands (`go -C apps/backend ...`);
pnpm/Turborepo cover `apps/*` and `packages/*` for JS/TS packages (`apps/frontend`).

```bash
pnpm install   # JS/TS tooling (turbo); no effect on Go builds
pnpm build     # go -C apps/backend build ./...
pnpm dev       # turbo + go -C apps/backend run ./cmd/ai-hub
pnpm lint      # go -C apps/backend vet ./...
pnpm test      # go -C apps/backend test ./... -count=1 -race
pnpm check     # go -C apps/backend vet ./... && go -C apps/backend test ./... -count=1 -race && pnpm --filter @connexto/frontend check
```

## Run locally

```bash
cp .env.example .env   # then adjust values; never commit .env
# AIHUB_GATEWAY_API_KEY is mandatory and must differ from the provider key.
# Generate a private value with `openssl rand -base64 48`; .env.example leaves
# it empty intentionally so startup fails closed until the operator sets it.
go -C apps/backend run ./cmd/ai-hub
```

With an explicit env file:

```bash
go -C apps/backend run ./cmd/ai-hub -env-file /path/to/.env
```

The server binds to `127.0.0.1:8080` by default.

```bash
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/readyz

curl -i -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  http://127.0.0.1:8080/v1/models

# Single provider (native upstream body, byte-for-byte when upstream succeeds):
curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  -H "X-AIHub-Provider: cursor" \
  http://127.0.0.1:8080/v1/models
```

## Endpoints

| Method | Path                    | Purpose                                                                  |
| ------ | ----------------------- | ------------------------------------------------------------------------ |
| GET    | `/healthz`              | Liveness. Process is running; no dependency is evaluated.                  |
| GET    | `/readyz`               | Readiness. Runs registered checks; `503` when any fails.                   |
| POST   | `/v1/responses`         | Responses pass-through to OpenCode Go (E02; `503` when provider disabled). |
| GET    | `/v1/models`            | Aggregated OpenAI-style model list from every enabled bound provider; optional `X-AIHub-Provider` scopes to one upstream passthrough. |
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
| `AIHUB_SERVER_READ_TIMEOUT`            | `2m`                                 |
| `AIHUB_SERVER_WRITE_TIMEOUT`           | `10m`                                |
| `AIHUB_SERVER_IDLE_TIMEOUT`            | `10m`                                |
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
| `AIHUB_DATABASE_URL`                   | *(empty — no database)*              |
| `AIHUB_DATABASE_MAX_CONNS`             | `4`                                  |
| `AIHUB_DATABASE_MIN_CONNS`             | `0`                                  |
| `AIHUB_DATABASE_CONNECT_TIMEOUT`       | `5s`                                 |
| `AIHUB_CATALOG_PERSISTENCE`            | `postgres` when a database URL is set, `file` otherwise |
| `AIHUB_GATEWAY_API_KEY`                | *(required, no default)*             |
| `AIHUB_GATEWAY_ALLOWED_HOSTS`          | loopback hosts when unset            |
| `AIHUB_GATEWAY_ALLOWED_ORIGINS`        | *(empty — no browser Origin allowed)*|
| `AIHUB_GATEWAY_MAX_BODY_BYTES`         | `33554432` (32 MiB)                  |

Durations use Go syntax (`250ms`, `15s`, `1m30s`). See [.env.example](.env.example).

Database and catalog persistence (Phase B, ADR-020):

- `AIHUB_DATABASE_URL` accepts URL-form PostgreSQL DSNs only
  (`postgres://` or `postgresql://` with a non-empty host) and is redacted in
  logs and errors. When it is empty the server runs local-first with the JSON
  snapshot file of ADR-010 and PostgreSQL is never touched.
- When a database URL is configured, startup connects, migrates the embedded
  schema and pings; any failure aborts the process. The catalog snapshot is
  then persisted in PostgreSQL (full-replace per sync) instead of
  `catalog.json`, and `/readyz` gains a `database` check.
- `AIHUB_CATALOG_PERSISTENCE` selects the mode explicitly (`file` or
  `postgres`); `postgres` requires a database URL.
- `AIHUB_DATABASE_MAX_CONNS` is 1–64, `AIHUB_DATABASE_MIN_CONNS` is 0–32 and
  must stay below `MAX_CONNS`, and `AIHUB_DATABASE_CONNECT_TIMEOUT` ranges
  from 100 ms to 60 s.

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
`supported` only when the provider declares it (including bare `image`/`audio`
tokens and SPEC A2 ingest rules in §5.4), and absence is `unknown`, never
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

After a hub upgrade that changes modality normalization (SPEC E03 amendment A2),
re-sync catalog providers that feed routing (`cursor`, `openrouter`,
`opencode-go`) and spot-check vision models, for example:

```bash
curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
  "http://127.0.0.1:8080/v1/catalog/models/cursor/composer-2.5" | jq '.capabilities.image_input'
```

Connexto Proxy operators should send `modalities.input` for text-only models so
the hub does not apply cursor default vision modalities (see SPEC §5.4).

`GET /v1/models` without `X-AIHub-Provider` merges each enabled provider's native
`data[]` entries into one OpenAI-style list (`object: list`). Upstream failures
for one provider are skipped so other providers still appear. With an explicit
selector, the handler passthroughs that provider's native response unchanged.
For normalized capability metadata, use `GET /v1/catalog/models` (E03).

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
  go -C apps/backend test ./internal/providers/ -run TestLiveOpenCodeGoSmoke -v
```

It is skipped automatically without both variables. The provider-only smoke
also honors `AIHUB_PROVIDER_OPENCODEGO_BASE_URL` when set and does not require
the gateway API key; mock-based green tests do not count as real integration
success.

See also the [Codex E07 report](docs/validation/E07-codex-first-harness-integration.md).

Cursor IDE users who enable **Override OpenAI Base URL** must use the hub API root (`https://<host>/v1`), the gateway client key (not a Cursor.com key), model `aihub/auto`, and prefer **Ask** mode for BYOK smoke tests. The Go hub exposes `POST /v1/chat/completions` (adapter) and `POST /v1/responses` (Codex and other harnesses). See [Cursor IDE BYOK with AI Hub](docs/validation/E02-cursor-ide-byok-aihub.md).

### Codex CLI support (validated release and environment)

E07 validated Codex CLI `0.164.0-alpha.11.1` on Linux `amd64` as the AI Hub's
first supported harness, using the Responses wire API and OpenCode Go through
the AI Hub. The tested model was `deepseek-v4.1-flash`. This support claim is
limited to that exact CLI release and environment class; it does not imply
support for other Codex releases or harnesses. Gateway-hop streaming passed;
Codex stdout progressive rendering remains `NOT VERIFIED`. See the
[E07 validation report](docs/validation/E07-codex-first-harness-integration.md)
for scenario evidence and limitations.

Prerequisites are Bash, Python 3, Go 1.23, bubblewrap, readable Linux `/proc`,
the recorded Codex CLI release, a newly provisioned OpenCode Go provider
credential, and the selected model. This manual smoke uses separate terminal
sessions. It does not edit `~/.codex`, project `.env` files, or tracked Codex
configuration.

Provision new, distinct gateway and provider credentials before the run; do
not reuse existing environment values. The commands below use Bash's
non-echoing `read -s` prompt in separate terminal sessions, a method permitted
by the E07 plan. Build the Hub binary before entering either credential. The
Hub terminal exports the two credentials only inside a sanitized subshell that
is replaced by the Hub process; the Codex terminal receives only the gateway
credential as `AIHUB_CLIENT_KEY`.

```bash
AIHUB_E07_BUILD_DIR="$(mktemp -d)"
AIHUB_E07_BINARY="$AIHUB_E07_BUILD_DIR/ai-hub"
AIHUB_DATA_DIR="$(mktemp -d)"
export AIHUB_E07_BUILD_DIR AIHUB_E07_BINARY AIHUB_DATA_DIR
mkdir -m 700 -p "$AIHUB_E07_BUILD_DIR/home" "$AIHUB_E07_BUILD_DIR/tmp"
env -i PATH="$PATH" HOME="$AIHUB_E07_BUILD_DIR/home" \
  GOPATH="$AIHUB_E07_BUILD_DIR/go" \
  GOMODCACHE="$AIHUB_E07_BUILD_DIR/go/pkg/mod" \
  GOCACHE="$AIHUB_E07_BUILD_DIR/cache" \
  go -C apps/backend build -o "$AIHUB_E07_BINARY" ./cmd/ai-hub
```

In Terminal A, enter the credentials without echo. The Hub receives only the
allowlisted runtime variables and its two credentials; the values are never
arguments, shell-history entries, or files:

```bash
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY AIHUB_CLIENT_KEY OPENCODE_API_KEY
read -r -s -p 'Gateway API key: ' AIHUB_GATEWAY_API_KEY; printf '\n'
read -r -s -p 'OpenCode Go provider key: ' AIHUB_PROVIDER_OPENCODEGO_API_KEY; printf '\n'
[[ -n "$AIHUB_PROVIDER_OPENCODEGO_API_KEY" && \
   ${#AIHUB_GATEWAY_API_KEY} -ge 32 && ${#AIHUB_GATEWAY_API_KEY} -le 512 && \
   "$AIHUB_GATEWAY_API_KEY" =~ ^[A-Za-z0-9._~+/-]+=*$ && \
   "$AIHUB_GATEWAY_API_KEY" != "$AIHUB_PROVIDER_OPENCODEGO_API_KEY" ]] || {
  printf 'Credential preflight failed.\n' >&2
  unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY
  exit 1
}
E07_HUB_GATE="$AIHUB_E07_BUILD_DIR/hub-launch-gate"
E07_HUB_READY="$AIHUB_E07_BUILD_DIR/hub-ready"
mkfifo "$E07_HUB_GATE"
(
  e07_path="$PATH"
  e07_home="$AIHUB_E07_BUILD_DIR/home"
  e07_tmpdir="$AIHUB_E07_BUILD_DIR/tmp"
  e07_binary="$AIHUB_E07_BINARY"
  e07_data_dir="$AIHUB_DATA_DIR"
  while IFS= read -r e07_name; do
    case "$e07_name" in
      PATH|HOME|TMPDIR|AIHUB_DATA_DIR|AIHUB_GATEWAY_API_KEY|AIHUB_PROVIDER_OPENCODEGO_API_KEY) ;;
      *) unset "$e07_name" ;;
    esac
  done < <(compgen -e)
  export PATH="$e07_path" HOME="$e07_home" TMPDIR="$e07_tmpdir" \
    AIHUB_DATA_DIR="$e07_data_dir" AIHUB_GATEWAY_API_KEY \
    AIHUB_PROVIDER_OPENCODEGO_API_KEY
  exec bash --noprofile --norc -c \
    ': > "$1"; IFS= read -r -t 30 _ < "$2" || exit 1; exec "$3" -env-file /dev/null' \
    e07-hub-launch "$E07_HUB_READY" "$E07_HUB_GATE" "$e07_binary"
) &
AIHUB_E07_HUB_PID=$!
for e07_attempt in {1..100}; do
  [[ -f "$E07_HUB_READY" ]] && break
  sleep 0.05
done
if [[ ! -f "$E07_HUB_READY" ]]; then
  kill -TERM "$AIHUB_E07_HUB_PID"
  wait "$AIHUB_E07_HUB_PID"
  rm -f -- "$E07_HUB_GATE" "$E07_HUB_READY"
  unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY \
    AIHUB_E07_HUB_PID E07_HUB_GATE E07_HUB_READY
  exit 1
fi
if ! env -i PATH="$PATH" HOME="$AIHUB_E07_BUILD_DIR/home" \
  python3 - "$AIHUB_E07_HUB_PID" <<'PY'
from pathlib import Path
import sys

pid = sys.argv[1]
try:
    entries = Path(f"/proc/{pid}/environ").read_bytes().split(b"\0")
except OSError:
    print("AIHUB_GATEWAY_API_KEY=NOT_VERIFIABLE")
    print("AIHUB_PROVIDER_OPENCODEGO_API_KEY=NOT_VERIFIABLE")
    print("AIHUB_CLIENT_KEY=NOT_VERIFIABLE")
    print("OPENCODE_API_KEY=NOT_VERIFIABLE")
    raise SystemExit(2)
names = {entry.partition(b"=")[0] for entry in entries if b"=" in entry}
expected = {
    "AIHUB_GATEWAY_API_KEY": True,
    "AIHUB_PROVIDER_OPENCODEGO_API_KEY": True,
    "AIHUB_CLIENT_KEY": False,
    "OPENCODE_API_KEY": False,
}
passed = True
for name, should_exist in expected.items():
    present = name.encode() in names
    status = "PRESENT" if present else "ABSENT"
    print(f"{name}={status}")
    passed = passed and present == should_exist
raise SystemExit(0 if passed else 1)
PY
then
  kill -TERM "$AIHUB_E07_HUB_PID"
  wait "$AIHUB_E07_HUB_PID"
  rm -f -- "$E07_HUB_GATE" "$E07_HUB_READY"
  unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY \
    AIHUB_E07_HUB_PID E07_HUB_GATE E07_HUB_READY
  exit 1
fi
rm -f -- "$E07_HUB_READY"
printf 'launch\n' > "$E07_HUB_GATE"
rm -f -- "$E07_HUB_GATE"
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY
```

The PID check runs before the launch gate opens, so it verifies the exact
allowlisted environment before the Hub starts. The wrapper's PID and
environment remain unchanged by `exec`, so that PID becomes the Hub process.
The check reads `/proc/<pid>/environ` and prints only `PRESENT`/`ABSENT`; no
provider request is made. Keep Terminal A separate from Terminal B.

The gateway key must be 32–512 characters in E04's `b64token` grammar and
must differ from the provider key. Do not place either value in a command,
Codex TOML, or repository file.

In Terminal B, enter the same gateway credential without echo. Do not enter
the provider credential in this terminal. Keep the gateway variable unexported
in the parent shell; the existing environment allowlist exports it only to the
Codex process and excludes it from Codex shell tools. Create `CODEX_HOME`
before writing its configuration. The provider uses the local Responses
endpoint and reads the gateway key from the environment variable named by
`env_key`. Codex state, SQLite state, XDG paths, and temporary files stay under
this temporary home:

```bash
unset AIHUB_CLIENT_KEY AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY OPENCODE_API_KEY
read -r -s -p 'Gateway API key for Codex: ' AIHUB_CLIENT_KEY; printf '\n'
test -n "$AIHUB_CLIENT_KEY"
export CODEX_HOME="$(mktemp -d)"
export CODEX_SQLITE_HOME="$CODEX_HOME"
export TMPDIR="$CODEX_HOME/tmp"
export XDG_CONFIG_HOME="$CODEX_HOME/xdg/config"
export XDG_DATA_HOME="$CODEX_HOME/xdg/data"
export XDG_CACHE_HOME="$CODEX_HOME/xdg/cache"
export XDG_STATE_HOME="$CODEX_HOME/xdg/state"
export XDG_RUNTIME_DIR="$CODEX_HOME/xdg/runtime"
mkdir -m 700 -p "$TMPDIR" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" \
  "$XDG_CACHE_HOME" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR"
env -i PATH="$PATH" CODEX_HOME="$CODEX_HOME" TMPDIR="$TMPDIR" python3 - <<'PY'
import json
import os
from pathlib import Path

home = Path(os.environ["CODEX_HOME"])
shell_values = {
    "PATH": os.environ["PATH"],
    "HOME": str(home),
    "TMPDIR": os.environ["TMPDIR"],
}
shell_set = ", ".join(
    f"{json.dumps(name)} = {json.dumps(value)}"
    for name, value in shell_values.items()
)
config = f'''model = "deepseek-v4.1-flash"
model_provider = "aihub"
sandbox_mode = "workspace-write"

[model_providers.aihub]
name = "AI Hub OpenCode Go"
base_url = "http://127.0.0.1:8080/v1"
wire_api = "responses"
env_key = "AIHUB_CLIENT_KEY"
request_max_retries = 0
stream_max_retries = 0

[shell_environment_policy]
inherit = "none"
ignore_default_excludes = false
set = {{ {shell_set} }}
experimental_use_profile = false
'''
(home / "config.toml").write_text(config, encoding="utf-8")
os.chmod(home / "config.toml", 0o600)
PY
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY OPENCODE_API_KEY
test -n "$AIHUB_CLIENT_KEY"
WORKDIR="$(mktemp -d "$TMPDIR/e07-work.XXXXXX")"
E07_CODEX_GATE="$CODEX_HOME/launch-gate"
E07_CODEX_READY="$CODEX_HOME/codex-ready"
mkfifo "$E07_CODEX_GATE"
(
  unset e07_name e07_path e07_home e07_sqlite_home e07_tmpdir \
    e07_xdg_config e07_xdg_data e07_xdg_cache e07_xdg_state \
    e07_xdg_runtime e07_client_key
  e07_path="$PATH"
  e07_home="$CODEX_HOME"
  e07_sqlite_home="$CODEX_SQLITE_HOME"
  e07_tmpdir="$TMPDIR"
  e07_xdg_config="$XDG_CONFIG_HOME"
  e07_xdg_data="$XDG_DATA_HOME"
  e07_xdg_cache="$XDG_CACHE_HOME"
  e07_xdg_state="$XDG_STATE_HOME"
  e07_xdg_runtime="$XDG_RUNTIME_DIR"
  e07_client_key="$AIHUB_CLIENT_KEY"
  while IFS= read -r e07_name; do
    if [[ "$e07_name" != PATH ]]; then
      unset "$e07_name"
    fi
  done < <(compgen -e)
  export PATH="$e07_path" HOME="$e07_home" CODEX_HOME="$e07_home" \
    CODEX_SQLITE_HOME="$e07_sqlite_home" TMPDIR="$e07_tmpdir" \
    XDG_CONFIG_HOME="$e07_xdg_config" XDG_DATA_HOME="$e07_xdg_data" \
    XDG_CACHE_HOME="$e07_xdg_cache" XDG_STATE_HOME="$e07_xdg_state" \
    XDG_RUNTIME_DIR="$e07_xdg_runtime" AIHUB_CLIENT_KEY="$e07_client_key"
  cd "$WORKDIR"
  exec bash --noprofile --norc -c \
    'cd "$1" || exit 1; : > "$2"; IFS= read -r -t 30 _ < "$3" || exit 1; exec codex exec --sandbox workspace-write --skip-git-repo-check - <<< "Reply with exactly E07_BASIC_OK and no other text."' \
    e07-codex-launch "$WORKDIR" "$E07_CODEX_READY" "$E07_CODEX_GATE"
) &
e07_codex_pid=$!
for e07_attempt in {1..100}; do
  [[ -f "$E07_CODEX_READY" ]] && break
  sleep 0.05
done
if [[ ! -f "$E07_CODEX_READY" ]]; then
  kill -TERM "$e07_codex_pid"
  wait "$e07_codex_pid"
  rm -f -- "$E07_CODEX_GATE" "$E07_CODEX_READY"
  unset AIHUB_CLIENT_KEY e07_codex_pid E07_CODEX_GATE E07_CODEX_READY
  exit 1
fi
if ! env -i PATH="$PATH" HOME="$CODEX_HOME" \
  python3 - "$e07_codex_pid" <<'PY'
from pathlib import Path
import sys

pid = sys.argv[1]
expected = {
    "AIHUB_CLIENT_KEY": True,
    "AIHUB_GATEWAY_API_KEY": False,
    "AIHUB_PROVIDER_OPENCODEGO_API_KEY": False,
    "OPENCODE_API_KEY": False,
}
try:
    entries = Path(f"/proc/{pid}/environ").read_bytes().split(b"\0")
except OSError:
    for name in expected:
        print(f"{name}=NOT_VERIFIABLE")
    raise SystemExit(2)
names = {entry.partition(b"=")[0] for entry in entries if b"=" in entry}
passed = True
for name, should_exist in expected.items():
    present = name.encode() in names
    status = "PRESENT" if present else "ABSENT"
    print(f"{name}={status}")
    passed = passed and present == should_exist
raise SystemExit(0 if passed else 1)
PY
then
  kill -TERM "$e07_codex_pid"
  wait "$e07_codex_pid"
  rm -f -- "$E07_CODEX_GATE" "$E07_CODEX_READY"
  unset AIHUB_CLIENT_KEY e07_codex_pid E07_CODEX_GATE E07_CODEX_READY
  exit 1
fi
rm -f -- "$E07_CODEX_READY"
printf 'launch\n' > "$E07_CODEX_GATE"
wait "$e07_codex_pid"
rm -f -- "$E07_CODEX_GATE"
unset AIHUB_CLIENT_KEY e07_codex_pid E07_CODEX_GATE E07_CODEX_READY \
  CODEX_SQLITE_HOME TMPDIR XDG_CONFIG_HOME XDG_DATA_HOME \
  XDG_CACHE_HOME XDG_STATE_HOME XDG_RUNTIME_DIR
rm -r -- "$CODEX_HOME" "$WORKDIR"
unset CODEX_HOME WORKDIR
```

The Codex process receives the gateway credential and temporary runtime paths
through an explicit environment allowlist. Shell tools receive only `PATH`,
temporary `HOME`, and `TMPDIR`; neither the gateway nor provider credential is
passed to them. The Codex PID check reads `/proc/<pid>/environ` while the
process waits at a launch gate; `exec` retains that PID and environment. The
gate opens only when the expected credential states pass, before the Codex
request starts. Keep sandbox and approval behavior enabled. After the manual
smoke, stop the Hub gracefully and remove only the temporary `CODEX_HOME`,
`WORKDIR`, AI Hub build/cache, and data directories created for this run:

```bash
kill -INT "$AIHUB_E07_HUB_PID"
wait "$AIHUB_E07_HUB_PID"
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY AIHUB_E07_HUB_PID
rm -r -- "$AIHUB_E07_BUILD_DIR" "$AIHUB_DATA_DIR"
```

The opt-in E07 validator exercises records, probes, streaming, restart, and
before/after integrity checks in one bounded pass. Build its test binary in a
separate credential-free environment, using temporary home and Go caches:

```bash
AIHUB_E07_TEST_DIR="$(mktemp -d)"
AIHUB_DATA_DIR="$(mktemp -d)"
mkdir -m 700 -p "$AIHUB_E07_TEST_DIR/home" "$AIHUB_E07_TEST_DIR/tmp"
export AIHUB_E07_TEST_DIR AIHUB_DATA_DIR
env -i PATH="$PATH" HOME="$AIHUB_E07_TEST_DIR/home" \
  GOPATH="$AIHUB_E07_TEST_DIR/go" \
  GOMODCACHE="$AIHUB_E07_TEST_DIR/go/pkg/mod" \
  GOCACHE="$AIHUB_E07_TEST_DIR/cache" \
  go -C apps/backend test -c -o "$AIHUB_E07_TEST_DIR/ai-hub-e07.test" ./cmd/ai-hub/e07/live
```

In Terminal C, enter new, distinct gateway and provider credentials using
separate non-echoing prompts. Do not reuse `OPENCODE_API_KEY` or allow the
harness to generate a replacement credential. Keep the values unexported in
the parent shell, then run the prebuilt test binary in a sanitized subshell.
Its explicit environment includes `PATH` so `exec.LookPath("codex")` can find
the CLI, plus only the test flag, dedicated data directory, and credentials
needed by the harness. The harness constructs a separate Codex child
environment that excludes the provider credential. Do not put values in
command text, arguments, files, or run `go test` while credentials are present:

```bash
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY OPENCODE_API_KEY AIHUB_CLIENT_KEY
read -r -s -p 'Gateway API key: ' AIHUB_GATEWAY_API_KEY; printf '\n'
read -r -s -p 'OpenCode Go provider key: ' AIHUB_PROVIDER_OPENCODEGO_API_KEY; printf '\n'
[[ -n "$AIHUB_PROVIDER_OPENCODEGO_API_KEY" && \
   ${#AIHUB_GATEWAY_API_KEY} -ge 32 && ${#AIHUB_GATEWAY_API_KEY} -le 512 && \
   "$AIHUB_GATEWAY_API_KEY" =~ ^[A-Za-z0-9._~+/-]+=*$ && \
   "$AIHUB_GATEWAY_API_KEY" != "$AIHUB_PROVIDER_OPENCODEGO_API_KEY" ]] || {
  printf 'Credential preflight failed.\n' >&2
  unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY
  exit 1
}
(
  e07_path="$PATH"
  e07_home="$AIHUB_E07_TEST_DIR/home"
  e07_tmpdir="$AIHUB_E07_TEST_DIR/tmp"
  e07_test_binary="$AIHUB_E07_TEST_DIR/ai-hub-e07.test"
  e07_data_dir="$AIHUB_DATA_DIR"
  while IFS= read -r e07_name; do
    case "$e07_name" in
      PATH|HOME|TMPDIR|AIHUB_E07_LIVE|AIHUB_DATA_DIR|AIHUB_GATEWAY_API_KEY|AIHUB_PROVIDER_OPENCODEGO_API_KEY) ;;
      *) unset "$e07_name" ;;
    esac
  done < <(compgen -e)
  export PATH="$e07_path" HOME="$e07_home" TMPDIR="$e07_tmpdir" \
    AIHUB_E07_LIVE=1 AIHUB_DATA_DIR="$e07_data_dir" \
    AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY
  exec "$e07_test_binary" -test.run '^TestE07LiveCodexOpenCodeGo$' -test.count=1 -test.v
)
unset AIHUB_GATEWAY_API_KEY AIHUB_PROVIDER_OPENCODEGO_API_KEY
```

After the test process exits, remove the temporary test-build and data
directories from the setup shell after the credential variables have been unset:

```bash
rm -r -- "$AIHUB_E07_TEST_DIR" "$AIHUB_DATA_DIR"
```

The validator creates and removes isolated Codex, data, and tool-workspace
directories. Its report emits scenario statuses and aggregate measurements
only. A skipped or failed required scenario leaves E07 incomplete; the support
claim above applies only to the validated release and environment.

## Tests and validation

```bash
go -C apps/backend test ./... -count=1 -race
go -C apps/backend vet ./...
go -C apps/backend run honnef.co/go/tools/cmd/staticcheck ./... # or staticcheck inside apps/backend
golangci-lint run ./... # run inside apps/backend
```

The isolated Codex shell canary is opt-in because it requires the exact
validated CLI release and a global Codex tree that no ambient Codex process is
modifying. It is skipped by the default suite and fails closed when enabled:

```bash
AIHUB_E07_CANARY=1 go -C apps/backend test ./cmd/ai-hub/e07/codex \
  -run '^TestE07CodexShellEnvironmentPolicyIsEffective$' -count=1 -v
```

The real-provider validator remains separately gated by `AIHUB_E07_LIVE=1`.

## Documentation

- [`docs/roadmap/AI-HUB-ROADMAP.md`](docs/roadmap/AI-HUB-ROADMAP.md) — current
  delivery status and product roadmap.
- [`docs/frontend/FRONTEND-MVP.md`](docs/frontend/FRONTEND-MVP.md) — frontend
  preparation, backend contracts, security boundary, and open decisions.
- [`docs/development-workspace.md`](docs/development-workspace.md) — workspace
  guide (pnpm + Turborepo and how the frontend joins the pipeline).
- [`docs/architecture.md`](docs/architecture.md) — architecture baseline and
  links to later runtime increments.
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
- [`docs/validation/E07-codex-first-harness-integration.md`](docs/validation/E07-codex-first-harness-integration.md) — completed E07 scenario evidence and scoped support result.
- [`docs/validation/E01-E07-execution-history.md`](docs/validation/E01-E07-execution-history.md) — cross-epic delivery history, validation status, recorded corrections and lessons.
- [`docs/discovery/E06-discovery.md`](docs/discovery/E06-discovery.md) — E06 discovery evidence.
- [`docs/research/opencode-go-protocol-spike.md`](docs/research/opencode-go-protocol-spike.md) — validated integration evidence.
- ADRs are indexed at [`docs/adr/README.md`](docs/adr/README.md) (including
  ADR-019 — hybrid monorepo workspace tooling).

## Known limitations

- The gateway uses one static client key. Key rotation, multiple concurrent
  keys and a secret manager are not implemented (E04 D15).
- TLS termination, rate limiting and CORS preflight are out of E04 scope. A
  non-loopback listener requires explicit configuration and must run behind a
  trusted network, VPN or TLS-terminating proxy.
- Provider sessions are per-request in E02 and are not stored in E06 records;
  cross-turn continuity of generated `x-opencode-session` values depends on a
  future AI Hub context/session layer.
- Failed Responses executions are not automatically retried and do not trigger
  silent provider fallback. Resource quotas and concurrency limits are
  enforced in PostgreSQL mode (E09).
- Execution records are at-most-once and best-effort: queue overflow, storage
  failure, crash or flush expiry loses history, and `usage` stays `null` in
  schema v1.
- File persistence mode uses the catalog snapshot and execution-record
  segments. PostgreSQL mode also persists provider configuration and E09
  resource/usage state; readiness checks the database in that mode. Provider
  health does not determine application readiness.
- The catalog is derived data: it reflects the last accepted sync, not a live
  execution probe. Synchronization runs at startup (healthy provider only) and
  on demand; there is no periodic refresh. One process owns a data directory.
- A successful catalog sync records `upstream_status` as 200; non-200 2xx
  variants from the models endpoint are not propagated in E03.
- No metrics or OpenTelemetry export (deferred beyond E06; E06 exposes only
  process-local degradation counters).
- `X-Request-Id` is the only correlation header accepted.
- No CI pipeline and no container image.
