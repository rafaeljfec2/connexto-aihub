import { type RefObject, useCallback, useRef } from 'react'
import { Brain } from 'lucide-react'
import { CopyTextButton } from '../../../shared/ui'
import { formatPricingCell } from '../lib/formatCatalogPricing'
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
      <col className="catalog-col-model" />
      <col className="catalog-col-provider" />
      <col className="catalog-col-id" />
      <col className="catalog-col-price" />
      <col className="catalog-col-price" />
      <col className="catalog-col-price" />
      <col className="catalog-col-price" />
      <col className="catalog-col-listing" />
      {withCompare ? <col className="catalog-col-compare" /> : null}
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
        <th scope="col" className="catalog-col-model">
          Modelo
        </th>
        <th scope="col" className="catalog-col-provider">
          Provedor
        </th>
        <th scope="col" className="catalog-col-id">
          Identificador
        </th>
        <th scope="col" className="catalog-col-price">
          Entrada
        </th>
        <th scope="col" className="catalog-col-price">
          Saída
        </th>
        <th scope="col" className="catalog-col-price">
          Cache leitura
        </th>
        <th scope="col" className="catalog-col-price">
          Cache escrita
        </th>
        <th scope="col" className="catalog-col-listing">
          Catálogo
        </th>
        {withCompare ? (
          <th scope="col" className="catalog-col-compare">
            Comparar
          </th>
        ) : null}
      </tr>
      <tr className="catalog-column-filters">
        <th scope="col" className="catalog-col-model">
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
        <th scope="col" className="catalog-col-provider">
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
        <th scope="col" className="catalog-col-id">
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
        <th scope="col" className="catalog-col-price" aria-hidden />
        <th scope="col" className="catalog-col-price" aria-hidden />
        <th scope="col" className="catalog-col-price" aria-hidden />
        <th scope="col" className="catalog-col-price" aria-hidden />
        <th scope="col" className="catalog-col-listing">
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
        {withCompare ? <th scope="col" className="catalog-col-compare" aria-hidden /> : null}
      </tr>
    </thead>
  )

  return (
    <div className="catalog-table-shell">
      <div
        ref={headScrollRef}
        className="catalog-table-head scrollbar-slim"
        onScroll={() => syncHorizontalScroll('head')}
      >
        <table className="catalog-modeltable">
          <CatalogTableColGroup withCompare={withCompare} />
          {tableHead}
        </table>
      </div>
      <div
        ref={bodyScrollRef}
        className="catalog-table-body grid-table-scroll scrollbar-slim"
        onScroll={() => syncHorizontalScroll('body')}
      >
        <table className="catalog-modeltable">
          <CatalogTableColGroup withCompare={withCompare} />
          <tbody>
            {showEmptyBody ? (
              <tr>
                <td colSpan={withCompare ? 9 : 8} className="py-8 text-center text-xs text-muted">
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
                    <td className="catalog-col-model">
                      <div className="catalog-modelcell">
                        <span className="catalog-modelglyph">
                          <Brain className="size-[15px]" aria-hidden />
                        </span>
                        <div className="min-w-0">
                          <div className="catalog-modelname">{model.name}</div>
                          <div className="catalog-modelid-row">
                            <div className="catalog-modelid">{model.model_id}</div>
                            <CopyTextButton
                              text={model.model_id}
                              label={`Copiar identificador ${model.model_id}`}
                            />
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="catalog-col-provider catalog-providerlabel">{model.provider}</td>
                    <td
                      className="catalog-col-id catalog-col-id-cell catalog-modelid"
                      title={model.model_id}
                    >
                      {model.model_id}
                    </td>
                    <td className="catalog-col-price catalog-pricing-cell">
                      {formatPricingCell(model.pricing, 'input')}
                    </td>
                    <td className="catalog-col-price catalog-pricing-cell">
                      {formatPricingCell(model.pricing, 'output')}
                    </td>
                    <td className="catalog-col-price catalog-pricing-cell">
                      {formatPricingCell(model.pricing, 'cache_read')}
                    </td>
                    <td className="catalog-col-price catalog-pricing-cell">
                      {formatPricingCell(model.pricing, 'cache_write')}
                    </td>
                    <td className="catalog-col-listing">
                      <span className="catalog-state">
                        <i aria-hidden />
                        {listingLabel(model)}
                      </span>
                    </td>
                    {withCompare ? (
                      <td className="catalog-col-compare" onClick={(event) => event.stopPropagation()}>
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
