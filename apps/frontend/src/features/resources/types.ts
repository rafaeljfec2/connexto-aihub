export type ResourceScope = 'global' | 'provider' | 'model'

export type ResourcePolicy = {
  id: number
  scope: ResourceScope
  provider_id?: string
  model_id?: string
  daily_limit: number
  enabled: boolean
  updated_at: string
}

export type ResourcePoliciesResponse = {
  policies: ResourcePolicy[]
}

export type QuotaUtilizationEntry = {
  policy_id: number
  scope: ResourceScope
  provider_id?: string
  model_id?: string
  daily_limit: number
  enabled: boolean
  period_start: string
  reset_at: string
  admitted_count: number
  remaining_requests: number
  utilization_percent: number
}

export type QuotaUtilizationResponse = {
  quota_utilization: QuotaUtilizationEntry[]
}

export type ConcurrencyPolicy = {
  id: number
  scope: ResourceScope
  provider_id?: string
  model_id?: string
  concurrency_limit: number
  enabled: boolean
  updated_at: string
}

export type ConcurrencyPoliciesResponse = {
  policies: ConcurrencyPolicy[]
}

export type ConcurrencySnapshot = {
  policy_id: number
  scope: ResourceScope
  provider_id?: string
  model_id?: string
  concurrency_limit: number
  enabled: boolean
  active_count: number
  available: number
}

export type ConcurrencySnapshotResponse = {
  concurrency: ConcurrencySnapshot[]
}

export type AnalyticsDateRange = {
  from: string
  to: string
}

export type AnalyticsScopeFilter = {
  provider_id?: string
  model_id?: string
}

export type AdmissionsDailyRow = {
  period_start: string
  admitted_count: number
}

export type AdmissionsResponse = {
  from: string
  to: string
  total_admitted: number
  daily: AdmissionsDailyRow[]
}

export type NullableTokenCount = number | null

export type ProviderUsageDailyRow = {
  period_start: string
  input_tokens: NullableTokenCount
  output_tokens: NullableTokenCount
  total_tokens: NullableTokenCount
  execution_count: number
}

export type ProviderUsageResponse = {
  from: string
  to: string
  input_tokens: NullableTokenCount
  output_tokens: NullableTokenCount
  total_tokens: NullableTokenCount
  reported_executions: number
  unavailable_executions: number
  partial_executions: number
  invalid_executions: number
  daily: ProviderUsageDailyRow[]
}

export type ProviderUsageProviderRow = {
  period_start: string
  provider_id: string
  input_tokens: NullableTokenCount
  output_tokens: NullableTokenCount
  total_tokens: NullableTokenCount
  executions: number
}

export type ProviderUsageModelRow = {
  period_start: string
  provider_id: string
  model_id: string
  input_tokens: NullableTokenCount
  output_tokens: NullableTokenCount
  total_tokens: NullableTokenCount
  executions: number
}

export type ProviderUsageDistributionResponse = {
  from: string
  to: string
  by_provider: ProviderUsageProviderRow[]
  by_model: ProviderUsageModelRow[]
}

export type ResourcesTabId = 'overview' | 'policies' | 'concurrency' | 'provider-usage'

export type ResourcesUiFilter = AnalyticsDateRange & {
  provider_id: string
  model_id: string
}
