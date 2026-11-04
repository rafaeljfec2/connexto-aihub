import { Link } from 'react-router-dom'
import { Panel } from '../../../shared/ui'
import type { ProviderDiagnostic } from '../api/queries'
import type { QuotaUtilizationEntry } from '../../resources/types'

type OverviewProvidersPanelProps = {
  providers?: ProviderDiagnostic[]
  quotas?: QuotaUtilizationEntry[]
  isLoading?: boolean
}

function getProviderInitials(id: string): string {
  const parts = id.replace(/[^a-zA-Z0-9]/g, ' ').trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return id.slice(0, 2).toUpperCase()
}

function formatProviderName(id: string): string {
  if (id === 'opencode-go' || id === 'opencode_go') return 'OpenCode'
  if (id === 'openrouter') return 'OpenRouter'
  if (id === 'openai') return 'OpenAI'
  if (id === 'anthropic') return 'Anthropic'
  if (id === 'google-ai' || id === 'google') return 'Google AI'
  if (id === 'azure-openai' || id === 'azure') return 'Azure OpenAI'

  return id
    .split(/[-_]/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

export function OverviewProvidersPanel({
  providers = [],
  quotas = [],
  isLoading,
}: OverviewProvidersPanelProps) {
  const quotaByProvider = new Map<string, QuotaUtilizationEntry>()
  for (const quota of quotas) {
    if (quota.provider_id) {
      quotaByProvider.set(quota.provider_id.toLowerCase(), quota)
    }
  }

  const visibleProviders = providers.slice(0, 4)

  return (
    <Panel className="overview-panel">
      <div className="overview-panel-head">
        <div>
          <h3 className="text-xs font-semibold text-[#E7ECF5]">Consumo de franquias</h3>
          <p className="mt-0.5 text-[10px] text-[#8D9AAF]">Uso e saldo disponível por provedor</p>
        </div>
        <Link to="/providers" className="overview-panel-link">
          Ver franquias ↗
        </Link>
      </div>

      {isLoading ? (
        <div className="py-4 text-center text-xs text-muted animate-pulse">
          Carregando franquias dos provedores…
        </div>
      ) : visibleProviders.length === 0 ? (
        <div className="py-4 text-center text-xs text-muted">Nenhum provedor configurado no momento.</div>
      ) : (
        <div className="min-h-0 flex-1 overflow-hidden">
          {visibleProviders.map((provider) => {
            const quota = quotaByProvider.get(provider.id.toLowerCase())
            const initials = getProviderInitials(provider.id)
            const name = formatProviderName(provider.id)

            let metaText = 'Franquia ainda não monitorada'
            let statusLabel = 'Sem leitura'
            let ballColor = 'bg-[#738078]'
            let statusTextColor = 'text-[#7C8A80]'

            if (quota) {
              if (!quota.enabled) {
                metaText = 'Política desabilitada'
                statusLabel = 'Não aplicável'
              } else if (quota.daily_limit <= 0) {
                metaText = 'Cota sem limite diário'
                statusLabel = 'Sem limite'
                ballColor = 'bg-healthy'
                statusTextColor = 'text-[#A7B3C3]'
              } else {
                const usedPercent = Math.min(100, Math.round(quota.utilization_percent))
                const availablePercent = Math.max(0, 100 - usedPercent)
                metaText = `${usedPercent}% consumido · cota diária`
                statusLabel = `${availablePercent}% disponível`

                if (usedPercent >= 80) {
                  ballColor = 'bg-warning'
                  statusTextColor = 'text-[#D8B77E]'
                } else {
                  ballColor = 'bg-healthy'
                  statusTextColor = 'text-[#A7B3C3]'
                }
              }
            }

            return (
              <div key={provider.id} className="overview-provider-row">
                <div className="flex items-center gap-2">
                  <span className="overview-provider-icon">{initials}</span>
                  <div>
                    <div className="text-[11px] font-semibold text-[#E1E8F3]">{name}</div>
                    <div className="text-[9px] text-[#909DB0]">{metaText}</div>
                  </div>
                </div>
                <span className={`flex items-center gap-1 text-[9px] ${statusTextColor}`}>
                  <i className={`size-1.5 rounded-full ${ballColor}`} aria-hidden />
                  {statusLabel}
                </span>
              </div>
            )
          })}
        </div>
      )}
    </Panel>
  )
}
