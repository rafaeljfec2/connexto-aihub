import { useQuery } from '@tanstack/react-query'
import {
  getHealth,
  getProviderDiagnostics,
  getReadiness,
  overviewQueryKeys,
} from '../api/queries'

export function useOverviewQueries() {
  const health = useQuery({
    queryKey: overviewQueryKeys.health,
    queryFn: getHealth,
    refetchInterval: false,
    retry: 1,
  })

  const readiness = useQuery({
    queryKey: overviewQueryKeys.readiness,
    queryFn: getReadiness,
    refetchInterval: false,
    retry: 1,
  })

  const providers = useQuery({
    queryKey: overviewQueryKeys.providerDiagnostics,
    queryFn: getProviderDiagnostics,
    refetchInterval: false,
    retry: 1,
  })

  const backendUnavailable =
    health.isError || readiness.isError || providers.isError

  return { health, readiness, providers, backendUnavailable }
}
