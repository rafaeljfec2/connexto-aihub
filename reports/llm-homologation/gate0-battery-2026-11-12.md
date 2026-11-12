# Homologação Gate 0 — Bateria `aihub/auto` (2026-11-12)

- **Data (UTC):** 2026-11-12 (janela ~17:56–18:02 UTC)
- **Gateway:** `http://127.0.0.1:8080` (`pnpm dev`, `/healthz` 200)
- **Escopo:** Dez identificadores da matriz histórica do Auto; bateria curta por modelo (texto, stream, `json_schema` strict sem stream)
- **Artefatos:** `reports/llm-homologation/gate0-battery-2026-11-12/run.log`, `results.json`, `run_gate0_battery.py`

---

## 1. Contrato da bateria

| Caso | Payload | Critério de sucesso |
| --- | --- | --- |
| `text` | `stream: false`, prompt `GATE0_OK` | HTTP 200 e texto não vazio |
| `stream` | `stream: true` | HTTP 200 e corpo SSE/JSON de stream |
| `json_schema` | `text.format` `json_schema` `strict: true`, prompt com valores **Connexto AI-Hub / Go / API** | HTTP 200 e JSON com os três campos exigidos |

Sem retries, sem fallback, uma chamada por caso.

---

## 2. Resultados — sete modelos na allowlist vigente

Após correção do prompt de `json_schema` (alinhado ao reteste de 2026-11-10), **todos os sete modelos** da allowlist MVP passaram nos três casos:

| Modelo | text | stream | json_schema |
| --- | --- | --- | --- |
| `anthropic/claude-haiku-4.5` | PASS | PASS | PASS |
| `glm-5.3-flash` | PASS | PASS | PASS |
| `deepseek-v4.1-flash` | PASS | PASS | PASS |
| `claude-haiku-5-5` | PASS | PASS | PASS |
| `gpt-6-luna` | PASS | PASS | PASS |
| `composer-2.5` | PASS | PASS | PASS |
| `antigravity:gemini-3.8-flash` | PASS | PASS | PASS |

Evidência `Gate0Approved: true` atualizada em `apps/backend/internal/routing/auto_evidence.go` com nota `gate0_battery_2026-11-12`.

---

## 3. Reteste — três Antigravity fora da allowlist

| Modelo | text | stream | json_schema | Observação |
| --- | --- | --- | --- | --- |
| `antigravity:claude-opus-4-6-thinking` | FAIL 429 | FAIL 429 | FAIL 502 | Cota upstream (mesmo padrão de 2026-11-10) |
| `antigravity:claude-sonnet-4-6` | FAIL 429 | FAIL 429 | FAIL 502 | Idem |
| `antigravity:gpt-oss-120b-medium` | FAIL 429 | FAIL 429 | FAIL 502 | Idem |

**Decisão:** redução formal da `AutoMVPAllowlist` de dez para **sete** modelos; os três ids permanecem no catálogo geral, mas não entram no Auto até nova homologação.

Ver: `docs/decisions/e08-auto-gate0-allowlist-reduction-2026-11-12.md`.

---

## 4. Consolidação Gate 0

- `AutoGate0Satisfied()` → **verdadeiro** para a allowlist de sete modelos.
- Gate 0 é **somente evidência** (`auto_evidence.go`); não bloqueia `aihub/auto` em runtime.
- Lacuna conhecida: registros de execução ainda podem expor `usage: null` em parte dos provedores — não bloqueia este Gate 0 estrutural, mas permanece item de telemetria.

---

## 5. Comandos de verificação

```bash
cd apps/backend && go test ./internal/routing/... -count=1
cd apps/backend && go test ./internal/gateway/... -run VirtualAuto -count=1
python3 reports/llm-homologation/gate0-battery-2026-11-12/run_gate0_battery.py
```
