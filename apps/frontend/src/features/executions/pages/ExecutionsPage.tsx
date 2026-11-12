import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { RefreshButton } from '../../../shared/ui'
import { useAppPageMeta } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { catalogQueryKeys, listCatalogProviders } from '../../catalog'
import { getRuntimeMeta, overviewQueryKeys } from '../../overview/api/queries'
import { executionsErrorMessage } from '../hooks/executionsFilter'
import { getGeneration, generationsQueryKeys, listGenerations } from '../api/generations'
import { GenerationDetailsDrawer } from '../components/GenerationDetailsDrawer'
import { GenerationsActivityPanel } from '../components/GenerationsActivityPanel'
import { GenerationsFilters } from '../components/GenerationsFilters'
import type { GenerationFilterDraft } from '../components/GenerationsFilters'
import { GenerationsTable } from '../components/GenerationsTable'
import { formatDataOriginTag } from '../lib/formatDataOriginTag'
import {
  inferPeriodPreset,
  resolvePeriodPresetRange,
  type GenerationPeriodPreset,
} from '../lib/generationPeriodPresets'
import type { ExecutionOutcome, Generation, GenerationsFilter } from '../types'
import { validateGenerationFilterDraft } from '../lib/validateGenerationFilters'
import { generationMatchesSearch } from '../lib/formatGeneration'
import '../generations.css'
import '../generation-drawer.css'

const PAGE_LIMIT = 50

function buildDefaultFilters(): GenerationFilterDraft {
  const range = resolvePeriodPresetRange('30d')
  return {
    provider_id: '',
    model_id: '',
    outcome: 'all',
    start_date: range.start_date,
    end_date: range.end_date,
  }
}

