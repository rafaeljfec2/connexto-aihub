# Cursor IDE + AI Hub — erro `invalid_input` (2026-11-13)

## Resumo

O toast **Unable to reach the model provider** com `400`, `"type":"client"`, `"reason":"invalid_input"` e mensagem truncada (`Invalid i…`) vem do **cliente Cursor**, não do envelope de erro do AI Hub.

No print analisado:

- Base URL: `https://aihub.connexro.com` **sem** `/v1`
- Modelo: `aihub/auto`, modo **Agent**
- Request ID (Cursor): `34d0de95-7d82-4f0f-9751-e980e3888a9c`

## Ação imediata (configuração no Cursor)

1. **Cursor → Settings → Models → API Keys**
2. **Override OpenAI Base URL:** `https://aihub.connexro.com/v1` (só a raiz da API; não colar `/chat/completions`).
3. **OpenAI API Key:** chave **cliente do gateway** do AI Hub (a mesma família do `AIHUB_CLIENT_KEY` usada no Codex), **não** chave da API do site cursor.com.
4. Manter o modelo customizado **`aihub/auto`** ligado e selecioná-lo no seletor do chat (evitar só **Auto**).
5. Trocar o chat para modo **Ask** e enviar `oi` de novo. O modo **Agent** costuma falhar em BYOK mesmo com provedor correto (campo `model` / formato Chat Completions).

Se ainda falhar: no toast, usar **Copy Request** e guardar a mensagem **completa** (a UI corta em `Invalid i…`).

## Por que a URL importa

O Cursor monta `POST {BaseURL}/chat/completions`. Com base `https://aihub.connexro.com` o destino vira `/chat/completions` na raiz do host, fora do contrato `/v1/…` do hub.

## Contrato de API

| Cliente | API | Endpoint hub (Go) |
| --- | --- | --- |
| Codex (`wire_api = responses`) | Responses | `POST /v1/responses` — **suportado** |
| Cursor (override OpenAI) | Chat Completions | `POST /v1/chat/completions` — **implementado** (adapter → pipeline Responses) |

Homologação recente de `aihub/auto` foi feita via **Responses** (`reports/auto-homolog-2026-11-12/RELATORIO.md`). O Cursor agora pode usar o adapter Chat Completions após deploy do hub com esta alteração.

Corrigir a Base URL (`/v1`) e a chave do gateway é necessário; bugs de BYOK no modo **Agent** do Cursor ainda podem ocorrer — preferir **Ask** no smoke test.

## Evidência externa (rede)

A partir deste ambiente de desenvolvimento, `aihub.connexro.com` não resolveu em DNS; `aihub.connexto.com` respondeu com redirecionamentos inconsistentes. Use o host exato do seu deploy e os curls do guia em inglês:

`docs/validation/E02-cursor-ide-byok-aihub.md`

## Documentação no repositório

- Guia operacional (EN): [docs/validation/E02-cursor-ide-byok-aihub.md](../../docs/validation/E02-cursor-ide-byok-aihub.md)

## Entrega de código (2026-11-13)

- `apps/backend/internal/gateway/chat_completions_*.go` — adapter Chat Completions
- `apps/backend/cmd/ai-hub/gateway_routes.go` — rota `POST /v1/chat/completions`
- Testes: `chat_completions_*_test.go`
