import type { TFunction } from 'i18next'

export type AppPageMetaKey = {
  key: string
}

export type AppPageMeta = {
  crumb: string
  eyebrow: string
  title: string
  lead?: string
}

const unknownMetaKey: AppPageMetaKey = { key: 'unknown' }

export const pageMetaKeyByPath: Record<string, AppPageMetaKey> = {
  '/': { key: 'overview' },
  '/providers': { key: 'providers' },
  '/catalog': { key: 'catalog' },
  '/playground': { key: 'playground' },
  '/executions': { key: 'executions' },
  '/resources': { key: 'resources' },
  '/intelligence': { key: 'intelligence' },
  '/intelligence/help': { key: 'intelligenceHelp' },
  '/profile': { key: 'profile' },
  '/settings': { key: 'settings' },
}

export function resolvePageMeta(pathname: string, t: TFunction<'meta'>): AppPageMeta {
  const entry = pageMetaKeyByPath[pathname] ?? unknownMetaKey
  const base = `${entry.key}`
  const lead = t(`${base}.lead`, { defaultValue: '' })
  return {
    crumb: t(`${base}.crumb`),
    eyebrow: t(`${base}.eyebrow`),
    title: t(`${base}.title`),
    ...(lead ? { lead } : {}),
  }
}
