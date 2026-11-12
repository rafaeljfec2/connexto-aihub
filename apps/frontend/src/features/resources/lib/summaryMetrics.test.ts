import { describe, expect, it } from 'vitest'
import {
  computeResourcesSummary,
  globalConcurrencyOccupancy,
  globalQuotaUtilizationPercent,
} from './summaryMetrics'

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
          scope: 'global',
          enabled: true,
          concurrency_limit: 4,
          active_count: 4,
          available: 0,
        },
      ],
    })

    expect(summary.configuredPolicies).toBe(3)
    expect(summary.globalUtilizationPercent).toBe(92)
    expect(summary.concurrencyActive).toBe(4)
    expect(summary.concurrencyLimit).toBe(4)
    expect(summary.nearLimitAlerts).toBe(2)
  })

  it('does not sum nested scopes for concurrency or aggregate utilization', () => {
    const summary = computeResourcesSummary({
      resourcePolicyCount: 3,
      concurrencyPolicyCount: 2,
      quotaUtilization: [
        {
          policy_id: 1,
          scope: 'global',
          daily_limit: 100,
          enabled: true,
          period_start: '',
          reset_at: '',
          admitted_count: 10,
          remaining_requests: 90,
          utilization_percent: 10,
        },
        {
          policy_id: 2,
          scope: 'provider',
          provider_id: 'openai',
          daily_limit: 50,
          enabled: true,
          period_start: '',
          reset_at: '',
          admitted_count: 45,
          remaining_requests: 5,
          utilization_percent: 95,
        },
      ],
      concurrencySnapshots: [
        {
          scope: 'global',
          enabled: true,
          concurrency_limit: 10,
          active_count: 2,
          available: 8,
        },
        {
          scope: 'provider',
          enabled: true,
          concurrency_limit: 5,
          active_count: 5,
          available: 0,
        },
      ],
    })

    expect(summary.globalUtilizationPercent).toBe(10)
    expect(summary.concurrencyActive).toBe(2)
    expect(summary.concurrencyLimit).toBe(10)
    expect(
      globalConcurrencyOccupancy([
        {
          scope: 'global',
          enabled: true,
          concurrency_limit: 10,
          active_count: 2,
          available: 8,
        },
        {
          scope: 'provider',
          enabled: true,
          concurrency_limit: 5,
          active_count: 5,
          available: 0,
        },
      ]),
    ).toEqual({ active: 2, limit: 10 })
  })

  it('returns null global utilization when no global quota policy is enabled', () => {
    expect(
      globalQuotaUtilizationPercent([
        {
          policy_id: 2,
          scope: 'provider',
          provider_id: 'anthropic',
          daily_limit: 20,
          enabled: true,
          period_start: '',
          reset_at: '',
          admitted_count: 18,
          remaining_requests: 2,
          utilization_percent: 90,
        },
      ]),
    ).toBeNull()
  })
})
