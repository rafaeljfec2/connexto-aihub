# Relatório: composição Codex → Cursor SDK

**Data:** 2026-11-13  
**Escopo:** separar `instructions`, ordem de composição, complemento MCP enxuto, testes e verificação.

## Entregas

1. **split-instructions** — `prepareResponsesPassthrough` usa `instructionsToPlainText`; `prompt.text` / `message` carregam só o input serializado; `clientInstructions` opcional em `CursorExecutionContext` e repasse até `CursorSdkInProcessInput`.
2. **compose-order** — `composeEdgeOwnedUserMessageText` monta: instruções → complemento MCP → input → workdir (só com tools de cliente).
3. **simplify-prompt** — `edgeOwnedBridgePromptPrefix` descreve apenas transporte MCP (sem persona GPT-6, sem `git status -sb`, sem bloco `mandatoryContractPromptBlock` no prompt).
4. **regression-tests** — novos/atualizados testes de plain text, passthrough, payload Cursor, prompt, lifecycle SDK; validador `client-tool-contract` inalterado e coberto pelos testes existentes.
5. **verify** — ver seção abaixo.

## Tamanho do prompt (amostra exec + instruções Codex)

| Artefato | Caracteres |
|----------|------------|
| Complemento MCP (`exec` grammar) | 1398 |
| Mensagem composta (instruções + complemento + input + workdir) | 1551 |

Antes, o mesmo fluxo embutia instruções no meio de um bloco grande (persona GPT-6, contratos repetidos e exemplos de shell), encobrindo regras como `Always prefix shell commands with rtk`.

## Verificação automatizada

Comandos executados (pacotes tocados):

- `vitest run` — protocol/passthrough, `client-mcp/prompt`, `cursor-execution-payload`, `sdk-inprocess-runner.lifecycle` — **OK**
- `typecheck` — `@agent-runtime-gateway/compatibility-openai`, `core`, `providers/cursor` — **OK**
- `typecheck` do pacote `apps/api` — **falha pré-existente** em `chat-completions-stream.ts` (`finish_reason: "length"`), fora deste escopo
- `pnpm test` (suite completa do proxy) — **5 falhas** em testes *real* Antigravity / provider-errors, não relacionados a esta mudança; lifecycle Cursor corrigido e passando

## Smoke test Composer 2.5 + Codex

**Status: pendente** — requer ambiente hub + Codex com Composer 2.5 ativo (skill `senior-code-review-judge`, `rtk`, `exec`/`wait`, segunda rodada). Não executado nesta sessão.

## Arquivos modificados (principais)

- `apps/proxy/packages/compatibility/openai/src/protocol/prepare.ts`
- `apps/proxy/packages/core/src/contracts/execution.ts`
- `apps/proxy/apps/api/src/utils/cursor-execution-payload.ts`
- `apps/proxy/packages/providers/cursor/src/client-mcp/prompt.ts`
- `apps/proxy/packages/providers/cursor/src/sdk-inprocess-lifecycle.ts`
- `apps/proxy/packages/providers/cursor/src/sdk-inprocess-types.ts`
- `apps/proxy/packages/providers/cursor/src/provider.ts`
- Testes em `compatibility/openai`, `cursor`, `apps/api`

Sem commit, push ou deploy (conforme plano).
