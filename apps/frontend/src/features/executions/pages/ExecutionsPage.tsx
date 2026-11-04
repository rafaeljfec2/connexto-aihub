import { useMemo, useState } from 'react'
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { listProviders, providersQueryKeys } from '../../providers/api/queries'
import { Button } from '../../../shared/ui/button'
import {
  executionsQueryKeys,
  getExecutionRecord,
  listExecutionRecords,
} from '../api/queries'
import { ExecutionDetailPanel } from '../components/ExecutionDetailPanel'
import { ExecutionsFilters } from '../components/ExecutionsFilters'
import { ExecutionsTable } from '../components/ExecutionsTable'
import {
  DEFAULT_PAGE_LIMIT,
  defaultExecutionsUiFilter,
  executionsErrorMessage,
  toApiFilter,
} from '../hooks/executionsFilter'
import type { ExecutionsUiFilter } from '../types'

export function ExecutionsPage() {
  const [draftFilter, setDraftFilter] = useState<ExecutionsUiFilter>(defaultExecutionsUiFilter)
  const [appliedFilter, setAppliedFilter] = useState<ExecutionsUiFilter>(defaultExecutionsUiFilter)
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const apiFilter = useMemo(() => toApiFilter(appliedFilter), [appliedFilter])

  const providers = useQuery({
    queryKey: providersQueryKeys.list,
    queryFn: listProviders,
    refetchInterval: false,
    retry: 1,
  })

  const listQuery = useInfiniteQuery({
    queryKey: executionsQueryKeys.list({ limit: DEFAULT_PAGE_LIMIT, ...apiFilter }),
    queryFn: ({ pageParam }) =>
      listExecutionRecords({
        limit: DEFAULT_PAGE_LIMIT,
        ...apiFilter,
        cursor: pageParam,
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
    refetchInterval: false,
    retry: 1,
  })

  const records = useMemo(
    () => listQuery.data?.pages.flatMap((page) => page.records) ?? [],
    [listQuery.data?.pages],
  )

  const listErrorMessage = listQuery.isError
    ? executionsErrorMessage(
        listQuery.error,
        'Execution records are unavailable. Verify gateway access and the records store.',
      )
    : undefined

  const degraded = listQuery.data?.pages.some((page) => page.degraded) ?? false
  const loadedCount = records.length

  const paginationErrorMessage = listQuery.isFetchNextPageError
    ? executionsErrorMessage(listQuery.error, 'Could not load the next page of records.')
    : undefined

  const detailQuery = useQuery({
    queryKey: executionsQueryKeys.detail(selectedId ?? ''),
    queryFn: () => getExecutionRecord(selectedId ?? ''),
    enabled: selectedId !== null && selectedId.length > 0,
    refetchInterval: false,
    retry: 1,
  })

  const providerOptions = useMemo(() => {
    const ids = providers.data?.providers.map((provider) => provider.id) ?? []
    return [...new Set(ids)].sort()
  }, [providers.data?.providers])

  function patchApplied(apiPatch: Partial<ExecutionsUiFilter>) {
    setAppliedFilter((previous) => ({ ...previous, ...apiPatch }))
    setSelectedId(null)
  }

  function handleOutcomeChange(outcome: ExecutionsUiFilter['outcome']) {
    const next = { ...draftFilter, outcome }
    setDraftFilter(next)
    patchApplied({ outcome })
  }

  function handleProviderChange(provider_id: string) {
    const next = { ...draftFilter, provider_id }
    setDraftFilter(next)
    patchApplied({ provider_id })
  }

  function applyModelFilter() {
    patchApplied({ model_id: draftFilter.model_id })
  }

  async function refreshAll() {
    await listQuery.refetch()
    if (selectedId) {
      await detailQuery.refetch()
    }
  }

  return (
    <div className="page executions-page">
      <AppPageHeader
        actions={
          <Button
            variant="ghost"
            size="sm"
            disabled={listQuery.isFetching}
            onClick={() => void refreshAll()}
          >
            <RefreshCw className="size-3.5" aria-hidden="true" />
            Refresh
          </Button>
        }
      />

      {degraded ? (
        <div className="banner banner-warning" role="status">
          Records index is degraded — history may be incomplete until the store recovers.
        </div>
      ) : null}

      <ExecutionsFilters
        value={draftFilter}
        providerOptions={providerOptions}
        onChange={setDraftFilter}
        onOutcomeChange={handleOutcomeChange}
        onProviderChange={handleProviderChange}
        onApplyModelFilter={applyModelFilter}
      />

      <div className="executions-grid">
        <section className="executions-list-panel">
          <div className="executions-panel-head">
            <strong>Execution log</strong>
            <span className="font-mono text-[9px] uppercase text-muted">
              {loadedCount} loaded · limit {DEFAULT_PAGE_LIMIT}
            </span>
          </div>

          <ExecutionsTable
            records={records}
            search={draftFilter.search}
            selectedId={selectedId}
            onSelect={setSelectedId}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={listErrorMessage}
          />

          {paginationErrorMessage ? (
            <div className="banner banner-error mx-3 mb-2" role="alert">
              {paginationErrorMessage}
            </div>
          ) : null}

          {listQuery.hasNextPage ? (
            <div className="executions-list-footer">
              <Button
                variant="ghost"
                size="sm"
                disabled={listQuery.isFetchingNextPage}
                onClick={() => void listQuery.fetchNextPage()}
              >
                {listQuery.isFetchingNextPage ? 'Loading…' : 'Load more'}
              </Button>
            </div>
          ) : null}

          <div className="executions-footnote">
            <span>API · /v1/records/executions</span>
            <span>Content-free audit metadata</span>
          </div>
        </section>

        <ExecutionDetailPanel
          record={detailQuery.data}
          isLoading={detailQuery.isLoading && selectedId !== null}
          isError={detailQuery.isError}
          errorMessage={
            detailQuery.isError
              ? executionsErrorMessage(
                  detailQuery.error,
                  'Could not load execution metadata.',
                )
              : undefined
          }
          selectedId={selectedId}
        />
      </div>
    </div>
  )
}
