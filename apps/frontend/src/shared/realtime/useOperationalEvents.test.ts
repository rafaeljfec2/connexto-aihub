import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { QueryClient } from '@tanstack/react-query'
import { renderHook, act } from '@testing-library/react'
import { useOperationalEvents } from './useOperationalEvents'

type MockListener = (event: MessageEvent<string>) => void

class MockEventSource {
  static instances: MockEventSource[] = []
  readonly url: string
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  private readonly listeners = new Map<string, MockListener[]>()

  constructor(url: string) {
    this.url = url
    MockEventSource.instances.push(this)
  }

  addEventListener(type: string, listener: EventListener): void {
    const handlers = this.listeners.get(type) ?? []
    handlers.push(listener as MockListener)
    this.listeners.set(type, handlers)
  }

  removeEventListener(type: string, listener: EventListener): void {
    const handlers = this.listeners.get(type) ?? []
    this.listeners.set(
      type,
      handlers.filter((item) => item !== listener),
    )
  }

  close(): void {
    // no-op for mock
  }

  emitOpen(): void {
    this.onopen?.()
    this.listeners.get('open')?.forEach((handler) => {
      handler(new MessageEvent('open'))
    })
  }

  emit(type: string, data: unknown): void {
    const event = new MessageEvent(type, { data: JSON.stringify(data) })
    this.listeners.get(type)?.forEach((handler) => handler(event))
  }

  emitError(): void {
    this.onerror?.()
    this.listeners.get('error')?.forEach((handler) => {
      handler(new MessageEvent('error'))
    })
  }
}

describe('useOperationalEvents', () => {
  beforeEach(() => {
    MockEventSource.instances = []
    vi.stubGlobal('EventSource', MockEventSource)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('updates readiness and provider caches from SSE payloads', () => {
    const queryClient = new QueryClient()
    const cacheKeys = {
      readiness: ['overview', 'readiness'] as const,
      providerDiagnostics: ['overview', 'providerDiagnostics'] as const,
    }

    renderHook(() => useOperationalEvents(queryClient, cacheKeys))

    const source = MockEventSource.instances[0]
    expect(source.url).toBe('/api/events')

    act(() => {
      source.emitOpen()
    })

    act(() => {
      source.emit('readiness_changed', { schema_version: 1, status: 'up', checks: { database: 'ok' } })
      source.emit('provider_status', {
        schema_version: 1,
        providers: [{ id: 'openai', enabled: true, state: 'healthy' }],
      })
    })

    expect(queryClient.getQueryData(cacheKeys.readiness)).toEqual({
      status: 'up',
      checks: { database: 'ok' },
    })
    expect(queryClient.getQueryData(cacheKeys.providerDiagnostics)).toEqual({
      providers: [{ id: 'openai', enabled: true, state: 'healthy' }],
    })
  })

  it('schedules reconnect after transport error', () => {
    vi.useFakeTimers()
    const queryClient = new QueryClient()
    const cacheKeys = {
      readiness: ['overview', 'readiness'] as const,
      providerDiagnostics: ['overview', 'providerDiagnostics'] as const,
    }

    renderHook(() => useOperationalEvents(queryClient, cacheKeys))
    expect(MockEventSource.instances).toHaveLength(1)

    act(() => {
      MockEventSource.instances[0].emitError()
    })

    act(() => {
      vi.advanceTimersByTime(1_000)
    })

    expect(MockEventSource.instances).toHaveLength(2)
  })
})
