import type { CatalogModel } from '../types'

export type CatalogColumnFilters = {
  model: string
  provider: string
  modelId: string
  listing: string
}

export const defaultCatalogColumnFilters: CatalogColumnFilters = {
  model: '',
  provider: '',
  modelId: '',
  listing: '',
}

function listingLabel(model: CatalogModel): string {
  return model.listing_state || 'Listado'
}

export function applyCatalogColumnFilters(
  models: readonly CatalogModel[],
  filters: CatalogColumnFilters,
): CatalogModel[] {
  const modelQuery = filters.model.trim().toLowerCase()
  const modelIdQuery = filters.modelId.trim().toLowerCase()
  const providerValue = filters.provider.trim()
  const listingValue = filters.listing.trim()

  return models.filter((model) => {
    if (modelQuery) {
      const matchesModel =
        model.name.toLowerCase().includes(modelQuery) ||
        model.model_id.toLowerCase().includes(modelQuery)
      if (!matchesModel) {
        return false
      }
    }

    if (providerValue && model.provider !== providerValue) {
      return false
    }

    if (modelIdQuery && !model.model_id.toLowerCase().includes(modelIdQuery)) {
      return false
    }

    if (listingValue && listingLabel(model) !== listingValue) {
      return false
    }

    return true
  })
}
