import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import {
  DEFAULT_LOCALE,
  htmlLangFor,
  persistLocale,
  resolveInitialLocale,
  type SupportedLocale,
} from './locale'
import { I18N_NAMESPACES, i18nResources } from './resources'

export { formatCurrencyUsd, formatDateTime, formatNumber, formatPercent } from './format'
export {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  SUPPORTED_LOCALES,
  htmlLangFor,
  intlLocaleFor,
  isSupportedLocale,
  persistLocale,
  readStoredLocale,
  resolveBrowserLocale,
  resolveInitialLocale,
  type SupportedLocale,
} from './locale'
export { I18N_NAMESPACES, i18nResources } from './resources'

function syncDocumentLanguage(locale: SupportedLocale): void {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = htmlLangFor(locale)
  }
}

export async function initI18n(initialLocale?: SupportedLocale): Promise<typeof i18n> {
  const locale = initialLocale ?? resolveInitialLocale()
  syncDocumentLanguage(locale)

  if (i18n.isInitialized) {
    await i18n.changeLanguage(locale)
    return i18n
  }

  await i18n.use(initReactI18next).init({
    resources: i18nResources,
    lng: locale,
    fallbackLng: DEFAULT_LOCALE,
    supportedLngs: ['pt-BR', 'en'],
    ns: [...I18N_NAMESPACES],
    defaultNS: 'common',
    interpolation: { escapeValue: false },
    react: { useSuspense: false },
  })

  i18n.on('languageChanged', (lng) => {
    const resolved = lng === 'en' ? 'en' : 'pt-BR'
    syncDocumentLanguage(resolved)
  })

  return i18n
}

export async function setLocale(locale: SupportedLocale): Promise<void> {
  persistLocale(locale)
  await i18n.changeLanguage(locale)
}

export default i18n
