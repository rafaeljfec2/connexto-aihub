import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { Button } from '../../../shared/ui'
import { catalogQueryKeys, listCatalogProviders } from '../../catalog'
import { executionsErrorMessage } from '../hooks/executionsFilter'
import { getGeneration, generationsQueryKeys, listGenerations } from '../api/generations'
import { GenerationDetailsDrawer } from '../components/GenerationDetailsDrawer'
import { GenerationsFilters } from '../components/GenerationsFilters'
import type { GenerationFilterDraft } from '../components/GenerationsFilters'
import { GenerationsTable } from '../components/GenerationsTable'
import type { ExecutionOutcome, Generation, GenerationsFilter } from '../types'
import { generationMatchesSearch } from '../lib/formatGeneration'
import '../generations.css'
import '../generation-drawer.css'

const PAGE_LIMIT = 50
const defaultFilters: GenerationFilterDraft = {
  provider_id: '',
  model_id: '',
  outcome: 'all',
  start_date: '',
  end_date: '',
}

export function ExecutionsPage() {
  const [search, setSearch] = useState('')
  const [draftFilters, setDraftFilters] = useState(defaultFilters)
  const [activeFilters, setActiveFilters] = useState(defaultFilters)
  const [filterError, setFilterError] = useState<string | null>(null)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const focusedRowRef = useRef<HTMLTableRowElement | null>(null)

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
    if (draftFilters.start_date && draftFilters.end_date && draftFilters.start_date > draftFilters.end_date) {
      setFilterError('The start date must be on or before the end date.')
      return
    }
    setFilterError(null)
    setActiveFilters({ ...draftFilters })
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
    document.querySelector<HTMLInputElement>('[aria-label="Search loaded generations"]')?.focus()
  }

  return (
    <div className={appPageClassName('executions-page generations-page', 'flex h-full flex-col gap-3 overflow-hidden')}>
      <div className="generations-pagehead">
        <AppPageHeader />
        <Button
          variant="secondary"
          size="sm"
          onClick={() => void listQuery.refetch()}
          disabled={listQuery.isFetching}
          className="cursor-pointer gap-1.5"
        >
          <RefreshCw className={`size-3.5 ${listQuery.isFetching ? 'animate-spin' : ''}`} aria-hidden="true" />
          <span>Refresh</span>
        </Button>
      </div>

      <nav className="generations-tabs" aria-label="Generation views" role="tablist">
        <button type="button" role="tab" aria-selected="true" className="generations-tab generations-tab-active">
          Generations
        </button>
      </nav>

      <GenerationsFilters
        search={search}
        onSearchChange={setSearch}
        value={draftFilters}
        providers={providerOptions}
        onChange={setDraftFilters}
        onApply={applyFilters}
      />
      {filterError ? <p className="text-xs text-failure" role="alert">{filterError}</p> : null}

      <GenerationsTable
        generations={visibleGenerations}
        selectedId={selectedId}
        onSelect={selectGeneration}
        onFocusRow={(row) => { focusedRowRef.current = row }}
        isLoading={listQuery.isLoading}
        isError={listQuery.isError}
        errorMessage={listQuery.error ? executionsErrorMessage(listQuery.error, 'Unable to load generations.') : undefined}
        enrichmentNotice={enrichmentNotice}
        hasNextPage={Boolean(listQuery.hasNextPage)}
        isFetchingNextPage={listQuery.isFetchingNextPage}
        onLoadMore={() => void listQuery.fetchNextPage()}
        loadedCount={generations.length}
      />

      <footer className="generations-footnote">
        <span>API · /v1/generations</span>
        <span>SEARCH COVERS LOADED RESULTS</span>
      </footer>

      <GenerationDetailsDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        generation={selectedGeneration}
        detail={detailQuery.data}
        isLoading={detailQuery.isLoading}
        isError={detailQuery.isError}
        errorMessage={detailQuery.error ? executionsErrorMessage(detailQuery.error, 'Unable to load generation details.') : undefined}
        onRestoreFocus={restoreFocus}
      />
    </div>
  )
}

function toApiFilter(filters: GenerationFilterDraft): Omit<GenerationsFilter, 'cursor' | 'limit'> {
  return {
    provider_id: filters.provider_id || undefined,
    model_id: filters.model_id.trim() || undefined,
    outcome: filters.outcome === 'all' ? undefined : filters.outcome as ExecutionOutcome,
    started_after: filters.start_date ? `${filters.start_date}T00:00:00.000Z` : undefined,
    started_before: filters.end_date ? `${filters.end_date}T23:59:59.999Z` : undefined,
  }
}
