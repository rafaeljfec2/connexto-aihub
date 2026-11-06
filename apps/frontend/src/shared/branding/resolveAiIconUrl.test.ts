import { describe, expect, it } from 'vitest'
import { aiIconFilenameUrl } from './aiIconGlob'
import { hubProviderIconSlug } from './hubProviderIconSlug'
import { resolveAiIconUrl } from './resolveAiIconUrl'

describe('hubProviderIconSlug', () => {
  it('maps hub provider ids and logos to ai-icons slugs', () => {
    expect(hubProviderIconSlug('cursor')).toBe('cursor')
    expect(hubProviderIconSlug('openrouter')).toBe('openrouter')
    expect(hubProviderIconSlug('opencode-go')).toBe('opencode')
    expect(hubProviderIconSlug('opencode')).toBe('opencode')
    expect(hubProviderIconSlug('antigravity')).toBe('google')
    expect(hubProviderIconSlug('google')).toBe('google')
  })
})

describe('resolveAiIconUrl', () => {
  it('resolves known hub provider slugs', () => {
    expect(resolveAiIconUrl(hubProviderIconSlug('cursor'))).toBeTruthy()
    expect(resolveAiIconUrl(hubProviderIconSlug('openrouter'))).toBeTruthy()
    expect(resolveAiIconUrl(hubProviderIconSlug('opencode-go'))).toBeTruthy()
    expect(resolveAiIconUrl(hubProviderIconSlug('antigravity'))).toBeTruthy()
  })

  it('returns undefined for unknown slugs', () => {
    expect(resolveAiIconUrl('definitely-not-a-real-brand-xyz')).toBeUndefined()
  })

  it('prefers color when available, otherwise mono', () => {
    const openrouterAuto = resolveAiIconUrl('openrouter', { variant: 'auto' })
    const openrouterColor = resolveAiIconUrl('openrouter', { variant: 'color' })
    expect(openrouterAuto).toBe(openrouterColor)
    expect(openrouterColor).toBe(aiIconFilenameUrl('openrouter-color.svg'))

    // cursor has mono only in this vendor pin
    expect(aiIconFilenameUrl('cursor-color.svg')).toBeUndefined()
    expect(resolveAiIconUrl('cursor', { variant: 'auto' })).toBe(
      aiIconFilenameUrl('cursor.svg'),
    )
    expect(resolveAiIconUrl('cursor', { variant: 'color' })).toBeUndefined()
    expect(resolveAiIconUrl('cursor', { variant: 'mono' })).toBe(
      aiIconFilenameUrl('cursor.svg'),
    )
  })
})
