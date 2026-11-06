# E02 — Codex multimodal (image) via AI Hub and OpenCode Go

Date: 2026-11-06 | Related: [E02-codex-opencode-go-e2e.md](./E02-codex-opencode-go-e2e.md), SPEC E03 amendment A2.

This document records reproducible evidence and operational checks for **vision**
requests when Codex uses `model_provider = "aihub"` and a Go model such as
`gpt-6-luna`.

## Typical Codex configuration

```toml
model = "gpt-6-luna"
model_provider = "aihub"

[model_providers.aihub]
name = "Connexto AI Hub"
base_url = "http://127.0.0.1:8080/v1"
wire_api = "responses"
env_key = "AIHUB_GATEWAY_API_KEY"
```

The shell that launches Codex must export `AIHUB_GATEWAY_API_KEY` with the same
value as `AIHUB_GATEWAY_API_KEY` in the hub `.env`. The hub holds the real
OpenCode credential in `AIHUB_PROVIDER_OPENCODEGO_API_KEY` (or equivalent).

Hub OpenCode base URL (not the inactive Codex `[model_providers.opencode-go]`
block):

```bash
AIHUB_PROVIDER_OPENCODEGO_BASE_URL=https://opencode.ai/inference/go
```

The adapter posts to `{base}/openai/v1/responses`.

## Failure modes (measured)

| Symptom | HTTP | Cause |
| --- | --- | --- |
| Routing before upstream | **503** `routing_unavailable` | Catalog missing `image_input` for the selected model (E08). Fix: E03 A2 enrichment + `POST /v1/catalog/sync`. |
| Upstream rejects body | **400** `invalid_value` on `input[0].type` = `input_text` | OpenCode Go expects `input[]` items of type `message` (or `role` + `content`), not flat `input_text` / `input_image` at the top level. Fix: hub normalizes payload for `opencode-go` only (`internal/gateway/opencode_input_normalize.go`). |
| Model answers without vision | **200** but generic “cannot see image” | Often Codex chat UI or payload without base64 in API; verify with curl below. |

### Reproduction (flat vs message)

Minimal 1×1 PNG base64 (`iVBORw0KGgo...`):

**Flat input (fails on OpenCode and on hub without normalizer):**

```json
"input": [
  {"type": "input_text", "text": "What color is the pixel?"},
  {"type": "input_image", "image_url": "data:image/png;base64,..."}
]
```

→ **400** — `Invalid value: 'input_text'. Supported values are: ... 'message' ...`

**Message wrapper (succeeds):**

```json
"input": [
  {
    "type": "message",
    "role": "user",
    "content": [
      {"type": "input_text", "text": "What color is the pixel?"},
      {"type": "input_image", "image_url": "data:image/png;base64,..."}
    ]
  }
]
```

→ **200** — assistant describes the pixel color (vision path exercised).

Codex often sends the **message** shape for HTTP Responses; some clients and
tests use the **flat** shape — the hub adapter covers the flat case for
`opencode-go`.

## Post-upgrade checklist

1. Deploy hub build including E03 A2 and OpenCode input normalization.
2. Restart the hub process.
3. Sync catalog (all providers that affect routing):

   ```bash
   curl -s -X POST -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{}' http://127.0.0.1:8080/v1/catalog/sync
   ```

4. Verify vision capability:

   ```bash
   curl -s -H "Authorization: Bearer $AIHUB_GATEWAY_API_KEY" \
     "http://127.0.0.1:8080/v1/catalog/models/opencode-go/gpt-6-luna" \
     | jq '.data.capabilities.image_input'
   ```

5. Smoke POST via hub with flat multimodal body; expect **200** and a descriptive
   assistant message.

6. **`GET /v1/models`** (no header) should list models from every enabled provider
   (OpenCode, Cursor via proxy, OpenRouter, etc.). Use `X-AIHub-Provider: cursor`
   only when you need the proxy's native list body unchanged.

## Cursor / Connexto Proxy path (backlog)

If `model_provider` targets **cursor** via the Connexto Proxy (`9090`), catalog
E03 A2 defaults apply, but the proxy HTTP layer must still forward parsed images
to the Cursor SDK (see `connexto-proxy-cli-llms` — `prepareResponsesRequest`
sets `prompt.images` while routes historically passed `message` text only).
That work is **out of scope** for the OpenCode Go path documented here.

## Screenshots in agent chat

Attaching an image in Codex/Cursor chat is not the same as a multimodal
Responses API body. For debugging, save PNGs under `reports/` or capture the
HTTP error body from the hub when testing with curl.
