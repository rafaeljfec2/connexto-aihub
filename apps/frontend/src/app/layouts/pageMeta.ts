export type AppPageMeta = {
  crumb: string
  eyebrow: string
  title: string
  lead?: string
}

const overviewMeta: AppPageMeta = {
  crumb: 'Visão geral',
  eyebrow: 'CONTROL PLANE / OVERVIEW',
  title: 'Visão geral',
  lead: 'Acompanhe provedores, franquias e operação do gateway em um só lugar.',
}

const unknownMeta: AppPageMeta = {
  crumb: 'Page',
  eyebrow: 'CONTROL PLANE',
  title: 'Page',
}

export const pageMetaByPath: Record<string, AppPageMeta> = {
  '/': overviewMeta,
  '/providers': {
    crumb: 'Provedores',
    eyebrow: 'PROVIDER GOVERNANCE / QUOTAS',
    title: 'Franquias dos provedores',
    lead: 'Consumo, saldo disponível e limites de cada conexão em um só lugar.',
  },
  '/catalog': {
    crumb: 'Modelos',
    eyebrow: 'CATALOG / MODEL DISCOVERY',
    title: 'Catálogo de modelos',
    lead: 'Encontre e explore os modelos disponíveis nos seus provedores.',
  },
  '/playground': {
    crumb: 'Playground',
    eyebrow: 'INFERENCE / STREAMING',
    title: 'Playground',
    lead: 'Stream inference through POST /v1/responses (in-memory only).',
  },
  '/executions': {
    crumb: 'Execuções',
    eyebrow: 'OBSERVABILITY / GENERATIONS',
    title: 'Execuções',
    lead: 'Inspecione cada geração processada pelo gateway.',
  },
  '/resources': {
    crumb: 'Resources',
    eyebrow: 'GOVERNANCE / RESOURCES',
    title: 'Policies & concurrency',
    lead:
      'Cross-gateway quota policies, concurrency occupancy, and provider-reported usage from E09 admin APIs.',
  },
  '/intelligence': {
    crumb: 'Inteligência',
    eyebrow: 'INTELLIGENCE / LEARNING ENGINE',
    title: 'Aprendizagem & decisões',
    lead: 'Entenda como o contexto orienta a escolha de modelos — e quais evidências sustentam cada decisão.',
  },
}

export function resolvePageMeta(pathname: string): AppPageMeta {
  return pageMetaByPath[pathname] ?? unknownMeta
}
