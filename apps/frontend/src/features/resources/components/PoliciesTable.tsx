import { useTranslation } from 'react-i18next'
import type { QuotaUtilizationEntry, ResourcePolicy } from '../types'
import { PoliciesTableBody } from './PoliciesTableShared'

type PoliciesTableProps = {
  policies: ResourcePolicy[]
  utilization: QuotaUtilizationEntry[]
  isLoading?: boolean
  fetchedAtMs?: number
}

export function PoliciesTable({ policies, utilization, isLoading, fetchedAtMs }: PoliciesTableProps) {
  const { t } = useTranslation('resources')

  return (
    <PoliciesTableBody
      policies={policies}
      utilization={utilization}
      isLoading={isLoading}
      fetchedAtMs={fetchedAtMs}
      title={t('policies.title')}
      subtitle={t('policies.subtitle')}
    />
  )
}
