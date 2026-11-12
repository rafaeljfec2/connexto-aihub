import { describe, expect, it } from 'vitest'
import { formatGenerationThroughput, formatGenerationTokens, formatProviderReportedCost } from './formatGeneration'
import type { Generation } from '../types'

const generation: Generation = {
  execution_id: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  request_id: null,
  provider_id: 'opencode-go',
  model_id: 'vendor/model',
  started_at: '2026-11-03T12:00:00Z',
  completed_at: '2026-11-03T12:00:01Z',
  duration_ns: 1_000_000_000,
  outcome: 'completed',
  phase: 'forwarding',
  cause: null,
  upstream_status: 200,
  usage: {
    status: 'reported',
    source: 'provider_test',
    input_tokens: 0,
    output_tokens: 0,
    provider_reported_cost: '0.000000000000',
    provider_list_cost: null,
    currency: 'USD',
  },
  telemetry: {
    streaming: false,
    provider_latency_ns: 0,
    routing_overhead_ns: null,
    time_to_first_token_ns: null,
    output_stream_duration_ns: 1_000_000_000,
    finish_reason: null,
    client_app: null,
  },
  enrichment_status: 'available',
}

describe('generation value formatting', () => {
  it('preserves explicit zero and labels unknown token values', () => {
    expect(formatGenerationTokens(0)).toBe('0')
    expect(formatGenerationTokens(null)).toBe('Not reported')
  })

  it('preserves provider-reported zero cost and currency', () => {
    expect(formatProviderReportedCost('0.000000000000', 'USD')).toBe('USD 0.000000000000')
    expect(formatProviderReportedCost(null, null)).toBe('Not reported')
  })

  it('calculates throughput only with an observed positive output interval', () => {
    expect(formatGenerationThroughput(generation)).toBe('0.0 tok/s')
    expect(formatGenerationThroughput({
      ...generation,
      telemetry: { ...generation.telemetry, output_stream_duration_ns: 0 },
    })).toBe('Not reported')
    expect(formatGenerationThroughput({
      ...generation,
      usage: { ...generation.usage, output_tokens: null },
    })).toBe('Not reported')
  })
})
