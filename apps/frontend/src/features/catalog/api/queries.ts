import { fetchJson } from '../../../shared/api/client'
import type { CatalogFilter, CatalogModelsResponse, CatalogSyncResponse } from '../types'

export const catalogQueryKeys = {
  models: (filter: CatalogFilter) => ['catalog', 'models', filter] as const,
}

function buildCatalogQuery(filter: CatalogFilter): string {
  const params = new URLSearchParams()
  if (filter.provider) {
    params.set('provider', filter.provider)
  }
  if (filter.capability) {
    params.set('capability', filter.capability)
  }
  if (filter.presence) {
    params.set('presence', filter.presence)
  }
  if (filter.listing_state) {
    params.set('listing_state', filter.listing_state)
  }
  const query = params.toString()
  return query.length > 0 ? `/v1/catalog/models?${query}` : '/v1/catalog/models'
}

export async function listCatalogModels(filter: CatalogFilter): Promise<CatalogModelsResponse> {
  return fetchJson<CatalogModelsResponse>(buildCatalogQuery(filter))
}

export async function syncCatalog(provider?: string): Promise<CatalogSyncResponse> {
  return fetchJson<CatalogSyncResponse>('/v1/catalog/sync', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(provider ? { provider } : {}),
  })
}
