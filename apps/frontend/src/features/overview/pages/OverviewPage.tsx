import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { BackendUnavailableBanner } from '../components/BackendUnavailableBanner'
import { ProviderDiagnosticsSection } from '../components/ProviderDiagnosticsSection'
import { ReadinessCard } from '../components/ReadinessCard'
import { StatusCard } from '../components/StatusCard'
import { useOverviewQueries } from '../hooks/useOverviewQueries'

export function OverviewPage() {
  const { health, readiness, providers, backendUnavailable } = useOverviewQueries()

  return (
    <div className="overview">
      <AppPageHeader />
      {backendUnavailable ? <BackendUnavailableBanner /> : null}

      <section className="card-grid">
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
      </section>

      <ProviderDiagnosticsSection
        providers={providers.data?.providers ?? []}
        isLoading={providers.isLoading}
      />
    </div>
  )
}
