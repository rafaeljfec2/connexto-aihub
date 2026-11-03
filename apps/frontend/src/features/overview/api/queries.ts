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

export async function getHealth(): Promise<HealthReport> {
  return fetchJson<HealthReport>('/healthz')
}

export async function getReadiness(): Promise<ReadinessReport> {
  return fetchJson<ReadinessReport>('/readyz')
}

export async function getProviderDiagnostics(): Promise<ProviderDiagnosticsResponse> {
  return fetchJson<ProviderDiagnosticsResponse>('/diagnostics/providers')
}

export const overviewQueryKeys = {
  health: ['overview', 'health'] as const,
  readiness: ['overview', 'readiness'] as const,
  providerDiagnostics: ['overview', 'providerDiagnostics'] as const,
}

export type OverviewOperationalCacheKeys = {
  readiness: typeof overviewQueryKeys.readiness
  providerDiagnostics: typeof overviewQueryKeys.providerDiagnostics
}
