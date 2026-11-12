import type { ReactNode } from 'react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'

type ResourcesPageHeadProps = {
  actions?: ReactNode
}

export function ResourcesPageHead({ actions }: ResourcesPageHeadProps) {
  return <AppPageHeader actions={actions} className="resources-pagehead" />
}
