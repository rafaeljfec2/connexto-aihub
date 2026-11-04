import type { AdminProvider } from '../types'
import { countProvidersWithoutQuotaReading } from '../lib/providerPresentation'

type ProviderGovernanceMetricsProps = {
  providers: AdminProvider[]
}

export function ProviderGovernanceMetrics({ providers }: ProviderGovernanceMetricsProps) {
  const monitored = providers.length
  const withoutQuotaReading = countProvidersWithoutQuotaReading(providers)

  return (
    <div className="providers-metrics">
      <div className="providers-metric">
        <div>
          <span>Provedores monitorados</span>
          <small>Conexões cadastradas no gateway</small>
        </div>
        <strong>{String(monitored).padStart(2, '0')}</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>Próximos do limite</span>
          <small>Acima de 80% de uso</small>
        </div>
        <strong className="text-[#D9B87E]">00</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>Limite atingido</span>
          <small>Requer atenção imediata</small>
        </div>
        <strong>00</strong>
      </div>
      <div className="providers-metric">
        <div>
          <span>Sem leitura de franquia</span>
          <small>Monitoramento não disponível na API</small>
        </div>
        <strong>{String(withoutQuotaReading).padStart(2, '0')}</strong>
      </div>
    </div>
  )
}
