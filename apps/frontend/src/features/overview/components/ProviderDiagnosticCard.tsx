import type { ProviderDiagnostic } from '../api/queries'
import { statusClass } from '../lib/statusClass'

type ProviderDiagnosticCardProps = {
  provider: ProviderDiagnostic
}

export function ProviderDiagnosticCard({ provider }: ProviderDiagnosticCardProps) {
  return (
    <article className="card provider-card">
      <h3>{provider.id}</h3>
      <p>Enabled: {provider.enabled ? 'yes' : 'no'}</p>
      <p className={statusClass(provider.state)}>{provider.state}</p>
    </article>
  )
}
