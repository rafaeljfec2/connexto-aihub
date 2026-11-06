# Revisão de código — Refino de UI Execuções / Generations

**Data:** 2026-11-05  
**Escopo:** barra de filtros, header PT, presets de período, painel decorativo, tag de origem (`apps/frontend/src/features/executions/`)  
**SPEC:** [FE-EXECUTIONS-GENERATIONS-REFINEMENT](../../docs/specs/FE-EXECUTIONS-GENERATIONS-REFINEMENT.md) (emendada 2026-11-05)  
**Validação:** [FE-EXECUTIONS-GENERATIONS-REFINEMENT](../../docs/validation/FE-EXECUTIONS-GENERATIONS-REFINEMENT.md)

## Round 1 — REJECTED

**Veredicto:** `REJECTED` — interpretação estrita da SPEC original (AC-GEN-06 proibia qualquer gráfico; AC-GEN-05 exigia filtro de modelo sempre visível).

**Verificação:** `pnpm --filter @connexto/frontend check` — PASS (157 testes).

| ID | Severidade | Achado | Status Round 1 |
| --- | --- | --- | --- |
| F01 | CRITICAL | `GenerationsActivityPanel` com barras estáticas e rótulo “VISUALIZAÇÃO ILUSTRATIVA” vs AC-GEN-06 literal (“omit chart”) | OPEN |
| F02 | IMPORTANT | `model_id` só na faixa **Personalizado** vs filtro de modelo sempre exposto na SPEC §5 | OPEN |
| F03 | IMPORTANT | Date picker vs sufixo UTC na API; presets em UTC | OPEN |
| F04 | IMPORTANT | Módulos novos referenciados fora do primeiro diff git | OPEN |
| F05 | MINOR | `aria-label` da linha usa `provider_id` bruto | OPEN |
| F06 | MINOR | Duplicação `formatDataOriginTag` vs catálogo | OPEN |

## Resolução — emenda de SPEC + aceite de produto

O Product Lead confirmou satisfação com a tela **como implementada** (painel decorativo + filtros compactos). A SPEC foi emendada em 2026-11-05 (decisão §13.8, AC-GEN-06 refinado, §5 item 12) para permitir painel **estático e rotulado**, proibindo apenas agregados API-backed ou dados operacionais fictícios na tabela/drawer.

| ID | Resolução | Status final |
| --- | --- | --- |
| F01 | **Emenda SPEC** — painel decorativo permitido com label explícito; componente documentado em §8 | **CLOSED** |
| F02 | **Emenda SPEC** — modelo + datas apenas em **Personalizado** com apply/Enter; outcome/provedor/presets imediatos | **CLOSED** |
| F03 | Comportamento UTC documentado na SPEC §5 item 6; rótulo na UI permanece opcional | **CLOSED** (doc) |
| F04 | Artefatos presentes no working tree; commit único recomendado quando autorizado | **CLOSED** (governança) |
| F05 | `aria-label` alinhado a `catalogProviderLabel` | **CLOSED** — `GenerationsTable.tsx` |
| F06 | Helper único `shared/lib/formatDataOriginTag.ts` com variantes `executions` / `catalog` | **CLOSED** |

**Verificação pós-correção:** `pnpm --filter @connexto/frontend check` — PASS (**159** testes após F05/F06).

## Round 2 — APPROVED

**Veredicto:** `APPROVED` — aderência à **SPEC emendada**, aceite do Product Lead, achados MINOR F05/F06 resolvidos.

**Follow-up opcional (não bloqueante):** nenhum item aberto no ledger deste refinamento.
