import { afterEach, describe, expect, it } from 'vitest'
import { LOCALE_STORAGE_KEY, resolveBrowserLocale, resolveInitialLocale } from './locale'

describe('resolveInitialLocale', () => {
  afterEach(() => {
    window.localStorage.removeItem(LOCALE_STORAGE_KEY)
  })

  it('prefers stored locale', () => {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, 'en')
    expect(resolveInitialLocale()).toBe('en')
  })

  it('falls back to pt-BR when storage is empty and navigator is Portuguese', () => {
    const original = navigator.language
    Object.defineProperty(navigator, 'language', { value: 'pt-BR', configurable: true })
    expect(resolveInitialLocale()).toBe('pt-BR')
    Object.defineProperty(navigator, 'language', { value: original, configurable: true })
  })
})

describe('resolveBrowserLocale', () => {
  it('maps en locales to en', () => {
    const original = navigator.language
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true })
    expect(resolveBrowserLocale()).toBe('en')
    Object.defineProperty(navigator, 'language', { value: original, configurable: true })
  })
})
