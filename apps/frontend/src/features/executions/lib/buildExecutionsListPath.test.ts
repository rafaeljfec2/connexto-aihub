import { describe, expect, it } from 'vitest'
import { buildExecutionsListPath } from '../api/queries'

describe('buildExecutionsListPath', () => {
  it('builds query string for supported filters only', () => {
    const path = buildExecutionsListPath({
      limit: 25,
      provider_id: 'opencode-go',
      model_id: 'vendor/model',
      outcome: 'failed',
      cursor: 'abc',
    })
    expect(path).toBe(
      '/v1/records/executions?limit=25&provider_id=opencode-go&model_id=vendor%2Fmodel&outcome=failed&cursor=abc',
    )
  })

  it('returns base path when no filters are set', () => {
    expect(buildExecutionsListPath({})).toBe('/v1/records/executions')
  })
})
