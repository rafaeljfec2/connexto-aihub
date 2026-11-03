import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { Button } from '../../../shared/ui'
import { catalogQueryKeys, listCatalogModels } from '../api/queries'
import { CatalogFilters } from '../components/CatalogFilters'
import { CatalogModelsTable } from '../components/CatalogModelsTable'
import { useCatalogSync } from '../hooks/useCatalogSync'
import { formatCatalogSyncFeedback } from '../lib/syncFeedback'
import type { CatalogFilter } from '../types'

export function CatalogPage() {
  const [draftFilter, setDraftFilter] = useState<CatalogFilter>({})
  const [appliedFilter, setAppliedFilter] = useState<CatalogFilter>({})
  const models = useQuery({
    queryKey: catalogQueryKeys.models(appliedFilter),
    queryFn: () => listCatalogModels(appliedFilter),
    refetchInterval: false,
    retry: 1,
  })
  const sync = useCatalogSync(appliedFilter)
  const syncFeedback = formatCatalogSyncFeedback(sync.error)

  return (
    <div className="page catalog-page">
      <AppPageHeader
        actions={
          <Button
            variant="primary"
            disabled={sync.isPending}
            onClick={() => sync.mutate(undefined)}
          >
            {sync.isPending ? 'Sync in progress…' : 'Sync catalog'}
          </Button>
        }
      />

      <CatalogFilters value={draftFilter} onChange={setDraftFilter} />
      <Button variant="secondary" size="sm" onClick={() => setAppliedFilter(draftFilter)}>
        Apply filters
      </Button>

      {sync.isPending ? (
        <div className="banner banner-info" role="status">
          Catalog synchronization is running.
        </div>
      ) : null}

      {syncFeedback ? (
        <div
          className={
            syncFeedback.tone === 'warning'
              ? 'banner banner-warning'
              : syncFeedback.tone === 'info'
                ? 'banner banner-info'
                : 'banner banner-error'
          }
          role="alert"
        >
          {syncFeedback.message}
        </div>
      ) : null}

      <CatalogModelsTable
        models={models.data?.data ?? []}
        isLoading={models.isLoading}
        isError={models.isError}
      />
    </div>
  )
}
