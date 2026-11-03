import type { ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { PageHeader } from '../../shared/ui'
import { resolvePageMeta } from './pageMeta'

export function useAppPageMeta() {
  const { pathname } = useLocation()
  return resolvePageMeta(pathname)
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
