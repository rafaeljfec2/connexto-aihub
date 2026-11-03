export type CatalogCapability = {
  state: string
  provenance: string
  confidence: string
  verification: string
}

export type CatalogModel = {
  provider: string
  model_id: string
  canonical_id?: string
  name: string
  type?: string
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
