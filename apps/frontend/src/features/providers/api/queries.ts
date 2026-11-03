import { fetchJson } from '../../../shared/api/client'
import type { AdminProvider, ProviderListResponse, UpdateProviderInput } from '../types'

export const providersQueryKeys = {
  list: ['providers', 'list'] as const,
  detail: (id: string) => ['providers', 'detail', id] as const,
}

export async function listProviders(): Promise<ProviderListResponse> {
  return fetchJson<ProviderListResponse>('/api/providers')
}

export async function getProvider(id: string): Promise<AdminProvider> {
  return fetchJson<AdminProvider>(`/api/providers/${encodeURIComponent(id)}`)
}

export async function validateProvider(id: string): Promise<{ status: string }> {
  return fetchJson<{ status: string }>(`/api/providers/${encodeURIComponent(id)}/validate`, {
    method: 'POST',
  })
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
