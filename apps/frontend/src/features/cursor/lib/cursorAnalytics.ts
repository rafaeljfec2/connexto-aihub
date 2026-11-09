import type { CursorDailyMetric, CursorUsageNamedCount } from '../types'

export type CursorDailyMetricOption = 'agent' | 'chat' | 'lines'

const numericSummaryKeys = [
  'linesAdded',
  'linesDeleted',
  'acceptedLinesAdded',
  'acceptedLinesDeleted',
  'totalApplies',
  'totalAccepts',
  'totalRejects',
  'totalTabsShown',
  'totalTabsAccepted',
  'chatRequests',
  'agentRequests',
  'subscriptionIncludedReqs',
  'usageBasedReqs',
  'cmdkUsages',
] as const

type NumericSummaryKey = (typeof numericSummaryKeys)[number]

export type CursorDailySummary = Readonly<
  Partial<Record<NumericSummaryKey, number>> & {
    activeDays: number
    firstActivityAt?: string
    lastActivityAt?: string
    modelUsages?: number
  }
>

interface MutableCursorDailySummary {
  activeDays: number
  firstActivityAt?: string
  lastActivityAt?: string
  modelUsages?: number
  values: Map<NumericSummaryKey, number>
}

function hasActivity(day: CursorDailyMetric): boolean {
  return (
    numericSummaryKeys.some((key) => day[key] !== undefined) ||
    (day.modelUsage?.length ?? 0) > 0 ||
    (day.extensionUsage?.length ?? 0) > 0 ||
    (day.tabExtensionUsage?.length ?? 0) > 0
  )
}

function modelUsageTotal(entries: readonly CursorUsageNamedCount[]): number {
  return entries.reduce((total, entry) => total + (entry.count ?? 1), 0)
}

/**
 * Aggregates the console daily metrics into totals for the summary cards.
 * Fields the source never reports stay undefined (never rendered as zero),
 * per the provider-reported-only contract.
 */
export function summarizeCursorDailyMetrics(
  metrics: readonly CursorDailyMetric[],
): CursorDailySummary {
  const accumulator: MutableCursorDailySummary = { activeDays: 0, values: new Map() }

  for (const day of metrics) {
    if (!hasActivity(day)) continue
    if (accumulator.activeDays === 0) accumulator.firstActivityAt = day.date
    accumulator.lastActivityAt = day.date
    accumulator.activeDays += 1
    for (const key of numericSummaryKeys) {
      const value = day[key]
      if (value === undefined) continue
      accumulator.values.set(key, (accumulator.values.get(key) ?? 0) + value)
    }
    if (day.modelUsage && day.modelUsage.length > 0) {
      accumulator.modelUsages = (accumulator.modelUsages ?? 0) + modelUsageTotal(day.modelUsage)
    }
  }

  const values = Object.fromEntries(accumulator.values) as Partial<Record<NumericSummaryKey, number>>
  return {
    ...values,
    activeDays: accumulator.activeDays,
    firstActivityAt: accumulator.firstActivityAt,
    lastActivityAt: accumulator.lastActivityAt,
    modelUsages: accumulator.modelUsages,
  }
}

export interface CursorModelUsageRankingEntry {
  name: string
  count: number
}

/**
 * Ranks model usage across days. Entries without a name cannot be ranked and are
 * skipped; entries without a count count as a single usage (console behavior).
 */
export function rankCursorModelUsage(
  metrics: readonly CursorDailyMetric[],
  limit: number,
): CursorModelUsageRankingEntry[] {
  if (limit <= 0) return []
  const counts = new Map<string, number>()

  for (const day of metrics) {
    for (const entry of day.modelUsage ?? []) {
      if (!entry.name?.trim()) continue
      const name = entry.name.trim()
      counts.set(name, (counts.get(name) ?? 0) + (entry.count ?? 1))
    }
  }

  const ranked = [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((left, right) => right.count - left.count)

  if (ranked.length <= limit) return ranked

  const remaining = ranked.slice(limit).reduce((total, entry) => total + entry.count, 0)
  return remaining > 0 ? [...ranked.slice(0, limit), { name: 'Others', count: remaining }] : ranked.slice(0, limit)
}

export interface CursorDailyPoint {
  date: string
  value?: number
  hasData: boolean
}

const dailyMetricValues: Record<CursorDailyMetricOption, (day: CursorDailyMetric) => number | undefined> = {
  agent: (day) => day.agentRequests,
  chat: (day) => day.chatRequests,
  lines: (day) => day.linesAdded,
}

/** Renders an ISO instant in the operator locale, falling back to the raw value. */
export function formatCursorTimestamp(iso: string): string {
  const timestamp = Date.parse(iso)
  if (Number.isNaN(timestamp)) return iso
  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(timestamp))
}

/** Builds a per-day series; days without the requested metric become gaps. */
export function buildCursorDailySeries(
  metrics: readonly CursorDailyMetric[],
  option: CursorDailyMetricOption,
): CursorDailyPoint[] {
  const selector = dailyMetricValues[option]
  return [...metrics]
    .sort((left, right) => Number(left.date) - Number(right.date))
    .map((day) => {
      const value = selector(day)
      return { date: day.date, value, hasData: value !== undefined }
    })
}
