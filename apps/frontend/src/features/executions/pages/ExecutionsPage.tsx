import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw, Search } from 'lucide-react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { Button, FilterChips, MasterDetailLayout, Panel, PanelHeader } from '../../../shared/ui'
import { executionsQueryKeys, getExecutionRecord, listExecutionRecords } from '../api/queries'
import { ExecutionDetailPanel } from '../components/ExecutionDetailPanel'
import { ExecutionsMetrics } from '../components/ExecutionsMetrics'
import { ExecutionsTable } from '../components/ExecutionsTable'
import type { ExecutionRecord, ExecutionRecordsFilter } from '../types'

export function ExecutionsPage() {
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'completed' | 'failed'>('all')
  const [providerFilter, setProviderFilter] = useState('all')
  const [selectedRecord, setSelectedRecord] = useState<ExecutionRecord | null>(null)

  const queryFilter = useMemo<ExecutionRecordsFilter>(() => {
    const f: ExecutionRecordsFilter = { limit: 50 }
    if (statusFilter !== 'all') {
      f.outcome = statusFilter
    }
    if (providerFilter !== 'all') {
      f.provider_id = providerFilter
    }
    return f
  }, [statusFilter, providerFilter])

  const listQuery = useQuery({
    queryKey: executionsQueryKeys.list(queryFilter),
    queryFn: () => listExecutionRecords(queryFilter),
    refetchInterval: false,
    retry: 1,
  })

  const records = listQuery.data?.records ?? []

  const activeSelectedId =
    selectedRecord?.execution_id ??
    selectedRecord?.request_id ??
    records[0]?.execution_id ??
    records[0]?.request_id ??
    null

  const detailQuery = useQuery({
    queryKey: executionsQueryKeys.detail(activeSelectedId ?? ''),
    queryFn: () => getExecutionRecord(activeSelectedId!),
    enabled: Boolean(activeSelectedId),
    retry: 1,
  })

  const providerOptions = useMemo(() => {
    const providers = Array.from(
      new Set(records.map((r) => r.provider_id).filter(Boolean)),
    )
    return ['all', ...providers]
  }, [records])

  const statusChips = [
    { id: 'all', label: 'Todas' },
    { id: 'completed', label: 'Concluídas' },
    { id: 'failed', label: 'Falhas' },
  ]

  const activeRecord = detailQuery.data ?? selectedRecord ?? records[0]

  return (
    <div className={appPageClassName('executions-page', 'space-y-4')}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <AppPageHeader />
        <Button
          variant="secondary"
          size="sm"
          onClick={() => void listQuery.refetch()}
          disabled={listQuery.isFetching}
          className="cursor-pointer gap-1.5"
        >
          <RefreshCw className={`size-3.5 ${listQuery.isFetching ? 'animate-spin' : ''}`} />
          <span>Atualizar</span>
        </Button>
      </div>

      <ExecutionsMetrics records={records} isLoading={listQuery.isLoading} />

      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line/40 pb-3">
        <div className="relative max-w-xs flex-1">
          <Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted" />
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar por ID, modelo ou provedor…"
            className="w-full rounded-md border border-line bg-surface py-1.5 pl-8 pr-3 text-xs text-primary placeholder:text-muted focus:border-cobalt focus:outline-none"
          />
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <FilterChips
            options={statusChips}
            selectedId={statusFilter}
            onSelect={(id) => setStatusFilter(id as typeof statusFilter)}
          />

          <select
            value={providerFilter}
            onChange={(e) => setProviderFilter(e.target.value)}
            className="rounded border border-line bg-surface px-2.5 py-1.5 text-xs text-secondary focus:border-cobalt focus:outline-none"
          >
            <option value="all">Todos os provedores</option>
            {providerOptions
              .filter((p) => p !== 'all')
              .map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
          </select>
        </div>
      </div>

      <MasterDetailLayout
        master={
          <Panel>
            <PanelHeader
              title="Registro de execuções"
              subtitle={`${records.length} chamadas registradas no histórico recente`}
            />
            <ExecutionsTable
              records={records}
              search={search}
              selectedId={activeSelectedId}
              onSelect={(r) => setSelectedRecord(r)}
              isLoading={listQuery.isLoading}
              isError={listQuery.isError}
            />
          </Panel>
        }
        detail={
          <ExecutionDetailPanel
            record={activeRecord}
            isLoading={detailQuery.isLoading}
            isError={detailQuery.isError}
            errorMessage={detailQuery.error ? String(detailQuery.error) : undefined}
            selectedId={activeSelectedId}
          />
        }
      />
    </div>
  )
}
