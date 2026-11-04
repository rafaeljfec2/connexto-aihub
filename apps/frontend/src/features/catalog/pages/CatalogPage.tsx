import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { RefreshCw, Search } from 'lucide-react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { Button, FilterChips, MasterDetailLayout, Panel, PanelHeader } from '../../../shared/ui'
import { catalogQueryKeys, listCatalogModels } from '../api/queries'
import { CatalogModelDetailPanel } from '../components/CatalogModelDetailPanel'
import { CatalogModelsTable } from '../components/CatalogModelsTable'
import { ModelCompareModal } from '../components/ModelCompareModal'
import { useCatalogSync } from '../hooks/useCatalogSync'
import { formatCatalogSyncFeedback } from '../lib/syncFeedback'
import type { CatalogFilter, CatalogModel } from '../types'

export function CatalogPage() {
  const [search, setSearch] = useState('')
  const [providerFilter, setProviderFilter] = useState('all')
  const [sortBy, setSortBy] = useState<'name' | 'provider'>('name')
  const [selectedModel, setSelectedModel] = useState<CatalogModel | null>(null)
  const [compareSet, setCompareSet] = useState<Set<string>>(new Set())
  const [compareModalOpen, setCompareModalOpen] = useState(false)
  const searchInputRef = useRef<HTMLInputElement>(null)

  const appliedFilter = useMemo<CatalogFilter>(() => {
    return providerFilter !== 'all' ? { provider: providerFilter } : {}
  }, [providerFilter])

  const modelsQuery = useQuery({
    queryKey: catalogQueryKeys.models(appliedFilter),
    queryFn: () => listCatalogModels(appliedFilter),
    refetchInterval: false,
    retry: 1,
  })

  const sync = useCatalogSync(appliedFilter)
  const syncFeedback = formatCatalogSyncFeedback(sync.error)

  const rawModels = modelsQuery.data?.data ?? []

  // Atalho de teclado '/' para focar na busca
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (
        e.key === '/' &&
        document.activeElement !== searchInputRef.current &&
        !(document.activeElement instanceof HTMLInputElement) &&
        !(document.activeElement instanceof HTMLTextAreaElement)
      ) {
        e.preventDefault()
        searchInputRef.current?.focus()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  // Provedores únicos para os chips de filtro
  const providerOptions = useMemo(() => {
    const providers = Array.from(new Set(rawModels.map((m) => m.provider))).filter(Boolean)
    return [
      { id: 'all', label: 'Todos' },
      ...providers.map((p) => ({ id: p, label: p })),
    ]
  }, [rawModels])

  // Filtragem e ordenação local
  const filteredAndSortedModels = useMemo(() => {
    let result = rawModels.filter((m) => {
      const q = search.toLowerCase().trim()
      if (!q) return true
      return (
        m.name.toLowerCase().includes(q) ||
        m.model_id.toLowerCase().includes(q) ||
        m.provider.toLowerCase().includes(q)
      )
    })

    result = [...result].sort((a, b) => {
      if (sortBy === 'provider') {
        return a.provider.localeCompare(b.provider) || a.name.localeCompare(b.name)
      }
      return a.name.localeCompare(b.name)
    })

    return result
  }, [rawModels, search, sortBy])

  const activeSelected =
    selectedModel !== null
      ? filteredAndSortedModels.find(
          (m) => `${m.provider}:${m.model_id}` === `${selectedModel.provider}:${selectedModel.model_id}`,
        ) ?? filteredAndSortedModels[0] ?? null
      : filteredAndSortedModels[0] ?? null

  const handleToggleCompare = (model: CatalogModel) => {
    const key = `${model.provider}:${model.model_id}`
    setCompareSet((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        if (next.size < 3) {
          next.add(key)
        }
      }
      return next
    })
  }

  const comparedModelsList = useMemo(() => {
    return rawModels.filter((m) => compareSet.has(`${m.provider}:${m.model_id}`))
  }, [rawModels, compareSet])

  return (
    <div className="page catalog-page space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <AppPageHeader />
        <div className="flex items-center gap-2">
          <Button
            variant="primary"
            size="sm"
            disabled={sync.isPending}
            onClick={() => sync.mutate(undefined)}
            className="cursor-pointer gap-1.5"
          >
            <RefreshCw className={`size-3.5 ${sync.isPending ? 'animate-spin' : ''}`} />
            <span>{sync.isPending ? 'Sincronizando…' : 'Sincronizar catálogo'}</span>
          </Button>
        </div>
      </div>

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

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative max-w-md flex-1">
          <Search className="absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted" />
          <input
            ref={searchInputRef}
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar por nome do modelo ou identificador…"
            className="w-full rounded-lg border border-line bg-surface py-2 pl-8 pr-8 text-xs text-primary placeholder:text-muted focus:border-cobalt focus:outline-none"
          />
          <span className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded border border-line px-1.5 py-0.5 font-mono text-[9px] text-muted">
            /
          </span>
        </div>

        <span className="font-mono text-[10px] text-muted">
          {filteredAndSortedModels.length} REGISTROS DISPONÍVEIS
        </span>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line/40 pb-3">
        <FilterChips
          label="PROVEDOR"
          options={providerOptions}
          selectedId={providerFilter}
          onSelect={(id: string) => setProviderFilter(id)}
        />

        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 font-mono text-[9px] text-muted uppercase">
            <span>Ordenar</span>
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as 'name' | 'provider')}
              className="rounded border border-line bg-surface px-2 py-1 text-xs text-secondary focus:border-cobalt focus:outline-none"
            >
              <option value="name">Nome A–Z</option>
              <option value="provider">Provedor A–Z</option>
            </select>
          </label>

          <Button
            variant="secondary"
            size="sm"
            disabled={compareSet.size < 2}
            onClick={() => setCompareModalOpen(true)}
            className="text-[10px]"
          >
            <span>Comparar modelos</span>
            <strong className="font-mono ml-1">{compareSet.size}/3</strong>
          </Button>
        </div>
      </div>

      <MasterDetailLayout
        master={
          <Panel>
            <PanelHeader
              title="Modelos disponíveis no catálogo"
              subtitle="Catálogo normalizado recebido dos conectores habilitados"
            />
            <CatalogModelsTable
              models={filteredAndSortedModels}
              isLoading={modelsQuery.isLoading}
              isError={modelsQuery.isError}
              selectedModelId={
                activeSelected ? `${activeSelected.provider}:${activeSelected.model_id}` : null
              }
              onSelectModel={(m) => setSelectedModel(m)}
              compareSet={compareSet}
              onToggleCompare={handleToggleCompare}
            />
          </Panel>
        }
        detail={<CatalogModelDetailPanel model={activeSelected} />}
      />

      <ModelCompareModal
        models={comparedModelsList}
        open={compareModalOpen}
        onClose={() => setCompareModalOpen(false)}
      />
    </div>
  )
}
