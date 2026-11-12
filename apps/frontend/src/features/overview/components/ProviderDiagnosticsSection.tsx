import { useTranslation } from 'react-i18next'
import type { ProviderDiagnostic } from '../api/queries'
import { Panel, PanelHeader } from '../../../shared/ui'
import { ProviderDiagnosticCard } from './ProviderDiagnosticCard'

type ProviderDiagnosticsSectionProps = {
  providers: ProviderDiagnostic[]
  isLoading: boolean
}

export function ProviderDiagnosticsSection({ providers, isLoading }: ProviderDiagnosticsSectionProps) {
  const { t } = useTranslation('overview')

  return (
    <Panel className="mt-4" aria-labelledby="overview-providers-heading">
      <PanelHeader title={t('diagnostics.sectionTitle')} subtitle={t('diagnostics.subtitle')} />
      <h2 id="overview-providers-heading" className="sr-only">
        {t('diagnostics.sectionTitle')}
      </h2>
      {isLoading ? <p className="text-sm text-muted">{t('diagnostics.loading')}</p> : null}
      <div className="provider-grid">
        {providers.map((provider) => (
          <ProviderDiagnosticCard key={provider.id} provider={provider} />
        ))}
      </div>
    </Panel>
  )
}
