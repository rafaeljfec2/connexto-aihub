import { Panel, PanelHeader } from '../../../shared/ui/panel'
import { formatTokenCount } from '../lib/formatTokens'
import type { ProviderUsageDistributionResponse, ProviderUsageResponse } from '../types'
import { SectionFetchedAt } from './SectionFetchedAt'

type ProviderUsageSectionProps = {
  usage: ProviderUsageResponse | undefined
  distribution: ProviderUsageDistributionResponse | undefined
  isLoading?: boolean
  isError?: boolean
  isDistributionError?: boolean
  usageFetchedAtMs?: number
  distributionFetchedAtMs?: number
}

export function ProviderUsageSection({
  usage,
  distribution,
  isLoading,
  isError,
  isDistributionError,
  usageFetchedAtMs,
  distributionFetchedAtMs,
}: ProviderUsageSectionProps) {
  if (isLoading) {
    return (
      <Panel>
        <PanelHeader
          title="Provider-reported usage"
          subtitle="Metering aggregates (E09.5) — not admission quotas"
        />
        <p className="text-[10px] text-muted">Loading provider usage analytics…</p>
      </Panel>
    )
  }

  if (isError || !usage) {
    return (
      <Panel>
        <PanelHeader
          title="Provider-reported usage"
          subtitle="Metering aggregates (E09.5) — not admission quotas"
        />
        <p className="text-[10px] text-muted">
          Provider usage data is unavailable for the selected range.
        </p>
      </Panel>
    )
  }

  return (
    <div className="grid gap-3">
      <Panel>
        <PanelHeader
          title="Provider-reported usage"
          subtitle="GET /api/resource-analytics/provider-usage — separate from admission analytics"
        />
        <div className="grid gap-3 sm:grid-cols-3">
          <TokenStat label="Input tokens" value={usage.input_tokens} />
          <TokenStat label="Output tokens" value={usage.output_tokens} />
          <TokenStat label="Total tokens" value={usage.total_tokens} />
        </div>
        <p className="mt-3 text-[9px] leading-relaxed text-[#758196]">
          Null fields mean unknown provider metrics, not zero. Cache tokens are not exposed by the
          current API contract.
        </p>
        <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <MeasurementStat label="Reported" value={usage.reported_executions} tone="ok" />
          <MeasurementStat label="Partial" value={usage.partial_executions} tone="warn" />
          <MeasurementStat label="Unavailable" value={usage.unavailable_executions} tone="muted" />
          <MeasurementStat label="Invalid" value={usage.invalid_executions} tone="muted" />
        </div>
        <SectionFetchedAt updatedAtMs={usageFetchedAtMs} />
      </Panel>

      {isDistributionError ? (
        <p className="text-[10px] text-warning" role="status">
          Provider usage summary loaded, but the distribution endpoint failed. Breakdown tables
          may be empty until you refresh.
        </p>
      ) : null}

      <div className="grid gap-3 xl:grid-cols-2">
        <DistributionTable
          title="By provider"
          subtitle="GET /api/resource-analytics/provider-usage/distribution"
          unavailable={isDistributionError}
          fetchedAtMs={distributionFetchedAtMs}
          rows={(distribution?.by_provider ?? []).map((row) => ({
            key: `${row.period_start}-${row.provider_id}`,
            primary: row.provider_id,
            secondary: row.period_start.slice(0, 10),
            input: row.input_tokens,
            output: row.output_tokens,
            total: row.total_tokens,
            executions: row.executions,
          }))}
        />
        <DistributionTable
          title="By model"
          subtitle="Model-level provider usage rows"
          unavailable={isDistributionError}
          fetchedAtMs={distributionFetchedAtMs}
          rows={(distribution?.by_model ?? []).map((row) => ({
            key: `${row.period_start}-${row.provider_id}-${row.model_id}`,
            primary: `${row.provider_id}/${row.model_id}`,
            secondary: row.period_start.slice(0, 10),
            input: row.input_tokens,
            output: row.output_tokens,
            total: row.total_tokens,
            executions: row.executions,
          }))}
        />
      </div>
    </div>
  )
}

function TokenStat({ label, value }: { label: string; value: number | null }) {
  return (
    <div className="rounded-[8px] border border-line/60 bg-[#171E29] px-3 py-2.5">
      <span className="block text-[10px] text-[#A0ADBF]">{label}</span>
      <strong className="mt-1 block text-lg font-semibold tracking-[-0.03em]">
        {formatTokenCount(value)}
      </strong>
    </div>
  )
}

function MeasurementStat({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: 'ok' | 'warn' | 'muted'
}) {
  const toneClass =
    tone === 'ok' ? 'text-healthy' : tone === 'warn' ? 'text-warning' : 'text-muted'
  return (
    <div className="rounded-[7px] border border-line/50 bg-[#171E29] px-2.5 py-2">
      <span className="block font-mono text-[9px] uppercase text-[#8290A5]">{label}</span>
      <strong className={`mt-1 block text-base font-semibold ${toneClass}`}>{value}</strong>
      <span className="mt-0.5 block text-[9px] text-muted">executions</span>
    </div>
  )
}

function DistributionTable({
  title,
  subtitle,
  rows,
  unavailable,
  fetchedAtMs,
}: {
  title: string
  subtitle: string
  rows: Array<{
    key: string
    primary: string
    secondary: string
    input: number | null
    output: number | null
    total: number | null
    executions: number
  }>
  unavailable?: boolean
  fetchedAtMs?: number
}) {
  return (
    <Panel>
      <PanelHeader title={title} subtitle={subtitle} />
      {unavailable ? (
        <p className="text-[10px] text-muted">Distribution data unavailable.</p>
      ) : rows.length === 0 ? (
        <p className="text-[10px] text-muted">No distribution rows for this range and filter.</p>
      ) : (
        <div className="max-h-[320px] overflow-auto">
          <table className="w-full min-w-[420px] border-collapse text-left">
            <thead className="sticky top-0 bg-surface">
              <tr className="border-b border-line">
                {['Scope', 'Day', 'In', 'Out', 'Total', 'Exec'].map((heading) => (
                  <th
                    key={heading}
                    className="px-2 py-2 font-mono text-[9px] uppercase text-[#8290A5]"
                  >
                    {heading}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.key} className="border-b border-line/30">
                  <td className="px-2 py-2 text-[10px] text-[#E1E8F2]">{row.primary}</td>
                  <td className="px-2 py-2 font-mono text-[9px] text-muted">{row.secondary}</td>
                  <td className="px-2 py-2 text-[10px]">{formatTokenCount(row.input)}</td>
                  <td className="px-2 py-2 text-[10px]">{formatTokenCount(row.output)}</td>
                  <td className="px-2 py-2 text-[10px]">{formatTokenCount(row.total)}</td>
                  <td className="px-2 py-2 font-mono text-[9px]">{row.executions}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <SectionFetchedAt updatedAtMs={fetchedAtMs} />
    </Panel>
  )
}
