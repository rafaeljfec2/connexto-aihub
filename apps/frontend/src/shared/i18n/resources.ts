import accountEn from './locales/en/account.json'
import authEn from './locales/en/auth.json'
import catalogEn from './locales/en/catalog.json'
import commonEn from './locales/en/common.json'
import executionsEn from './locales/en/executions.json'
import intelligenceEn from './locales/en/intelligence.json'
import metaEn from './locales/en/meta.json'
import navEn from './locales/en/nav.json'
import overviewEn from './locales/en/overview.json'
import playgroundEn from './locales/en/playground.json'
import providersEn from './locales/en/providers.json'
import resourcesEn from './locales/en/resources.json'
import accountPtBR from './locales/pt-BR/account.json'
import authPtBR from './locales/pt-BR/auth.json'
import catalogPtBR from './locales/pt-BR/catalog.json'
import commonPtBR from './locales/pt-BR/common.json'
import executionsPtBR from './locales/pt-BR/executions.json'
import intelligencePtBR from './locales/pt-BR/intelligence.json'
import metaPtBR from './locales/pt-BR/meta.json'
import navPtBR from './locales/pt-BR/nav.json'
import overviewPtBR from './locales/pt-BR/overview.json'
import playgroundPtBR from './locales/pt-BR/playground.json'
import providersPtBR from './locales/pt-BR/providers.json'
import resourcesPtBR from './locales/pt-BR/resources.json'

export const i18nResources = {
  'pt-BR': {
    common: commonPtBR,
    nav: navPtBR,
    meta: metaPtBR,
    account: accountPtBR,
    auth: authPtBR,
    overview: overviewPtBR,
    providers: providersPtBR,
    catalog: catalogPtBR,
    intelligence: intelligencePtBR,
    executions: executionsPtBR,
    playground: playgroundPtBR,
    resources: resourcesPtBR,
  },
  en: {
    common: commonEn,
    nav: navEn,
    meta: metaEn,
    account: accountEn,
    auth: authEn,
    overview: overviewEn,
    providers: providersEn,
    catalog: catalogEn,
    intelligence: intelligenceEn,
    executions: executionsEn,
    playground: playgroundEn,
    resources: resourcesEn,
  },
} as const

export const I18N_NAMESPACES = [
  'common',
  'nav',
  'meta',
  'account',
  'auth',
  'overview',
  'providers',
  'catalog',
  'intelligence',
  'executions',
  'playground',
  'resources',
] as const
