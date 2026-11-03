import { useMutation, useQueryClient } from '@tanstack/react-query'
import { catalogQueryKeys, syncCatalog } from '../api/queries'
import type { CatalogFilter } from '../types'

export function useCatalogSync(activeFilter: CatalogFilter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (provider?: string) => syncCatalog(provider),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['catalog', 'models'] })
      await queryClient.invalidateQueries({ queryKey: catalogQueryKeys.models(activeFilter) })
    },
  })
}
