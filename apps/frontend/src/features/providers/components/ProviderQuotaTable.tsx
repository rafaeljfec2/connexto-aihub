import { useMemo, useState } from 'react'
import { GridPagination, Panel, PanelHeader, ProgressBar } from '../../../shared/ui'
import { useGridPagination } from '../../../shared/hooks/useGridPagination'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import {
  providerConnectionSubtitle,
  providerInitials,
  providerSituation,
} from '../lib/providerPresentation'
import type { AdminProvider } from '../types'

type ProviderQuotaTableProps = {
  providers: AdminProvider[]
  snapshots: ProviderQuotaSnapshot[]
  selectedId: string | null
  onSelect: (provider: AdminProvider) => void
  isLoading?: boolean
}

function primarySnapshot(
  providerId: string,
  snapshots: ProviderQuotaSnapshot[],
): ProviderQuotaSnapshot | undefined {
  const matches = snapshots.filter((snapshot) => snapshot.account_id === providerId)
  return (
    matches.find((snapshot) => snapshot.scope === 'subscription') ??
    matches.find((snapshot) => snapshot.scope === 'account') ??
    matches.find((snapshot) => snapshot.scope === 'api_key') ??
    matches[0]
  )
}

function franchiseTitle(snapshot: ProviderQuotaSnapshot | undefined): { title: string; value: string } {
  if (!snapshot || snapshot.status !== 'available') {
    return { title: 'Não monitorada', value: snapshot?.error?.message ?? 'Sem fonte de quota configurada' }
  }
  const rolling = snapshot.quotas.find((metric) => metric.metric === 'go_rolling')
  if (rolling?.consumed_percent != null) {
    return {
      title: 'Franquia Go (rolling)',
      value: `${rolling.consumed_percent.toLocaleString('pt-BR')}% reportado pelo provedor`,
    }
  }
  const usage = snapshot.quotas.find((metric) => metric.metric === 'total_usage' || metric.metric === 'usage')
  if (usage?.used != null) {
    return { title: 'Leitura oficial', value: `Uso reportado: ${usage.used}` }
  }
  return { title: 'Leitura oficial', value: `${snapshot.quotas.length} métrica(s) do provedor` }
}

function consumedPercent(snapshot: ProviderQuotaSnapshot | undefined): number | undefined {
  if (!snapshot || snapshot.status !== 'available') return undefined
  const rolling = snapshot.quotas.find((metric) => metric.metric === 'go_rolling')
  return rolling?.consumed_percent
}

export function ProviderQuotaTable({
  providers,
  snapshots,
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
                const snapshot = primarySnapshot(provider.id, snapshots)
                const franchise = franchiseTitle(snapshot)
                const percent = consumedPercent(snapshot)

                return (
                  <tr
                    key={provider.id}
                    className={`cursor-pointer transition-colors hover:bg-white/[0.02] ${
                      isSelected ? 'selected' : ''
                    }`}
                    onClick={() => onSelect(provider)}
                    data-testid={`providers-quota-row-${provider.id}`}
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
                      <div className="providers-quota-title">{franchise.title}</div>
                      <div className="providers-quota-value">{franchise.value}</div>
                    </td>
                    <td>
                      <div>
                        <div className="providers-progress-head">
                          <span>{percent != null ? 'Percentual do provedor' : 'Sem dados'}</span>
                          <span>{percent != null ? `${percent.toLocaleString('pt-BR')}%` : '—'}</span>
                        </div>
                        {percent != null ? (
                          <ProgressBar value={percent} max={100} />
                        ) : (
                          <ProgressBar value={0} max={100} variant="na" />
                        )}
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
