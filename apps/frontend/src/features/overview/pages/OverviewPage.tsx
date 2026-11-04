import { RefreshCw } from 'lucide-react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { Button, EmptyTelemetryState, Panel, PanelHeader } from '../../../shared/ui'
import { BackendUnavailableBanner } from '../components/BackendUnavailableBanner'
import { OverviewProvidersPanel } from '../components/OverviewProvidersPanel'
import { OverviewRecentExecutionsPanel } from '../components/OverviewRecentExecutionsPanel'
import { OverviewStatCards } from '../components/OverviewStatCards'
import { ProviderDiagnosticsSection } from '../components/ProviderDiagnosticsSection'
import { ReadinessCard } from '../components/ReadinessCard'
import { StatusCard } from '../components/StatusCard'
import { useOverviewQueries } from '../hooks/useOverviewQueries'

export function OverviewPage() {
  const {
    health,
    readiness,
    providers,
    catalog,
    recentExecutions,
    backendUnavailable,
    refetchAll,
  } = useOverviewQueries()

  const isOperational =
    readiness.data?.status === 'ready' || health.data?.status === 'healthy'

  const modelsList = catalog.data?.data ?? []
  const executionsList = recentExecutions.data?.records ?? []

  return (
    <div className="overview space-y-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <AppPageHeader />
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => void refetchAll()}
            className="cursor-pointer gap-1.5"
            disabled={health.isFetching || readiness.isFetching}
          >
            <RefreshCw
              className={`size-3.5 ${
                health.isFetching || readiness.isFetching ? 'animate-spin' : ''
              }`}
            />
            <span>Atualizar</span>
          </Button>
        </div>
      </div>

      {backendUnavailable ? <BackendUnavailableBanner /> : null}

      <OverviewStatCards
        isOperational={isOperational}
        providers={providers.data?.providers ?? []}
        modelCount={modelsList.length}
        executionCount={executionsList.length}
        isLoading={health.isLoading || readiness.isLoading}
      />

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.2fr_0.8fr]">
        <Panel>
          <PanelHeader
            title="Volume de requisições"
            subtitle="Tráfego processado pelo gateway nas últimas 24 horas"
          />
          <EmptyTelemetryState
            title="Telemetria de tráfego agregado não instrumentada"
            description="O gateway emite métricas e logs operacionais individuais, mas a agregação de série temporal de 24 horas ainda não está consolidada em endpoint de leitura no backend. Nenhum gráfico artificial é renderizado."
          />
        </Panel>

        <OverviewProvidersPanel
          providers={providers.data?.providers ?? []}
          isLoading={providers.isLoading}
        />
      </div>

      <OverviewRecentExecutionsPanel
        executions={executionsList}
        isLoading={recentExecutions.isLoading}
      />

      <section className="mt-4 pt-2 border-t border-line/40">
        <h3 className="mb-2 font-mono text-[10px] uppercase tracking-[0.1em] text-muted">
          Diagnóstico técnico operacional
        </h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <StatusCard
            title="System liveness"
            status={health.data?.status}
            isLoading={health.isLoading}
          />
          <ReadinessCard
            status={readiness.data?.status}
            checks={readiness.data?.checks}
            isLoading={readiness.isLoading}
          />
        </div>
        <ProviderDiagnosticsSection
          providers={providers.data?.providers ?? []}
          isLoading={providers.isLoading}
        />
      </section>
    </div>
  )
}
