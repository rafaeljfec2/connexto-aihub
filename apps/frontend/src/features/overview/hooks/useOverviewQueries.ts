import { useQuery } from '@tanstack/react-query'
import {
  getHealth,
  getProviderDiagnostics,
  getReadiness,
  overviewQueryKeys,
} from '../api/queries'
import { listCatalogModels } from '../../catalog/api/queries'
import { listExecutionRecords } from '../../executions/api/queries'

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

  const catalog = useQuery({
    queryKey: ['catalog', 'models', 'overview-summary'],
    queryFn: () => listCatalogModels({}),
    refetchInterval: false,
    retry: 1,
  })

  const recentExecutions = useQuery({
    queryKey: ['executions', 'list', { limit: 5 }],
    queryFn: () => listExecutionRecords({ limit: 5 }),
    refetchInterval: false,
    retry: 1,
  })

  const backendUnavailable =
    health.isError || readiness.isError || providers.isError

  const refetchAll = async () => {
    await Promise.all([
      health.refetch(),
      readiness.refetch(),
      providers.refetch(),
      catalog.refetch(),
      recentExecutions.refetch(),
    ])
  }

  return {
    health,
    readiness,
    providers,
    catalog,
    recentExecutions,
    backendUnavailable,
    refetchAll,
  }
}
