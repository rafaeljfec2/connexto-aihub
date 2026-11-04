import { Link } from 'react-router-dom'
import { Panel } from '../../../shared/ui'
import type { ProviderQuotaSnapshot, ProviderQuotaMetric } from '../types/quota'

export type OverviewProvidersPanelProps = {
  snapshots?: ProviderQuotaSnapshot[]
  isLoading?: boolean
}

function getInitials(label: string, id: string): string {
  const source = label.trim() !== '' ? label : id
  const parts = source.replace(/[^a-zA-Z0-9]/g, ' ').trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return source.slice(0, 2).toUpperCase()
}

function formatCurrency(amount: number, unit?: string): string {
  const formatted = amount.toLocaleString('pt-BR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  if (unit === 'USD') {
    return `US$ ${formatted}`
  }
  if (unit) {
    return `${formatted} ${unit}`
  }
  return formatted
}

function findMetric(metrics: ProviderQuotaMetric[], name: string): ProviderQuotaMetric | undefined {
  return metrics.find((m) => m.metric === name)
}

export function OverviewProvidersPanel({
  snapshots = [],
  isLoading,
}: OverviewProvidersPanelProps) {
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
        <div className="space-y-3 py-3 animate-pulse" data-testid="quota-loading">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="size-7 rounded-lg bg-raised" />
              <div className="space-y-1">
                <div className="h-3 w-20 rounded bg-raised" />
                <div className="h-2 w-32 rounded bg-raised" />
              </div>
            </div>
            <div className="h-3 w-16 rounded bg-raised" />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="size-7 rounded-lg bg-raised" />
              <div className="space-y-1">
                <div className="h-3 w-24 rounded bg-raised" />
                <div className="h-2 w-28 rounded bg-raised" />
              </div>
            </div>
            <div className="h-3 w-14 rounded bg-raised" />
          </div>
        </div>
      ) : snapshots.length === 0 ? (
        <div className="py-4 text-center text-xs text-muted" data-testid="quota-empty">
          Nenhuma conta de provedor configurada no momento.
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto space-y-0.5">
          {snapshots.map((s) => {
            const displayName = s.account_label || s.provider_name || s.account_id
            const initials = getInitials(displayName, s.account_id)

            let metaText = 'Franquia ainda não monitorada'
            let statusLabel = 'Sem leitura'
            let ballColor = 'bg-[#738078]'
            let statusTextColor = 'text-[#7C8A80]'
            let officialConsumedPercent: number | undefined

            if (s.status === 'unconfigured_credential') {
              metaText = 'Credencial não configurada'
              statusLabel = 'Sem leitura'
            } else if (s.status === 'unauthorized') {
              metaText = 'Credencial sem permissão de leitura'
              statusLabel = 'Sem leitura'
            } else if (s.status === 'rate_limited') {
              metaText = 'Limite da API atingido'
              statusLabel = 'Rate limit'
              ballColor = 'bg-[#D6AD72]'
              statusTextColor = 'text-[#D8B77E]'
            } else if (s.status === 'timeout') {
              metaText = 'Tempo limite na consulta excedido'
              statusLabel = 'Indisponível'
            } else if (s.status === 'error') {
              metaText = 'Consulta ao provedor indisponível'
              statusLabel = 'Sem leitura'
            } else if (s.status === 'available') {
              const quotas = s.quotas ?? []

              // Official percent from provider, if explicitly reported
              const percentMetric = quotas.find((m) => m.consumed_percent != null || m.remaining_percent != null)
              if (percentMetric?.consumed_percent != null) {
                officialConsumedPercent = percentMetric.consumed_percent
              }

              // Account-level metrics from official provider API
              const accountUsageMetric = findMetric(quotas, 'total_usage')
              const accountCreditsMetric = findMetric(quotas, 'total_credits')

              // Key-level metrics
              const keyUsageMetric = findMetric(quotas, 'usage')
              const remainingMetric = findMetric(quotas, 'limit_remaining') ?? quotas.find((m) => m.remaining != null)
              const limitMetric = findMetric(quotas, 'limit') ?? quotas.find((m) => m.limit != null && m.metric !== 'total_credits')

              // Build meta text without calculations
              if (accountUsageMetric?.used != null && accountUsageMetric.used > 0) {
                const usedFormatted = formatCurrency(accountUsageMetric.used, accountUsageMetric.unit)
                if (limitMetric?.limit != null) {
                  const limitFormatted = formatCurrency(limitMetric.limit, limitMetric.unit)
                  metaText = `${usedFormatted} consumido · Limite ${limitFormatted}`
                } else if (accountCreditsMetric?.limit != null) {
                  const creditsFormatted = formatCurrency(accountCreditsMetric.limit, accountCreditsMetric.unit)
                  metaText = `${usedFormatted} consumido · Total ${creditsFormatted}`
                } else {
                  metaText = `${usedFormatted} consumido (conta)`
                }
              } else if (keyUsageMetric?.used != null) {
                const usedFormatted = formatCurrency(keyUsageMetric.used, keyUsageMetric.unit)
                const resetPeriod = limitMetric?.period ? ` · ciclo ${limitMetric.period}` : ''
                if (limitMetric?.limit != null) {
                  const limitFormatted = formatCurrency(limitMetric.limit, limitMetric.unit)
                  metaText = `${usedFormatted} consumido · Limite ${limitFormatted}${resetPeriod}`
                } else if (accountCreditsMetric?.limit != null) {
                  const creditsFormatted = formatCurrency(accountCreditsMetric.limit, accountCreditsMetric.unit)
                  metaText = `${usedFormatted} consumido · Total ${creditsFormatted}`
                } else {
                  metaText = `${usedFormatted} consumido${resetPeriod}`
                }
              } else if (limitMetric?.limit != null) {
                const limitFormatted = formatCurrency(limitMetric.limit, limitMetric.unit)
                metaText = `Limite ${limitFormatted}`
              } else if (accountCreditsMetric?.limit != null) {
                const creditsFormatted = formatCurrency(accountCreditsMetric.limit, accountCreditsMetric.unit)
                metaText = `Total ${creditsFormatted} em créditos`
              } else {
                metaText = 'Conectado · Leitura oficial'
              }

              // Build right status label without calculations
              if (percentMetric?.remaining_percent != null) {
                statusLabel = `${percentMetric.remaining_percent}% disponível`
                ballColor = 'bg-healthy'
                statusTextColor = 'text-[#A7B3C3]'
              } else if (remainingMetric?.remaining != null) {
                statusLabel = `${formatCurrency(remainingMetric.remaining, remainingMetric.unit)} disponível`
                ballColor = 'bg-healthy'
                statusTextColor = 'text-[#A7B3C3]'
              } else if (accountCreditsMetric?.limit != null) {
                statusLabel = `${formatCurrency(accountCreditsMetric.limit, accountCreditsMetric.unit)} em créditos`
                ballColor = 'bg-healthy'
                statusTextColor = 'text-[#A7B3C3]'
              } else if (percentMetric?.consumed_percent != null) {
                statusLabel = `${percentMetric.consumed_percent}% consumido`
                ballColor = 'bg-healthy'
                statusTextColor = 'text-[#A7B3C3]'
              } else if (limitMetric?.limit == null) {
                statusLabel = 'Chave sem limite'
                ballColor = 'bg-healthy'
                statusTextColor = 'text-[#A7B3C3]'
              } else {
                statusLabel = 'Disponível'
                ballColor = 'bg-healthy'
                statusTextColor = 'text-[#A7B3C3]'
              }
            }

            return (
              <div key={s.account_id} className="overview-provider-row" data-testid={`provider-row-${s.account_id}`}>
                <div className="flex items-center gap-2">
                  <span className="overview-provider-icon">{initials}</span>
                  <div>
                    <div className="text-[11px] font-semibold text-[#E1E8F3]">{displayName}</div>
                    <div className="text-[9px] text-[#909DB0]">{metaText}</div>
                  </div>
                </div>

                <div className="flex flex-col items-end gap-1">
                  <span className={`flex items-center gap-1 text-[9px] ${statusTextColor}`}>
                    <i className={`size-1.5 rounded-full ${ballColor}`} aria-hidden />
                    {statusLabel}
                  </span>
                  {/* Progress bar strictly rendered only if official percentage is present */}
                  {officialConsumedPercent != null && (
                    <div className="h-1 w-16 overflow-hidden rounded-full bg-raised">
                      <div
                        className="h-full bg-healthy transition-all"
                        style={{ width: `${Math.min(100, Math.max(0, officialConsumedPercent))}%` }}
                      />
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </Panel>
  )
}
