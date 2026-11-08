import { useMemo, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useAppPageMeta } from '../../../app/layouts/AppPageHeader'
import { getProviderQuotas, getProviderQuotasForce, listProviders, providersQueryKeys } from '../api/queries'
import type { ProviderQuotaSnapshot, ProviderQuotasResponse } from '../../overview/types/quota'
import { ProviderQuotasPageView } from './ProviderQuotasPageView'
import type { AdminProvider } from '../types'
import { primaryQuotaSnapshot } from '../lib/quotaPresentation'

export function ProvidersPage() {
  const meta = useAppPageMeta()
  const queryState = useProvidersPageQueries()
  const [selectedProvider, setSelectedProvider] = useState<AdminProvider | null>(null)
  const [manageTarget, setManageTarget] = useState<AdminProvider | null>(null)
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const { refresh, ...viewState } = queryState
  const activeSelected = selectCurrentProvider(selectedProvider, queryState.providers)
  const activeSnapshot = useActiveSnapshot(activeSelected, queryState.snapshots)
  const manageProvider = currentManageProvider(manageTarget, queryState.providers)

  return (
    <ProviderQuotasPageView
      meta={meta}
      {...viewState}
      selectedProvider={activeSelected}
      selectedSnapshot={activeSnapshot}
      manageProvider={manageProvider}
      isAddModalOpen={isAddModalOpen}
      hasProviders={queryState.providers.length > 0}
      onSelectProvider={setSelectedProvider}
      onOpenManage={setManageTarget}
      onCloseManage={() => setManageTarget(null)}
      onAddProvider={() => setIsAddModalOpen(true)}
      onAddModalChange={setIsAddModalOpen}
      onRefresh={refresh}
      onProviderAdded={(provider) => {
        setSelectedProvider(provider)
        refresh()
      }}
    />
  )
}

function useProvidersPageQueries() {
  const queryClient = useQueryClient()
  const providersQuery = useQuery({
    queryKey: providersQueryKeys.list,
    queryFn: listProviders,
    refetchInterval: false,
    retry: 1,
  })
  const quotasQuery = useQuery({
    queryKey: providersQueryKeys.quotas,
    queryFn: getProviderQuotas,
    refetchInterval: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: 1,
  })

  const [isManualRefreshing, setIsManualRefreshing] = useState(false)
  const [manualRefreshFailed, setManualRefreshFailed] = useState(false)
  const refreshInFlightRef = useRef(false)

  const hasInitialError =
    (providersQuery.isError && providersQuery.data === undefined) ||
    (quotasQuery.isError && quotasQuery.data === undefined)
  const isInitialLoading =
    !hasInitialError && (providersQuery.isLoading || quotasQuery.isLoading)
  const isRefreshing =
    providersQuery.isFetching || quotasQuery.isFetching || isManualRefreshing

  return {
    providers: providersQuery.data?.providers ?? [],
    snapshots: quotasQuery.data?.snapshots ?? [],
    hasInitialError,
    isInitialLoading,
    hasRefetchError:
      (providersQuery.isRefetchError && providersQuery.data !== undefined) ||
      (quotasQuery.isRefetchError && quotasQuery.data !== undefined) ||
      manualRefreshFailed,
    isRefreshing,
    isBackgroundRefreshing: !isInitialLoading && isRefreshing,
    refresh: () => {
      if (refreshInFlightRef.current) return
      refreshInFlightRef.current = true
      setIsManualRefreshing(true)
      setManualRefreshFailed(false)
      void runForcedQuotaRefresh(queryClient)
        .then((completed) => {
          if (!completed) setManualRefreshFailed(true)
        })
        .catch(() => setManualRefreshFailed(true))
        .finally(() => {
          refreshInFlightRef.current = false
          setIsManualRefreshing(false)
        })
      void providersQuery.refetch()
    },
  }
}

const quotaRefreshPollIntervalMs = 1500
/** Upper bound matches the slowest provider collection budget (Cursor browser automation, ~120s). */
const quotaRefreshTimeoutMs = 120_000

function quotaRefreshDelay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function isQuotaCollectionRunning(response: ProviderQuotasResponse): boolean {
  return response.snapshots.some((snapshot) => snapshot.refreshing === true)
}

/**
 * Forces the backend to collect quotas from the providers physically, then polls the
 * regular endpoint until every background collection settles. Cached snapshots remain
 * visible the whole time; the cache only swaps once the actual collected data arrives.
 */
async function runForcedQuotaRefresh(
  queryClient: ReturnType<typeof useQueryClient>,
): Promise<boolean> {
  const forced = await getProviderQuotasForce()
  queryClient.setQueryData(providersQueryKeys.quotas, forced)
  if (!isQuotaCollectionRunning(forced)) return true

  const deadline = Date.now() + quotaRefreshTimeoutMs
  while (Date.now() < deadline) {
    await quotaRefreshDelay(quotaRefreshPollIntervalMs)
    const latest = await getProviderQuotas()
    queryClient.setQueryData(providersQueryKeys.quotas, latest)
    if (!isQuotaCollectionRunning(latest)) return true
  }
  return false
}

function selectCurrentProvider(
  selectedProvider: AdminProvider | null,
  providers: AdminProvider[],
): AdminProvider | null {
  if (selectedProvider === null) return providers[0] ?? null
  return providers.find((provider) => provider.id === selectedProvider.id) ?? providers[0] ?? null
}

function useActiveSnapshot(
  provider: AdminProvider | null,
  snapshots: ProviderQuotaSnapshot[],
): ProviderQuotaSnapshot | undefined {
  return useMemo(
    () => (provider ? primaryQuotaSnapshot(provider.id, snapshots) : undefined),
    [provider, snapshots],
  )
}

function currentManageProvider(
  manageTarget: AdminProvider | null,
  providers: AdminProvider[],
): AdminProvider | null {
  if (manageTarget === null) return null
  return providers.find((provider) => provider.id === manageTarget.id) ?? manageTarget
}
