import { fetchJson } from '../../../shared/api/client'
import type {
  AdmissionsResponse,
  AnalyticsDateRange,
  AnalyticsScopeFilter,
  ConcurrencyPoliciesResponse,
  ConcurrencySnapshotResponse,
  ProviderUsageDistributionResponse,
  ProviderUsageResponse,
  QuotaUtilizationResponse,
  ResourcePoliciesResponse,
} from '../types'

export const resourcesQueryKeys = {
  resourcePolicies: ['resources', 'resourcePolicies'] as const,
  quotaUtilization: (filter: AnalyticsScopeFilter) =>
    ['resources', 'quotaUtilization', filter] as const,
  concurrencyPolicies: ['resources', 'concurrencyPolicies'] as const,
  concurrencySnapshot: (filter: AnalyticsScopeFilter) =>
    ['resources', 'concurrencySnapshot', filter] as const,
  admissions: (range: AnalyticsDateRange, filter: AnalyticsScopeFilter) =>
    ['resources', 'admissions', range, filter] as const,
  providerUsage: (range: AnalyticsDateRange, filter: AnalyticsScopeFilter) =>
    ['resources', 'providerUsage', range, filter] as const,
  providerUsageDistribution: (range: AnalyticsDateRange, filter: AnalyticsScopeFilter) =>
    ['resources', 'providerUsageDistribution', range, filter] as const,
}

function appendScopeParams(params: URLSearchParams, filter: AnalyticsScopeFilter): void {
  if (filter.provider_id) {
    params.set('provider_id', filter.provider_id)
  }
  if (filter.model_id) {
    params.set('model_id', filter.model_id)
  }
}

export function buildAnalyticsRangePath(
  basePath: string,
  range: AnalyticsDateRange,
  filter: AnalyticsScopeFilter,
): string {
  const params = new URLSearchParams()
  params.set('from', range.from)
  params.set('to', range.to)
  appendScopeParams(params, filter)
  return `${basePath}?${params.toString()}`
}

export function buildQuotaUtilizationPath(filter: AnalyticsScopeFilter): string {
  const params = new URLSearchParams()
  appendScopeParams(params, filter)
  const query = params.toString()
  return query.length > 0
    ? `/api/resource-analytics/quota-utilization?${query}`
    : '/api/resource-analytics/quota-utilization'
}

export function buildConcurrencySnapshotPath(filter: AnalyticsScopeFilter): string {
  const params = new URLSearchParams()
  appendScopeParams(params, filter)
  const query = params.toString()
  return query.length > 0
    ? `/api/resource-analytics/concurrency?${query}`
    : '/api/resource-analytics/concurrency'
}

export async function listResourcePolicies(): Promise<ResourcePoliciesResponse> {
  return fetchJson<ResourcePoliciesResponse>('/api/resource-policies')
}

export async function getQuotaUtilization(
  filter: AnalyticsScopeFilter,
): Promise<QuotaUtilizationResponse> {
  return fetchJson<QuotaUtilizationResponse>(buildQuotaUtilizationPath(filter))
}

export async function listConcurrencyPolicies(): Promise<ConcurrencyPoliciesResponse> {
  return fetchJson<ConcurrencyPoliciesResponse>('/api/concurrency-policies')
}

export async function getConcurrencySnapshot(
  filter: AnalyticsScopeFilter,
): Promise<ConcurrencySnapshotResponse> {
  return fetchJson<ConcurrencySnapshotResponse>(buildConcurrencySnapshotPath(filter))
}

export async function getAdmissionsAnalytics(
  range: AnalyticsDateRange,
  filter: AnalyticsScopeFilter,
): Promise<AdmissionsResponse> {
  return fetchJson<AdmissionsResponse>(
    buildAnalyticsRangePath('/api/resource-analytics/admissions', range, filter),
  )
}

export async function getProviderUsageAnalytics(
  range: AnalyticsDateRange,
  filter: AnalyticsScopeFilter,
): Promise<ProviderUsageResponse> {
  return fetchJson<ProviderUsageResponse>(
    buildAnalyticsRangePath('/api/resource-analytics/provider-usage', range, filter),
  )
}

export async function getProviderUsageDistribution(
  range: AnalyticsDateRange,
  filter: AnalyticsScopeFilter,
): Promise<ProviderUsageDistributionResponse> {
  return fetchJson<ProviderUsageDistributionResponse>(
    buildAnalyticsRangePath('/api/resource-analytics/provider-usage/distribution', range, filter),
  )
}
