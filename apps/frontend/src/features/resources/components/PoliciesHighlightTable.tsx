import { useTranslation } from 'react-i18next'
import type { QuotaUtilizationEntry, ResourcePolicy } from '../types'
import { PoliciesTableBody } from './PoliciesTableShared'

type PoliciesHighlightTableProps = {
  policies: ResourcePolicy[]
  utilization: QuotaUtilizationEntry[]
  isLoading?: boolean
  fetchedAtMs?: number
  onViewPolicies?: () => void
}

export function PoliciesHighlightTable({
  policies,
  utilization,
  isLoading,
  fetchedAtMs,
  onViewPolicies,
}: PoliciesHighlightTableProps) {
  const { t } = useTranslation('resources')

  return (
    <PoliciesTableBody
      policies={policies}
      utilization={utilization}
      compact
      isLoading={isLoading}
      fetchedAtMs={fetchedAtMs}
      title={t('policies.highlightTitle')}
      subtitle={t('policies.highlightSubtitle')}
      headerAction={
        onViewPolicies ? (
          <button type="button" className="resources-select" onClick={onViewPolicies}>
            {t('policies.viewPolicies')}
          </button>
        ) : null
      }
    />
  )
}
