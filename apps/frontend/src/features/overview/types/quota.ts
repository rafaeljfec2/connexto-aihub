export type ProviderQuotaStatus =
  | 'available'
  | 'unsupported'
  | 'unconfigured_credential'
  | 'unauthorized'
  | 'rate_limited'
  | 'timeout'
  | 'error'

export type DataScope =
  | 'api_key'
  | 'account'
  | 'workspace'
  | 'organization'
  | 'subscription'
  | 'unknown'

export type MetricNature =
  | 'quota'
  | 'usage'
  | 'cost'
  | 'rate_limit'

export interface ProviderQuotaMetric {
  metric: string
  label: string
  nature: MetricNature

  used?: number
  limit?: number
  remaining?: number

  consumed_percent?: number
  remaining_percent?: number
  remaining_fraction?: number

  unit?: string
  period?: string

  starts_at?: string
  ends_at?: string
  resets_at?: string
  scope?: string
  include_byok?: boolean
}

export interface ProviderQuotaSource {
  type: string
  name: string
  scope: DataScope
  fetched_at: string
  provider_reported_at?: string
}

export interface ProviderQuotaError {
  code: string
  message: string
}

export interface ProviderUsageChartPoint {
  date: string
  cost_micro_cents: number
  cost_usd: number
  tokens?: number
  requests?: number
}

export interface ProviderUsageChart {
  title: string
  subtitle: string
  total_micro_cents: number
  total_usd: number
  unit: string
  period_start: string
  period_end: string
  points: ProviderUsageChartPoint[]
}

export interface ProviderQuotaSnapshot {
  account_id: string
  account_label: string
  provider_id: string
  provider_name: string
  status: ProviderQuotaStatus
  scope: DataScope
  quotas: ProviderQuotaMetric[]
  source: ProviderQuotaSource
  error?: ProviderQuotaError
  provider_expires_at?: string
  workspace_id?: string
  workspace_name?: string
  usage_url?: string
  usage_chart?: ProviderUsageChart
}

export interface ProviderQuotasResponse {
  snapshots: ProviderQuotaSnapshot[]
  fetched_at: string
}

