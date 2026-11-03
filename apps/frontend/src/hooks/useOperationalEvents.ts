import { useEffect, useRef, useState } from 'react'
import type { QueryClient } from '@tanstack/react-query'
import { queryKeys, type ProviderDiagnostic, type ReadinessReport } from '../api/client'

type OperationalEnvelope = {
  schema_version: number
}

type ReadinessEvent = OperationalEnvelope & ReadinessReport

type ProviderStatusEvent = OperationalEnvelope & {
  providers: ProviderDiagnostic[]
}

export type OperationalConnectionState = 'connecting' | 'live' | 'degraded'

export function useOperationalEvents(queryClient: QueryClient): {
  connectionState: OperationalConnectionState
  reconnect: () => void
} {
  const [connectionState, setConnectionState] = useState<OperationalConnectionState>('connecting')
  const [generation, setGeneration] = useState(0)
  const sourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    const source = new EventSource('/api/events')
    sourceRef.current = source
    setConnectionState('connecting')

    const onOpen = (): void => {
      setConnectionState('live')
    }

    const onError = (): void => {
      setConnectionState('degraded')
    }

    const onReadiness = (event: MessageEvent<string>): void => {
      try {
        const payload = JSON.parse(event.data) as ReadinessEvent
        queryClient.setQueryData(queryKeys.readiness, {
          status: payload.status,
          checks: payload.checks,
        } satisfies ReadinessReport)
      } catch {
        setConnectionState('degraded')
      }
    }

    const onProviders = (event: MessageEvent<string>): void => {
      try {
        const payload = JSON.parse(event.data) as ProviderStatusEvent
        queryClient.setQueryData(queryKeys.providerDiagnostics, {
          providers: payload.providers,
        })
      } catch {
        setConnectionState('degraded')
      }
    }

    source.addEventListener('open', onOpen)
    source.addEventListener('error', onError)
    source.addEventListener('readiness_changed', onReadiness as EventListener)
    source.addEventListener('provider_status', onProviders as EventListener)

    return () => {
      source.removeEventListener('open', onOpen)
      source.removeEventListener('error', onError)
      source.removeEventListener('readiness_changed', onReadiness as EventListener)
      source.removeEventListener('provider_status', onProviders as EventListener)
      source.close()
      sourceRef.current = null
    }
  }, [queryClient, generation])

  const reconnect = (): void => {
    sourceRef.current?.close()
    setGeneration((value) => value + 1)
  }

  return { connectionState, reconnect }
}