export function ExecutionsPage() {
  const meta = useAppPageMeta()
  const defaultFilters = useMemo(() => buildDefaultFilters(), [])
  const [search, setSearch] = useState('')
  const [draftFilters, setDraftFilters] = useState(defaultFilters)
  const [activeFilters, setActiveFilters] = useState(defaultFilters)
  const [periodPreset, setPeriodPreset] = useState<GenerationPeriodPreset>('30d')
  const [filterError, setFilterError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const focusedRowRef = useRef<HTMLTableRowElement | null>(null)
  const draftFiltersRef = useRef(draftFilters)
  draftFiltersRef.current = draftFilters

  const apiFilter = useMemo(() => toApiFilter(activeFilters), [activeFilters])
  const listFilter = useMemo(() => ({ ...apiFilter, limit: PAGE_LIMIT }), [apiFilter])
  const listQuery = useInfiniteQuery({
    queryKey: generationsQueryKeys.list(listFilter),
    queryFn: ({ pageParam }) => listGenerations({ ...listFilter, cursor: pageParam ?? undefined }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
    refetchInterval: false,
    retry: 1,
  })
  const providersQuery = useQuery({
    queryKey: catalogQueryKeys.providers,
    queryFn: listCatalogProviders,
    refetchInterval: false,
    retry: 1,
  })
  const runtimeMeta = useQuery({
    queryKey: overviewQueryKeys.runtimeMeta,
    queryFn: getRuntimeMeta,
    refetchInterval: false,
    retry: 1,
  })
  const generations = useMemo(
    () => listQuery.data?.pages.flatMap((page) => page.generations) ?? [],
    [listQuery.data],
  )
  const visibleGenerations = useMemo(
    () => generations.filter((generation) => generationMatchesSearch(generation, search)),
    [generations, search],
  )
  const selectedGeneration = visibleGenerations.find((generation) => generation.execution_id === selectedId) ?? null
  const providerOptions = useMemo(() => {
    const values = new Set([
      ...(providersQuery.data?.data ?? []).map((provider) => provider.provider),
      ...generations.map((generation) => generation.provider_id),
    ])
    if (activeFilters.provider_id) values.add(activeFilters.provider_id)
    return [...values].sort((left, right) => left.localeCompare(right))
  }, [activeFilters.provider_id, generations, providersQuery.data])
  const detailQuery = useQuery({
    queryKey: generationsQueryKeys.detail(selectedId ?? ''),
    queryFn: () => getGeneration(selectedId ?? ''),
    enabled: Boolean(selectedId && drawerOpen),
    refetchInterval: false,
    retry: 1,
  })
  const enrichmentNotice = useMemo(() => {
    const pages = listQuery.data?.pages ?? []
    if (pages.some((page) => page.degraded)) return 'ENRICHMENT DEGRADED'
    if (pages.some((page) => page.generations.some((generation) => generation.enrichment_status === 'partial'))) {
      return 'ENRICHMENT PARTIAL'
    }
    if (pages.some((page) => page.generations.some((generation) => generation.enrichment_status === 'unavailable'))) {
      return 'ENRICHMENT UNAVAILABLE'
    }
    return null
  }, [listQuery.data])

  const originTag = formatDataOriginTag(runtimeMeta.data?.data_origin)

  const commitFilters = useCallback(
    (next: GenerationFilterDraft) => {
      const validation = validateGenerationFilterDraft(next)
      if (!validation.ok) {
        setFilterError(validation.message)
        return false
      }
      setFilterError(null)
      setDraftFilters(next)
      setActiveFilters(next)
      if (periodPreset !== 'custom') {
        setPeriodPreset(inferPeriodPreset(next.start_date, next.end_date))
      }
      return true
    },
    [periodPreset],
  )

  useEffect(() => {
    setSelectedId(null)
    setDrawerOpen(false)
  }, [apiFilter])

  useEffect(() => {
    if (selectedId && !visibleGenerations.some((generation) => generation.execution_id === selectedId)) {
      setSelectedId(null)
      setDrawerOpen(false)
    }
  }, [selectedId, visibleGenerations])

  const applyFilters = () => {
    commitFilters(draftFilters)
  }

  const handlePeriodPresetChange = (preset: GenerationPeriodPreset) => {
    setPeriodPreset(preset)
    if (preset === 'custom') {
      return
    }
    const range = resolvePeriodPresetRange(preset)
    commitFilters({ ...draftFiltersRef.current, ...range })
  }

  const handleOutcomeChange = (outcome: GenerationFilterDraft['outcome']) => {
    commitFilters({ ...draftFiltersRef.current, outcome })
  }

  const handleProviderChange = (providerId: string) => {
    commitFilters({ ...draftFiltersRef.current, provider_id: providerId })
  }

  const handleDraftChange = (next: GenerationFilterDraft) => {
    setDraftFilters(next)
    if (periodPreset !== 'custom') {
      setPeriodPreset(inferPeriodPreset(next.start_date, next.end_date))
    }
  }

  const selectGeneration = (generation: Generation) => {
    setSelectedId(generation.execution_id)
    setDrawerOpen(true)
  }

  const restoreFocus = () => {
    if (focusedRowRef.current?.isConnected) {
      focusedRowRef.current.focus()
      return
    }
    document.querySelector<HTMLInputElement>('[aria-label="Buscar gerações carregadas"]')?.focus()
  }

  return (
    <div className={appPageClassName('executions-page generations-page', 'flex h-full flex-col gap-2 overflow-hidden')}>
      <div className="generations-page-top">
        <header className="generations-pagehead">
          <div className="min-w-0 flex-1">
            <p className="generations-page-eyebrow">{meta.eyebrow}</p>
            <h1 className="generations-page-title">{meta.title}</h1>
            {meta.lead ? <p className="generations-page-lead">{meta.lead}</p> : null}
          </div>
          <div className="generations-headright">
            <span className="overview-backend-tag">{originTag}</span>
            <RefreshButton
              className="overview-refresh"
              onRefresh={() => listQuery.refetch()}
              isRefreshing={listQuery.isFetching}
            />
          </div>
        </header>

        <nav className="generations-tabs" aria-label="Visões de gerações" role="tablist">
          <button type="button" role="tab" aria-selected="true" className="generations-tab generations-tab-active">
            Generations
          </button>
        </nav>

        <GenerationsActivityPanel />
      </div>

      <GenerationsFilters
        search={search}
        onSearchChange={setSearch}
        value={draftFilters}
        periodPreset={periodPreset}
        onPeriodPresetChange={handlePeriodPresetChange}
        providers={providerOptions}
        onDraftChange={handleDraftChange}
        onOutcomeChange={handleOutcomeChange}
        onProviderChange={handleProviderChange}
        onApply={applyFilters}
      />
      {filterError ? (
        <p className="text-xs text-failure" role="alert">
          {filterError}
        </p>
      ) : null}

      <GenerationsTable
        generations={visibleGenerations}
        selectedId={selectedId}
        onSelect={selectGeneration}
        onFocusRow={(row) => {
          focusedRowRef.current = row
        }}
        isLoading={listQuery.isLoading}
        isError={listQuery.isError}
        errorMessage={
          listQuery.error ? executionsErrorMessage(listQuery.error, 'Não foi possível carregar as gerações.') : undefined
        }
        enrichmentNotice={enrichmentNotice}
        hasNextPage={Boolean(listQuery.hasNextPage)}
        isFetchingNextPage={listQuery.isFetchingNextPage}
        onLoadMore={() => void listQuery.fetchNextPage()}
        loadedCount={generations.length}
        originTag={originTag}
      />

      <footer className="generations-footnote">
        <span>API · /v1/generations</span>
        <span>A BUSCA COBRE APENAS OS RESULTADOS CARREGADOS</span>
      </footer>

      <GenerationDetailsDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        generation={selectedGeneration}
        detail={detailQuery.data}
        isLoading={detailQuery.isLoading}
        isError={detailQuery.isError}
        errorMessage={
          detailQuery.error
            ? executionsErrorMessage(detailQuery.error, 'Não foi possível carregar os detalhes da geração.')
            : undefined
        }
        onRestoreFocus={restoreFocus}
      />
    </div>
  )
}

function toApiFilter(filters: GenerationFilterDraft): Omit<GenerationsFilter, 'cursor' | 'limit'> {
  return {
    provider_id: filters.provider_id || undefined,
    model_id: filters.model_id.trim() || undefined,
    outcome: filters.outcome === 'all' ? undefined : (filters.outcome as ExecutionOutcome),
    started_after: filters.start_date ? `${filters.start_date}T00:00:00.000Z` : undefined,
    started_before: filters.end_date ? `${filters.end_date}T23:59:59.999Z` : undefined,
  }
}
