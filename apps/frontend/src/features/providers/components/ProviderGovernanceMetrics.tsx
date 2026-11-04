import { AlertTriangle, CheckCircle, HelpCircle } from 'lucide-react'
import { StatCard } from '../../../shared/ui'
import type { AdminProvider } from '../types'

type ProviderGovernanceMetricsProps = {
  providers: AdminProvider[]
}

export function ProviderGovernanceMetrics({ providers }: ProviderGovernanceMetricsProps) {
  const total = providers.length
  const enabled = providers.filter((p) => p.enabled).length
  const missingCreds = providers.filter((p) => !p.credential_configured).length
  const configured = providers.filter((p) => p.credential_configured).length

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        label="Provedores monitorados"
        icon={<CheckCircle className="size-3.5" />}
        value={String(total).padStart(2, '0')}
        note="Total de conexões cadastradas"
      />

      <StatCard
        label="Conexões ativas"
        icon={<CheckCircle className="size-3.5 text-healthy" />}
        value={<span className="text-healthy">{String(enabled).padStart(2, '0')}</span>}
        note="Habilitadas para requisições"
      />

      <StatCard
        label="Credenciais pendentes"
        icon={<AlertTriangle className="size-3.5 text-warning" />}
        value={<span className="text-warning">{String(missingCreds).padStart(2, '0')}</span>}
        note="Sem chave ou token configurado"
      />

      <StatCard
        label="Sem leitura de franquia"
        icon={<HelpCircle className="size-3.5 text-muted" />}
        value={<span className="text-muted">{String(configured).padStart(2, '0')}</span>}
        note="Leitura automática não fornecida pela API"
      />
    </div>
  )
}
