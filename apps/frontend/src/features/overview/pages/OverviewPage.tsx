import { useTranslation } from 'react-i18next'
import { useAppPageMeta } from '../../../app/layouts/AppPageHeader'
import { formatOverviewBackendTag } from '../lib/formatOverviewBackendTag'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { RefreshButton } from '../../../shared/ui'
import { BackendUnavailableBanner } from '../components/BackendUnavailableBanner'
import { OverviewProvidersPanel } from '../components/OverviewProvidersPanel'
import { OverviewRecentExecutionsPanel } from '../components/OverviewRecentExecutionsPanel'
import { OverviewStatCards } from '../components/OverviewStatCards'
import { OverviewTrafficChartPanel } from '../components/OverviewTrafficChartPanel'
import { useOverviewQueries } from '../hooks/useOverviewQueries'
import '../overviewResponsive.css'

export function OverviewPage() {
  const { t } = useTranslation('overview')
  const meta = useAppPageMeta()
  const {
    health,
    readiness,
    providers,
    providerQuotas,
    runtimeMeta,
    trafficSummary,
    quotaUtilization,
    catalog,
    recentExecutions,
    backendUnavailable,
    refetchAll,
  } = useOverviewQueries()

  const isOperational =
    readiness.data?.status === 'ready' ||
    readiness.data?.status === 'up' ||
    health.data?.status === 'healthy' ||
    health.data?.status === 'up'

  const modelsList = catalog.data?.data ?? []
  const executionsList = recentExecutions.data?.records ?? []
  const quotasList = quotaUtilization.data?.quota_utilization ?? []

  const isRefreshing = health.isFetching || readiness.isFetching || trafficSummary.isFetching || providerQuotas.isFetching
  const dataOrigin = runtimeMeta.data?.data_origin ?? 'live'
  const versionString = runtimeMeta.data?.version ?? 'PREVIEW 01'

  const originTagLabel = formatOverviewBackendTag(dataOrigin)

  return (
    <div className={appPageClassName('overview')}>
      <header className="overview-pagehead">
        <div className="min-w-0">
          <p className="overview-page-eyebrow">{meta.eyebrow}</p>
          <h1 className="overview-page-title">{meta.title}</h1>
          {meta.lead ? <p className="overview-page-lead">{meta.lead}</p> : null}
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <span className="overview-backend-tag">{originTagLabel}</span>
          <RefreshButton
            className="overview-refresh"
            label={t('refreshOverview')}
            onRefresh={refetchAll}
            isRefreshing={isRefreshing}
          />
        </div>
      </header>

      {backendUnavailable ? (
        <div className="shrink-0">
          <BackendUnavailableBanner />
        </div>
      ) : null}

      <OverviewStatCards
        isOperational={isOperational}
        providers={providers.data?.providers ?? []}
        quotas={quotasList}
        modelCount={modelsList.length}
        trafficSummary={trafficSummary.data}
        isLoading={health.isLoading || readiness.isLoading}
      />

      <div className="overview-middle">
        <div className="overview-dashboard-grid">
          <OverviewTrafficChartPanel
            traffic={trafficSummary.data}
            isLoading={trafficSummary.isLoading}
          />
          <OverviewProvidersPanel
            snapshots={providerQuotas.data?.snapshots ?? []}
            isLoading={providerQuotas.isLoading}
            isError={providerQuotas.isError}
          />
        </div>

        <OverviewRecentExecutionsPanel
          executions={executionsList}
          isLoading={recentExecutions.isLoading}
        />
      </div>

      <footer className="overview-footnote">
        <span>{t('page.endpointsFootnote')}</span>
        <span>{t('page.brandFootnote', { version: versionString })}</span>
      </footer>
    </div>
  )
}
