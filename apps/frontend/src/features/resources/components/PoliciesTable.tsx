import type { QuotaUtilizationEntry, ResourcePolicy } from '../types'
import { PoliciesTableBody } from './PoliciesTableShared'

type PoliciesTableProps = {
  policies: ResourcePolicy[]
  utilization: QuotaUtilizationEntry[]
  isLoading?: boolean
  fetchedAtMs?: number
}

export function PoliciesTable({ policies, utilization, isLoading, fetchedAtMs }: PoliciesTableProps) {
  return (
    <PoliciesTableBody
      policies={policies}
      utilization={utilization}
      isLoading={isLoading}
      fetchedAtMs={fetchedAtMs}
      title="Políticas de recursos"
      subtitle="Regras retornadas por /api/resource-policies"
    />
  )
}
