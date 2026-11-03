# Relatório — UI Foundation (Doop)

**Data:** 2026-11-03  
**Escopo:** Materializar identidade visual oficial do Connexto AI Hub a partir do canvas Doop.

## Fontes Doop

| Artefato | ID / nome |
|---|---|
| Canvas | `Qactw8snfC` — Rafael's first canvas |
| Style guide | `ai-hub-visual-system` |
| Frames consultados | Overview, Provedores, Catálogo (+ referência Execuções/Recursos/Config/Inteligência) |

## Entregas

1. **Design tokens** — paleta graphite/cobalt, tipografia Newsreader + DM Sans + IBM Plex Mono via `@theme` (Tailwind v4).
2. **AppShell** — sidebar 224px (recolhível), workspace, navegação MVP, rodapé operacional, top bar com breadcrumb, ambiente, SSE e ações globais; drawer em viewport estreita.
3. **`shared/ui`** — botões, badges, cards, painéis, cabeçalho de página, inputs, banners, skeleton, empty state, tooltip e dialog (Radix).
4. **Telas** — Overview, Providers, Catalog e Playground com `PageHeader` e superfícies alinhadas ao guia visual.

## Comparação visual

- Composição geral (sidebar + top bar + área de conteúdo) segue frames 1280×800.
- **Providers:** o frame Doop prioriza franquias/cotas; a rota atual permanece **administração de credenciais** (contrato MVP) — apenas shell/tokens do design oficial, sem números ilustrativos do Doop.
- **Playground:** ausente no protótipo Doop listado; usa o mesmo AppShell e tokens.

## Validação

| Comando | Resultado |
|---|---|
| `pnpm --filter @connexto/frontend check` | OK |
| `pnpm --filter @connexto/frontend build` | OK |
| `go test ./... -count=1` | OK |

## Pendências

- QA visual fino contra PNGs do Doop (screenshot MCP do Overview expirou por timeout).
- Aprovação do Lead antes do Setor 7.
- Telas Execuções/Recursos e visão de cotas em Providers quando o backend/MVP autorizar.
