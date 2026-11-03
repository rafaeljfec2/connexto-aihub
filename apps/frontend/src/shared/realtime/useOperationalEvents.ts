import { useEffect, useRef, useState } from 'react'
import type { QueryClient } from '@tanstack/react-query'
import {
  operationalEventNames,
  type ProviderStatusEvent,
  type ReadinessChangedEvent,
} from './operationalEvents'

const INITIAL_BACKOFF_MS = 1_000
const MAX_BACKOFF_MS = 30_000

export type OperationalConnectionState = 'connecting' | 'live' | 'degraded'

export type OperationalCacheKeys = {
  readiness: readonly unknown[]
  providerDiagnostics: readonly unknown[]
}

export function useOperationalEvents(
  queryClient: QueryClient,
  cacheKeys: OperationalCacheKeys,
): {
  connectionState: OperationalConnectionState
  reconnect: () => void
} {
  const [connectionState, setConnectionState] = useState<OperationalConnectionState>('connecting')
  const [generation, setGeneration] = useState(0)
  const sourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    let cancelled = false
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined
    let backoffMs = INITIAL_BACKOFF_MS
    let detachSource: (() => void) | undefined

    const scheduleReconnect = (): void => {
      if (cancelled) {
        return
      }
      reconnectTimer = setTimeout(() => {
        backoffMs = Math.min(backoffMs * 2, MAX_BACKOFF_MS)
        connect()
      }, backoffMs)
    }

    const connect = (): void => {
      if (cancelled) {
        return
      }
      detachSource?.()
      detachSource = undefined

      const source = new EventSource('/api/events')
      sourceRef.current = source
      setConnectionState('connecting')

      const onOpen = (): void => {
        backoffMs = INITIAL_BACKOFF_MS
        setConnectionState('live')
      }

      const onError = (): void => {
        setConnectionState('degraded')
        detachSource?.()
        detachSource = undefined
        sourceRef.current = null
        scheduleReconnect()
      }

      const onReadiness = (event: MessageEvent<string>): void => {
        try {
          const payload = JSON.parse(event.data) as ReadinessChangedEvent
          queryClient.setQueryData(cacheKeys.readiness, {
            status: payload.status,
            checks: payload.checks,
          })
        } catch {
          setConnectionState('degraded')
        }
      }

      const onProviders = (event: MessageEvent<string>): void => {
        try {
          const payload = JSON.parse(event.data) as ProviderStatusEvent
          queryClient.setQueryData(cacheKeys.providerDiagnostics, {
            providers: payload.providers,
          })
        } catch {
          setConnectionState('degraded')
        }
      }

      source.addEventListener('open', onOpen)
      source.addEventListener('error', onError)
      source.addEventListener(operationalEventNames.readinessChanged, onReadiness as EventListener)
      source.addEventListener(operationalEventNames.providerStatus, onProviders as EventListener)

      detachSource = (): void => {
        source.removeEventListener('open', onOpen)
        source.removeEventListener('error', onError)
        source.removeEventListener(operationalEventNames.readinessChanged, onReadiness as EventListener)
        source.removeEventListener(operationalEventNames.providerStatus, onProviders as EventListener)
        source.close()
      }
    }

    connect()

    return () => {
      cancelled = true
      if (reconnectTimer) {
        clearTimeout(reconnectTimer)
      }
      detachSource?.()
      detachSource = undefined
      sourceRef.current = null
    }
  }, [cacheKeys.providerDiagnostics, cacheKeys.readiness, generation, queryClient])

  const reconnect = (): void => {
    sourceRef.current?.close()
    setGeneration((value) => value + 1)
  }

  return { connectionState, reconnect }
}
