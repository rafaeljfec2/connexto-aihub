# Cloudflare Tunnel — aihub.connexto.com.br (Cursor BYOK)

Public ingress for Cursor OpenAI Base URL override. Tunnel `aihub-cursor` (`b1f35f39-5627-438d-846d-862b85781d73`) is **remotely managed** (`config_src=cloudflare`).

## Allowed paths (ingress)

| Path pattern | Purpose |
| --- | --- |
| `^/v1/(models\|responses\|chat/completions)/?$` | Models list, Codex Responses, Cursor Chat Completions adapter |
| catch-all `http_status:404` | Deny everything else |

Reference copy: `~/.cloudflared/aihub-cursor.yml`.

## Deploy checklist (backend + tunnel)

1. Rebuild and recreate the hub backend:
   ```bash
   docker compose build backend
   docker compose up -d backend --force-recreate
   ```
2. Ensure `AIHUB_GATEWAY_ALLOWED_HOSTS` includes `aihub.connexto.com.br` in `.env`.
3. Update tunnel ingress in Cloudflare when adding hub routes (API `PUT /accounts/{account_id}/cfd_tunnel/{tunnel_id}/configurations`).
4. Connector: `systemctl --user status cloudflared-aihub-cursor.service` (token `~/.cloudflared/aihub-cursor.token`).

## Cursor settings

- Base URL: `https://aihub.connexto.com.br/v1`
- Model: `aihub/auto`
- API key: gateway client key (`AIHUB_GATEWAY_API_KEY`), not cursor.com

## Security

- Do not put Cloudflare Access on this hostname (breaks Bearer auth from Cursor).
- Rotate tunnel token if leaked.
