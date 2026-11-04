# SPEC — Typography Normalization (Reference: OpenRouter)

**Document:** `docs/specs/FE-TYPOGRAPHY-NORMALIZATION.md`  
**Status:** Approved — Product Lead Authorization via Prompt Specification  
**Date:** 2026-11-04  
**Classification:** Frontend Global Design System & Doop Typography Normalization  
**Scope:** Complete frontend (`apps/frontend/`) and 7 Doop canvas frames (`Qactw8snfC`)  

---

## 1. Executive Summary & Intent

The purpose of this specification is to globally normalize typography across the Connexto AI Hub application, aligning with the typographic language observed and audited from OpenRouter (https://openrouter.ai/).

This is strictly a **typography normalization and optical adjustment task**. It is NOT a screen redesign, color scheme change, or layout refactor. The midnight blueprint / graphite + cobalt identity (`#10141B`, `#131A24`, `#191F2A`, `#202A38`, cobalt `#7896E8`, line `rgba(180,194,218,0.14)`) is fully preserved.

---

## 2. Audited Reference Evidence (OpenRouter)

On 2026-11-04, a live headless Chrome inspection of `https://openrouter.ai/` and its network font requests was executed via Playwright.

### 2.1 Fonts Actually Loaded by OpenRouter
1. `https://openrouter.ai/_next/static/immutable/media/gordita_bold-s.p.3v-a8bgqavs7d.woff2`
   - Family: `gordita`
   - Weight: `700`
   - Role: Display H1 hero titles.
   - Fallback chain: `"gordita Fallback"`, `"Plus Jakarta Sans"`, `jakarta`, ui-sans-serif, system-ui.
   - Letter-spacing: `-1.4px` on `56px` (~`-0.025em`).
2. `https://openrouter.ai/_next/static/immutable/media/PlusJakartaSans_VariableFont_wght_latin-s.p.1e067vq8lbwt6.woff2`
   - Family: `Plus Jakarta Sans`
   - Weight range: `200 800`
   - Role: Core interface font — body, paragraphs, navigation, buttons, subheadings, table headers, forms.
   - Computed styles: Body 14px (line-height 22.75px, font-weight 450), Paragraphs 16px (line-height 21.6px, font-weight 450), H2 16–20px (font-weight 500–600), Buttons 14px (font-weight 500).
   - Global modifier: `body.tabular-nums` activates proportional/tabular numeric alignment.
3. `https://openrouter.ai/_next/static/immutable/media/GeistMono_Variable.2u63thiw2vdry.woff2`
   - Family: `geistMono`
   - Weight range: `100 900`
   - Role: Technical data, shortcuts (`<kbd>`), code snippets, IDs, telemetry latencies.
   - Computed styles: 12px, font-weight 500, tabular-nums.

### 2.2 Licensing Verification & Font Family Decision
- **Gordita**: Proprietary commercial font by Typefaces / Fabian Harb. A codebase scan confirmed that no Gordita license or font files exist in the repository. Per project governance rules:
  > *"Caso não exista licença para Gordita, utilizar Plus Jakarta Sans em títulos com pesos 600–700."*
- **Plus Jakarta Sans**: Designed by Gumpita Rahayu (Tokotype). Licensed under the **SIL Open Font License (OFL)**. Free for commercial and open-source use.
- **Geist Mono**: Designed by Vercel in collaboration with Basement Studio. Licensed under the **SIL Open Font License (OFL)**. Free for commercial and open-source use.

**Final Family Selection:**
- `--font-display`: `"Plus Jakarta Sans", ui-sans-serif, system-ui, sans-serif;` (weights 600–700)
- `--font-interface`: `"Plus Jakarta Sans", ui-sans-serif, system-ui, sans-serif;` (weights 400, 500, 600)
- `--font-mono`: `"Geist Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;` (weights 400, 500, 600)

---

## 3. Current State vs. Proposed Design Tokens

### 3.1 Current Codebase State
- `apps/frontend/index.html` imports:
  - `Newsreader:opsz,wght@6..72,400;6..72,500`
  - `DM Sans:wght@400;500;600;700`
  - `IBM Plex Mono:wght@400;500;600`
- `apps/frontend/src/styles.css` defines:
  - `--font-display: "Newsreader", ui-serif, Georgia, serif;`
  - `--font-sans: "DM Sans", ui-sans-serif, system-ui, sans-serif;`
  - `--font-mono: "IBM Plex Mono", ui-monospace, monospace;`
- Multiple components reference `font-display`, `font-mono`, `font-sans`, and hardcoded classes.
- Doop canvas guidelines (`ai-hub-visual-system`) and 7 frames currently mandate Newsreader (titles), DM Sans (interface), and IBM Plex Mono (telemetry).

### 3.2 Proposed Global Tokens
In `:root` and `@theme` (`apps/frontend/src/styles.css`):
```css
:root {
  --font-display: "Plus Jakarta Sans", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  --font-interface: "Plus Jakarta Sans", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  --font-sans: "Plus Jakarta Sans", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  --font-mono: "Geist Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}
```

### 3.3 Typographic Scale & Optical Rules

| Token | Size | Line Height | Weight | Letter Spacing | Target Elements |
|---|---:|---:|---:|---:|---|
| **Display** | 36px | 44px | 700 | -0.02em | Hero display titles, main marketing headings |
| **H1** | 28px | 36px | 700 | -0.02em | Page headers (`AppPageHeader`), top section headings |
| **H2** | 20px | 28px | 600 | -0.01em | Panel headers, card section titles |
| **H3** | 16px | 24px | 600 | normal | Subheadings, modal titles, drawer section titles |
| **Body Large** | 14px | 22px | 400 / 500 | normal | Primary descriptions, prominent form labels |
| **Body** | 13px | 20px | 400 | normal | Default body, table rows, form inputs |
| **Label** | 12px | 16px | 500 | normal | Controls, secondary buttons, badges, tabs |
| **Caption** | 10px | 14px | 500 | normal | Footnotes, minor status text, metadata |
| **Technical Data** | 11px | 16px | 400–500 | normal / 0.02em | Latency, throughput, IDs, timestamps, tokens (with `tabular-nums`) |

---

## 4. Component Mapping & Codebase Inventory

| File | Current Font Usage | Target Replacement & Optical Adjustment |
|---|---|---|
| `apps/frontend/index.html` | Google Fonts import for `DM Sans`, `IBM Plex Mono`, `Newsreader` | Replace with Google Fonts import for `Plus Jakarta Sans` (400, 500, 600, 700, italics) and `Geist Mono` (400, 500, 600, 700) with `display=swap`. |
| `apps/frontend/src/styles.css` | `--font-display: Newsreader`, `--font-sans: DM Sans`, `--font-mono: IBM Plex Mono` | Map `--font-display` to `Plus Jakarta Sans`, `--font-sans` and `--font-interface` to `Plus Jakarta Sans`, and `--font-mono` to `Geist Mono`. Activate `tabular-nums` on body and mono. |
| `apps/frontend/src/shared/ui/page-header.tsx` | `font-display font-normal text-[clamp(1.75rem,4vw,2.15rem)] tracking-[-0.03em]` | Retain `font-display`, update weight from `font-normal` (400) to `font-bold` (700) or `font-semibold` (600), tracking `-0.02em`. |
| `apps/frontend/src/features/executions/generation-drawer.css` | `font-family: var(--font-display); font-weight: 400;` | Change to `font-weight: 600; tracking: -0.01em;` to reflect Plus Jakarta Sans display hierarchy. |
| `apps/frontend/src/features/executions/generations.css` | References `var(--font-mono)` and `var(--font-sans)` | Automatically inherits `Geist Mono` and `Plus Jakarta Sans`. |
| `apps/frontend/src/app/layouts/AppSidebar.tsx` | Logo and workspace items | Uses `font-semibold` with `Plus Jakarta Sans` and `font-mono` with `Geist Mono`. |
| `apps/frontend/src/app/layouts/AppTopBar.tsx` | Breadcrumbs and environment tags | Inherits `Plus Jakarta Sans` and `Geist Mono` with `tabular-nums`. |
| Shared UI primitives (`card.tsx`, `panel.tsx`, `badge.tsx`, `button.tsx`, `dialog.tsx`, `filter-chips.tsx`, `grid-pagination.tsx`) | Use semantic `font-mono`, `text-xs font-semibold`, etc. | Automatically inherits new system tokens without code duplication. |

---

## 5. Doop Canvas Normalization Plan

Canvas: `Qactw8snfC`  
Style Guide: `ai-hub-visual-system`  

1. **Update `ai-hub-visual-system` Guideline**:
   - Replace Newsreader with `Plus Jakarta Sans` (weights 600–700, letter-spacing -0.02em).
   - Replace DM Sans with `Plus Jakarta Sans` (weights 400–600).
   - Replace IBM Plex Mono with `Geist Mono` (weights 400–500, tabular-nums).
   - Note that both `Plus Jakarta Sans` and `Geist Mono` are native hosted fonts in Doop.
2. **Apply to 7 Frames**:
   - `gcZJvg0UgN` — Overview
   - `wt9PBPBMhi` — Providers
   - `hV8tWirXfF` — Catalog
   - `PgFh4Y5Uka` — Executions
   - `Q9sTNaOnH5` — Resources
   - `ZD_0jujch7` — Settings
   - `JlQLIdZw8c` — Intelligence
3. **Preservation Rules**:
   - Preserves HTML structure, layout coordinates, graphite + cobalt palette, interactive behaviors, and `data-doop-*` attributes.
   - Adjusts only CSS font-family rules (`font-family: 'Plus Jakarta Sans', sans-serif;` and `font-family: 'Geist Mono', monospace;`) and font weights/letter-spacing.
   - Capture screenshots of all 7 frames. Note that modifying frames transitions their approval status to `needs_review` requiring human sign-off.

---

## 6. Acceptance Criteria

1. **Single Source of Truth**: All font families are derived from `--font-display`, `--font-interface`, `--font-sans`, and `--font-mono` tokens.
2. **Elimination of Legacy Fonts**: Zero remaining references to Newsreader, DM Sans, or IBM Plex Mono in `index.html`, `styles.css`, or components.
3. **Verified Font Availability**: Google Fonts stylesheet imports `Plus Jakarta Sans` (400, 500, 600, 700) and `Geist Mono` (400, 500, 600, 700) with robust system fallbacks.
4. **Tabular Numerics**: `tabular-nums` is active globally on `body` and technical/tabular cells to prevent jitter.
5. **Layout Stability**: No text overflow, broken buttons, unintended line wraps, or clipping at 1280px (desktop), 768px (tablet), and 390px (mobile).
6. **Fidelity between Frontend and Doop**: Both the live frontend code and the 7 Doop canvas frames share the identical typography system.
7. **Verification**: TypeScript check (`pnpm --filter @connexto/frontend check`) and production build succeed with zero errors.
8. **Evidence**: Full record of decisions, before/after analysis, test results, and screenshots documented in AI Memory.

