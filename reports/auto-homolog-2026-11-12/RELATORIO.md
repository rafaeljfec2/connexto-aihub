# Homologação Auto AI-Hub — 2026-11-12

## Escopo

Validar `aihub/auto` no hub local (`http://127.0.0.1:8080`) com:

1. Máscara de `body.model` → sempre `aihub/auto` (sem sticky de provedor).
2. Headers `X-AIHub-Resolved-*` livres (AW-RR).
3. Tools / `additional_tools` no path Auto.
4. Sessão real via Codex CLI (`codex exec -m aihub/auto`).

## Resultado geral

**APROVADO** para os critérios de máscara, tools API e conclusão Codex.

| Porta | Critério | Resultado |
|---|---|---|
| API texto ×10 | HTTP 2xx + `body.model=aihub/auto` | **10/10** |
| API texto ×10 | Diversidade física (AW-RR livre) | cursor / opencode-go / antigravity |
| API `additional_tools` ×5 | HTTP 200 + sinal de tool + máscara | **5/5** |
| Codex CLI | `AUTO_HOMOLOG_OK` + `EXIT:0` | **OK** |
| Codex debug | Máscara de resposta (H6) | **8/8** |
| Codex debug | `custom_tool_*` preservado no prepare OpenCode | **5 kept / 0 stripped** |

## Evidências

### A) Máscara API (`api-mask.tsv`)

Todos os 10 turns retornaram `body.model=aihub/auto` enquanto o resolved variou:

- `cursor/composer-2.5`
- `opencode-go/gpt-6-luna`
- `antigravity/antigravity:gemini-3.8-flash`

Isso confirma: Auto **não fixa** provedor; só **não vaza** o modelo físico no corpo da resposta.

### B) Tools API (`api-additional-tools.tsv`)

5/5 com `additional_tools` Codex-shaped: HTTP 200, `body.model=aihub/auto`, sinal de tool presente. Resolved neste lote ficou em `cursor/composer-2.5` (AW-RR + filtro Antigravity-with-tools).

### C) Codex CLI (`codex-cli.log`)

- Modelo da sessão: `aihub/auto`
- Tools shell executaram (`ls` / `head` do `virtual_auto_model_sink.go`)
- Veredito: **AUTO_HOMOLOG_OK**
- Exit: **0**

Debug da janela Codex (`codex-debug-summary.json`):

- 8 turns Auto
- 8 máscaras de `model` na resposta
- 4 flips servidor composer ↔ luna (esperado sem sticky)
- `custom_tool_*`: 5 preservados, 0 removidos no prepare OpenCode
- 1 cancel ~100s do cliente (residual operacional, não quebrou a homologação)

## Resíduos conhecidos

### 1. `tool exec invoked with incompatible payload` (Codex CLI)

- **Onde:** cliente Codex (`codex_core::tools::router`), não o hub.
- **Padrão observado:** o modelo emite um tool call que o executor local rejeita → Codex narra de novo (“Listando…”) → depois emite `exec` válido e roda.
- **Nesta bateria:** ~10 erros + 2 `exec` OK no 1º run; 2º run também concluiu `AUTO_HOMOLOG_OK`.
- **Severidade:** média (ruído/retry). Não quebra máscara Auto; atrasa o agente.
- **Hipótese principal:** formato misto `custom_tool_call` vs `function_call` / `local_shell` quando o AW-RR troca composer ↔ luna no meio do thread.
- **Mitigação sem sticky:** usar modelo explícito no Codex para trabalho longo; ou (futuro) normalizar tool calls outbound por provedor.

### 2. Flip AW-RR composer ↔ luna (servidor)

- **Onde:** seletor Auto (`rule:auto_awrr`).
- **Produto:** Auto **não** pode sticky — flip é comportamento esperado.
- **Efeito:** novo LLM pode “reiniciar Fase 1” / re-ler contexto.
- **O que já mitiga:** máscara `body.model → aihub/auto` (some o “Model changed to Composer 2.5”).
- **Severidade:** baixa para API/chat curto; alta para agente multi-turn com tools (Codex).

### 3. Cancel ~100s (`cause=client`, `duration≈1m40s`)

- **Onde:** cliente Codex aborta o stream; hub registra `outcome=canceled`.
- **Padrão:** turn Cursor/`composer-2.5` aberto ~100s → cliente corta → próximo turn Auto pode cair noutro pool.
- **Severidade:** média (reinício perceptível na UI).
- **Não é timeout do hub** (`StreamInactivityTimeout` = 2 min); é timeout/abort do Codex.

## Artefatos

- `api-mask.tsv`
- `api-additional-tools.tsv`
- `codex-cli.log`
- `codex-debug-summary.json`
- `debug-api-phase.log` (fase API, se presente)

## Conclusão

Homologação Auto **passou** nos gates de máscara de identidade virtual, tools API e execução Codex CLI. Sticky permanece desabilitado por regra de produto.
