import { Activity, CheckCircle2, Layers, Plug } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { StatCard } from '../../../shared/ui'
import { formatNumber } from '../../../shared/i18n/format'
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
    return `${formatNumber(count / 1_000_000, { maximumFractionDigits: 1 })}M`
  }
  if (count >= 1_000) {
    return `${formatNumber(count / 1_000, { maximumFractionDigits: 1 })}k`
  }
  return formatNumber(count)
}

export function OverviewStatCards({
  isOperational,
  providers = [],
  quotas = [],
  modelCount,
  trafficSummary,
  isLoading,
}: OverviewStatCardsProps) {
  const { t } = useTranslation('overview')
  const degradedProviders = providers.filter(
    (p) => p.state === 'degraded' || p.state === 'unreachable',
  )
  const highQuotaCount = quotas.filter((q) => q.utilization_percent >= 80).length
  const attentionCount = Math.max(degradedProviders.length, highQuotaCount)
  const totalMonitored = Math.max(providers.length, quotas.length)

  const attentionNote =
    highQuotaCount > 0
      ? t('stats.quotaNearLimit')
      : degradedProviders.length > 0
        ? t('stats.providersDegraded', { count: degradedProviders.length })
        : t('stats.providersOk')

  const totalRequests = trafficSummary?.total_requests ?? 0
  const trendPercent = trafficSummary?.trend_percentage ?? 0

  return (
    <div className="overview-stats">
      <StatCard
        label={t('stats.gatewayState')}
        icon={<CheckCircle2 className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">{t('stats.loading')}</span>
          ) : isOperational ? (
            <span className="text-[1.4375rem] font-medium tracking-[-0.02em] text-[#F0F3EC]">
              {t('stats.operational')}
            </span>
          ) : (
            <span className="text-[1.4375rem] font-medium tracking-[-0.02em] text-[#D88186]">
              {t('stats.degraded')}
            </span>
          )
        }
        note={
          <div className="flex items-center gap-1.5">
            <span
              className={`size-1.5 rounded-full ${isOperational ? 'bg-healthy' : 'bg-failure'}`}
            />
            <span>
              {isOperational ? t('stats.gatewayReady') : t('stats.checkConnectivity')}
            </span>
          </div>
        }
      />

      <StatCard
        label={t('stats.quotasAttention')}
        icon={<Plug className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">{t('stats.loading')}</span>
          ) : (
            <MotionNumberPair attention={attentionCount} total={totalMonitored} />
          )
        }
        note={attentionNote}
      />

      <StatCard
        label={t('stats.modelsAvailable')}
        icon={<Layers className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">{t('stats.loading')}</span>
          ) : (
            <span className="text-[1.75rem] font-medium tracking-[-0.04em] text-[#F0F3EC]">
              {modelCount !== undefined ? modelCount : '—'}
            </span>
          )
        }
        note={
          <Link
            to="/catalog"
            className="inline-flex items-center gap-1 text-[0.625rem] text-accent-light hover:text-primary hover:underline"
          >
            {t('stats.exploreCatalog')}
          </Link>
        }
      />

      <StatCard
        label={t('stats.requests24h')}
        icon={<Activity className="size-3.5" />}
        value={
          isLoading ? (
            <span className="text-base text-muted animate-pulse">{t('stats.loading')}</span>
          ) : (
            <span className="text-[1.75rem] font-medium tracking-[-0.04em] text-[#F0F3EC]">
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
              {t('stats.trendVsPrevious')}
            </span>
          ) : (
            <span>{t('stats.traffic24hNote')}</span>
          )
        }
      />
    </div>
  )
}

function MotionNumberPair({ attention, total }: { attention: number; total: number }) {
  return (
    <div className="flex items-baseline gap-1 text-[1.75rem] font-medium tracking-[-0.04em] text-[#F0F3EC]">
      <span>{String(attention).padStart(2, '0')}</span>
      <span className="text-base font-normal text-[#74877A]">/ {String(total).padStart(2, '0')}</span>
    </div>
  )
}
