export type AppPageMeta = {
  crumb: string
  eyebrow: string
  title: string
  lead?: string
}

const overviewMeta: AppPageMeta = {
  crumb: 'Overview',
  eyebrow: 'CONTROL PLANE / OVERVIEW',
  title: 'Operational overview',
  lead: 'Gateway readiness, provider diagnostics, and operational signals in one place.',
}

const unknownMeta: AppPageMeta = {
  crumb: 'Page',
  eyebrow: 'CONTROL PLANE',
  title: 'Page',
}

export const pageMetaByPath: Record<string, AppPageMeta> = {
  '/': overviewMeta,
  '/providers': {
    crumb: 'Providers',
    eyebrow: 'PROVIDER GOVERNANCE / ADMIN',
    title: 'Provider administration',
    lead: 'Manage connectivity credentials and enablement. Secrets are write-only and never displayed from the API.',
  },
  '/catalog': {
    crumb: 'Models',
    eyebrow: 'CATALOG / MODEL DISCOVERY',
    title: 'Model catalog',
    lead: 'Browse normalized models and trigger catalog synchronization.',
  },
  '/playground': {
    crumb: 'Playground',
    eyebrow: 'INFERENCE / STREAMING',
    title: 'Playground',
    lead: 'Stream inference through POST /v1/responses (in-memory only).',
  },
  '/executions': {
    crumb: 'Executions',
    eyebrow: 'OBSERVABILITY / EXECUTIONS',
    title: 'Executions',
    lead: 'Review content-free gateway execution records and audit metadata.',
  },
  '/resources': {
    crumb: 'Resources',
    eyebrow: 'GOVERNANCE / RESOURCES',
    title: 'Policies & concurrency',
    lead:
      'Cross-gateway quota policies, concurrency occupancy, and provider-reported usage from E09 admin APIs.',
  },
}

export function resolvePageMeta(pathname: string): AppPageMeta {
  return pageMetaByPath[pathname] ?? unknownMeta
}
