import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { useAppPageMeta } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { GridPanel } from '../../../shared/ui'
import { useGridPagination } from '../../../shared/hooks/useGridPagination'
import { getRuntimeMeta, overviewQueryKeys } from '../../overview/api/queries'
import { catalogQueryKeys, listCatalogModels } from '../api/queries'
import { CatalogModelDetailPanel } from '../components/CatalogModelDetailPanel'
import { CatalogModelsTable } from '../components/CatalogModelsTable'
import { ModelCompareModal } from '../components/ModelCompareModal'
import { useCatalogSync } from '../hooks/useCatalogSync'
import {
  applyCatalogColumnFilters,
  defaultCatalogColumnFilters,
  type CatalogColumnFilters,
} from '../lib/catalogColumnFilters'
import { formatCatalogSyncFeedback } from '../lib/syncFeedback'
import { formatDataOriginTag } from '../../../shared/lib/formatDataOriginTag'
import type { CatalogModel } from '../types'

function listingLabel(model: CatalogModel): string {
  return model.listing_state || 'Listado'
}

export function CatalogPage() {
  const meta = useAppPageMeta()
  const [columnFilters, setColumnFilters] = useState<CatalogColumnFilters>(
    defaultCatalogColumnFilters,
  )
  const [sortBy, setSortBy] = useState<'name' | 'provider'>('name')
  const [selectedModel, setSelectedModel] = useState<CatalogModel | null>(null)
  const [compareSet, setCompareSet] = useState<Set<string>>(new Set())
  const [compareModalOpen, setCompareModalOpen] = useState(false)
  const modelFilterInputRef = useRef<HTMLInputElement>(null)

  const catalogListFilter = useMemo(() => ({}), [])

  const modelsQuery = useQuery({
    queryKey: catalogQueryKeys.models(catalogListFilter),
    queryFn: () => listCatalogModels(catalogListFilter),
    refetchInterval: false,
    retry: 1,
  })

  const runtimeMeta = useQuery({
    queryKey: overviewQueryKeys.runtimeMeta,
    queryFn: getRuntimeMeta,
    refetchInterval: false,
    retry: 1,
  })

  const sync = useCatalogSync(catalogListFilter)
  const syncFeedback = formatCatalogSyncFeedback(sync.error)

  const rawModels = modelsQuery.data?.data ?? []

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (
        event.key === '/' &&
        document.activeElement !== modelFilterInputRef.current &&
        !(document.activeElement instanceof HTMLInputElement) &&
        !(document.activeElement instanceof HTMLTextAreaElement)
      ) {
        event.preventDefault()
        modelFilterInputRef.current?.focus()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  const providerFilterOptions = useMemo(() => {
    return Array.from(new Set(rawModels.map((model) => model.provider))).filter(Boolean).sort()
  }, [rawModels])

  const listingFilterOptions = useMemo(() => {
    return Array.from(new Set(rawModels.map((model) => listingLabel(model)))).sort()
  }, [rawModels])

  const filteredAndSortedModels = useMemo(() => {
    const filtered = applyCatalogColumnFilters(rawModels, columnFilters)

    return [...filtered].sort((a, b) => {
      if (sortBy === 'provider') {
        return a.provider.localeCompare(b.provider) || a.name.localeCompare(b.name)
      }
      return a.name.localeCompare(b.name)
    })
  }, [rawModels, columnFilters, sortBy])

  const gridPagination = useGridPagination(filteredAndSortedModels, {
    resetKey: [
      columnFilters.model,
      columnFilters.provider,
      columnFilters.modelId,
      columnFilters.listing,
      sortBy,
    ].join('|'),
  })

  const activeSelected = useMemo(() => {
    if (selectedModel !== null) {
      return (
        filteredAndSortedModels.find(
          (model) =>
            `${model.provider}:${model.model_id}` ===
            `${selectedModel.provider}:${selectedModel.model_id}`,
        ) ?? null
      )
    }
    return gridPagination.pageItems[0] ?? filteredAndSortedModels[0] ?? null
  }, [selectedModel, filteredAndSortedModels, gridPagination.pageItems])

  const handleToggleCompare = (model: CatalogModel) => {
    const key = `${model.provider}:${model.model_id}`
    setCompareSet((previous) => {
      const next = new Set(previous)
      if (next.has(key)) {
        next.delete(key)
      } else if (next.size < 3) {
        next.add(key)
      }
      return next
    })
  }

  const comparedModelsList = useMemo(() => {
    return rawModels.filter((model) => compareSet.has(`${model.provider}:${model.model_id}`))
  }, [rawModels, compareSet])

  const originTag = formatDataOriginTag(runtimeMeta.data?.data_origin, 'catalog')
  const versionLabel = runtimeMeta.data?.version ?? 'PREVIEW 03'
  const visibleCountLabel = `${String(filteredAndSortedModels.length).padStart(2, '0')} REGISTROS`

  const handleColumnFilterChange = (patch: Partial<CatalogColumnFilters>) => {
    setColumnFilters((previous) => ({ ...previous, ...patch }))
  }

  return (
    <div className={appPageClassName('catalog-page')}>
      <div className="catalog-page-top">
        <div className="catalog-pagehead">
          <div className="min-w-0 flex-1">
            <p className="catalog-page-eyebrow">{meta.eyebrow}</p>
            <h1 className="catalog-page-title">{meta.title}</h1>
            {meta.lead ? <p className="catalog-page-lead">{meta.lead}</p> : null}
            <p className="catalog-pricing-note">
              Preços por 1M tokens quando informados pelo catálogo oficial do provedor (sem estimativa do Hub).
            </p>
          </div>
          <div className="catalog-headright">
            <span className="overview-backend-tag">{originTag}</span>
            <button
              type="button"
              className="catalog-sync-btn"
              disabled={sync.isPending}
              onClick={() => sync.mutate(undefined)}
            >
              <RefreshCw className={`inline size-3.5 ${sync.isPending ? 'animate-spin' : ''}`} />
              <span>{sync.isPending ? 'Sincronizando…' : 'Sincronizar catálogo'}</span>
            </button>
          </div>
        </div>

        {(sync.isPending || syncFeedback) && (
          <div className="catalog-alerts">
            {sync.isPending ? (
              <div className="banner banner-info" role="status">
                Sincronização do catálogo em andamento…
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
          </div>
        )}

        <div className="catalog-toolbar-bar">
          <span className="catalog-result-count">
            {filteredAndSortedModels.length.toLocaleString('pt-BR')} REGISTROS VISÍVEIS
          </span>
          <div className="catalog-actions">
            <label className="catalog-sort-label">
              ORDENAR
              <select
                value={sortBy}
                onChange={(event) => setSortBy(event.target.value as 'name' | 'provider')}
                className="catalog-sort"
              >
                <option value="name">Nome A–Z</option>
                <option value="provider">Provedor A–Z</option>
              </select>
            </label>
            <button
              type="button"
              className="catalog-compare-open"
              disabled={compareSet.size < 2}
              onClick={() => setCompareModalOpen(true)}
            >
              Comparar modelos <b className="ml-1 font-mono">{compareSet.size}/3</b>
            </button>
          </div>
        </div>
      </div>

      <div className="catalog-layout">
        <GridPanel
          className="catalog-list-panel"
          title="Modelos disponíveis no catálogo"
          countLabel={visibleCountLabel}
          pagination={{
            meta: gridPagination.meta,
            onPreviousPage: gridPagination.goToPreviousPage,
            onNextPage: gridPagination.goToNextPage,
            pageSize: gridPagination.pageSize,
            onPageSizeChange: gridPagination.setPageSize,
          }}
          footnoteLeft="CATÁLOGO SINCRONIZADO"
          footnoteRight={modelsQuery.isLoading ? 'CARREGANDO…' : 'DADOS DO BACKEND'}
        >
          <CatalogModelsTable
            models={gridPagination.pageItems}
            isLoading={modelsQuery.isLoading}
            isError={modelsQuery.isError}
            selectedModelId={
              activeSelected ? `${activeSelected.provider}:${activeSelected.model_id}` : null
            }
            onSelectModel={(model) => setSelectedModel(model)}
            compareSet={compareSet}
            onToggleCompare={handleToggleCompare}
            columnFilters={columnFilters}
            onColumnFilterChange={handleColumnFilterChange}
            providerOptions={providerFilterOptions}
            listingOptions={listingFilterOptions}
            modelFilterInputRef={modelFilterInputRef}
          />
        </GridPanel>

        <CatalogModelDetailPanel model={activeSelected} />
      </div>

      <footer className="catalog-bottomnote">
        <span>API · /v1/catalog/models &nbsp; /v1/catalog/providers &nbsp; /v1/catalog/sync</span>
        <span>CONNEXTO AI HUB / {versionLabel}</span>
      </footer>

      <ModelCompareModal
        models={comparedModelsList}
        open={compareModalOpen}
        onClose={() => setCompareModalOpen(false)}
      />
    </div>
  )
}
