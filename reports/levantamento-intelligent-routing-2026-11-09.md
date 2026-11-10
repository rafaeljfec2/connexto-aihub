# Levantamento — Intelligent Routing (E08)

**Data:** 2026-11-09  
**Repositório:** `connexto-aihub` @ `60d0d963e0950f7c63894c7e6d8857ab6085766d` (`main`)  
**Escopo:** estado de produto, incrementos E08, implementação em runtime, documentação, memória de projeto e alinhamento com outros épicos.

---

## 1. Resumo executivo — em qual fase estamos

| Dimensão | Situação |
| --- | --- |
| **Fase de produto (E08)** | **Pós-MVP fechado:** incrementos **E08.1–E08.5 aceitos** (decisão de projeto 2026-11-03). O roteamento determinístico está **implementado e ligado** ao gateway `POST /v1/responses`. |
| **Fase seguinte (E08.6–E08.8)** | **Não autorizada.** Classificação de tarefa, perfis Efficiency/Balance/Intelligence e ranking entre candidatos elegíveis permanecem **visão futura** (SPEC e ADR-022). |
| **Próximo fronte de desenvolvimento autorizado (roadmap)** | **Frontend MVP** — não E08.6 (ver `docs/roadmap/AI-HUB-ROADMAP.md`). |
| **“Auto” / roteamento por custo ou quota (produto)** | **Fora do E08 MVP** e, em parte, **Etapa 2** da decisão Unified Quota Management — sinais de quota para roteamento futuro, independente do motor E08.2–E08.5. |

**Conclusão em uma linha:** o AI Hub está na fase **“Intelligent Routing MVP determinístico entregue e em produção de código em `main`”**, aguardando decisão de produto explícita antes de qualquer **E08.6+** ou roteamento otimizado por sinais de quota/custo.

---

## 2. Modelo de fases (pipeline E08)

Pipeline documentado na SPEC E08:

```text
Request Understanding  →  Eligibility Engine  →  Intelligent Selector  →  Provider + Model  →  Execution Lifecycle
     (E08.6+)                 (E08.2)              (E08.3 MVP; E08.7–E08.8+)         (E08.4)            (E05/E06)
```

| Incremento | Nome | Autorização | Estado (2026-11-09) |
| --- | --- | --- | --- |
| E08.1 + E08.1.1 | Contrato, SPEC, ADR-022 | Aceito 2026-11-03 | Artefatos em `docs/specs/E08-*.md`, `docs/adr/ADR-022-*` |
| E08.2 | Motor de elegibilidade | Aceito | `internal/routing/eligibility.go` + testes |
| E08.3 | Seletor determinístico | Aceito | `internal/routing/selector.go` — sem desempate lexicográfico entre múltiplos elegíveis |
| E08.4 | Integração gateway Responses | Aceito | `internal/gateway/responses_routing.go`, wiring em `cmd/ai-hub/gateway_routes.go` |
| E08.5 | Observabilidade + E2E | Aceito | `internal/routing/observability.go`, testes de gateway |
| E08.6 | Request Understanding (metadados estruturais) | **Não autorizado** | Nenhum código de classificador |
| E08.7 | Optimization Profiles | **Não autorizado** | Conceito na SPEC §2.1 apenas |
| E08.8 | Seletor com sinais + perfis | **Não autorizado** | Nenhuma implementação |

Comportamento **intencional do MVP:** com **dois ou mais** modelos elegíveis após elegibilidade, o resultado é **`routing_ambiguous`** — não há ranking implícito no catálogo.

---

## 3. Evidência de implementação (código)

### 3.1 Pacote `internal/routing` (17 arquivos)

| Área | Arquivos principais | Função |
| --- | --- | --- |
| Validação estrutural | `validate.go` | Formas de `model`, `max_output_tokens`, etc. |
| Snapshot | `snapshot.go`, `project.go` | Um snapshot consistente por decisão (contrato §2.4) |
| Elegibilidade | `eligibility.go` | Projeção de capacidades, catálogo, runtime, limites |
| Seleção | `selector.go`, `selector_helpers.go` | Precedência contrato §4; regra opencode-go; candidato único |
| Observabilidade | `observability.go` | Log content-free `intelligent_routing=true` |
| Tipos / motivos | `types.go`, `decision.go`, `reason.go` | Códigos estáveis de rejeição e exclusão |

