import { initI18n } from './index'
import type { SupportedLocale } from './locale'

let testLocalePromise: Promise<unknown> | null = null

/** Initialize i18n for unit tests with an explicit locale (default pt-BR). */
export function ensureTestI18n(locale: SupportedLocale = 'pt-BR'): Promise<unknown> {
  if (!testLocalePromise) {
    testLocalePromise = initI18n(locale)
  }
  return testLocalePromise
}
