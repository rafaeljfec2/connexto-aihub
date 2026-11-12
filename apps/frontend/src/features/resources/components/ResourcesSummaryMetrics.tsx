import { cn } from '../../../shared/ui/cn'

type ResourcesSummaryMetricsProps = {
  configuredPolicies: number
  peakUtilizationPercent: number | null
  concurrencyActive: number
  concurrencyLimit: number
  nearLimitAlerts: number
  isLoading?: boolean
}

export function ResourcesSummaryMetrics({
  configuredPolicies,
  peakUtilizationPercent,
  concurrencyActive,
  concurrencyLimit,
  nearLimitAlerts,
  isLoading,
}: ResourcesSummaryMetricsProps) {
  const loadingValue = '…'

  return (
    <div
      className="mb-3.5 grid gap-2.5 sm:grid-cols-2 xl:grid-cols-4"
      aria-busy={isLoading ? true : undefined}
    >
      <MetricCard
        label="Configured policies"
        detail="Resource + concurrency rules"
        value={isLoading ? loadingValue : String(configuredPolicies)}
      />
      <MetricCard
        label="Peak quota utilization"
        detail="Highest enabled daily limit"
        value={
          isLoading
            ? loadingValue
            : peakUtilizationPercent === null
              ? '—'
              : `${peakUtilizationPercent.toFixed(1)}%`
        }
      />
      <MetricCard
        label="Concurrency in use"
        detail="Active slots / configured limit"
        value={
          isLoading
            ? loadingValue
            : concurrencyLimit > 0
              ? `${concurrencyActive} / ${concurrencyLimit}`
              : `${concurrencyActive}`
        }
      />
      <MetricCard
        label="Near-limit signals"
        detail="Quota ≥90% or saturated concurrency"
        value={isLoading ? loadingValue : String(nearLimitAlerts)}
        valueClassName={!isLoading && nearLimitAlerts > 0 ? 'text-warning' : undefined}
      />
    </div>
  )
}

function MetricCard({
  label,
  detail,
  value,
  valueClassName,
}: {
  label: string
  detail: string
  value: string
  valueClassName?: string
}) {
  return (
    <div className="flex items-center justify-between rounded-[9px] border border-line bg-surface px-3.5 py-2.5">
      <div>
        <span className="block text-[0.625rem] text-[#A0ADBF]">{label}</span>
        <small className="mt-0.5 block text-[0.5625rem] text-[#8290A5]">{detail}</small>
      </div>
      <strong className={cn('text-xl font-semibold tracking-[-0.04em] text-[#EDF2FA]', valueClassName)}>
        {value}
      </strong>
    </div>
  )
}
