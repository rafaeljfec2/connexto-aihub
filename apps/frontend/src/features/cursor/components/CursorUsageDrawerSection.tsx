import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDateTime } from '../../../shared/i18n/format'
import { RefreshButton } from '../../../shared/ui'
import { useCursorUsageQueries } from '../hooks/useCursorUsageQueries'
import {
  buildCursorDailySeries,
  rankCursorModelUsage,
  summarizeCursorDailyMetrics,
} from '../lib/cursorAnalytics'
import type { CursorUsageAnalyticsEnvelope } from '../types'

export function CursorUsageDrawerSection({ providerId }: Readonly<{ providerId: string }>) {
  const { t } = useTranslation('overview')
  const { envelope, hasRefetchError, isInitialLoading, isRefreshing, refresh } =
    useCursorUsageQueries(providerId)

  if (isInitialLoading) return <DrawerStatus message={t('cursor.loading')} />
  if (!envelope) return <DrawerStatus message={t('cursor.consoleUnavailable')} />
  if (envelope.status === 'collecting') {
    return <DrawerStatus message={t('cursor.collectingGrok')} />
  }
  if (envelope.analytics === undefined) {
    return (
      <DrawerStatus message={envelope.error?.message ?? t('cursor.noConsoleActivity')} />
    )
  }

  return (
    <CursorUsageDrawerContent
      envelope={envelope}
      hasRefetchError={hasRefetchError}
      isRefreshing={isRefreshing}
      onRefresh={refresh}
    />
  )
}

function DrawerStatus({ message }: Readonly<{ message: string }>) {
  return (
    <section className="cursor-drawer-usage" data-testid="cursor-drawer-usage-status" role="status">
      <DrawerHeading />
      <p className="text-xs text-muted">{message}</p>
    </section>
  )
}

function CursorUsageDrawerContent({
  envelope,
  hasRefetchError,
  isRefreshing,
  onRefresh,
}: Readonly<{
  envelope: CursorUsageAnalyticsEnvelope
  hasRefetchError: boolean
  isRefreshing: boolean
  onRefresh: () => void
}>) {
  const { t } = useTranslation('overview')
  const metrics = envelope.analytics?.dailyMetrics ?? []
  const summary = useMemo(() => summarizeCursorDailyMetrics(metrics), [metrics])
  const series = useMemo(() => buildCursorDailySeries(metrics, 'agent').slice(-14), [metrics])
  const models = useMemo(() => rankCursorModelUsage(metrics, 5), [metrics])
  const maxRequests = Math.max(...series.map((point) => point.value ?? 0), 0)

  return (
    <section className="cursor-drawer-usage" data-testid="cursor-drawer-usage">
      <div className="flex items-start justify-between gap-2">
        <DrawerHeading />
        <RefreshButton
          className="providers-primary-action !p-2"
          aria-label={t('cursor.refreshAria')}
          onRefresh={onRefresh}
          isRefreshing={isRefreshing}
          iconOnly
          iconClassName="size-3"
        />
      </div>
      <div className="cursor-drawer-usage-metrics">
        <UsageMetric
          label={t('cursor.agentRequests')}
          value={summary.agentRequests}
          secondaryLabel={t('cursor.chat')}
          secondary={summary.chatRequests}
        />
        <UsageMetric
          label={t('cursor.linesAdded')}
          value={summary.linesAdded}
          secondaryLabel={t('cursor.accepted')}
          secondary={summary.acceptedLinesAdded}
        />
        <UsageMetric
          label={t('cursor.applies')}
          value={summary.totalApplies}
          secondaryLabel={t('cursor.accepts')}
          secondary={summary.totalAccepts}
        />
        <UsageMetric
          label={t('cursor.includedRequests')}
          value={summary.subscriptionIncludedReqs}
          secondaryLabel={t('cursor.usageBased')}
          secondary={summary.usageBasedReqs}
        />
      </div>
      {envelope.error || hasRefetchError ? (
        <p className="mt-2 text-[0.625rem] text-warning" role="status" data-testid="cursor-drawer-refresh-warning">
          {t('cursor.refetchWarning')}
        </p>
      ) : null}
      <div className="mt-3">
        <p className="cursor-drawer-usage-label">{t('cursor.dailyChartTitle')}</p>
        {series.length === 0 ? (
          <p className="mt-2 text-[0.625rem] text-muted">{t('cursor.noDailyActivity')}</p>
        ) : (
          <div className="cursor-drawer-usage-chart" role="img" aria-label={t('cursor.chartAria')}>
            {series.map((point) => (
              <div key={point.date} className="flex min-w-0 flex-1 flex-col items-center justify-end gap-1">
                <div
                  className="w-full max-w-5 rounded-t-sm bg-cobalt/75"
                  style={{ height: barHeight(point.value, maxRequests) }}
                  title={point.value === undefined ? t('cursor.noReadingReported') : String(point.value)}
                />
                <span className="font-mono text-[0.4375rem] text-muted">{dayOfMonth(point.date)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
      <div className="mt-3">
        <p className="cursor-drawer-usage-label">{t('cursor.usageByModelTitle')}</p>
        {models.length === 0 ? (
          <p className="mt-2 text-[0.625rem] text-muted">{t('cursor.noModelUsage')}</p>
        ) : (
          <ul className="mt-2 space-y-1.5">
            {models.map((model) => (
              <li key={model.name} className="flex justify-between gap-3 text-[0.625rem]">
                <span className="truncate text-secondary">{model.name}</span>
                <span className="shrink-0 font-mono text-primary">{model.count}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
      {envelope.sourceCollectedAt ? (
        <p className="mt-3 text-[0.5625rem] text-muted">
          {t('cursor.lastCollect', {
            at: formatDateTime(envelope.sourceCollectedAt),
          })}
        </p>
      ) : null}
    </section>
  )
}

function DrawerHeading() {
  const { t } = useTranslation('overview')
  return <p className="cursor-drawer-usage-label">{t('cursor.sectionTitle')}</p>
}

function UsageMetric({
  label,
  value,
  secondaryLabel,
  secondary,
}: Readonly<{ label: string; value?: number; secondaryLabel: string; secondary?: number }>) {
  return (
    <div className="cursor-drawer-usage-metric">
      <span>{label}</span>
      <strong>{value ?? '—'}</strong>
      {secondary !== undefined ? (
        <small>
          {secondaryLabel}: {secondary}
        </small>
      ) : null}
    </div>
  )
}

function barHeight(value: number | undefined, max: number): number {
  if (value === undefined) return 0
  if (value <= 0 || max <= 0) return 0
  return Math.max(4, Math.round((value / max) * 46))
}

function dayOfMonth(timestamp: string): string {
  const value = Number(timestamp)
  if (!Number.isFinite(value)) return '—'
  return new Intl.DateTimeFormat(undefined, { day: '2-digit', timeZone: 'UTC' }).format(new Date(value))
}
