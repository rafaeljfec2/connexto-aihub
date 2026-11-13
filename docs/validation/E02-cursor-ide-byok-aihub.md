# Cursor IDE — OpenAI Base URL override with AI Hub

Status: **configuration guidance** (no hub code change in this record).

Report date: 2026-11-13.

## Context

Cursor can route **custom OpenAI-format models** through **Models → API Keys → Override OpenAI Base URL**. That path speaks **Chat Completions** (`POST …/chat/completions`), not the Responses wire API.

AI Hub’s public gateway (Go) exposes **Responses** at `POST /v1/responses`, **Chat Completions** at `POST /v1/chat/completions` (adapter → Responses pipeline), and **models** at `GET /v1/models` (`apps/backend/cmd/ai-hub/gateway_routes.go`). The Node **Connexto Proxy** also implements Chat Completions for local Cursor SDK execution (`apps/proxy/apps/api/src/routes/chat-completions.ts`).

Codex is the **supported** harness: `wire_api = "responses"` and `base_url = "…/v1"` (see repository `README.md`).

## Symptom (Cursor UI)

- Toast: **Unable to reach the model provider**
- Body shape: `{"error":{"type":"client","reason":"invalid_input","message":"…"}}`
- HTTP **400** with a Cursor **Request ID** (client envelope, not AI Hub’s `{"error":{"type":"invalid_request",…}}`).

Common truncated message prefix: `Invalid i…` (full text only visible via **Copy Request** in the toast).

Typical misconfiguration from screenshots:

- Base URL **without** `/v1` (Cursor appends `/chat/completions` to the base).
- **Agent** mode with BYOK (known Cursor issues dropping or mis-shaping the `model` field).
- Expecting **Agent** tools/streaming parity with Codex Responses on the hub edge.

## Required Cursor settings

| Setting | Value |
| --- | --- |
| **Override OpenAI Base URL** | `https://<your-hub-host>/v1` — API root only; **do not** paste `/v1/chat/completions`. |
| **OpenAI API Key** | AI Hub **gateway client key** (`Authorization: Bearer …`), same family as `AIHUB_CLIENT_KEY` for Codex — **not** a Cursor.com API key. |
| **Custom model** | `aihub/auto` (must appear in the chat model picker). |
| **Chat mode** | Prefer **Ask** for first BYOK smoke test; **Agent** has separate BYOK bugs in some Cursor versions. |

Models that stay on Cursor infrastructure (Composer, Claude, Grok, Tab, Apply) **ignore** this override. Only the selected custom OpenAI-key model uses the hub base URL.

## Smoke tests (operator)

Replace `HUB` and `KEY` with your deployment.

### 1. Models list (hub)

```bash
curl -sS -H "Authorization: Bearer $KEY" "https://HUB/v1/models" | head -c 2000
```

Expect `aihub/auto` in the JSON payload (aggregated catalog).

### 2. Responses (supported harness contract)

```bash
curl -sS -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"model":"aihub/auto","input":"oi"}' \
  "https://HUB/v1/responses" | head -c 2000
```

Expect HTTP **2xx** when the gateway and upstream providers are healthy.

### 3. Chat Completions (what Cursor’s override calls)

```bash
curl -sS -o /tmp/cc-body.txt -w "%{http_code}\n" \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"model":"aihub/auto","messages":[{"role":"user","content":"oi"}]}' \
  "https://HUB/v1/chat/completions"
cat /tmp/cc-body.txt | head -c 500
```

On the **Go hub edge**, this route is implemented by the Chat Completions adapter (`apps/backend/internal/gateway/chat_completions_handler.go`). Expect **401** without a valid gateway key, or **2xx** when providers are healthy.

### 4. Wrong base URL (reproduces path bugs)

```bash
curl -sS -o /dev/null -w "%{http_code}\n" -X POST \
  -H "Content-Type: application/json" \
  -d '{}' "https://HUB/chat/completions"
```

Using `https://HUB` (no `/v1`) in Cursor produces this class of path.

## Cursor in-app check

1. Apply settings above; restart Cursor if the custom model list does not refresh.
2. Select **`aihub/auto`** in the chat model dropdown (not **Auto**).
3. Switch to **Ask** mode; send `oi`.
4. If it fails, use **Copy Request** on the toast and capture the full `message` and Request ID.
5. Correlate server-side only if the request reached the hub (gateway access logs / generations).

## Limitations and follow-up engineering

| Topic | Notes |
| --- | --- |
| Chat Completions on hub | `POST /v1/chat/completions` adapts to Responses; streaming SSE is translated to Chat chunks. |
| Agent + BYOK | Cursor may return `invalid_input` / `model is required` even when curl to the provider works; prefer Ask or wait for Cursor fixes. |
| `aihub/auto` | Routed via hub intelligent selector on **Responses**; Chat shim must map messages → Responses if added later. |

## References

- Hub routes: `apps/backend/cmd/ai-hub/gateway_routes.go`
- Hub errors: `apps/backend/internal/gateway/errors.go`
- Proxy Chat Completions: `apps/proxy/apps/api/src/routes/chat-completions.ts`
- Codex harness: `README.md` (`wire_api = "responses"`)
- Homologation report for `aihub/auto`: `reports/auto-homolog-2026-11-12/RELATORIO.md`
