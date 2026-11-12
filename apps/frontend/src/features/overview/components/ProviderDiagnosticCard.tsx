import { useTranslation } from 'react-i18next'
import type { ProviderDiagnostic } from '../api/queries'
import { statusClass } from '../lib/statusClass'

type ProviderDiagnosticCardProps = {
  provider: ProviderDiagnostic
}

export function ProviderDiagnosticCard({ provider }: ProviderDiagnosticCardProps) {
  const { t } = useTranslation('overview')

  return (
    <article className="card provider-card">
      <h3>{provider.id}</h3>
      <p>
        {t('diagnostics.enabled')}: {provider.enabled ? t('diagnostics.yes') : t('diagnostics.no')}
      </p>
      <p className={statusClass(provider.state)}>{provider.state}</p>
    </article>
  )
}
