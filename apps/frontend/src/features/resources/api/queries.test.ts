import { describe, expect, it } from 'vitest'
import {
  buildAnalyticsRangePath,
  buildConcurrencySnapshotPath,
  buildQuotaUtilizationPath,
} from '../api/queries'

describe('resources API paths', () => {
  it('builds analytics range paths with filters', () => {
    const path = buildAnalyticsRangePath(
      '/api/resource-analytics/provider-usage',
      { from: '2026-11-01', to: '2026-11-07' },
      { provider_id: 'opencode-go', model_id: 'm1' },
    )
    expect(path).toBe(
      '/api/resource-analytics/provider-usage?from=2026-11-01&to=2026-11-07&provider_id=opencode-go&model_id=m1',
    )
  })

  it('builds quota utilization path without empty query', () => {
    expect(buildQuotaUtilizationPath({})).toBe('/api/resource-analytics/quota-utilization')
    expect(buildConcurrencySnapshotPath({ provider_id: 'p1' })).toBe(
      '/api/resource-analytics/concurrency?provider_id=p1',
    )
  })
})