**Volume aproximado:** ~2 698 linhas somando `internal/routing/*.go` e `internal/gateway/responses_routing.go` (contagem local 2026-11-09).

### 3.2 Integração no gateway

- `POST /v1/responses` instancia `ResponsesRouting` com catálogo, statuses de providers, `ProviderService` e `StaleAfter` (`gateway_routes.go`).
- Pipeline em `routeResponses`: `ValidateStructure` → `ProjectRequirements` → `CaptureSnapshot` → `EvaluateEligibility` → `SelectDeterministic` → preparação do body (injeção de `model` quando seleção automática).
- Handler: `internal/gateway/responses_handler.go` chama roteamento e `routing.LogDecision` antes da execução.

### 3.3 Extensões pós-fechamento MVP (em `main`)

Commits relevantes após o núcleo E08 (histórico `internal/routing` / gateway):

| Commit (resumo) | Impacto no roteamento |
| --- | --- |
| `820eb74` fix BUG-001 | Projeção de capacidades Cursor proxy para payloads Codex (elegibilidade E08) |
| `01ea20e` | Normalização OpenCode Responses + enriquecimento catálogo `image_input` |
| `selector.go` (Antigravity) | Prefixo `antigravity:` no model resolve provider explícito |

Não alteram o contrato de **não ranking** entre múltiplos elegíveis.

---

## 4. Verificação automatizada (2026-11-09)

| Comando | Resultado |
| --- | --- |
| `go test ./internal/routing/... -count=1` | **59** testes — PASS |
| `go test ./... -count=1` | **1100** testes, **39** pacotes — PASS |
| Testes filtrados Routing/Eligibility/Selector no gateway | **35** — PASS |

Fechamento original (2026-11-03) citava **866** testes com `-race`; a suíte cresceu com E09, frontend tooling no monorepo Go, etc. O MVP E08 permanece coberto por pacotes dedicados e testes de integração em `internal/gateway/*routing*`.

---

## 5. Contratos, decisões e memória

### 5.1 Repositório (inglês — decisões/specs)

| Documento | Conteúdo |
| --- | --- |
| `docs/specs/E08-intelligent-routing.md` | SPEC mestre; §7 lista incrementos aceitos |
| `docs/specs/E08-routing-contract.md` | Contrato de precedência e elegibilidade |
| `docs/adr/ADR-022-deterministic-intelligent-routing.md` | ADR determinismo e limites |
| `docs/decisions/e08-mvp-closure-2026-11-03.md` | Fechamento formal MVP |
| `docs/decisions/e08.6-scope-assessment-2026-11-03.md` | **Recomendação: não iniciar E08.6** |
| `docs/decisions/e08.3-formal-acceptance-2026-11-03.md` | Aceite E08.3 |
| `docs/decisions/e08.4-gateway-routing-integration.md` | Integração gateway |
| `docs/decisions/e08.5-formal-acceptance-2026-11-03.md` / `e08.5-routing-observability.md` | Observabilidade |
| `docs/roadmap/AI-HUB-ROADMAP.md` | E08.1–E08.5 entregues; próximo fronte = Frontend MVP |
| `docs/architecture.md` | E08 listado nos incrementos pós-baseline |

### 5.2 AI Memory (`default` / `rafaeljfec2-connexto-aihub`)

| Página | Nota |
| --- | --- |
| `decisions/e08-mvp-closure-2026-11-03.md` | **Pinned** — veredito MVP CLOSED; cadeia E08.1–E08.5 |
| `decisions/e08-intelligent-routing.md` | Direção do épico |
| `decisions/e08-formal-acceptance-2026-11-03.md` | Aceite E08.1 / início E08.2 |
| `decisions/e08.3-formal-acceptance-audit-2026-11-03.md` | Auditoria seletor |
| `decisions/e08.4-formal-acceptance-independent-review-2026-11-03.md` | Review independente E08.4 |
| `decisions/e08.5-independent-review-formal-acceptance-2026-11-03.md` | E08.5 |
| `decisions/e08.6-scope-assessment-2026-11-03.md` | Não autorizar E08.6 no MVP |

**Atualização deste levantamento:** nova página `decisions/intelligent-routing-status-audit-2026-11-09.md` (pinned) com snapshot de 2026-11-09.

