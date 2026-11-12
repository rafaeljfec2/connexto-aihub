import { describe, expect, it } from 'vitest'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import {
  cursorMonthlyAdditionalUsage,
  cursorPrepaidBalanceUsd,
  cursorWeeklyByProduct,
  cursorWeeklyIncludedStatus,
  formatCursorWeeklyPercent,
} from './cursorQuotaMetrics'

const expandedCursorSnapshot: ProviderQuotaSnapshot = {
  account_id: 'cursor-1',
  account_label: 'Cursor',
  provider_id: 'cursor',
  provider_name: 'Cursor',
  status: 'available',
  scope: 'account',
  quotas: [
    { metric: 'total_usage', label: 'Total account usage', nature: 'cost', used: 0.18, unit: 'USD' },
    {
      metric: 'total_credits',
      label: 'Weekly included budget',
      nature: 'quota',
      limit: 161.59,
      unit: 'USD',
      consumed_percent: 0.11139,
      starts_at: '2026-11-09T21:45:07.488Z',
    },
    {
      metric: 'weekly_used_percent',
      label: 'Weekly used percent',
      nature: 'usage',
      consumed_percent: 0.11139,
    },
    {
      metric: 'weekly_included_status',
      label: 'Weekly included status',
      nature: 'usage',
      scope: 'weekly_status:metered',
    },
    {
      metric: 'weekly_product_cursor',
      label: 'Weekly usage · Cursor',
      nature: 'usage',
      used: 0.18,
      consumed_percent: 0.11139,
      unit: 'USD',
      scope: 'product:cursor',
    },
    {
      metric: 'monthly_additional_usage',
      label: 'Monthly additional usage',
      nature: 'cost',
      used: 9.22,
      unit: 'USD',
      starts_at: '2026-11-01T00:00:00.000Z',
      resets_at: '2026-12-01T00:00:00.000Z',
      scope: 'additional_enabled',
    },
    {
      metric: 'monthly_credits_used',
      label: 'Monthly credits used',
      nature: 'cost',
      used: 0,
      unit: 'USD',
    },
    {
      metric: 'prepaid_available_credits',
      label: 'Prepaid credits available',
      nature: 'quota',
      remaining: 0.79,
      unit: 'USD',
    },
  ],
  source: {
    type: 'provider_api',
    name: 'cursor_proxy_usage_collect',
    scope: 'account',
    fetched_at: '2026-11-11T23:30:00Z',
  },
}

describe('cursorQuotaMetrics', () => {
  it('rounds positive sub-1% values up to 1% and keeps standard rounding >= 1%', () => {
    expect(formatCursorWeeklyPercent(0)).toBe(0)
    expect(formatCursorWeeklyPercent(0.11139)).toBe(1)
    expect(formatCursorWeeklyPercent(90.31)).toBe(90)
    expect(formatCursorWeeklyPercent(100.02)).toBe(100)
  })

  it('extracts per-product weekly usage, monthly additional usage, status, and prepaid balance', () => {
    expect(cursorWeeklyIncludedStatus(expandedCursorSnapshot)).toBe('metered')
    expect(cursorWeeklyByProduct(expandedCursorSnapshot)).toEqual([
      {
        product: 'cursor',
        label: 'Cursor',
        usedUsd: 0.18,
        percent: 1,
      },
    ])
    expect(cursorMonthlyAdditionalUsage(expandedCursorSnapshot)).toEqual({
      usedUsd: 9.22,
      limitUsd: undefined,
      creditsUsedUsd: 0,
      additionalUsageOff: false,
      startsAt: '2026-11-01T00:00:00.000Z',
      resetsAt: '2026-12-01T00:00:00.000Z',
    })
    expect(cursorPrepaidBalanceUsd(expandedCursorSnapshot)).toBe(0.79)
  })
})
