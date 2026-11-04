import { describe, expect, it } from 'vitest'
import {
  formatUtilizationDisplay,
  isNearQuotaLimit,
  quotaUtilizationDisplay,
} from './quotaUtilization'
import type { QuotaUtilizationEntry } from '../types'

const baseEntry: QuotaUtilizationEntry = {
  policy_id: 1,
  scope: 'global',
  daily_limit: 100,
  enabled: true,
  period_start: '2026-11-03T00:00:00Z',
  reset_at: '2026-11-04T00:00:00Z',
  admitted_count: 90,
  remaining_requests: 10,
  utilization_percent: 90,
}

describe('quotaUtilizationDisplay', () => {
  it('returns N/A for disabled policies', () => {
    const display = quotaUtilizationDisplay({ ...baseEntry, enabled: false })
    expect(display.kind).toBe('na')
    expect(formatUtilizationDisplay(display)).toBe('—')
  })

  it('returns N/A when daily limit is zero', () => {
    const display = quotaUtilizationDisplay({ ...baseEntry, daily_limit: 0 })
    expect(display.kind).toBe('na')
  })

  it('returns percent when enabled with limit', () => {
    const display = quotaUtilizationDisplay(baseEntry)
    expect(display).toEqual({ kind: 'percent', value: 90 })
    expect(formatUtilizationDisplay(display)).toBe('90.0%')
  })

  it('detects near-limit utilization', () => {
    expect(isNearQuotaLimit(baseEntry)).toBe(true)
    expect(isNearQuotaLimit({ ...baseEntry, utilization_percent: 50 })).toBe(false)
  })
})
