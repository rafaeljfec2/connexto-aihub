import { useMemo, useState } from 'react'
import { GridPagination, Panel, PanelHeader, ProgressBar } from '../../../shared/ui'
import { useGridPagination } from '../../../shared/hooks/useGridPagination'
import {
  providerConnectionSubtitle,
  providerInitials,
  providerSituation,
} from '../lib/providerPresentation'
import type { AdminProvider } from '../types'

type ProviderQuotaTableProps = {
  providers: AdminProvider[]
  selectedId: string | null
  onSelect: (provider: AdminProvider) => void
  isLoading?: boolean
}

export function ProviderQuotaTable({
  providers,
  selectedId,
  onSelect,
  isLoading,
}: ProviderQuotaTableProps) {
  const [search, setSearch] = useState('')

  const filtered = useMemo(
    () =>
      providers.filter((provider) => {
        const query = search.toLowerCase().trim()
        return (
          provider.name.toLowerCase().includes(query) || provider.id.toLowerCase().includes(query)
        )
      }),
    [providers, search],
  )

  const pagination = useGridPagination(filtered, { resetKey: search })

  return (
    <Panel className="providers-panel min-w-0">
      <PanelHeader
        className="!mb-2 shrink-0"
        title="Consumo por provedor"
        subtitle="Cada provedor pode ter unidade e ciclo diferentes · ordenação alfabética"
        action={
          <input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="⌕  Buscar provedor..."
            className="providers-search"
            aria-label="Buscar provedor"
          />
        }
      />

      {isLoading ? (
        <div className="py-8 text-center text-xs text-muted animate-pulse">
          Carregando lista de provedores…
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-8 text-center text-xs text-muted">Nenhum provedor encontrado.</div>
      ) : (
        <div className="providers-table-body">
          <div className="grid-table-scroll scrollbar-slim">
          <table className="providers-quota-table w-full">
            <thead>
              <tr>
                <th>Provedor</th>
                <th>Franquia monitorada</th>
                <th>Consumo</th>
                <th>Disponível</th>
                <th>Situação</th>
              </tr>
            </thead>
            <tbody>
              {pagination.pageItems.map((provider) => {
                const isSelected = provider.id === selectedId
                const situation = providerSituation(provider)

                return (
                  <tr
                    key={provider.id}
                    className={`cursor-pointer transition-colors hover:bg-white/[0.02] ${
                      isSelected ? 'selected' : ''
                    }`}
                    onClick={() => onSelect(provider)}
                  >
                    <td>
                      <div className="providers-provider-name">
                        <span className="providers-provider-icon">{providerInitials(provider)}</span>
                        <div>
                          <strong className="block text-[11px] font-semibold text-[#E1E8F3]">
                            {provider.name}
                          </strong>
                          <small className="mt-[3px] block text-[9px] text-[#909DB0]">
                            {providerConnectionSubtitle(provider)}
                          </small>
                        </div>
                      </div>
                    </td>
                    <td>
                      <div className="providers-quota-title">Não monitorada</div>
                      <div className="providers-quota-value">Sem fonte de quota configurada</div>
                    </td>
                    <td>
                      <div>
                        <div className="providers-progress-head">
                          <span>Sem dados</span>
                          <span>—</span>
                        </div>
                        <ProgressBar value={0} max={100} variant="na" />
                      </div>
                    </td>
                    <td className="providers-remaining">—</td>
                    <td>
                      <span className={`providers-state ${situation.state}`}>
                        <i aria-hidden />
                        {situation.label}
                      </span>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          </div>
        </div>
      )}

      {!isLoading && filtered.length > 0 ? (
        <GridPagination
          meta={pagination.meta}
          onPreviousPage={pagination.goToPreviousPage}
          onNextPage={pagination.goToNextPage}
          pageSize={pagination.pageSize}
          onPageSizeChange={pagination.setPageSize}
          className="grid-panel-pagination shrink-0"
        />
      ) : null}

      <footer className="providers-panel-footnote shrink-0">
        <span>
          {providers.length} PROVEDOR{providers.length === 1 ? '' : 'ES'} · UNIDADES POR FRANQUIA
        </span>
        <span>DADOS DO BACKEND</span>
      </footer>
    </Panel>
  )
}
