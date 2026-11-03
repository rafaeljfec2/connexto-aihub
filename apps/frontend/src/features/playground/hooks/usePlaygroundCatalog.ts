import { useQuery } from '@tanstack/react-query'
import { catalogQueryKeys, listCatalogModels, type CatalogModel } from '../../catalog'

export function usePlaygroundCatalogModels() {
  return useQuery({
    queryKey: catalogQueryKeys.models({ presence: 'present' }),
    queryFn: () => listCatalogModels({ presence: 'present' }),
    refetchInterval: false,
    retry: 1,
  })
}

export function groupCatalogModels(models: CatalogModel[]): Map<string, CatalogModel[]> {
  const grouped = new Map<string, CatalogModel[]>()
  for (const model of models) {
    const list = grouped.get(model.provider) ?? []
    list.push(model)
    grouped.set(model.provider, list)
  }
  return grouped
}

export function defaultSelection(grouped: Map<string, CatalogModel[]>): {
  providerId: string
  modelId: string
} {
  const firstProvider = [...grouped.keys()].sort()[0]
  if (!firstProvider) {
    return { providerId: '', modelId: '' }
  }
  const models = grouped.get(firstProvider) ?? []
  const firstModel = models[0]
  return {
    providerId: firstProvider,
    modelId: firstModel?.model_id ?? '',
  }
}
