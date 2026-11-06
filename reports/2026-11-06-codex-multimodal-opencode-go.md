# Relatório: multimodal Codex → AI Hub → OpenCode Go (gpt-6-luna)

Date: 2026-11-06

## Resumo

- **Causa raiz:** OpenCode Go rejeita `input[]` com itens `input_text` / `input_image` no top-level (`400 invalid_value`); exige `type: message` com `content[]`.
- **Correção:** normalizador no hub (`internal/gateway/opencode_input_normalize.go`) + catálogo E03 A2 (`image_input`).

## Reprodução em camadas (2026-11-06, hub reiniciado com build local)

| Camada | Payload flat 1×1 PNG | HTTP | Resultado |
| --- | --- | --- | --- |
| OpenCode direto (baseline, sessão anterior) | flat | 400 | `input_text` inválido no top-level |
| OpenCode direto | message wrapper | 200 | descreve pixel |
| AI Hub `:8080` (pós-normalizer) | flat via Codex-style | **200** `completed` | assistente: **Black** |
| Codex smoke | — | manual | usar mesmo `config.toml` aihub; esperado igual ao curl via hub |

## Catálogo pós-sync

```text
GET /v1/catalog/models/opencode-go/gpt-6-luna
capabilities.image_input.state = supported
```

## Operação

1. Deploy build com normalizer + E03 A2.
2. `POST /v1/catalog/sync`.
3. Codex: `AIHUB_GATEWAY_API_KEY` exportada; `base_url = http://127.0.0.1:8080/v1`.

Detalhes em inglês: [docs/validation/E02-codex-multimodal-via-aihub.md](../docs/validation/E02-codex-multimodal-via-aihub.md).
