# Third-party: ai-icons

## Source

- Repository: [gokuscraper/ai-icons](https://github.com/gokuscraper/ai-icons)
- License: MIT (see `apps/frontend/vendor/ai-icons/LICENSE`)
- Vendor path: `apps/frontend/vendor/ai-icons/`
- Pin file: `apps/frontend/vendor/ai-icons/VERSION`

The upstream project is an SVG icon set for AI / LLM brands (mono, color, and text variants). It does not publish an npm package; Connexto AI Hub vendors a full copy for offline Vite embedding into the Go static binary.

## Contents

| Path | Purpose |
|------|---------|
| `svgs/` | Flat SVG files (`{name}.svg`, `{name}-color.svg`, `{name}-text.svg`) |
| `icon-toc.json` | Catalog metadata (docsUrl, group, color flags) |
| `LICENSE` | MIT license text |
| `VERSION` | Pinned commit SHA and sync timestamp |

## Refresh

```bash
# Re-sync the SHA recorded in VERSION (or the script default)
pnpm sync:ai-icons

# Sync a specific commit, branch, or tag
node scripts/sync-ai-icons.mjs <sha-or-ref>
```

After refreshing, update frontend hub→slug mappings if new connectors require icons, then run:

```bash
pnpm --filter @connexto/frontend check
pnpm --filter @connexto/frontend build
```

## Runtime usage

Frontend resolution lives under `apps/frontend/src/shared/branding/`:

- Vite `import.meta.glob` over vendored SVGs (`?url`)
- Hub provider id / logo → file slug map
- `AiBrandIcon` React component with initials fallback

Do not load icons from GitHub raw URLs in production builds.
