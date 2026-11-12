import { useTranslation } from 'react-i18next'
import { cn } from '../../../shared/ui/cn'

type ResourcesMetricGridProps = {
  configuredPolicies: number
  globalUtilizationPercent: number | null
  concurrencyActive: number
  concurrencyLimit: number
  nearLimitAlerts: number
  isLoading?: boolean
}

export function ResourcesMetricGrid({
  configuredPolicies,
  globalUtilizationPercent,
  concurrencyActive,
  concurrencyLimit,
  nearLimitAlerts,
  isLoading,
}: ResourcesMetricGridProps) {
  const { t } = useTranslation('resources')
  const loadingValue = '…'
  const concurrencyValueText = concurrencyMetricValue(
    isLoading,
    loadingValue,
    concurrencyActive,
    concurrencyLimit,
  )

  return (
    <div className="resources-metrics" aria-busy={isLoading ? true : undefined}>
      <MetricCard
        label={t('metrics.configuredPolicies.label')}
        detail={t('metrics.configuredPolicies.detail')}
        value={isLoading ? loadingValue : String(configuredPolicies)}
      />
      <MetricCard
        label={t('metrics.aggregateUsage.label')}
        detail={t('metrics.aggregateUsage.detail')}
        value={
          isLoading
            ? loadingValue
            : globalUtilizationPercent === null
              ? '—'
              : globalUtilizationPercent.toFixed(0)
        }
        suffix={!isLoading && globalUtilizationPercent !== null ? '%' : undefined}
      />
      <MetricCard
        label={t('metrics.concurrencyInUse.label')}
        detail={t('metrics.concurrencyInUse.detail')}
        value={concurrencyValueText}
        suffix={
          !isLoading && concurrencyLimit > 0 ? ` / ${concurrencyLimit}` : undefined
        }
      />
      <MetricCard
        label={t('metrics.policyAlerts.label')}
        detail={t('metrics.policyAlerts.detail')}
        value={isLoading ? loadingValue : String(nearLimitAlerts).padStart(2, '0')}
        valueClassName={!isLoading && nearLimitAlerts > 0 ? 'is-warn' : undefined}
      />
    </div>
  )
}

function concurrencyMetricValue(
  isLoading: boolean | undefined,
  loadingValue: string,
  active: number,
  limit: number,
): string {
  if (isLoading) {
    return loadingValue
  }
  if (limit > 0 || active > 0) {
    return String(active)
  }
  return '—'
}

function MetricCard({
  label,
  detail,
  value,
  suffix,
  valueClassName,
}: {
  label: string
  detail: string
  value: string
  suffix?: string
  valueClassName?: string
}) {
  return (
    <div className="resources-metric">
      <div>
        <span>{label}</span>
        <small>{detail}</small>
      </div>
      <strong className={cn(valueClassName)}>
        {value}
        {suffix ? <span className="resources-metric-suffix">{suffix}</span> : null}
      </strong>
    </div>
  )
}
