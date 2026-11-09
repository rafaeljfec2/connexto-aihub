import { useEffect, useRef, useState } from 'react'
import { useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import {
  cursorQueryKeys,
  getCursorUsageAnalytics,
  getCursorUsageAnalyticsForce,
} from '../api/queries'
import type { CursorUsageAnalyticsEnvelope } from '../types'

const usageRefreshPollIntervalMs = 1500
/** Upper bound mirrors the slowest proxy collection budget (about 120s). */
const usageRefreshTimeoutMs = 120_000

export type CursorUsageQueryState = Readonly<{
  envelope?: CursorUsageAnalyticsEnvelope
  hasInitialError: boolean
  isInitialLoading: boolean
  hasRefetchError: boolean
  isRefreshing: boolean
  isBackgroundRefreshing: boolean
  refresh: () => void
  retry: () => void
}>

function refreshDelay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function isCollectionRunning(envelope: CursorUsageAnalyticsEnvelope): boolean {
  return envelope.refreshing === true || envelope.status === 'collecting'
}

/**
 * Forces a backend collection and polls the regular endpoint until the refresh
 * settles. Cached content stays visible the whole time; the cache only swaps
 * once the physical collection lands.
 */
async function runForcedUsageRefresh(queryClient: QueryClient, providerId: string): Promise<boolean> {
  const queryKey = cursorQueryKeys.usageAnalytics(providerId)
  const forced = await getCursorUsageAnalyticsForce(providerId)
  queryClient.setQueryData(queryKey, forced)
  if (!isCollectionRunning(forced)) return forced.status === 'available'

  const deadline = Date.now() + usageRefreshTimeoutMs
  while (Date.now() < deadline) {
    await refreshDelay(usageRefreshPollIntervalMs)
    const latest = await getCursorUsageAnalytics(providerId)
    queryClient.setQueryData(queryKey, latest)
    if (!isCollectionRunning(latest)) {
      return latest.status === 'available' && latest.analytics !== undefined && latest.error === undefined
    }
  }
  return false
}

export function useCursorUsageQueries(providerId: string): CursorUsageQueryState {
  const queryClient = useQueryClient()
  const usageQuery = useQuery({
    queryKey: cursorQueryKeys.usageAnalytics(providerId),
    queryFn: () => getCursorUsageAnalytics(providerId),
    refetchInterval: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: 1,
  })

  const [isManualRefreshing, setIsManualRefreshing] = useState(false)
  const [manualRefreshFailed, setManualRefreshFailed] = useState(false)
  const refreshInFlightRef = useRef(false)

  // Cold backend start: the first envelope arrives with status collecting; keep
  // polling the regular endpoint until the background collection lands.
  useEffect(() => {
    if (usageQuery.data?.status !== 'collecting') return
    let cancelled = false
    const timer = setInterval(() => {
      if (cancelled) return
      void getCursorUsageAnalytics(providerId)
        .then((next) => {
          if (cancelled) return undefined
          queryClient.setQueryData(cursorQueryKeys.usageAnalytics(providerId), next)
          return next
        })
        .catch(() => undefined)
    }, usageRefreshPollIntervalMs)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [providerId, queryClient, usageQuery.data?.status])

  const hasInitialError = usageQuery.isError && usageQuery.data === undefined

  return {
    envelope: usageQuery.data,
    hasInitialError,
    isInitialLoading: !hasInitialError && usageQuery.isLoading,
    hasRefetchError:
      (usageQuery.isRefetchError && usageQuery.data !== undefined) || manualRefreshFailed,
    isRefreshing: usageQuery.isFetching || isManualRefreshing,
    isBackgroundRefreshing: !usageQuery.isLoading && (usageQuery.isFetching || isManualRefreshing),
    refresh: () => {
      if (refreshInFlightRef.current) return
      refreshInFlightRef.current = true
      setIsManualRefreshing(true)
      setManualRefreshFailed(false)
      void runForcedUsageRefresh(queryClient, providerId)
        .then((settledWithFreshData) => {
          if (!settledWithFreshData) setManualRefreshFailed(true)
        })
        .catch(() => setManualRefreshFailed(true))
        .finally(() => {
          refreshInFlightRef.current = false
          setIsManualRefreshing(false)
        })
    },
    retry: () => {
      void usageQuery.refetch()
    },
  }
}
