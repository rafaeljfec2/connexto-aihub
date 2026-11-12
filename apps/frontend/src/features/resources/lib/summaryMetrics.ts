import type { QuotaUtilizationEntry, ResourcePolicy, ResourceScope } from '../types'

export function mergePoliciesWithUtilization(
  policies: ResourcePolicy[],
  utilization: QuotaUtilizationEntry[],
): Array<{ policy: ResourcePolicy; utilization?: QuotaUtilizationEntry }> {
  const byPolicyId = new Map(utilization.map((row) => [row.policy_id, row]))
  return policies.map((policy) => ({
    policy,
    utilization: byPolicyId.get(policy.id),
  }))
}

type ConcurrencySnapshotInput = {
  scope: ResourceScope
  enabled: boolean
  concurrency_limit: number
  active_count: number
  available: number
}

function enabledQuotaRows(quotaUtilization: QuotaUtilizationEntry[]): QuotaUtilizationEntry[] {
  return quotaUtilization.filter((row) => row.enabled && row.daily_limit > 0)
}

export function globalQuotaUtilizationPercent(
  quotaUtilization: QuotaUtilizationEntry[],
): number | null {
  const globalRows = enabledQuotaRows(quotaUtilization).filter((row) => row.scope === 'global')
  if (globalRows.length === 0) {
    return null
  }
  return Math.max(...globalRows.map((row) => row.utilization_percent))
}

export function globalConcurrencyOccupancy(
  snapshots: ConcurrencySnapshotInput[],
): { active: number; limit: number } {
  const globalSnapshots = snapshots.filter((row) => row.scope === 'global' && row.enabled)
  if (globalSnapshots.length === 0) {
    return { active: 0, limit: 0 }
  }
  const withLimit = globalSnapshots.find((row) => row.concurrency_limit > 0)
  const primary = withLimit ?? globalSnapshots[0]
  return {
    active: primary.active_count,
    limit: primary.concurrency_limit,
  }
}

export function computeResourcesSummary(input: {
  resourcePolicyCount: number
  concurrencyPolicyCount: number
  quotaUtilization: QuotaUtilizationEntry[]
  concurrencySnapshots: ConcurrencySnapshotInput[]
}): {
  configuredPolicies: number
  globalUtilizationPercent: number | null
  concurrencyActive: number
  concurrencyLimit: number
  nearLimitAlerts: number
} {
  const globalUtilizationPercent = globalQuotaUtilizationPercent(input.quotaUtilization)
  const { active: concurrencyActive, limit: concurrencyLimit } = globalConcurrencyOccupancy(
    input.concurrencySnapshots,
  )

  const enabledQuota = enabledQuotaRows(input.quotaUtilization)
  let nearLimitAlerts = 0
  for (const row of enabledQuota) {
    if (row.utilization_percent >= 90) {
      nearLimitAlerts += 1
    }
  }
  for (const snapshot of input.concurrencySnapshots) {
    if (snapshot.enabled && snapshot.concurrency_limit > 0 && snapshot.available <= 0) {
      nearLimitAlerts += 1
    }
  }

  return {
    configuredPolicies: input.resourcePolicyCount + input.concurrencyPolicyCount,
    globalUtilizationPercent,
    concurrencyActive,
    concurrencyLimit,
    nearLimitAlerts,
  }
}
