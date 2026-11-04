import { Activity, CheckCircle2, Layers, Plug } from 'lucide-react'
import { Link } from 'react-router-dom'
import { StatCard } from '../../../shared/ui'
import type { ProviderDiagnostic, TrafficSummary } from '../api/queries'
import type { QuotaUtilizationEntry } from '../../resources/types'

type OverviewStatCardsProps = {
  isOperational: boolean
  readinessChecks?: Record<string, string>
  providers?: ProviderDiagnostic[]
  quotas?: QuotaUtilizationEntry[]
  modelCount?: number
  trafficSummary?: TrafficSummary
  isLoading?: boolean
}

function formatRequestsCount(count: number): string {
  if (count >= 1_000_000) {
    return `${(count / 1_000_000).toLocaleString('pt-BR', { maximumFractionDigits: 1 })}M`
  }
  if (count >= 1_000) {
    return `${(count / 1_000).toLocaleString('pt-BR', { maximumFractionDigits: 1 })}k`
  }
  return count.toLocaleString('pt-BR')
}

export function OverviewStatCards({
  isOperational,
  providers = [],
  quotas = [],
  modelCount,
  trafficSummary,
  isLoading,
}: OverviewStatCardsProps) {
  // Determine attention quotas: providers with utilization >= 80% or degraded state
  const degradedProviders = providers.filter(
    (p) => p.state === 'degraded' || p.state === 'unreachable',
  )
  const highQuotaCount = quotas.filter((q) => q.utilization_percent >= 80).length
  const attentionCount = Math.max(degradedProviders.length, highQuotaCount)
  const totalMonitored = Math.max(providers.length, quotas.length)

  const attentionNote =
    highQuotaCount > 0
      ? 'Franquia próxima do limite'
      : degradedProviders.length > 0
        ? `${degradedProviders.length} conexão requer verificação`
        : 'Todas as conexões em estado normal'

  const totalRequests = trafficSummary?.total_requests ?? 0
  const trendPercent = trafficSummary?.trend_percentage ?? 0

  return (
    <div className="overview-stats">
      <StatCard
        label="Estado do gateway"
        icon={<CheckCircle2 className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : isOperational ? (
            <span className="text-[23px] font-medium tracking-[-0.02em] text-[#F0F3EC]">
              Operacional
            </span>
          ) : (
            <span className="text-[23px] font-medium tracking-[-0.02em] text-[#D88186]">
              Degradado
            </span>
          )
        }
        note={
          <div className="flex items-center gap-1.5">
            <span
              className={`size-1.5 rounded-full ${
                isOperational ? 'bg-healthy' : 'bg-failure'
              }`}
            />
            <span>
              {isOperational
                ? 'Pronto para receber requisições'
                : 'Verifique a conectividade dos serviços'}
            </span>
          </div>
        }
      />

      <StatCard
        label="Franquias em atenção"
        icon={<Plug className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : (
            <div className="flex items-baseline gap-1 text-[28px] font-medium tracking-[-0.04em] text-[#F0F3EC]">
              <span>{String(attentionCount).padStart(2, '0')}</span>
              <span className="text-base font-normal text-[#74877A]">
                / {String(totalMonitored).padStart(2, '0')}
              </span>
            </div>
          )
        }
        note={attentionNote}
      />

      <StatCard
        label="Modelos disponíveis"
        icon={<Layers className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : (
            <span className="text-[28px] font-medium tracking-[-0.04em] text-[#F0F3EC]">
              {modelCount !== undefined ? modelCount : '—'}
            </span>
          )
        }
        note={
          <Link
            to="/catalog"
            className="inline-flex items-center gap-1 text-[10px] text-accent-light hover:text-primary hover:underline"
          >
            Explorar catálogo ↗
          </Link>
        }
      />

      <StatCard
        label="Requisições · 24h"
        icon={<Activity className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">Carregando…</span>
          ) : (
            <span className="text-[28px] font-medium tracking-[-0.04em] text-[#F0F3EC]">
              {formatRequestsCount(totalRequests)}
            </span>
          )
        }
        note={
          trendPercent !== 0 ? (
            <span className="flex items-center gap-1">
              <span className="font-semibold text-healthy">
                ↗ {trendPercent > 0 ? `+${trendPercent.toFixed(1)}%` : `${trendPercent.toFixed(1)}%`}
              </span>{' '}
              em relação ao período anterior
            </span>
          ) : (
            <span>Tráfego processado nas últimas 24h</span>
          )
        }
      />
    </div>
  )
}
