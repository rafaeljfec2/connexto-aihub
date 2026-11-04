import { describe, expect, it } from 'vitest'
import { computeResourcesSummary } from './summaryMetrics'

describe('computeResourcesSummary', () => {
  it('aggregates policy counts and near-limit alerts', () => {
    const summary = computeResourcesSummary({
      resourcePolicyCount: 2,
      concurrencyPolicyCount: 1,
      quotaUtilization: [
        {
          policy_id: 1,
          scope: 'global',
          daily_limit: 10,
          enabled: true,
          period_start: '',
          reset_at: '',
          admitted_count: 9,
          remaining_requests: 1,
          utilization_percent: 92,
        },
      ],
      concurrencySnapshots: [
        {
          enabled: true,
          concurrency_limit: 4,
          active_count: 4,
          available: 0,
        },
      ],
    })

    expect(summary.configuredPolicies).toBe(3)
    expect(summary.peakUtilizationPercent).toBe(92)
    expect(summary.concurrencyActive).toBe(4)
    expect(summary.concurrencyLimit).toBe(4)
    expect(summary.nearLimitAlerts).toBe(2)
  })
})
