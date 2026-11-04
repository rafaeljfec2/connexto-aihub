import type { ProviderQuotaMetric, ProviderQuotaSnapshot } from '../types/quota'

export type GoPlusMeterId = 'go_rolling' | 'go_weekly' | 'go_monthly'

export type GoPlusMeterView = Readonly<{
  id: GoPlusMeterId
  label: string
  percent: number
  resetsAt?: string
}>

const meterOrder: readonly GoPlusMeterId[] = ['go_rolling', 'go_weekly', 'go_monthly']

const meterLabels: Record<GoPlusMeterId, string> = {
  go_rolling: '5-hour window',
  go_weekly: 'Weekly',
  go_monthly: 'Monthly',
}

export function isGoPlusMeterId(metric: string): metric is GoPlusMeterId {
  return metric === 'go_rolling' || metric === 'go_weekly' || metric === 'go_monthly'
}

export function formatGoProductLabel(workspaceName?: string): string {
  const raw = workspaceName?.trim()
  if (!raw) return 'Go Plus'
  const normalized = raw.toLowerCase()
  if (normalized === 'go-plus' || normalized === 'go plus') return 'Go Plus'
  return raw
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

export function productUsageTitle(workspaceName?: string): string {
  return `${formatGoProductLabel(workspaceName)} usage`
}

export function consoleBudgetsFromSnapshot(
  snapshot: ProviderQuotaSnapshot,
): ProviderQuotaMetric[] {
  return snapshot.quotas.filter(
    (metric) => metric.metric.startsWith('budget_') && metric.nature === 'cost',
  )
}

export function selectOpenCodeGoSnapshot(
  snapshots: readonly ProviderQuotaSnapshot[],
): ProviderQuotaSnapshot | undefined {
  return snapshots.find(
    (snapshot) =>
      snapshot.status === 'available' &&
      (snapshot.provider_id === 'opencode-go' || snapshot.scope === 'subscription') &&
      snapshot.quotas.some(
        (metric) => isGoPlusMeterId(metric.metric) && metric.consumed_percent != null,
      ),
  )
}

export function goPlusMetersFromSnapshot(snapshot: ProviderQuotaSnapshot): GoPlusMeterView[] {
  const byId = new Map<string, ProviderQuotaMetric>()
  for (const metric of snapshot.quotas) {
    byId.set(metric.metric, metric)
  }

  const meters: GoPlusMeterView[] = []
  for (const id of meterOrder) {
    const metric = byId.get(id)
    if (metric?.consumed_percent == null) continue
    meters.push({
      id,
      label: meterLabels[id],
      percent: metric.consumed_percent,
      resetsAt: metric.resets_at,
    })
  }
  return meters
}

/** Compact chip follows Console top bar: prefer weekly percent when present. */
export function goPlusChipPercent(meters: readonly GoPlusMeterView[]): number | undefined {
  const weekly = meters.find((meter) => meter.id === 'go_weekly')
  if (weekly) return weekly.percent
  return meters[0]?.percent
}
