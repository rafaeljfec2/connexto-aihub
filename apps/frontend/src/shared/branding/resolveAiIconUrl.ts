import { aiIconFilenameUrl } from './aiIconGlob'

export type AiIconVariant = 'auto' | 'color' | 'mono'

export type ResolveAiIconUrlOptions = {
  variant?: AiIconVariant
}

/**
 * Resolves a vendor SVG URL for an ai-icons slug.
 * - `auto` prefers `{slug}-color.svg`, then `{slug}.svg`
 * - `color` only tries the color variant
 * - `mono` only tries the mono variant
 */
export function resolveAiIconUrl(
  slug: string,
  options: ResolveAiIconUrlOptions = {},
): string | undefined {
  const normalized = slug.trim().toLowerCase()
  if (normalized.length === 0) {
    return undefined
  }

  const variant = options.variant ?? 'auto'
  const colorFile = `${normalized}-color.svg`
  const monoFile = `${normalized}.svg`

  if (variant === 'color') {
    return aiIconFilenameUrl(colorFile)
  }
  if (variant === 'mono') {
    return aiIconFilenameUrl(monoFile)
  }

  return aiIconFilenameUrl(colorFile) ?? aiIconFilenameUrl(monoFile)
}
