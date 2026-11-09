export interface CursorUsageNamedCount {
  name?: string
  count?: number
}

export interface CursorDailyMetric {
  date: string
  linesAdded?: number
  linesDeleted?: number
  acceptedLinesAdded?: number
  acceptedLinesDeleted?: number
  totalApplies?: number
  totalAccepts?: number
  totalRejects?: number
  totalTabsShown?: number
  totalTabsAccepted?: number
  chatRequests?: number
  agentRequests?: number
  subscriptionIncludedReqs?: number
  usageBasedReqs?: number
  cmdkUsages?: number
  modelUsage?: CursorUsageNamedCount[]
  extensionUsage?: CursorUsageNamedCount[]
  tabExtensionUsage?: CursorUsageNamedCount[]
  clientVersionUsage?: CursorUsageNamedCount[]
}

export interface CursorUsageAnalyticsPeriod {
  startDate?: string
  endDate?: string
}

export interface CursorUserAnalytics {
  period?: CursorUsageAnalyticsPeriod
  dailyMetrics?: CursorDailyMetric[]
  applyLinesRank?: number
  tabsAcceptedRank?: number
  totalTeamMembers?: number
  totalApplyLines?: number
  teamAverageApplyLines?: number
  totalTabsAccepted?: number
  teamAverageTabsAccepted?: number
  totalMembersInTeam?: number
}

export type CursorUsageAnalyticsStatus =
  | 'available'
  | 'collecting'
  | 'authentication_required'
  | 'analytics_unavailable'
  | 'upstream_error'
  | 'provider_not_found'
  | 'unconfigured_gateway_key'

export interface CursorUsageError {
  code: string
  message: string
}

export type CursorUsageAnalyticsEnvelope = Readonly<{
  providerId: string
  providerName: string
  analytics?: CursorUserAnalytics
  fetchedAt: string
  sourceCollectedAt?: string
  status: CursorUsageAnalyticsStatus
  error?: CursorUsageError
  refreshing?: boolean
}>
