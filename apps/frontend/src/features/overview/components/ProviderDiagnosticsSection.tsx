import type { ProviderDiagnostic } from '../api/queries'
import { Panel, PanelHeader } from '../../../shared/ui'
import { ProviderDiagnosticCard } from './ProviderDiagnosticCard'

type ProviderDiagnosticsSectionProps = {
  providers: ProviderDiagnostic[]
  isLoading: boolean
}

export function ProviderDiagnosticsSection({ providers, isLoading }: ProviderDiagnosticsSectionProps) {
  return (
    <Panel className="mt-4" aria-labelledby="overview-providers-heading">
      <PanelHeader
        title="Provider diagnostics"
        subtitle="Connectivity and runtime state from /diagnostics/providers"
      />
      <h2 id="overview-providers-heading" className="sr-only">
        Provider diagnostics
      </h2>
      {isLoading ? <p className="text-sm text-muted">Loading…</p> : null}
      <div className="provider-grid">
        {providers.map((provider) => (
          <ProviderDiagnosticCard key={provider.id} provider={provider} />
        ))}
      </div>
    </Panel>
  )
}
