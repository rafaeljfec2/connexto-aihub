import { Panel, PanelHeader, GridPagination } from '../../../shared/ui'
import { useGridPagination } from '../../../shared/hooks/useGridPagination'
import { mergePoliciesWithUtilization } from '../lib/summaryMetrics'
import {
  formatUtilizationDisplay,
  isNearQuotaLimit,
  quotaUtilizationDisplay,
} from '../lib/quotaUtilization'
import { formatScopeLabel, policyDisplayName } from '../lib/scopeLabel'
import type { QuotaUtilizationEntry, ResourcePolicy } from '../types'
import { UtilizationStateBadge } from './UtilizationStateBadge'
import { SectionFetchedAt } from './SectionFetchedAt'

type QuotaPoliciesTableProps = {
  policies: ResourcePolicy[]
  utilization: QuotaUtilizationEntry[]
  compact?: boolean
  isLoading?: boolean
  fetchedAtMs?: number
}

export function QuotaPoliciesTable({
  policies,
  utilization,
  compact,
  isLoading,
  fetchedAtMs,
}: QuotaPoliciesTableProps) {
  const rows = mergePoliciesWithUtilization(policies, utilization)
  const pagination = useGridPagination(rows)
  const visibleRows = compact ? rows.slice(0, 5) : pagination.pageItems

  return (
    <Panel>
      <PanelHeader
        title={compact ? 'Highlighted quota policies' : 'Resource quota policies'}
        subtitle="GET /api/resource-policies + /api/resource-analytics/quota-utilization"
      />
      {isLoading ? (
        <p className="text-[10px] text-muted">Loading quota policies…</p>
      ) : rows.length === 0 ? (
        <p className="text-[10px] text-muted">No resource policies configured.</p>
      ) : (
        <div className="grid-table-scroll scrollbar-slim overflow-x-auto">
          <table className="w-full min-w-[520px] border-collapse text-left">
            <thead>
              <tr className="border-b border-line">
                {['Policy', 'Scope', 'Limit', 'Utilization', 'State'].map((heading) => (
                  <th
                    key={heading}
                    className="px-2 py-2 font-mono text-[9px] uppercase tracking-[0.03em] text-[#8290A5]"
                  >
                    {heading}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {visibleRows.map(({ policy, utilization: usageRow }) => {
                const display = usageRow ? quotaUtilizationDisplay(usageRow) : null
                const near = usageRow ? isNearQuotaLimit(usageRow) : false
                const stateVariant =
                  !policy.enabled || display?.kind === 'na'
                    ? 'muted'
                    : near
                      ? 'warn'
                      : 'ok'
                const stateLabel =
                  !policy.enabled
                    ? 'Disabled'
                    : display?.kind === 'na'
                      ? 'No limit'
                      : near
                        ? 'Near limit'
                        : 'Within limit'
                return (
                  <tr key={policy.id} className="border-b border-line/40">
                    <td className="px-2 py-2.5 text-[10px] font-medium text-[#E1E8F2]">
                      {policyDisplayName(policy.scope, policy.id, policy.provider_id, policy.model_id)}
                    </td>
                    <td className="px-2 py-2.5 font-mono text-[9px] text-[#9AA8BB]">
                      {formatScopeLabel(policy.scope, policy.provider_id, policy.model_id)}
                    </td>
                    <td className="px-2 py-2.5 text-[10px] text-secondary">
                      {policy.daily_limit > 0 ? policy.daily_limit.toLocaleString('en-US') : '—'}
                    </td>
                    <td className="px-2 py-2.5 text-[10px] text-secondary">
                      {display ? formatUtilizationDisplay(display) : '—'}
                      {usageRow && display?.kind === 'percent' ? (
                        <span className="mt-0.5 block text-[9px] text-muted">
                          {usageRow.admitted_count} admitted · {usageRow.remaining_requests} remaining
                        </span>
                      ) : null}
                    </td>
                    <td className="px-2 py-2.5">
                      <UtilizationStateBadge variant={stateVariant}>{stateLabel}</UtilizationStateBadge>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      {!compact && rows.length > 0 ? (
        <GridPagination
          meta={pagination.meta}
          onPreviousPage={pagination.goToPreviousPage}
          onNextPage={pagination.goToNextPage}
          pageSize={pagination.pageSize}
          onPageSizeChange={pagination.setPageSize}
          className="grid-panel-pagination"
        />
      ) : null}
      <SectionFetchedAt updatedAtMs={fetchedAtMs} />
    </Panel>
  )
}
