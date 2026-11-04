import type { ProviderQuotaMetric, DataScope } from '../types/quota'
import { formatQuotaValue } from '../lib/formatQuotaValue'

type OverviewQuotaMetricRowProps = Readonly<{
  accountId: string
  snapshotScope: DataScope
  metric: ProviderQuotaMetric
}>

function metricLabel(scope: DataScope, metric: ProviderQuotaMetric): string {
  if (scope === 'api_key' && metric.metric === 'usage') return 'Uso desta chave'
  if (scope === 'account' && metric.metric === 'total_usage') return 'Uso total da conta'
  if (scope === 'account' && metric.metric === 'total_credits') return 'Créditos comprados'
  return metric.label
}

function rawValues(metric: ProviderQuotaMetric): string[] {
  const values: string[] = []
  if (metric.used != null) values.push(`Usado ${formatQuotaValue(metric.used, metric.unit)}`)
  if (metric.limit != null) {
    const valueLabel = metric.metric === 'total_credits' ? 'Total' : 'Limite'
    values.push(`${valueLabel} ${formatQuotaValue(metric.limit, metric.unit)}`)
  }
  if (metric.remaining != null) values.push(`Restante ${formatQuotaValue(metric.remaining, metric.unit)}`)
  return values
}

export function OverviewQuotaMetricRow({ accountId, snapshotScope, metric }: OverviewQuotaMetricRowProps) {
  const values = rawValues(metric)
  const testId = `quota-metric-${accountId}-${snapshotScope}-${metric.metric}`

  return (
    <div className="flex flex-wrap items-baseline justify-between gap-x-2 gap-y-0.5" data-testid={testId}>
      <span className="text-[9px] font-medium text-[#CBD5E2]">{metricLabel(snapshotScope, metric)}</span>
      {values.length > 0 ? (
        <span className="flex flex-wrap justify-end gap-x-2 text-right text-[9px] text-[#A7B3C3]">
          {values.map((value) => <span key={value}>{value}</span>)}
        </span>
      ) : null}
    </div>
  )
}
