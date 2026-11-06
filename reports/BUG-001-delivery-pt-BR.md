# Delivery report — BUG-001 (Composer/Codex 503)

**Demand:** BUG-001 · **Branch:** `connexto/BUG-001` · **CDW:** DONE · **Review:** APPROVED (round 1)

## Objetivo

Codex via AI Hub (`POST /v1/responses`, tools + reasoning) deve rotear para `cursor/composer-2.5` sem **503 routing_unavailable** e seguir para execução no proxy.

## Entregue no connexto-aihub

| Área | Mudança |
|------|---------|
| Catálogo / E08 | `ToolsSupported` e `ReasoningSupported` explícitos em `connextoProxyDescriptor` |
| Cobertura | Legacy `connexto-proxy-cli-llms` excluído do cálculo de coverage; migration `0009` |
| Provider model | `cursor` como provedor lógico; transporte Connexto Proxy separado |
| Gateway | Teste Codex-like; normalização header legado → `cursor` |
| UX ops | Mensagem clara quando upstream proxy retorna **413** |
| CDW | `connexto.project.json`, run `.connexto/runs/BUG-001/` completo |

## Validação

- `go test ./... -count=1 -race` — OK (1007 testes)
- `go vet ./...` — OK
- Evidência operacional: logs com `reached_execution=true`, `coverage_complete_fresh=true`

## Fora do escopo BUG-001 (repo proxy)

Correções em **connexto-proxy-cli-llms**: body 32 MiB, `CURSOR_DEFAULT_WORKING_DIRECTORY`, Docker volume + uid — necessárias para Codex **responder**, não para o 503 do hub.

## Pendências do operador

1. **Commit** na branch `connexto/BUG-001` (adapter CDW: `allow_commit: false` até você pedir).
2. **Proxy local:** `pnpm dev` na 9090 *ou* Docker com workspace (ver `.env` ajustados).
3. **Codex:** `base_url` → hub `:8080`; chaves alinhadas (hub gateway + proxy gateway + Cursor).

## Artefatos

Run dir: `.connexto/runs/BUG-001/` (`research.md` Implemented: yes, `developer/report.json` v2, `review.json` verdict APPROVED).
