# Manter contexto original do agente (RTK) — 2026-11-13

## Decisão

O proxy **não** aplica política `rtk` em comandos shell. A regra `Always prefix shell commands with rtk` permanece nas **instructions** do Codex (`clientInstructions`), vindas do agente de origem (`AGENTS.md` / `RTK.md`).

Motivo: `rtk` pode não estar instalado na máquina do desenvolvedor; prefixar no hub alteraria o payload que o agente pediu e faria o cliente executar outro programa.

## Verificação de implementação

| Item do plano | Estado |
|---|---|
| Módulo `rtk-shell-policy.ts` | **Ausente** (não criado) |
| `enforceClientToolContract` acoplado a `clientInstructions` para prefixo | **Ausente** |
| Testes de enforcement de prefixo `rtk` | **Não adicionados** (só regressão: proxy não injeta `rtk`) |
| Composição `instructions` → complemento → `input` | **Mantida** |
| `prompt.ts` sem duplicar regra RTK | **Mantido** |

## Papel do Cursor SDK (`edge_owned`)

- `cwd`: diretório temporário `cursor-edge-owned-…`
- Tools SDK: apenas `mcp`
- Execução real de shell/arquivo: **cliente Codex** (`client_shell`, `exec`, `apply_patch`)

## Testes

- `packages/providers/cursor/tests/client-mcp/client-tool-contract.test.ts` — caso *never adds rtk to shell payloads*
- Suíte `client-mcp`: executada nesta rodada (ver log da sessão)

## Smoke real Composer 2.5 + Codex

**Pendente** — aderência a `rtk` depende do modelo e do ambiente Codex, não de transformação no proxy.

Sem commit, push ou deploy (conforme plano).
