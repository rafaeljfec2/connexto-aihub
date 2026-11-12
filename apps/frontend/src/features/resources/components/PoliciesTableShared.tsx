import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { GridPagination } from '../../../shared/ui'
import { useGridPagination } from '../../../shared/hooks/useGridPagination'
import { formatNumber } from '../../../shared/i18n/format'
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

type PoliciesTableBodyProps = {
  policies: ResourcePolicy[]
  utilization: QuotaUtilizationEntry[]
  compact?: boolean
  isLoading?: boolean
  fetchedAtMs?: number
  title: string
  subtitle: string
  headerAction?: ReactNode
}

export function PoliciesTableBody({
  policies,
  utilization,
  compact,
  isLoading,
  fetchedAtMs,
  title,
  subtitle,
  headerAction,
}: PoliciesTableBodyProps) {
  const { t } = useTranslation('resources')
  const rows = mergePoliciesWithUtilization(policies, utilization)
  const pagination = useGridPagination(rows)
  const visibleRows = compact ? rows.slice(0, 5) : pagination.pageItems

  const tableHeadings = compact
    ? ([
        t('policies.table.policy'),
        t('policies.table.scope'),
        t('policies.table.utilization'),
        t('policies.table.state'),
      ] as const)
    : ([
        t('policies.table.policy'),
        t('policies.table.scope'),
        t('policies.table.definedLimit'),
        t('policies.table.utilization'),
        t('policies.table.state'),
      ] as const)

  return (
    <section className="resources-panel">
      <div className="resources-panel-head">
        <div>
          <h3 className="resources-panel-title">{title}</h3>
          <p className="resources-panel-subtitle">{subtitle}</p>
        </div>
        {headerAction}
      </div>
      {isLoading ? (
        <p className="resources-empty">{t('policies.loading')}</p>
      ) : rows.length === 0 ? (
        <p className="resources-empty">{t('policies.empty')}</p>
      ) : (
        <div className="resources-table-scroll scrollbar-slim">
          <table className="resources-table">
            <thead>
              <tr className="border-b border-line">
                {tableHeadings.map((heading) => (
                  <th key={heading}>{heading}</th>
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
                const stateLabel = !policy.enabled
                  ? t('state.inactive')
                  : display?.kind === 'na'
                    ? t('state.inactive')
                    : near
                      ? compact
                        ? t('state.nearLimit')
                        : t('state.attention')
                      : compact
                        ? t('state.withinLimit')
                        : t('state.active')
                const utilizationLabel =
                  display?.kind === 'percent'
                    ? compact
                      ? t('utilization.percentOfLimitCompact', {
                          value: display.value.toFixed(0),
                        })
                      : formatUtilizationDisplay(display)
                    : '—'
                return (
                  <tr key={policy.id}>
                    <td className="font-medium text-[#E1E8F2]">
                      {policyDisplayName(policy.scope, policy.id, policy.provider_id, policy.model_id)}
                    </td>
                    <td className="font-mono text-[0.5625rem] uppercase text-[#9AA8BB]">
                      {formatScopeLabel(policy.scope, policy.provider_id, policy.model_id)}
                    </td>
                    {compact ? null : (
                      <td>
                        {policy.daily_limit > 0 ? formatNumber(policy.daily_limit) : '—'}
                      </td>
                    )}
                    <td>{utilizationLabel}</td>
                    <td>
                      <UtilizationStateBadge variant={stateVariant}>{stateLabel}</UtilizationStateBadge>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      {!compact ? (
        <p className="resources-measurement-note">{t('policies.measurementNote')}</p>
      ) : null}
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
    </section>
  )
}
