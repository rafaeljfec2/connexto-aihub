import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'
import { formatNumber } from '../../../shared/i18n/format'
import { globalConcurrencyOccupancy } from '../lib/summaryMetrics'
import { isConcurrencySaturated } from '../lib/quotaUtilization'
import { concurrencyPolicyDisplayName, formatScopeLabel } from '../lib/scopeLabel'
import type { ConcurrencyPolicy, ConcurrencySnapshot } from '../types'
import { UtilizationStateBadge } from './UtilizationStateBadge'
import { SectionFetchedAt } from './SectionFetchedAt'

type ConcurrencyPanelProps = {
  policies: ConcurrencyPolicy[]
  snapshots: ConcurrencySnapshot[]
  isLoading?: boolean
  snapshotFetchedAtMs?: number
  policiesFetchedAtMs?: number
}

export function ConcurrencyPanel({
  policies,
  snapshots,
  isLoading,
  snapshotFetchedAtMs,
  policiesFetchedAtMs,
}: ConcurrencyPanelProps) {
  const { t } = useTranslation('resources')
  const snapshotByPolicy = new Map(snapshots.map((row) => [row.policy_id, row]))
  const { active: globalActive, limit: globalLimit } = globalConcurrencyOccupancy(snapshots)
  const globalSaturated =
    globalLimit > 0 &&
    snapshots.some(
      (row) => row.scope === 'global' && row.enabled && isConcurrencySaturated(row),
    )
  const globalPercent =
    globalLimit > 0 ? Math.min(100, Math.round((globalActive / globalLimit) * 1000) / 10) : null

  const distributionRows = snapshots.filter((row) => row.enabled && row.concurrency_limit > 0)

  const tableHeadings = [
    t('concurrency.policies.table.policy'),
    t('concurrency.policies.table.scope'),
    t('concurrency.policies.table.limit'),
    t('concurrency.policies.table.inUse'),
    t('concurrency.policies.table.state'),
  ] as const

  return (
    <div className="resources-concurrency-stack">
      <div className="resources-overview-grid">
      <section className="resources-panel">
        <div className="resources-panel-head">
          <div>
            <h3 className="resources-panel-title">{t('concurrency.usage.title')}</h3>
            <p className="resources-panel-subtitle">{t('concurrency.usage.subtitle')}</p>
          </div>
          <UtilizationStateBadge
            variant={globalLimit <= 0 ? 'muted' : globalSaturated ? 'warn' : 'ok'}
          >
            {globalLimit <= 0
              ? t('state.noGlobalLimit')
              : globalSaturated
                ? t('state.limitReached')
                : t('state.withinLimit')}
          </UtilizationStateBadge>
        </div>

        <div className="resources-consumption-hero">
          <strong>
            {globalLimit > 0 ? (
              <>
                {globalActive}
                <span className="resources-metric-suffix"> / {globalLimit}</span>
              </>
            ) : (
              '—'
            )}
          </strong>
          {globalPercent !== null ? (
            <span>
              {t('concurrency.usage.percentInUse', {
                value: formatNumber(globalPercent),
              })}
            </span>
          ) : (
            <span>{t('concurrency.usage.noGlobalLimit')}</span>
          )}
        </div>

        {globalLimit > 0 ? (
          <>
            <div className="resources-concurrency-bar">
              <div
                className={cn('resources-concurrency-bar-fill', globalSaturated && 'is-warn')}
                style={{ width: `${Math.min(100, globalPercent ?? 0)}%` }}
              />
            </div>
            <div className="resources-chart-labels">
              <span>{t('concurrency.usage.chartZeroSlots')}</span>
              <span>{t('concurrency.usage.chartConfiguredSlots', { limit: globalLimit })}</span>
            </div>
          </>
        ) : null}

        <div className="resources-notice">
          <p>{t('concurrency.usage.notice')}</p>
        </div>
        <SectionFetchedAt updatedAtMs={snapshotFetchedAtMs} />
      </section>

      <section className="resources-panel">
        <div className="resources-panel-head">
          <div>
            <h3 className="resources-panel-title">{t('concurrency.distribution.title')}</h3>
            <p className="resources-panel-subtitle">{t('concurrency.distribution.subtitle')}</p>
          </div>
        </div>
        <div className="resources-scope-list">
          {distributionRows.map((snapshot) => {
            const percent =
              snapshot.concurrency_limit > 0
                ? (snapshot.active_count / snapshot.concurrency_limit) * 100
                : 0
            const saturated = snapshot.available <= 0
            return (
              <div key={snapshot.policy_id} className="resources-scope-card">
                <div className="mb-1.5 flex items-center justify-between gap-2">
                  <strong className="text-[10px] text-[#E1E8F2]">
                    {concurrencyPolicyDisplayName(
                      snapshot.scope,
                      snapshot.policy_id,
                      snapshot.provider_id,
                      snapshot.model_id,
                    )}
                  </strong>
                  <span className="font-mono text-[9px] text-[#9AA8BB]">
                    {snapshot.active_count} / {snapshot.concurrency_limit}
                  </span>
                </div>
                <div className="resources-distribution-track">
                  <div
                    className={cn('resources-distribution-fill', saturated && 'is-warn')}
                    style={{ width: `${Math.min(100, percent)}%` }}
                  />
                </div>
                <small className="mt-1.5 block text-[9px] text-[#7D8A9D]">
                  {formatScopeLabel(snapshot.scope, snapshot.provider_id, snapshot.model_id)}
                </small>
              </div>
            )
          })}
          {isLoading ? (
            <p className="resources-empty">{t('concurrency.distribution.loading')}</p>
          ) : distributionRows.length === 0 ? (
            <p className="resources-empty">{t('concurrency.distribution.empty')}</p>
          ) : null}
        </div>
      </section>
      </div>

      <section className="resources-panel">
        <div className="resources-panel-head">
          <div>
            <h3 className="resources-panel-title">{t('concurrency.policies.title')}</h3>
            <p className="resources-panel-subtitle">{t('concurrency.policies.subtitle')}</p>
          </div>
          <span className="font-mono text-[9px] text-[#8290A5]">/api/concurrency-policies</span>
        </div>
        {isLoading ? (
          <p className="resources-empty">{t('concurrency.policies.loading')}</p>
        ) : policies.length === 0 ? (
          <p className="resources-empty">{t('concurrency.policies.empty')}</p>
        ) : (
          <div className="resources-table-scroll scrollbar-slim">
            <table className="resources-table min-w-[480px]">
              <thead>
                <tr className="border-b border-line">
                  {tableHeadings.map((heading) => (
                    <th key={heading}>{heading}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {policies.map((policy) => {
                  const snapshot = snapshotByPolicy.get(policy.id)
                  const saturatedRow =
                    snapshot !== undefined && isConcurrencySaturated(snapshot)
                  return (
                    <tr key={policy.id}>
                      <td className="font-medium text-[#E1E8F2]">
                        {concurrencyPolicyDisplayName(
                          policy.scope,
                          policy.id,
                          policy.provider_id,
                          policy.model_id,
                        )}
                      </td>
                      <td className="font-mono text-[0.5625rem] text-[#9AA8BB]">
                        {formatScopeLabel(policy.scope, policy.provider_id, policy.model_id)}
                      </td>
                      <td>{policy.concurrency_limit}</td>
                      <td>{snapshot ? snapshot.active_count : '—'}</td>
                      <td>
                        <UtilizationStateBadge
                          variant={!policy.enabled ? 'muted' : saturatedRow ? 'warn' : 'ok'}
                        >
                          {!policy.enabled
                            ? t('state.inactive')
                            : saturatedRow
                              ? t('state.limitReached')
                              : t('state.active')}
                        </UtilizationStateBadge>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
        <SectionFetchedAt updatedAtMs={policiesFetchedAtMs} />
      </section>
    </div>
  )
}
