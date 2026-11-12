import { Panel, PanelHeader } from '../../../shared/ui/panel'
import { cn } from '../../../shared/ui/cn'
import { isConcurrencySaturated } from '../lib/quotaUtilization'
import {
  concurrencyPolicyDisplayName,
  formatScopeLabel,
} from '../lib/scopeLabel'
import type { ConcurrencyPolicy, ConcurrencySnapshot } from '../types'
import { UtilizationStateBadge } from './UtilizationStateBadge'
import { SectionFetchedAt } from './SectionFetchedAt'

type ConcurrencySectionProps = {
  policies: ConcurrencyPolicy[]
  snapshots: ConcurrencySnapshot[]
  isLoading?: boolean
  snapshotFetchedAtMs?: number
  policiesFetchedAtMs?: number
}

export function ConcurrencySection({
  policies,
  snapshots,
  isLoading,
  snapshotFetchedAtMs,
  policiesFetchedAtMs,
}: ConcurrencySectionProps) {
  const snapshotByPolicy = new Map(snapshots.map((row) => [row.policy_id, row]))
  const enabledSnapshots = snapshots.filter((row) => row.enabled && row.concurrency_limit > 0)
  const totalActive = enabledSnapshots.reduce((sum, row) => sum + row.active_count, 0)
  const totalLimit = enabledSnapshots.reduce((sum, row) => sum + row.concurrency_limit, 0)
  const saturated = enabledSnapshots.some(isConcurrencySaturated)
  const utilizationPercent =
    totalLimit > 0 ? Math.min(100, Math.round((totalActive / totalLimit) * 1000) / 10) : null

  return (
    <div className="grid gap-3 lg:grid-cols-[1.05fr_0.95fr]">
      <Panel>
        <PanelHeader
          title="Concurrency utilization"
          subtitle="GET /api/resource-analytics/concurrency"
          action={
            <UtilizationStateBadge variant={saturated ? 'warn' : 'ok'}>
              {saturated ? 'Saturated' : 'Within capacity'}
            </UtilizationStateBadge>
          }
        />
        <div className="mb-1 flex items-baseline gap-2">
          <strong className="text-[1.375rem] font-semibold tracking-[-0.04em]">
            {totalActive}
            {totalLimit > 0 ? (
              <span className="text-xs font-normal text-[#8290A5]"> / {totalLimit}</span>
            ) : null}
          </strong>
          {utilizationPercent !== null ? (
            <span className="font-mono text-[0.5625rem] text-[#8FA0B6]">
              {utilizationPercent}% IN USE
            </span>
          ) : (
            <span className="font-mono text-[0.5625rem] text-[#8FA0B6]">NO ENABLED LIMITS</span>
          )}
        </div>
        {totalLimit > 0 ? (
          <div className="my-3 h-2.5 overflow-hidden rounded-full bg-[#2A3443]">
            <div
              className={cn('h-full rounded-full bg-cobalt', saturated && 'bg-warning')}
              style={{ width: `${Math.min(100, utilizationPercent ?? 0)}%` }}
            />
          </div>
        ) : null}
        <div className="grid gap-2">
          {enabledSnapshots.map((snapshot) => {
            const percent =
              snapshot.concurrency_limit > 0
                ? (snapshot.active_count / snapshot.concurrency_limit) * 100
                : 0
            return (
              <div
                key={snapshot.policy_id}
                className="rounded-[7px] border border-line/50 bg-[#171E29] p-2.5"
              >
                <div className="mb-1.5 flex items-center justify-between gap-2">
                  <strong className="text-[0.625rem] text-[#E1E8F2]">
                    {concurrencyPolicyDisplayName(
                      snapshot.scope,
                      snapshot.policy_id,
                      snapshot.provider_id,
                      snapshot.model_id,
                    )}
                  </strong>
                  <span className="font-mono text-[0.5625rem] text-[#9AA8BB]">
                    {snapshot.active_count} / {snapshot.concurrency_limit}
                  </span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-[#2A3443]">
                  <div
                    className={cn('h-full rounded-full bg-cobalt', snapshot.available <= 0 && 'bg-warning')}
                    style={{ width: `${Math.min(100, percent)}%` }}
                  />
                </div>
              </div>
            )
          })}
        {isLoading ? (
          <p className="text-[0.625rem] text-muted">Loading concurrency snapshot…</p>
        ) : enabledSnapshots.length === 0 ? (
            <p className="text-[0.625rem] text-muted">No enabled concurrency limits for the current filter.</p>
          ) : null}
        </div>
        <SectionFetchedAt updatedAtMs={snapshotFetchedAtMs} />
      </Panel>

      <Panel>
        <PanelHeader
          title="Concurrency policies"
          subtitle="GET /api/concurrency-policies"
        />
        {isLoading ? (
          <p className="text-[0.625rem] text-muted">Loading concurrency policies…</p>
        ) : policies.length === 0 ? (
          <p className="text-[0.625rem] text-muted">No concurrency policies configured.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[480px] border-collapse">
              <thead>
                <tr className="border-b border-line">
                  {['Policy', 'Scope', 'Limit', 'In use', 'State'].map((heading) => (
                    <th
                      key={heading}
                      className="px-2 py-2 text-left font-mono text-[0.5625rem] uppercase text-[#8290A5]"
                    >
                      {heading}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {policies.map((policy) => {
                  const snapshot = snapshotByPolicy.get(policy.id)
                  const saturatedRow =
                    snapshot !== undefined && isConcurrencySaturated(snapshot)
                  return (
                    <tr key={policy.id} className="border-b border-line/40">
                      <td className="px-2 py-2.5 text-[0.625rem] font-medium text-[#E1E8F2]">
                        {concurrencyPolicyDisplayName(
                          policy.scope,
                          policy.id,
                          policy.provider_id,
                          policy.model_id,
                        )}
                      </td>
                      <td className="px-2 py-2.5 font-mono text-[0.5625rem] text-[#9AA8BB]">
                        {formatScopeLabel(policy.scope, policy.provider_id, policy.model_id)}
                      </td>
                      <td className="px-2 py-2.5 text-[0.625rem] text-secondary">
                        {policy.concurrency_limit}
                      </td>
                      <td className="px-2 py-2.5 text-[0.625rem] text-secondary">
                        {snapshot ? `${snapshot.active_count} (${snapshot.available} free)` : '—'}
                      </td>
                      <td className="px-2 py-2.5">
                        <UtilizationStateBadge
                          variant={!policy.enabled ? 'muted' : saturatedRow ? 'warn' : 'ok'}
                        >
                          {!policy.enabled ? 'Disabled' : saturatedRow ? 'Saturated' : 'Active'}
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
      </Panel>
    </div>
  )
}
