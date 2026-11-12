import { useTranslation } from 'react-i18next'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import { countGovernanceMetrics } from '../lib/quotaPresentation'
import type { AdminProvider } from '../types'

type ProviderGovernanceMetricsProps = {
  providers: AdminProvider[]
  snapshots: ProviderQuotaSnapshot[]
}

export function ProviderGovernanceMetrics({
  providers,
  snapshots,
}: ProviderGovernanceMetricsProps) {
  const { t } = useTranslation('providers')
  const metrics = countGovernanceMetrics(providers, snapshots)

  return (
    <div className="providers-metrics">
      <div className="providers-metric">
        <div>
          <span>{t('governance.monitoredTitle')}</span>
          <small>{t('governance.monitoredSubtitle')}</small>
        </div>
        <strong>{String(metrics.monitored).padStart(2, '0')}</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>{t('governance.nearLimitTitle')}</span>
          <small>{t('governance.nearLimitSubtitle')}</small>
        </div>
        <strong className="text-[#D9B87E]">{String(metrics.nearLimit).padStart(2, '0')}</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>{t('governance.atLimitTitle')}</span>
          <small>{t('governance.atLimitSubtitle')}</small>
        </div>
        <strong>{String(metrics.atLimit).padStart(2, '0')}</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>{t('governance.withoutReadingTitle')}</span>
          <small>{t('governance.withoutReadingSubtitle')}</small>
        </div>
        <strong>{String(metrics.withoutReading).padStart(2, '0')}</strong>
      </div>
    </div>
  )
}
