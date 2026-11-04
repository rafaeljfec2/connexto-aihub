import type { QuotaUtilizationEntry, ResourcePolicy } from '../types'

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

export function computeResourcesSummary(input: {
  resourcePolicyCount: number
  concurrencyPolicyCount: number
  quotaUtilization: QuotaUtilizationEntry[]
  concurrencySnapshots: Array<{
    enabled: boolean
    concurrency_limit: number
    active_count: number
    available: number
  }>
}): {
  configuredPolicies: number
  peakUtilizationPercent: number | null
  concurrencyActive: number
  concurrencyLimit: number
  nearLimitAlerts: number
} {
  const enabledQuota = input.quotaUtilization.filter((row) => row.enabled && row.daily_limit > 0)
  const peakUtilizationPercent =
    enabledQuota.length > 0
      ? Math.max(...enabledQuota.map((row) => row.utilization_percent))
      : null

  let concurrencyActive = 0
  let concurrencyLimit = 0
  for (const snapshot of input.concurrencySnapshots) {
    if (!snapshot.enabled) {
      continue
    }
    concurrencyActive += snapshot.active_count
    concurrencyLimit += snapshot.concurrency_limit
  }

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
    peakUtilizationPercent,
    concurrencyActive,
    concurrencyLimit,
    nearLimitAlerts,
  }
}
