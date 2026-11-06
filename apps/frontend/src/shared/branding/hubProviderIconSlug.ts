/**
 * Maps Connexto AI Hub provider identifiers / logo tokens onto ai-icons
 * file slugs (basename without variant suffix).
 *
 * When adding a connector in internal/providers/service.go, add a matching
 * entry here so the UI can resolve a brand icon.
 */
const HUB_TO_ICON_SLUG: Readonly<Record<string, string>> = {
  cursor: 'cursor',
  openrouter: 'openrouter',
  'opencode-go': 'opencode',
  opencode: 'opencode',
}

/**
 * Resolves a hub provider id, connection type, or AvailableProvider.logo value
 * to an ai-icons slug. Unknown values fall back to a sanitized form of the input.
 */
export function hubProviderIconSlug(providerKey: string): string {
  const key = providerKey.trim().toLowerCase()
  if (key.length === 0) {
    return ''
  }
  const mapped = HUB_TO_ICON_SLUG[key]
  if (mapped) {
    return mapped
  }
  // Soft fallback: strip common separators so "open-router" → still try openrouter-like keys later
  return key.replace(/_/g, '-')
}
