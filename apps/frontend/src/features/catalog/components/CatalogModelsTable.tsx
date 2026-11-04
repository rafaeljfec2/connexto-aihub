import { type RefObject, useCallback, useRef } from 'react'
import { Brain } from 'lucide-react'
import type { CatalogColumnFilters } from '../lib/catalogColumnFilters'
import type { CatalogModel } from '../types'

type CatalogModelsTableProps = {
  models: CatalogModel[]
  isLoading: boolean
  isError: boolean
  selectedModelId?: string | null
  onSelectModel?: (model: CatalogModel) => void
  compareSet?: Set<string>
  onToggleCompare?: (model: CatalogModel) => void
  columnFilters: CatalogColumnFilters
  onColumnFilterChange: (patch: Partial<CatalogColumnFilters>) => void
  providerOptions: string[]
  listingOptions: string[]
  modelFilterInputRef?: RefObject<HTMLInputElement | null>
}

function listingLabel(model: CatalogModel): string {
  if (model.listing_state) {
    return model.listing_state
  }
  return 'Listado'
}

function CatalogTableColGroup({ withCompare }: { withCompare: boolean }) {
  return (
    <colgroup>
      <col style={{ width: withCompare ? '34%' : '36%' }} />
      <col style={{ width: '14%' }} />
      <col style={{ width: withCompare ? '26%' : '28%' }} />
      <col style={{ width: '14%' }} />
      {withCompare ? <col style={{ width: '12%' }} /> : null}
    </colgroup>
  )
}

export function CatalogModelsTable({
  models,
  isLoading,
  isError,
  selectedModelId,
  onSelectModel,
  compareSet = new Set(),
  onToggleCompare,
  columnFilters,
  onColumnFilterChange,
  providerOptions,
  listingOptions,
  modelFilterInputRef,
}: CatalogModelsTableProps) {
  if (isLoading) {
    return <p className="py-8 text-center text-xs text-muted">Carregando modelos do catálogo…</p>
  }

  if (isError) {
    return (
      <div className="banner banner-error m-3" role="alert">
        Falha ao carregar o catálogo. Verifique o acesso ao gateway e os filtros aplicados.
      </div>
    )
  }

  const showEmptyBody = models.length === 0
  const withCompare = Boolean(onToggleCompare)
  const headScrollRef = useRef<HTMLDivElement>(null)
  const bodyScrollRef = useRef<HTMLDivElement>(null)
  const isSyncingScrollRef = useRef(false)

  const syncHorizontalScroll = useCallback((source: 'head' | 'body') => {
    const head = headScrollRef.current
    const body = bodyScrollRef.current
    if (!head || !body || isSyncingScrollRef.current) {
      return
    }
    isSyncingScrollRef.current = true
    if (source === 'body') {
      head.scrollLeft = body.scrollLeft
    } else {
      body.scrollLeft = head.scrollLeft
    }
    isSyncingScrollRef.current = false
  }, [])

  const tableHead = (
    <thead>
      <tr>
        <th scope="col">Modelo</th>
        <th scope="col">Provedor</th>
        <th scope="col">Identificador</th>
        <th scope="col">Catálogo</th>
        {withCompare ? <th scope="col">Comparar</th> : null}
      </tr>
      <tr className="catalog-column-filters">
        <th scope="col">
          <input
            ref={modelFilterInputRef}
            type="search"
            value={columnFilters.model}
            onChange={(event) => onColumnFilterChange({ model: event.target.value })}
            placeholder="Filtrar modelo…"
            className="catalog-column-filter"
            aria-label="Filtrar coluna modelo"
            onClick={(event) => event.stopPropagation()}
          />
        </th>
        <th scope="col">
          <select
            value={columnFilters.provider}
            onChange={(event) => onColumnFilterChange({ provider: event.target.value })}
            className="catalog-column-filter"
            aria-label="Filtrar coluna provedor"
            onClick={(event) => event.stopPropagation()}
          >
            <option value="">Todos os provedores</option>
            {providerOptions.map((provider) => (
              <option key={provider} value={provider}>
                {provider}
              </option>
            ))}
          </select>
        </th>
        <th scope="col">
          <input
            type="search"
            value={columnFilters.modelId}
            onChange={(event) => onColumnFilterChange({ modelId: event.target.value })}
            placeholder="Filtrar ID…"
            className="catalog-column-filter"
            aria-label="Filtrar coluna identificador"
            onClick={(event) => event.stopPropagation()}
          />
        </th>
        <th scope="col">
          <select
            value={columnFilters.listing}
            onChange={(event) => onColumnFilterChange({ listing: event.target.value })}
            className="catalog-column-filter"
            aria-label="Filtrar coluna catálogo"
            onClick={(event) => event.stopPropagation()}
          >
            <option value="">Todos os estados</option>
            {listingOptions.map((listing) => (
              <option key={listing} value={listing}>
                {listing}
              </option>
            ))}
          </select>
        </th>
        {withCompare ? <th scope="col" aria-hidden /> : null}
      </tr>
    </thead>
  )

  return (
    <div className="catalog-table-shell">
      <div
        ref={headScrollRef}
        className="catalog-table-head"
        onScroll={() => syncHorizontalScroll('head')}
      >
        <table className="catalog-modeltable catalog-modeltable-fixed">
          <CatalogTableColGroup withCompare={withCompare} />
          {tableHead}
        </table>
      </div>
      <div
        ref={bodyScrollRef}
        className="catalog-table-body grid-table-scroll scrollbar-slim"
        onScroll={() => syncHorizontalScroll('body')}
      >
        <table className="catalog-modeltable catalog-modeltable-fixed">
          <CatalogTableColGroup withCompare={withCompare} />
          <tbody>
            {showEmptyBody ? (
              <tr>
                <td colSpan={withCompare ? 5 : 4} className="py-8 text-center text-xs text-muted">
                  Nenhum modelo corresponde aos filtros das colunas.
                </td>
              </tr>
            ) : (
              models.map((model) => {
                const key = `${model.provider}:${model.model_id}`
                const isSelected = selectedModelId === key
                const isChecked = compareSet.has(key)

                return (
                  <tr
                    key={key}
                    onClick={() => onSelectModel?.(model)}
                    className={isSelected ? 'catalog-row-selected' : undefined}
                  >
                    <td>
                      <div className="catalog-modelcell">
                        <span className="catalog-modelglyph">
                          <Brain className="size-[15px]" aria-hidden />
                        </span>
                        <div className="min-w-0">
                          <div className="catalog-modelname">{model.name}</div>
                          <div className="catalog-modelid">{model.model_id}</div>
                        </div>
                      </div>
                    </td>
                    <td className="catalog-providerlabel">{model.provider}</td>
                    <td className="catalog-modelid">{model.model_id}</td>
                    <td>
                      <span className="catalog-state">
                        <i aria-hidden />
                        {listingLabel(model)}
                      </span>
                    </td>
                    {withCompare ? (
                      <td onClick={(event) => event.stopPropagation()}>
                        <input
                          type="checkbox"
                          checked={isChecked}
                          onChange={() => onToggleCompare?.(model)}
                          aria-label={`Adicionar ${model.name} à comparação`}
                          className="catalog-compare-check"
                        />
                      </td>
                    ) : null}
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
