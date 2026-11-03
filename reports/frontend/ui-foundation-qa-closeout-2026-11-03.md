# UI Foundation — QA visual, estabilização e aprovação (2026-11-03)

## Decisão da etapa

**Classificação: `APPROVED`** — AppShell e tokens alinhados ao Style Guide e frames Doop; pronto para **aprovação explícita do Lead** antes do Setor 7.

Não é `BLOCKED`: build/testes verdes, comparação visual realizada (com limitação MCP Doop), divergências de conteúdo são escopo de produto documentado.

---

## Estado Git (pré-commit)

- Branch: `main` (ahead 2 de `origin/main`).
- UI Foundation + feature-first: alterações locais extensas (`app/`, `features/`, `shared/`, `styles.css`, deps).
- `git diff --check`: sem erros de whitespace.

---

## Code Review (formal)

**Veredito:** `APPROVED_WITH_NOTES` (revisão independente + correções aplicadas).

| ID | Severidade | Status |
|---|---|---|
| Metadados duplicados `pageMeta` / `PageHeader` | IMPORTANT | **Corrigido** — `AppPageHeader` + `useAppPageMeta()` |
| Drawer mobile (Escape, focus trap, aria-modal) | IMPORTANT | **Corrigido** — `FocusScope`, Escape, foco restaurado |
| Rótulo “Gateway operational” = SSE | IMPORTANT | **Corrigido** — rodapé descreve SSE explicitamente |
| `StatusCard` cores conflitantes | MINOR | **Corrigido** |
| `Button` + `asChild` + `type` | MINOR | **Corrigido** |
| `Dialog` sem close | MINOR | **Corrigido** |
| Sem testes de layout | OBSERVATION | Pendente (não bloqueante) |
| Bundle > 500 kB | OBSERVATION | Documentado; sem split nesta etapa |

---

## QA visual — Doop

**Canvas:** `Qactw8snfC` · **Style guide:** `ai-hub-visual-system`

| Frame | MCP `get_frame_screenshot` | Referência alternativa |
|---|---|---|
| Overview `gcZJvg0UgN` | Timeout 30s | PNG baixado + browser |
| Providers `wt9PBPBMhi` | Timeout 30s | PNG baixado |
| Catalog `hV8tWirXfF` | Timeout 30s | PNG baixado |

Arquivos de referência: `reports/frontend/qa/doop-{overview,providers,catalog}.png`

### Comparação (shell vs conteúdo)

| Aspecto | Doop | App implementada | Ação |
|---|---|---|---|
| Sidebar ~224px, topbar ~62px, graphite/cobalt | Sim | Sim (sidebar ajustada para `224px` exatos) | OK |
| Workspace card, nav sections, breadcrumb | Sim | Sim | OK |
| Tipografia Newsreader + DM Sans + Plex Mono | Sim | Sim (Google Fonts) | OK |
| Overview: 4 stat cards + gráficos + cotas | Sim (ilustrativo) | 2 cards reais + diagnostics | **Escopo produto** — sem dados fictícios |
| Providers: tabela de franquias | Sim | Admin de credenciais MVP | **Escopo produto** |
| Catalog: busca + painel lateral | Sim | Filtros + tabela API | **Escopo produto** (layout parcial) |
| Playground | Sem frame | Tokens/AppShell | OK |
| Labels PT vs EN | PT no Doop | EN no produto | Documentado |

Comparação interativa: browser em `http://127.0.0.1:5173/` vs PNGs Doop (capturas de sessão Cursor).

---

## QA responsivo

| Viewport | Validação |
|---|---|
| Desktop 1280×800 | AppShell completo; sidebar visível; sem overflow horizontal aparente |
| Tablet 768×800 | Sidebar oculta em `<lg`; topbar com menu; conteúdo legível |
| Mobile 390×844 | Drawer abre via “Open navigation”; backdrop + foco no primeiro link; SSE no header |

Teclado: `Escape` fecha drawer (implementado na correção pós-review).

---

## Estabilização técnica

### `ERR_PNPM_IGNORED_BUILDS` (esbuild@0.25.12)

- **Causa:** política pnpm do monorepo bloqueia scripts de build de dependências (supply-chain).
- **Evidência:** `esbuild@0.25.12` via `vite`; `pnpm --filter @connexto/frontend build` **PASS**.
- **Decisão:** não executar `pnpm approve-builds` nesta etapa — build de produção já funciona; aprovação de scripts é decisão de governança do operador.

### Comandos

| Comando | Resultado |
|---|---|
| `pnpm --filter @connexto/frontend check` | PASS (21 testes) |
| `pnpm --filter @connexto/frontend build` | PASS — JS ~569 kB (gzip ~179 kB) |
| `go test ./... -count=1` | PASS (923) |
| `git diff --check` | PASS |

### Bundle > 500 kB

- Origem principal: React 19, react-router-dom, TanStack Query, lucide-react (ícones), Radix, zod.
- **Ação nesta etapa:** nenhum code-split (evitar otimização prematura). Reavaliar se o Lead exigir budget explícito.

### Regressões / contratos

- **Zero polling:** `refetchInterval: false` em queries e `queryClient` default.
- **SSE:** `useOperationalEvents` com cleanup de listeners e backoff; testes passando.
- **Segredos no bundle:** busca por `AIHUB_` / tokens no `dist/` — sem vazamento; auth via proxy Vite dev.
- **Playground streaming:** UI funcional no browser (comboboxes, composer); lógica inalterada nos testes existentes.

---

## Alterações pós-QA (esta sessão)

- Sidebar largura fixa `224px` (Style Guide).
- Relatório e PNGs Doop em `reports/frontend/qa/`.

---

## Pendências remanescentes (não bloqueiam APPROVED)

1. Aprovação **humana** do Lead sobre layout e divergências de conteúdo (Overview/Providers/Catalog vs frames ilustrativos).
2. Testes automatizados de AppShell / drawer (opcional).
3. MCP Doop `get_frame_screenshot` instável (timeout) — usar PNG `https://doop.design/i/<frameId>.png?scale=2`.
4. Setor 7: Execuções, Recursos, visão de cotas em Providers quando backend permitir.
5. Commit/push/PR aguardando autorização.

---

## Evidências anexas no repositório

- `reports/frontend/ui-foundation-doop-2026-11-03.md` (entrega anterior)
- `reports/frontend/qa/doop-*.png` (referência Doop)
- Este arquivo: `reports/frontend/ui-foundation-qa-closeout-2026-11-03.md`
