import { useTranslation } from 'react-i18next'
import { buildProviderUsageDistribution } from '../lib/usageDistribution'
import type { ProviderUsageDistributionResponse, ProviderUsageResponse } from '../types'
import { SectionFetchedAt } from './SectionFetchedAt'

type UsageDistributionProps = {
  distribution: ProviderUsageDistributionResponse | undefined
  usage: ProviderUsageResponse | undefined
  isLoading?: boolean
  isError?: boolean
  fetchedAtMs?: number
}

export function UsageDistribution({
  distribution,
  usage,
  isLoading,
  isError,
  fetchedAtMs,
}: UsageDistributionProps) {
  const { t } = useTranslation('resources')
  const slices = buildProviderUsageDistribution(distribution?.by_provider ?? [])

  return (
    <section className="resources-panel" aria-label={t('usageDistribution.ariaLabel')}>
      <div className="resources-panel-head">
        <div>
          <h3 className="resources-panel-title">{t('usageDistribution.title')}</h3>
          <p className="resources-panel-subtitle">{t('usageDistribution.subtitle')}</p>
        </div>
      </div>

      {isLoading ? (
        <p className="resources-empty">{t('usageDistribution.loading')}</p>
      ) : isError ? (
        <p className="resources-empty">{t('usageDistribution.unavailable')}</p>
      ) : slices.length === 0 ? (
        <p className="resources-empty">{t('usageDistribution.empty')}</p>
      ) : (
        <div className="resources-distribution-bars">
          {slices.map((slice, index) => (
            <div key={slice.providerId} className="resources-distribution-row">
              <label>{slice.providerId}</label>
              <div className="resources-distribution-track">
                <div
                  className="resources-distribution-fill"
                  style={{
                    width: `${Math.min(100, slice.percent)}%`,
                    opacity: Math.max(0.45, 1 - index * 0.15),
                  }}
                />
              </div>
              <strong>{Math.round(slice.percent)}%</strong>
            </div>
          ))}
        </div>
      )}

      <p className="resources-measurement-note">
        {t('usageDistribution.measurementSource')}
        {usage
          ? t('usageDistribution.measurementCounts', {
              reported: usage.reported_executions,
              partial: usage.partial_executions,
              unavailable: usage.unavailable_executions,
              invalid: usage.invalid_executions,
            })
          : null}
      </p>

      <SectionFetchedAt updatedAtMs={fetchedAtMs} />
    </section>
  )
}
