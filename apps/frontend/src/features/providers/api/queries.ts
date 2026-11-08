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

export async function getProviderQuotasForce(): Promise<ProviderQuotasResponse> {
  return fetchJson<ProviderQuotasResponse>('/api/ui/providers/quotas?refresh=true')
}

export async function listProviders(): Promise<ProviderListResponse> {
  return fetchJson<ProviderListResponse>('/api/providers')
}

export async function getProvider(id: string): Promise<AdminProvider> {
  return fetchJson<AdminProvider>(`/api/providers/${encodeURIComponent(id)}`)
}

export type ValidateProviderBody = {
  credential?: string
  management_credential?: string
}

export async function validateProvider(
  id: string,
  body?: ValidateProviderBody,
): Promise<{ status: string }> {
  return fetchJson<{ status: string }>(`/api/providers/${encodeURIComponent(id)}/validate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body && Object.keys(body).length > 0 ? JSON.stringify(body) : undefined,
  })
}

export type AntigravityProxyRuntimeSessionClearResponse = {
  cleared: boolean
  namespace: string
  mapped_acp_ids_count?: number
  listed_acp_sessions_count?: number
  acp_sessions_closed?: number
  acp_close_failures?: number
  warnings?: string[]
}

export async function clearAntigravityProxyRuntimeSessions(
  id: string,
): Promise<AntigravityProxyRuntimeSessionClearResponse> {
  return fetchJson<AntigravityProxyRuntimeSessionClearResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/runtime/sessions/clear`,
    { method: 'POST' },
  )
}

export type AntigravityProxyAuthStatusResponse = {
  authenticated: boolean
  account?: string | null
  lastCheckedAt: string
  viabilityGate: string
  limitations: string[]
  activeOperation?: {
    operationId?: string
    status?: string
    createdAt?: string
    expiresAt?: string
    account?: string | null
    error?: string | null
  }
}

export async function getAntigravityProxyAuthStatus(
  id: string,
): Promise<AntigravityProxyAuthStatusResponse> {
  return fetchJson<AntigravityProxyAuthStatusResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/antigravity/auth/status`,
  )
}

export type AntigravityProxyAuthStartResponse = {
  operation_id: string
  status: string
  auth_url: string
  state: string
  expires_at: string
}

export async function startAntigravityProxyAuth(
  id: string,
  body?: { redirect_uri?: string },
): Promise<AntigravityProxyAuthStartResponse> {
  return fetchJson<AntigravityProxyAuthStartResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/antigravity/auth/start`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body ?? {}),
    },
  )
}

export type CursorProxyRuntimeSessionClearResponse = {
  cleared: boolean
  namespace: string
}

export async function clearCursorUsageProxyRuntimeSessions(
  id: string,
): Promise<CursorProxyRuntimeSessionClearResponse> {
  return fetchJson<CursorProxyRuntimeSessionClearResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/cursor/runtime/sessions/clear`,
    { method: 'POST' },
  )
}

export type CursorProxyAuthStartResponse = {
  status: string
  sessionId: string
  expiresAt: string
  interactionUrl: string
  browserMode?: 'headed' | 'headless'
}

export async function startCursorProxyAuth(id: string): Promise<CursorProxyAuthStartResponse> {
  return fetchJson<CursorProxyAuthStartResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/cursor/auth/start`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    },
  )
}

export type CursorProxyAuthStatusResponse = {
  sessionId: string
  accountId: string
  status: string
  expiresAt: string
}

export type CursorProxyUsageSessionResponse = {
  status: 'authenticated' | 'authentication_required'
  accountId: string
  touchedAt?: string
}

export type CursorProxyUsageCollectResponse = {
  status: string
  provider?: string
  accountId: string
  collectedAt?: string
}

export async function getCursorProxyUsageSession(
  id: string,
): Promise<CursorProxyUsageSessionResponse> {
  return fetchJson<CursorProxyUsageSessionResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/cursor/usage/session`,
  )
}

export async function probeCursorProxyUsageCollect(
  id: string,
): Promise<CursorProxyUsageCollectResponse> {
  return fetchJson<CursorProxyUsageCollectResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/cursor/usage/collect`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
}

export async function getCursorProxyAuthStatus(
  id: string,
  sessionId: string,
): Promise<CursorProxyAuthStatusResponse> {
  return fetchJson<CursorProxyAuthStatusResponse>(
    `/api/providers/${encodeURIComponent(id)}/proxy/cursor/auth/${encodeURIComponent(sessionId)}`,
  )
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
