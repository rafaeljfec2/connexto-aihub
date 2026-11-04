import { useQuery } from '@tanstack/react-query'
import {
  getHealth,
  getProviderDiagnostics,
  getProviderQuotas,
  getReadiness,
  getRuntimeMeta,
  getTrafficSummary,
  overviewQueryKeys,
} from '../api/queries'
import { listCatalogModels } from '../../catalog/api/queries'
import { listExecutionRecords } from '../../executions/api/queries'
import { getQuotaUtilization } from '../../resources/api/queries'

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

  const runtimeMeta = useQuery({
    queryKey: overviewQueryKeys.runtimeMeta,
    queryFn: getRuntimeMeta,
    refetchInterval: false,
    retry: 1,
  })

  const trafficSummary = useQuery({
    queryKey: overviewQueryKeys.trafficSummary('24h'),
    queryFn: () => getTrafficSummary('24h'),
    refetchInterval: false,
    retry: 1,
  })

  const providerQuotas = useQuery({
    queryKey: overviewQueryKeys.providerQuotas,
    queryFn: getProviderQuotas,
    refetchInterval: false,
    retry: 1,
  })

  const quotaUtilization = useQuery({
    queryKey: ['resources', 'quota-utilization', 'overview'],
    queryFn: () => getQuotaUtilization({}),
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
      providerQuotas.refetch(),
      runtimeMeta.refetch(),
      trafficSummary.refetch(),
      quotaUtilization.refetch(),
      catalog.refetch(),
      recentExecutions.refetch(),
    ])
  }

  return {
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
  }
}
