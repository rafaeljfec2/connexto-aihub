import { fetchJson } from '../../../shared/api/client'

export type HealthReport = { status: string }

export type ReadinessReport = {
  status: string
  checks?: Record<string, string>
}

export type ProviderDiagnostic = {
  id: string
  enabled: boolean
  state: string
}

export type ProviderDiagnosticsResponse = {
  providers: ProviderDiagnostic[]
}

export type RuntimeMeta = {
  environment: string
  data_origin: string
  version: string
  gateway_status: string
}

export type TrafficBucket = {
  timestamp: string
  label: string
  count: number
}

export type TrafficSummary = {
  range: string
  total_requests: number
  trend_percentage: number
  buckets: TrafficBucket[]
  has_sufficient_history: boolean
}

export async function getHealth(): Promise<HealthReport> {
  return fetchJson<HealthReport>('/healthz')
}

export async function getReadiness(): Promise<ReadinessReport> {
  return fetchJson<ReadinessReport>('/readyz')
}

export async function getProviderDiagnostics(): Promise<ProviderDiagnosticsResponse> {
  return fetchJson<ProviderDiagnosticsResponse>('/diagnostics/providers')
}

export async function getRuntimeMeta(): Promise<RuntimeMeta> {
  return fetchJson<RuntimeMeta>('/api/runtime/meta')
}

export async function getTrafficSummary(range = '24h'): Promise<TrafficSummary> {
  return fetchJson<TrafficSummary>(`/api/telemetry/traffic-summary?range=${encodeURIComponent(range)}`)
}

export const overviewQueryKeys = {
  health: ['overview', 'health'] as const,
  readiness: ['overview', 'readiness'] as const,
  providerDiagnostics: ['overview', 'providerDiagnostics'] as const,
  runtimeMeta: ['overview', 'runtimeMeta'] as const,
  trafficSummary: (range: string) => ['overview', 'trafficSummary', range] as const,
}

export type OverviewOperationalCacheKeys = {
  readiness: typeof overviewQueryKeys.readiness
  providerDiagnostics: typeof overviewQueryKeys.providerDiagnostics
}
