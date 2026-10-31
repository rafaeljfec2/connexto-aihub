# E04 — Gateway Hardening & Access Control — Real E2E Validation

Date: 2026-10-31 | Baseline: `main @ c9444ce` plus the uncommitted E04 working
tree | Upstream: real OpenCode Go | Verdict: **E04 VALIDATED** with the
limitations listed below.

This validation exercised the real chain Codex CLI → AI Hub → OpenCode Go. No
mock upstream was used for the scenarios marked real. The project `.env` was
read by the application but was not modified. A temporary gateway key was
generated under `/tmp` with mode `0600`; neither the gateway key nor the
provider key is reproduced in this document or in repository files.

## 1. Environment

| Item | Value |
| --- | --- |
| Branch / baseline | `main` / `c9444ce` (E03), E04 changes uncommitted |
| Go | Go toolchain available on the host; `go build ./...` green |
| AI Hub loopback endpoint | `http://127.0.0.1:18080` |
| AI Hub explicit non-loopback endpoint | `0.0.0.0:18081`, exercised through `127.0.0.1:18081` |
| Codex CLI | `0.164.0-alpha.11.1` with isolated `CODEX_HOME` |
| Model | `deepseek-v4.1-flash` |
| Provider credential | Loaded from the existing project `.env`; never printed |
| Gateway credential | Temporary 48-character random key, distinct from the provider key |
| Data directory | Temporary directory outside the repository |
| Global Codex configuration | Not modified |

The first Codex invocation proved authentication before any functional scenario:
the isolated provider config used `env_key = "AIHUB_CLIENT_KEY"`, and that
variable carried the gateway key. Codex received `E04-CODEX-OK` from the real
upstream.

## 2. Access-control matrix

| Check | Result | Evidence |
| --- | --- | --- |
| `GET /healthz` without auth | **PASS** — `200` | Liveness remains unauthenticated for exact GET |
| `GET /readyz` without auth | **PASS** — `200` | Readiness remains unauthenticated for exact GET |
| `HEAD /healthz` without auth | **PASS** — `401` | HEAD is not authentication-exempt |
| `GET /v1/models` without auth | **PASS** — `401` | Local envelope `unauthorized`, `WWW-Authenticate`, `Cache-Control: no-store` |
| `GET /v1/models` with gateway key | **PASS** — `200` | 28 provider models returned |
| Provider key used as gateway key | **PASS** — `401` | Distinct credentials are enforced at runtime |
| Startup with provider key reused as gateway key | **PASS** — exit `1` | Error says keys must differ and does not print either value |
| Startup with a too-short gateway key | **PASS** — exit `1` | Error identifies `gateway.api_key` without echoing the value |
| Disallowed `Host` | **PASS** — `403` | `evil.test` rejected; body does not echo it |
| Disallowed `Origin` | **PASS** — `403` | `https://evil.test` rejected |
| Gateway key absent from response bodies | **PASS** | Scanned successful and rejected bodies |
| Gateway key absent from access logs | **PASS** | Logs contain method/path/status/bytes/duration/correlation/remote only |
| `GET /v1/catalog/models` with auth | **PASS** — `200` | 28 catalog entries; E03 response shape unchanged |

## 3. Request admission

| Check | Result |
| --- | --- |
| `POST /v1/responses` without `Content-Type` | **PASS** — `415 unsupported_media_type` |
| Body above 32 MiB | **PASS** — `413 payload_too_large`, rejected before upstream |
| Invalid `x-opencode-session` | **PASS** — `400 invalid_request` |
| Valid JSON request | **PASS** — `200`, provider response completed |
| Upstream error passthrough | **PASS** — unknown model preserved the upstream `400` envelope |
| Gateway key leaked in a successful or upstream-error body | **PASS** — no leak |

The generated session is not echoed in responses; automated integration tests
observe it on the upstream request. The E2E run proved the admission path did
not alter accepted bodies or upstream status/body semantics.

## 4. Real Responses and streaming

A non-streaming request returned `200` with `status:"completed"` and an output
payload. A streaming request returned `200` with
`Content-Type: text/event-stream` and included `response.completed`.

A measured streaming run produced 100 non-empty record lines over a 697 ms
window, with 92 distinct millisecond arrival buckets. This is direct evidence
that SSE arrived progressively through the E04 middleware rather than as one
buffered response. The gateway `Flush`/`Unwrap` preservation is also covered by
`internal/server/access_streaming_test.go`.

## 5. Codex CLI scenarios

| Scenario | Result | Notes |
| --- | --- | --- |
| Basic generation | **PASS** | `codex exec` returned exactly `E04-CODEX-OK` through the authenticated gateway |
| Multi-turn context | **PASS** | Turn 1 stored `E04-VIOLET`; explicit session resume returned exactly `E04-VIOLET` |
| Tool calling | **PASS** | Codex created `note.txt` containing `e04 works`, read it back and replied with the contents |
| Streaming at the Codex hop | **NOT INSTRUMENTED** | `codex exec` does not expose per-chunk rendering; progressive SSE was measured at the gateway hop instead |

The multi-turn run used the explicit session ID. In this Codex build,
`codex exec resume --last` selected a new session and therefore did not recall
the codeword. Resuming the recorded session ID worked. This is a Codex CLI
selection behaviour, not an AI Hub or provider failure.

