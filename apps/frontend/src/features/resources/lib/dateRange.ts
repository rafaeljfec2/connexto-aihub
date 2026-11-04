import type { AnalyticsDateRange } from '../types'

export function formatDateOnlyUtc(date: Date): string {
  return date.toISOString().slice(0, 10)
}

export function defaultAnalyticsRange(): AnalyticsDateRange {
  const to = new Date()
  const from = new Date(to)
  from.setUTCDate(from.getUTCDate() - 6)
  return {
    from: formatDateOnlyUtc(from),
    to: formatDateOnlyUtc(to),
  }
}
