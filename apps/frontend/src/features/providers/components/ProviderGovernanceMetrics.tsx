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
  const metrics = countGovernanceMetrics(providers, snapshots)

  return (
    <div className="providers-metrics">
      <div className="providers-metric">
        <div>
          <span>Provedores monitorados</span>
          <small>Conexões cadastradas no gateway</small>
        </div>
        <strong>{String(metrics.monitored).padStart(2, '0')}</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>Próximos do limite</span>
          <small>Sinal oficial do provedor</small>
        </div>
        <strong className="text-[#D9B87E]">{String(metrics.nearLimit).padStart(2, '0')}</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>Limite atingido</span>
          <small>Requer atenção imediata</small>
        </div>
        <strong>{String(metrics.atLimit).padStart(2, '0')}</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>Sem leitura de franquia</span>
          <small>Monitoramento não disponível na API</small>
        </div>
        <strong>{String(metrics.withoutReading).padStart(2, '0')}</strong>
      </div>
    </div>
  )
}
