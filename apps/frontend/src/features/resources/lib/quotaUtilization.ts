import type { QuotaUtilizationEntry } from '../types'

export type UtilizationDisplay =
  | { kind: 'percent'; value: number }
  | { kind: 'na'; reason: string }

export function quotaUtilizationDisplay(entry: QuotaUtilizationEntry): UtilizationDisplay {
  if (!entry.enabled) {
    return { kind: 'na', reason: 'Policy disabled' }
  }
  if (entry.daily_limit <= 0) {
    return { kind: 'na', reason: 'No daily limit configured' }
  }
  return { kind: 'percent', value: entry.utilization_percent }
}

export function formatUtilizationDisplay(display: UtilizationDisplay): string {
  if (display.kind === 'na') {
    return '—'
  }
  return `${display.value.toFixed(1)}%`
}

export const NEAR_LIMIT_PERCENT = 90

export function isNearQuotaLimit(entry: QuotaUtilizationEntry): boolean {
  const display = quotaUtilizationDisplay(entry)
  return display.kind === 'percent' && display.value >= NEAR_LIMIT_PERCENT
}

export function isConcurrencySaturated(snapshot: {
  enabled: boolean
  concurrency_limit: number
  available: number
}): boolean {
  if (!snapshot.enabled || snapshot.concurrency_limit <= 0) {
    return false
  }
  return snapshot.available <= 0
}
