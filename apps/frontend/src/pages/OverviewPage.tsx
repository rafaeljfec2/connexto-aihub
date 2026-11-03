import { useQuery } from '@tanstack/react-query'
import { getHealth, getProviderDiagnostics, getReadiness, queryKeys } from '../api/client'

function statusClass(state: string): string {
  if (state === 'healthy' || state === 'up' || state === 'ok') {
    return 'status-healthy'
  }
  if (state === 'disabled' || state === 'unknown') {
    return 'status-muted'
  }
  return 'status-unhealthy'
}

export function OverviewPage() {
  const health = useQuery({
    queryKey: queryKeys.health,
    queryFn: getHealth,
    refetchInterval: false,
    retry: 1,
  })

  const readiness = useQuery({
    queryKey: queryKeys.readiness,
    queryFn: getReadiness,
    refetchInterval: false,
    retry: 1,
  })

  const providers = useQuery({
    queryKey: queryKeys.providerDiagnostics,
    queryFn: getProviderDiagnostics,
    refetchInterval: false,
    retry: 1,
  })

  const backendUnavailable =
    health.isError || readiness.isError || providers.isError

  return (
    <div className="overview">
      <h1>Operational Overview</h1>
      {backendUnavailable ? (
        <div className="banner banner-error">
          Backend disconnected or unreachable. Start AI Hub on port 8080 and use Reconnect SSE after it is healthy.
        </div>
      ) : null}

      <section className="card-grid">
        <article className="card">
          <h2>System liveness</h2>
          {health.isLoading ? <p>Loading…</p> : null}
          {health.data ? (
            <p className={statusClass(health.data.status)}>{health.data.status.toUpperCase()}</p>
          ) : null}
        </article>
        <article className="card">
          <h2>Database readiness</h2>
          {readiness.isLoading ? <p>Loading…</p> : null}
          {readiness.data ? (
            <>
              <p className={statusClass(readiness.data.status)}>{readiness.data.status.toUpperCase()}</p>
              {readiness.data.checks ? (
                <ul>
                  {Object.entries(readiness.data.checks).map(([name, value]) => (
                    <li key={name}>
                      {name}: <span className={statusClass(value)}>{value}</span>
                    </li>
                  ))}
                </ul>
              ) : null}
            </>
          ) : null}
        </article>
      </section>

      <section>
        <h2>Providers</h2>
        {providers.isLoading ? <p>Loading…</p> : null}
        <div className="provider-grid">
          {(providers.data?.providers ?? []).map((provider) => (
            <article key={provider.id} className="card provider-card">
              <h3>{provider.id}</h3>
              <p>Enabled: {provider.enabled ? 'yes' : 'no'}</p>
              <p className={statusClass(provider.state)}>{provider.state}</p>
            </article>
          ))}
        </div>
      </section>
    </div>
  )
}
