export const LOCALE_STORAGE_KEY = 'connexto:locale'

export const SUPPORTED_LOCALES = ['pt-BR', 'en'] as const

export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number]

export const DEFAULT_LOCALE: SupportedLocale = 'pt-BR'

export function isSupportedLocale(value: string): value is SupportedLocale {
  return (SUPPORTED_LOCALES as readonly string[]).includes(value)
}

export function resolveBrowserLocale(): SupportedLocale {
  if (typeof navigator === 'undefined') {
    return DEFAULT_LOCALE
  }
  const tag = navigator.language.toLowerCase()
  if (tag.startsWith('pt')) {
    return 'pt-BR'
  }
  return 'en'
}

export function readStoredLocale(): SupportedLocale | null {
  if (typeof window === 'undefined') {
    return null
  }
  try {
    const stored = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    if (stored !== null && isSupportedLocale(stored)) {
      return stored
    }
  } catch {
    // Ignore storage read errors
  }
  return null
}

export function resolveInitialLocale(): SupportedLocale {
  return readStoredLocale() ?? resolveBrowserLocale()
}

export function persistLocale(locale: SupportedLocale): void {
  if (typeof window === 'undefined') {
    return
  }
  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // Ignore storage write errors
  }
}

/** BCP-47 tag for Intl APIs (en maps to en-US for number grouping). */
export function intlLocaleFor(locale: SupportedLocale): string {
  return locale === 'en' ? 'en-US' : 'pt-BR'
}

export function htmlLangFor(locale: SupportedLocale): string {
  return locale === 'en' ? 'en' : 'pt-BR'
}
