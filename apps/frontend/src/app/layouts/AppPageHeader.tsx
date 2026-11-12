import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation } from 'react-router-dom'
import { PageHeader } from '../../shared/ui'
import { resolvePageMeta } from './pageMeta'

/**
 * Page title block only. Horizontal layout is owned by AppShell (`appShellLayout.ts`).
 */
export function useAppPageMeta() {
  const { pathname } = useLocation()
  const { t } = useTranslation('meta')
  return resolvePageMeta(pathname, t)
}

type AppPageHeaderProps = {
  actions?: ReactNode
  className?: string
}

export function AppPageHeader({ actions, className }: AppPageHeaderProps) {
  const meta = useAppPageMeta()
  return (
    <PageHeader
      className={className}
      eyebrow={meta.eyebrow}
      title={meta.title}
      lead={meta.lead}
      actions={actions}
    />
  )
}
