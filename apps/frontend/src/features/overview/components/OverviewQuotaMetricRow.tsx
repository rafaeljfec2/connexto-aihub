import type { ProviderQuotaMetric, DataScope } from '../types/quota'
import { formatNumber } from '../../../shared/i18n/format'
import { formatQuotaValue } from '../lib/formatQuotaValue'
import { formatResetsIn } from '../lib/formatResetsIn'
import {
  overviewConsumedPercentLabel,
  overviewLimitValueLabel,
  overviewMetricLabel,
  overviewRemainingPercentLabel,
  overviewRemainingValueLabel,
  overviewTotalValueLabel,
  overviewUsedValueLabel,
} from '../lib/overviewQuotaLabels'

type OverviewQuotaMetricRowProps = Readonly<{
  accountId: string
  snapshotScope: DataScope
  metric: ProviderQuotaMetric
}>

function formatMetricValue(scope: DataScope, metric: ProviderQuotaMetric, value: number): string {
  if (scope === 'account' && metric.metric === 'total_usage' && metric.unit === 'USD') {
    return `US$ ${formatNumber(value, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
  }
  return formatQuotaValue(value, metric.unit)
}

function rawValues(scope: DataScope, metric: ProviderQuotaMetric): string[] {
  const values: string[] = []
  if (metric.remaining_percent != null) {
    values.push(overviewRemainingPercentLabel(metric.remaining_percent))
  } else if (metric.consumed_percent != null) {
    values.push(overviewConsumedPercentLabel(metric.consumed_percent))
  }
  if (metric.used != null) {
    const value = formatMetricValue(scope, metric, metric.used)
    values.push(
      scope === 'account' && metric.metric === 'total_usage'
        ? value
        : overviewUsedValueLabel(value),
    )
  }
  if (metric.limit != null) {
    const value = formatMetricValue(scope, metric, metric.limit)
    if (scope === 'account' && metric.metric === 'total_credits') {
      values.push(value)
    } else if (metric.metric === 'total_credits') {
      values.push(overviewTotalValueLabel(value))
    } else {
      values.push(overviewLimitValueLabel(value))
    }
  }
  if (metric.remaining != null) {
    values.push(overviewRemainingValueLabel(formatQuotaValue(metric.remaining, metric.unit)))
  }
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
      <span className="text-[0.5625rem] font-medium text-[#CBD5E2]">
        {overviewMetricLabel(snapshotScope, metric)}
      </span>
      {values.length > 0 ? (
        <span className="flex flex-wrap justify-end gap-x-2 text-right text-[0.5625rem] text-[#A7B3C3]">
          {values.map((value) => (
            <span key={value}>{value}</span>
          ))}
        </span>
      ) : null}
    </div>
  )
}
