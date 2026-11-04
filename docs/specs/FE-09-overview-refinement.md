# SPEC FE-09 — Refinamento Visual e Funcional da Tela Visão Geral (Overview)

- **Status:** Approved (Autorizado pelo Product Lead em 2026-11-03)
- **Data:** 2026-11-03
- **Referência Visual Fonte da Verdade:** Doop Canvas `Qactw8snfC`, Frame `gcZJvg0UgN` ("01 — Overview", 1280×800) e Imagem de Referência do Operador
- **Diretriz de Design:** Dark Graphite + Cobalt (`#10141B`, `#191F2A`, `#202A38`, `#7896E8`, `#AFC2F2`), tipografia híbrida editorial Newsreader + DM Sans + IBM Plex Mono

---

## 1. Decisões Incorporadas Formalmente

1. **Volume de requisições e Telemetria Real:**
   - Exclusivamente dados reais do backend. Nenhum dado artificial, linha de base fictícia ou interpolação no ambiente operacional.
   - Verificar a agregação no backend: total de requisições registradas vs. admissões de recursos. Criar/ajustar endpoint de agregação de volume temporal de requisições (`/api/overview/traffic-volume` ou extensão segura de telemetria) se o existente não cobrir o volume global de requisições do gateway nas últimas 24h.
   - Preservar valores zero e ausência de dados como estados distintos. Se não houver histórico suficiente, apresentar estado vazio elegante mantendo as proporções e a composição do painel visual.
2. **Consumo de franquias:**
   - Provedor sem política de cota: badge cinza `Sem leitura` com descrição `Franquia ainda não monitorada`.
   - Política com limite válido: apresentar consumo real (`% consumido · ciclo`, `% disponível`).
   - Política ilimitada: identificar como `Sem limite`.
   - Política desabilitada: apresentar estado não aplicável, nunca 0%.
   - Não confundir ausência de configuração com consumo zero.
3. **Diagnósticos e Rodapé:**
   - Remover da Overview os blocos redundantes de *System liveness* e *ReadinessCard*.
   - Preservar os endpoints `/healthz`, `/readyz` e `/diagnostics/providers`.
   - Manter a informação operacional essencial no card *Estado do gateway*.
   - Reproduzir a footnote minimalista do Doop com `/healthz`, `/readyz`, `/diagnostics/providers` e versão/ambiente.
4. **Tipografia:**
   - `Newsreader` (título editorial H1, 35px, 400).
   - `DM Sans` (textos, controles, labels e corpo).
   - `IBM Plex Mono` (métricas, identificadores, horários e metadados).
   - Fontes carregadas de forma segura e resiliente para produção.
5. **Metadados Públicos de Runtime e Origem dos Dados:**
   - Contrato seguro no backend (`GET /api/runtime/meta` ou `/api/system/context`) fornecendo:
     - `environment`: "development" | "staging" | "production"
     - `data_origin`: "live" | "demonstrative" | "partial" | "unavailable"
     - `version`: string (ex: "PREVIEW 01" ou build SHA)
   - Não inferir ambiente apenas por host ou banco; usar configuração explícita e segura.
6. **Fidelidade Visual e Escopo:**
   - Referência estrita ao Frame `gcZJvg0UgN` em 1280×800.
   - Escopo limitado exclusivamente à tela Visão Geral. Proibido iniciar próximas telas antes da aprovação do usuário.

---

## 2. Contratos de Backend e Modelos de Dados

### 2.1 Metadados de Runtime (`GET /api/runtime/meta`)
Endpoint não sensível para operadores e frontend:
```json
{
  "environment": "development",
  "data_origin": "live",
  "version": "PREVIEW 01",
  "gateway_status": "operational"
}
```

### 2.2 Volume Temporal de Tráfego (`GET /api/telemetry/traffic-summary?range=24h`)
Calculado a partir dos registros reais de execução (`execution_records`) persistidos pelo gateway:
```json
{
  "range": "24h",
  "total_requests": 82,
  "trend_percentage": 0.0,
  "buckets": [
    { "timestamp": "2026-11-03T18:00:00Z", "count": 12 },
    { "timestamp": "2026-11-03T19:00:00Z", "count": 25 }
  ],
  "has_sufficient_history": true
}
```
Se não houver execuções ou histórico, `total_requests: 0`, `buckets: []`, `has_sufficient_history: false`.

### 2.3 Quotas e Provedores (`GET /api/resource-analytics/quota-utilization`)
Utiliza o serviço existente de analytics e diagnósticos:
- Provedores com cota: calcula `% consumido`, `% disponível`.
- Provedores sem cota: `state: "unmonitored"`, label `"Sem leitura"`.

---

## 3. Plano Incremental de Execução

- **Fase 1 (Backend Contracts & Services):**
  - Implementar handler seguro `GET /api/runtime/meta`.
  - Implementar agregação real de tráfego de 24h a partir de `execution_records` / serviço de telemetria.
  - Testes unitários e de integração Go para as novas rotas.
- **Fase 2 (Frontend Tokens & Design System):**
  - Configurar fonte `Newsreader` e tokens no `@connexto/frontend`.
  - Ajustar Topbar com pill de ambiente dinâmico e tag de origem de dados reais/demonstrativos.
- **Fase 3 (Componentes do Overview):**
  - Refatorar `OverviewStatCards.tsx` (Cards: Gateway, Franquias, Catálogo, Requisições).
  - Implementar `OverviewTrafficChartPanel.tsx` com SVG dinâmico para dados reais ou estado vazio fiel sem distorcer o layout.
  - Refatorar `OverviewProvidersPanel.tsx` respeitando as regras de quota e "Sem leitura".
  - Refatorar `OverviewRecentExecutionsPanel.tsx` com as colunas e badges do Doop.
  - Adicionar a footnote técnica do Doop e remover os blocos redundantes.
- **Fase 4 (Testes Automatizados & Validação Visual):**
  - Testes unitários React e Go.
  - Teste visual com Playwright capturando screenshot em 1280×800.
  - Comparação com Doop e ajuste fino.
- **Fase 5 (Consolidação e AI Memory):**
  - Relatório final completo e gravação no AI Memory.
