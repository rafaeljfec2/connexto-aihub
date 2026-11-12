import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
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
import { formatCatalogOriginTag } from '../lib/catalogOriginTag'
import { catalogListingLabel } from '../lib/catalogListingLabel'
import { formatNumber } from '../../../shared/i18n/format'
import type { CatalogModel } from '../types'

export function CatalogPage() {
  const { t } = useTranslation('catalog')
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
    return Array.from(new Set(rawModels.map((model) => catalogListingLabel(model)))).sort()
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

  const originTag = formatCatalogOriginTag(runtimeMeta.data?.data_origin)
  const versionLabel = runtimeMeta.data?.version ?? 'PREVIEW 03'
  const visibleCountLabel = t('page.recordsLabel', {
    count: String(filteredAndSortedModels.length).padStart(2, '0'),
  })

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
            <p className="catalog-pricing-note">{t('page.pricingNoteFull')}</p>
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
              <span>{sync.isPending ? t('page.syncing') : t('page.syncCatalog')}</span>
            </button>
          </div>
        </div>

        {(sync.isPending || syncFeedback) && (
          <div className="catalog-alerts">
            {sync.isPending ? (
              <div className="banner banner-info" role="status">
                {t('sync.inProgress')}
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
            {t('page.visibleRecords', { count: formatNumber(filteredAndSortedModels.length) })}
          </span>
          <div className="catalog-actions">
            <label className="catalog-sort-label">
              {t('page.sortLabelUpper')}
              <select
                value={sortBy}
                onChange={(event) => setSortBy(event.target.value as 'name' | 'provider')}
                className="catalog-sort"
              >
                <option value="name">{t('page.sortName')}</option>
                <option value="provider">{t('page.sortProvider')}</option>
              </select>
            </label>
            <button
              type="button"
              className="catalog-compare-open"
              disabled={compareSet.size < 2}
              onClick={() => setCompareModalOpen(true)}
            >
              {t('page.compareModels')}{' '}
              <b className="ml-1 font-mono">{compareSet.size}/3</b>
            </button>
          </div>
        </div>
      </div>

      <div className="catalog-layout">
        <GridPanel
          className="catalog-list-panel"
          title={t('page.modelsTitle')}
          countLabel={visibleCountLabel}
          pagination={{
            meta: gridPagination.meta,
            onPreviousPage: gridPagination.goToPreviousPage,
            onNextPage: gridPagination.goToNextPage,
            pageSize: gridPagination.pageSize,
            onPageSizeChange: gridPagination.setPageSize,
          }}
          footnoteLeft={t('page.footnoteLeft')}
          footnoteRight={modelsQuery.isLoading ? t('page.loadingFootnote') : t('page.backendDataTag')}
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
        <span>{t('page.apiFootnote')}</span>
        <span>{t('page.brandFootnote', { version: versionLabel })}</span>
      </footer>

      <ModelCompareModal
        models={comparedModelsList}
        open={compareModalOpen}
        onClose={() => setCompareModalOpen(false)}
      />
    </div>
  )
}
