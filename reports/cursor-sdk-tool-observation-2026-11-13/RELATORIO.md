# Relatório — observação de tools do Cursor SDK 1.3.1

Data: 2026-11-13

## Escopo

Captura fora do hub (`scripts/observe-sdk-tool-events.mts`) com workspace temporário em `/tmp`. Evidência bruta em `observe-output.json`.

## Eventos capturados (capture-events)

| Cenário | Ferramenta | Sequência observada |
|---|---|---|
| `read` | `read` | `tool_call` `running` → `completed` (mesmo `call_id`) |
| `shell` | `shell` | `tool_call` `running` → `completed` (mesmo `call_id`) |
| `mcp_hang_cancel` | `mcp` | `tool_call` `running` → `run.cancel()` → `status` cancelado, **sem** `tool_call` `completed` |
| `customTools` | `mcp` + `local.customTools.observe_probe` | `tool_call` `running` → `completed` |

Chaves registradas em `args`/`result` apenas como nomes de campo; conteúdo não persistido.

## Correlação de IDs (confirm-ids)

- `read` / `shell`: o SDK emite só mensagens `tool_call`; `call_id` estável entre `running` e `completed` (prefixo `tool_` + UUID).
- `customTools`: `SDKCustomToolContext.toolCallId` coincide com `tool_call.call_id` (`tool_220a1790-…` na execução de referência).
- Nesta amostra não houve bloco `assistant.tool_use` nos fluxos `read`/`shell`; o contrato tipado (`ToolUseBlock.id`) permanece o identificador alternativo quando o assistente emite `tool_use` antes do evento `tool_call` (coberto por teste unitário com `call-read`).

## Cancelamento (confirm-cancel)

- **Produção (`sdk-inprocess-runner`)**: `await cleanup()` (inclui `run.cancel()` quando `running`) ocorre **antes** de `forwardedToolCallEvents`; teste novo garante que falha em `cancel` impede encaminhamento ao cliente.
- **SDK ao vivo (`mcp_hang_cancel`)**: após `running` + `cancel`, resultado `cancelled` e `messagesAfterCancel: 1` (apenas `status`); não chegou `completed` para a tool pendente.
- Servidor MCP “hang” pode receber `tools/call` antes do cancel terminar (efeito local no stdio); o gateway não encaminha MCP nativo em produção (`tools: ["mcp"]` apenas para cliente).

## onDelta / deltas (check-ondelta)

- Tipos em `@cursor/sdk@1.3.1`: `SendOptions.onDelta` presente em `agent.d.ts`; literais `partial-tool-call` e `tool-call-delta` em `delta-types.d.ts`.
- Na observação com `onDelta` ativo, **nenhum** delta `partial-tool-call` nem `tool-call-delta` foi emitido; apareceram `tool-call-started`, `tool-call-completed`, `text-delta`, etc.
- Nenhuma implementação de streaming incremental de tool no hub nesta etapa.

## Implementação (preserve-call-id)

- Campo opcional `sdkCallId` em `CursorToolCall`.
- `sdk-inprocess-runner` preenche a partir de `tool_call.call_id` ou `tool_use.id`.
- `edge-run.ts` usa `sdkCallId` como `callId` do evento `tool.call` (sem prefixo `call_mcp_`).

## Verificações automatizadas

- `vitest`: `tests/edge-run.test.ts`, `tests/sdk-inprocess-runner.test.ts` (inclui cancel falho e propagação de `sdkCallId`).
