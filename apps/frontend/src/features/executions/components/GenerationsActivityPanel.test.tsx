import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import type { Generation } from '../types'
import { GenerationsActivityPanel, buildGenerationDailySeries } from './GenerationsActivityPanel'

function makeGeneration(executionId: string, startedAt: string): Generation {
  return {
    execution_id: executionId,
    request_id: null,
    provider_id: 'openai',
    model_id: 'gpt-4.1',
    started_at: startedAt,
    completed_at: startedAt,
    duration_ns: 1_000_000,
    outcome: 'completed',
    phase: 'completed',
    cause: null,
    upstream_status: 200,
    usage: {
      status: 'reported',
      source: 'provider',
      input_tokens: 10,
      output_tokens: 20,
      provider_reported_cost: null,
      provider_list_cost: null,
      currency: null,
    },
    telemetry: {
      streaming: false,
      provider_latency_ns: 1_000_000,
      routing_overhead_ns: 10_000,
      time_to_first_token_ns: null,
      output_stream_duration_ns: null,
      finish_reason: 'stop',
      client_app: null,
    },
    enrichment_status: 'available',
  }
}

afterEach(() => {
  cleanup()
})

describe('GenerationsActivityPanel', () => {
  it('renders an empty state instead of hardcoded bars when no generations are loaded', () => {
    render(<GenerationsActivityPanel generations={[]} />)

    expect(screen.getByText('0 GERAÇÕES CARREGADAS · 30 DIAS')).toBeTruthy()
    expect(screen.getByText('Nenhuma geração carregada para exibir atividade no período.')).toBeTruthy()
  })

  it('aggregates loaded generations into a 30-day UTC series', () => {
    const now = new Date('2026-11-12T12:00:00Z')
    const series = buildGenerationDailySeries(
      [
        makeGeneration('g-1', '2026-11-12T08:00:00Z'),
        makeGeneration('g-2', '2026-11-12T10:00:00Z'),
        makeGeneration('g-3', '2026-11-10T09:00:00Z'),
      ],
      30,
      now,
    )

    expect(series).toHaveLength(30)
    const nov12 = series.find((day) => day.dateKey === '2026-11-12')
    const nov10 = series.find((day) => day.dateKey === '2026-11-10')
    const nov11 = series.find((day) => day.dateKey === '2026-11-11')

    expect(nov12?.count).toBe(2)
    expect(nov12?.heightPercent).toBe(100)
    expect(nov10?.count).toBe(1)
    expect(nov10?.heightPercent).toBe(50)
    expect(nov11?.count).toBe(0)
    expect(nov11?.heightPercent).toBe(0)
  })
})
