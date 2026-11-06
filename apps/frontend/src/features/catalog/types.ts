export type CatalogCapability = {
  state: string
  provenance: string
  confidence: string
  verification: string
}

export type CatalogModelLimits = {
  context?: number | null
  input?: number | null
  output?: number | null
}

export type CatalogModelPricing = {
  currency: string
  unit: string
  input?: number | null
  output?: number | null
  cache_read?: number | null
  cache_write?: number | null
  source: string
  complete: boolean
}

export type CatalogModel = {
  provider: string
  model_id: string
  canonical_id?: string
  name: string
  type?: string
  limits?: CatalogModelLimits
  pricing?: CatalogModelPricing | null
  capabilities: Record<string, CatalogCapability>
  presence: string
  freshness: string
  provider_state: string
  execution_verification: string
  listing_state: string
  listing_reasons: string[]
}

export type CatalogModelsResponse = {
  object: string
  data: CatalogModel[]
  meta: { count: number; generated_at: string }
}

export type CatalogProvidersResponse = {
  object: string
  data: Array<{ provider: string }>
}

export type CatalogSyncResponse = {
  providers: Array<{
    provider: string
    outcome: string
    counts: Record<string, number>
    duration_ms: number
  }>
}

export type CatalogFilter = {
  provider?: string
  capability?: string
  presence?: string
  listing_state?: string
}