## 6. Listener exposure

| Check | Result |
| --- | --- |
| Non-loopback bind without `AIHUB_GATEWAY_ALLOWED_HOSTS` | **PASS** — startup exit `1`, error identifies `allowed_hosts` |
| Explicit `0.0.0.0` plus allowed hosts, exercised from loopback | **PASS** — `GET /healthz` `200`; authenticated `GET /v1/models` `200` |
| Loopback default | **PASS** — primary run bound `127.0.0.1:18080` |

The post-bind actual-address check is additionally covered by
`internal/server/access_listener_test.go`, including wildcard, IPv6 and
unsupported address types. The E2E safe-exposure run was local only; the
listener was not exposed to an external network.

## 7. Quality gates

| Gate | Result |
| --- | --- |
| `gofmt -l .` | **PASS** — no output |
| `go test ./... -count=1` | **PASS** |
| `go test ./... -count=1 -race` | **PASS** |
| `go vet ./...` | **PASS** |
| `staticcheck ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** — 0 issues |
| `go build ./...` | **PASS** |
| Coverage, `internal/access` | **91.2%** |
| Coverage, `internal/server` | **89.5%** |
| Coverage, `internal/config` | **88.5%** |
| Pre-existing E02/E03 coverage | Unchanged or improved; gateway 93.3%, catalog 90.6% |

## 8. Reproduction outline

The run used process-only secrets and an isolated temporary environment. The
commands below show the shape without reproducing any credential:

```bash
GATEWAY_KEY="$(openssl rand -hex 24)"          # shell variable only
AIHUB_GATEWAY_API_KEY="$GATEWAY_KEY" \
AIHUB_SERVER_PORT=18080 \
AIHUB_DATA_DIR="<temporary-data-dir>" \
  go run ./cmd/ai-hub -env-file .env

curl -i http://127.0.0.1:18080/healthz
curl -i -H "Authorization: Bearer $GATEWAY_KEY" \
  http://127.0.0.1:18080/v1/models
curl -i -H "Authorization: Bearer $GATEWAY_KEY" \
  -H 'Content-Type: application/json' \
  --data-binary @request.json \
  http://127.0.0.1:18080/v1/responses
```

Codex used an isolated `CODEX_HOME` whose `config.toml` contained only the
loopback base URL, `wire_api = "responses"` and `env_key = "AIHUB_CLIENT_KEY"`.
The variable was exported from the temporary gateway key for the duration of the
command.

## 9. Review remediation and revalidation

Two independent `senior-code-review-judge` rounds returned
`APPROVED_WITH_NOTES`. The first-round findings were corrected before the final
gates:

| Finding | Correction | Revalidation |
| --- | --- | --- |
| Public placeholder gateway key in `.env.example` | Replaced with an empty value plus generation guidance; README/SPEC examples aligned | Startup now fails closed until the operator sets a private key |
| Empty or signed authority components accepted | Origin rejects empty port/query/fragment; request Host rejects signed ports; request-level tests added | Malformed Origin values returned `403`; `Host: 127.0.0.1:+18080` returned `403` |
| Session-generation failure used a 400 error type at status 500 | Added a dedicated `internal_error` envelope and an injectable generator seam with a failure test | `go test ./internal/access/...` green; failure path returns `500` without leaking details |

After remediation, the full quality-gate suite was rerun, the real upstream
Responses/SSE/catalog checks remained green, and Codex returned
`E04-REVALIDATED` through the authenticated gateway. No commit or push was made.

The second review round identified three additional notes. They were remediated
as follows:

| Finding | Correction | Revalidation |
| --- | --- | --- |
| Provider live smoke required the gateway key despite being documented as provider-only | The test now builds the provider configuration directly from the provider environment | Static suite green; the opt-in smoke no longer calls `config.Load()` |
| Architecture overstated gateway key storage | The document now distinguishes the runtime SHA-256 digest from the original key retained in process configuration/environment | Documentation reviewed against `cmd/ai-hub/main.go` and `internal/access/access.go` |
| SPEC/design phase metadata was stale | Phase/status metadata and ADR references were updated to the implemented/remediated state | Documentation reviewed against the current E04 deliverables |

These changes are limited to the opt-in smoke test and documentation. The
deterministic quality-gate suite was rerun; the real upstream E2E from the first
remediation was not repeated because no runtime code changed.

No review findings remain open after this remediation.

## 10. Scope boundaries confirmed

- TLS termination, rate limiting, key rotation and CORS/preflight were not
  implemented or claimed.
- Non-loopback exposure requires explicit configuration and remains an
  operational responsibility (trusted network, VPN or TLS-terminating proxy).
- No E03 technical-debt refactoring was performed.
- No commit or push was made; LIM-03 remains in force.
- The catalog remained derived data; the run observed 28 entries without a
  semantic change to E03 listing behaviour.

## 11. Verdict

E04 is validated end-to-end against the real OpenCode Go upstream and Codex CLI,
including gateway authentication, Host/Origin rejection, body admission,
upstream error passthrough, SSE streaming, safe explicit exposure and unsafe
startup refusal. The only unobserved item is progressive rendering inside the
Codex CLI process itself, which that client does not expose; the gateway hop was
measured directly and the automated streaming tests remain green.