### 5.3 Desatualização conhecida na memória de fechamento (E08)

A página pinned de fechamento (2026-11-03) ainda menciona implementação “local” em `ea198436` + diff não publicado. Em **2026-11-09**, `main` está em `60d0d963` e o código E08 está **no histórico remoto** (commits `ea19843`, `2c6e06e`, `820eb74`, etc.). O fechamento permanece válido em **aceite funcional**; apenas o item “git publish” ficou **obsoleto**.

---

## 6. O que **não** é Intelligent Routing E08 (evitar confusão)

| Tema | Relação com E08 |
| --- | --- |
| `reports/llm-homologation-remediation.md` §11 (“router not built”) | Refere-se a **pré-requisitos de homologação Auto/custo** e `openrouter/auto`, **não** à ausência do motor E08.2–E08.5 (já existente). Tratar como gate de **produto Auto**, não como status de E08. |
| Unified Quota Management **Etapa 2** | Roteamento automático **futuro** usando sinais de quota; SPEC E08.6–E08.8 também futuro — **épocas distintas**, mesma direção de produto de longo prazo. |
| `connexto-proxy-cli-llms` “routing helpers” | Refatoração modular do proxy; **não** é o épico E08 do AI Hub. |
| Frontend (`routing_overhead_ns`, eventos `routing_selected`) | **Telemetria/visualização** de execuções; não implementa seletor. |
| Kickoff `AI-Hub-Product-Architecture-Kickoff.md` (trechos “E08 not implemented”) | Documento histórico de visão; **supersedido** por `architecture.md` e roadmap para E08.1–E08.5. |

---

## 7. Limitações operacionais do MVP (comportamento esperado)

1. **Ambiguidade:** múltiplos modelos elegíveis → HTTP mapeado de `routing_ambiguous` (contrato).
2. **Sem fallback:** falha após seleção não troca provider/model (FR-12).
3. **Sem classificador LLM** e **sem otimização de custo/latência** no catálogo.
4. **Caminho legado em testes:** alguns handlers de teste passam `routingDeps == nil` — sem log `routing decision` (documentado em E08.5).
5. **Homologação de modelos (2026-11-09):** modelos `BLOCKED` (Antigravity quota) devem ser tratados como indisponíveis na **operação**, via elegibilidade/runtime — não é novo incremento E08.

---

## 8. Critérios de aceite SPEC — panorama

Os **AC-1–AC-14** da SPEC E08 estão endereçados pelo conjunto eligibility + selector + gateway + testes (matriz de fechamento 2026-11-03). Lacunas **documentais** (não funcionais) já listadas no fechamento:

- Opcional: espelhar aceites formais E08.2/E08.4 só na wiki vs arquivos `docs/decisions/*` (rastreabilidade).
- Roadmap snapshot no arquivo de roadmap usa commit antigo (`9338c53`); estado E08 não muda por isso.

---

## 9. Recomendações (somente planejamento)

| Prioridade | Ação | Gate |
| --- | --- | --- |
| P0 | Manter E08.3 como baseline; **não** iniciar E08.6 sem mini-SPEC aprovado | Lead + SPEC |
| P1 | Se produto pedir “modo Auto”: autorizar **E08.7** (perfis) antes ou junto de E08.6 — classificador sozinho não resolve `routing_ambiguous` | `e08.6-scope-assessment` |
| P2 | Reconciliar memória pinned de fechamento (git publish obsoleto) | Este audit 2026-11-09 |
| P3 | Etapa 2 Unified Quota: especificação **separada** de E08.6; alinhar sinais com ADR-022 (sem override de seleção explícita) | Decisão de produto |
| P4 | Homologação LLM: não confundir gate Auto (§11 remediation) com entrega E08 | Documentação operacional |

---

## 10. Referências rápidas

- SPEC: `docs/specs/E08-intelligent-routing.md`
- Fechamento MVP: `docs/decisions/e08-mvp-closure-2026-11-03.md`
- Decisão em inglês (espelho): `docs/decisions/intelligent-routing-status-2026-11-09.md`
- AI Memory: `decisions/intelligent-routing-status-audit-2026-11-09.md`
- Código entrada gateway: `internal/gateway/responses_routing.go`, `cmd/ai-hub/gateway_routes.go`
