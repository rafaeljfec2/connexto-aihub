import { describe, expect, it } from 'vitest'
import { buildProviderUsageDistribution } from './usageDistribution'

describe('buildProviderUsageDistribution', () => {
  it('uses token totals when every provider reports tokens', () => {
    const slices = buildProviderUsageDistribution([
      {
        period_start: '2026-11-01T00:00:00Z',
        provider_id: 'openai',
        input_tokens: 100,
        output_tokens: 50,
        total_tokens: 150,
        executions: 10,
      },
      {
        period_start: '2026-11-01T00:00:00Z',
        provider_id: 'anthropic',
        input_tokens: 50,
        output_tokens: 50,
        total_tokens: 100,
        executions: 8,
      },
    ])

    expect(slices).toHaveLength(2)
    expect(slices[0]?.percent).toBe(60)
    expect(slices[1]?.percent).toBe(40)
  })

  it('falls back to executions when any total_tokens is null', () => {
    const slices = buildProviderUsageDistribution([
      {
        period_start: '2026-11-01T00:00:00Z',
        provider_id: 'openai',
        input_tokens: null,
        output_tokens: null,
        total_tokens: null,
        executions: 3,
      },
      {
        period_start: '2026-11-01T00:00:00Z',
        provider_id: 'anthropic',
        input_tokens: 10,
        output_tokens: 10,
        total_tokens: 20,
        executions: 7,
      },
    ])

    expect(slices[0]?.percent).toBeCloseTo(30, 0)
    expect(slices[1]?.percent).toBeCloseTo(70, 0)
  })

  it('returns an empty list for no rows', () => {
    expect(buildProviderUsageDistribution([])).toEqual([])
  })
})
