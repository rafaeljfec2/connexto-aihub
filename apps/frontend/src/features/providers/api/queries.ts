import { fetchJson } from '../../../shared/api/client'
import type { ProviderQuotasResponse } from '../../overview/types/quota'
import type {
  AdminProvider,
  AvailableProvidersResponse,
  CreateProviderInput,
  ProviderConnectionStatus,
  ProviderListResponse,
  SyncModelsResponse,
  UpdateProviderInput,
} from '../types'

export const providersQueryKeys = {
  list: ['providers', 'list'] as const,
  available: ['providers', 'available'] as const,
  detail: (id: string) => ['providers', 'detail', id] as const,
  status: (id: string) => ['providers', 'status', id] as const,
  quotas: ['providers', 'quotas'] as const,
}

export async function listAvailableProviders(): Promise<AvailableProvidersResponse> {
  return fetchJson<AvailableProvidersResponse>('/api/providers/available')
}

export async function createProvider(input: CreateProviderInput): Promise<AdminProvider> {
  return fetchJson<AdminProvider>('/api/providers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export async function getProviderQuotas(): Promise<ProviderQuotasResponse> {
  return fetchJson<ProviderQuotasResponse>('/api/ui/providers/quotas')
}

export async function listProviders(): Promise<ProviderListResponse> {
  return fetchJson<ProviderListResponse>('/api/providers')
}

export async function getProvider(id: string): Promise<AdminProvider> {
  return fetchJson<AdminProvider>(`/api/providers/${encodeURIComponent(id)}`)
}

export async function validateProvider(id: string, credential?: string): Promise<{ status: string }> {
  return fetchJson<{ status: string }>(`/api/providers/${encodeURIComponent(id)}/validate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: credential ? JSON.stringify({ credential }) : undefined,
  })
}

export async function syncProviderModels(id: string): Promise<SyncModelsResponse> {
  return fetchJson<SyncModelsResponse>(`/api/providers/${encodeURIComponent(id)}/sync-models`, {
    method: 'POST',
  })
}

export async function getProviderStatus(id: string): Promise<ProviderConnectionStatus> {
  return fetchJson<ProviderConnectionStatus>(`/api/providers/${encodeURIComponent(id)}/status`)
}

export async function enableProvider(id: string): Promise<AdminProvider> {
  return fetchJson<AdminProvider>(`/api/providers/${encodeURIComponent(id)}/enable`, {
    method: 'POST',
  })
}

export async function disableProvider(id: string): Promise<AdminProvider> {
  return fetchJson<AdminProvider>(`/api/providers/${encodeURIComponent(id)}/disable`, {
    method: 'POST',
  })
}

export async function updateProvider(id: string, input: UpdateProviderInput): Promise<AdminProvider> {
  return fetchJson<AdminProvider>(`/api/providers/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}
