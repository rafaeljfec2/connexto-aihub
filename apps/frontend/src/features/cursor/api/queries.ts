import { fetchJson } from '../../../shared/api/client'
import type { CursorUsageAnalyticsEnvelope } from '../types'

export const cursorQueryKeys = {
  usageAnalytics: (providerId?: string) => ['cursor', 'usageAnalytics', providerId ?? 'default'] as const,
}

export async function getCursorUsageAnalytics(providerId?: string): Promise<CursorUsageAnalyticsEnvelope> {
  const suffix = providerId ? `?providerId=${encodeURIComponent(providerId)}` : ''
  return fetchJson<CursorUsageAnalyticsEnvelope>(`/api/ui/providers/cursor/usage-analytics${suffix}`)
}

export async function getCursorUsageAnalyticsForce(providerId?: string): Promise<CursorUsageAnalyticsEnvelope> {
  const providerParam = providerId ? `providerId=${encodeURIComponent(providerId)}&` : ''
  return fetchJson<CursorUsageAnalyticsEnvelope>(
    `/api/ui/providers/cursor/usage-analytics?${providerParam}refresh=true`,
  )
}
