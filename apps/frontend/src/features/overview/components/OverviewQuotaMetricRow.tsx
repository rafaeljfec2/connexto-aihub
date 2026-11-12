import type { ProviderQuotaMetric, DataScope } from '../types/quota'
import { formatQuotaValue } from '../lib/formatQuotaValue'
import { formatResetsIn } from '../lib/formatResetsIn'

type OverviewQuotaMetricRowProps = Readonly<{
  accountId: string
  snapshotScope: DataScope
  metric: ProviderQuotaMetric
}>

function metricLabel(scope: DataScope, metric: ProviderQuotaMetric): string {
  if (scope === 'api_key' && metric.metric === 'usage') return 'Uso desta chave'
  if (scope === 'account' && metric.metric === 'total_usage') return 'Uso total da conta'
  if (scope === 'account' && metric.metric === 'total_credits') return 'Créditos comprados'
  if (metric.metric === 'go_rolling') return 'Franquia rolling'
  if (metric.metric === 'go_weekly') return 'Franquia semanal'
  if (metric.metric === 'go_monthly') return 'Franquia mensal'
  return metric.label || metric.metric
}

function formatMetricValue(scope: DataScope, metric: ProviderQuotaMetric, value: number): string {
  if (scope !== 'account' || metric.metric !== 'total_usage' || metric.unit !== 'USD') {
    return formatQuotaValue(value, metric.unit)
  }

  const formatted = value.toLocaleString('pt-BR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return `US$ ${formatted}`
}

function rawValues(scope: DataScope, metric: ProviderQuotaMetric): string[] {
  const values: string[] = []
  if (metric.remaining_percent != null) {
    values.push(`${metric.remaining_percent.toLocaleString('pt-BR')}% restante`)
  } else if (metric.consumed_percent != null) {
    values.push(`${metric.consumed_percent.toLocaleString('pt-BR')}%`)
  }
  if (metric.used != null) {
    const value = formatMetricValue(scope, metric, metric.used)
    values.push(scope === 'account' && metric.metric === 'total_usage' ? value : `Usado ${value}`)
  }
  if (metric.limit != null) {
    const value = formatMetricValue(scope, metric, metric.limit)
    if (scope === 'account' && metric.metric === 'total_credits') {
      values.push(value)
    } else {
      const valueLabel = metric.metric === 'total_credits' ? 'Total' : 'Limite'
      values.push(`${valueLabel} ${value}`)
    }
  }
  if (metric.remaining != null) values.push(`Restante ${formatQuotaValue(metric.remaining, metric.unit)}`)
  if (metric.resets_at != null) {
    const resetLabel = formatResetsIn(metric.resets_at)
    if (resetLabel) values.push(resetLabel)
  }
  return values
}

export function OverviewQuotaMetricRow({ accountId, snapshotScope, metric }: OverviewQuotaMetricRowProps) {
  const values = rawValues(snapshotScope, metric)
  const testId = `quota-metric-${accountId}-${snapshotScope}-${metric.metric}`

  return (
    <div className="flex flex-wrap items-baseline justify-between gap-x-2 gap-y-0.5" data-testid={testId}>
      <span className="text-[0.5625rem] font-medium text-[#CBD5E2]">{metricLabel(snapshotScope, metric)}</span>
      {values.length > 0 ? (
        <span className="flex flex-wrap justify-end gap-x-2 text-right text-[0.5625rem] text-[#A7B3C3]">
          {values.map((value) => <span key={value}>{value}</span>)}
        </span>
      ) : null}
    </div>
  )
}
