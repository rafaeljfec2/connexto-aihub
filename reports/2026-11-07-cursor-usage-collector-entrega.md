# Cursor Usage Collector — relatório de entrega (2026-11-07)

## Resumo

Implementação ponta a ponta do coletor de uso Cursor via **playwright-core + Chromium** no proxy, persistência de sessão com namespace `cursor-usage`, rotas HTTP conforme o plano, integração no AI Hub (cota + auth + limpeza de sessões) e spike documentado com smoke.

## Verificações

- `pnpm test` (repositório `connexto-proxy-cli-llms`): sucesso (inclui testes Vitest do serviço).
- `pnpm exec tsc -p apps/api/tsconfig.json --noEmit`: sucesso.
- `go test ./... -short` (repositório `connexto-aihub`): sucesso.

## Riscos / próximos passos

1. Ajustar `CURSOR_USAGE_RESPONSE_URL_PATTERN` após spike real contra o dashboard Cursor (o padrão atual é genérico `/usage/i`).
2. Expor `interactionUrl` só atrás do gateway Hub com noVNC seguro (placeholder HTML no proxy).
3. E2E manual: login humano, segundo collect, restart com `SESSION_STORAGE_DRIVER=filesystem`, `POST .../runtime/sessions/clear` com `cursor-usage`.

## Evidência (sem segredos)

- Código: pacote `@agent-runtime-gateway/provider-cursor-usage`, rotas em `apps/api/src/routes/cursor-usage.ts`, Hub em `internal/providers/cursor_usage_*.go`, `cmd/ai-hub/provider_admin.go`.
