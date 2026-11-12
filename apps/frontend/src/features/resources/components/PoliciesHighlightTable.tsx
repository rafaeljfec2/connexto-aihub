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
  return (
    <PoliciesTableBody
      policies={policies}
      utilization={utilization}
      compact
      isLoading={isLoading}
      fetchedAtMs={fetchedAtMs}
      title="Políticas em destaque"
      subtitle="Regras configuradas para controlar recursos"
      headerAction={
        onViewPolicies ? (
          <button type="button" className="resources-select" onClick={onViewPolicies}>
            Ver políticas ↗
          </button>
        ) : null
      }
    />
  )
}
