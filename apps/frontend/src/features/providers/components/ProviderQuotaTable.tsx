import { useState } from 'react'
import { Search } from 'lucide-react'
import { Panel, PanelHeader } from '../../../shared/ui'
import type { AdminProvider } from '../types'

type ProviderQuotaTableProps = {
  providers: AdminProvider[]
  selectedId: string | null
  onSelect: (provider: AdminProvider) => void
  onOpenManage: (provider: AdminProvider) => void
  isLoading?: boolean
}

export function ProviderQuotaTable({
  providers,
  selectedId,
  onSelect,
  onOpenManage,
  isLoading,
}: ProviderQuotaTableProps) {
  const [search, setSearch] = useState('')

  const filtered = providers.filter((p) => {
    const q = search.toLowerCase().trim()
    return p.name.toLowerCase().includes(q) || p.id.toLowerCase().includes(q)
  })

  return (
    <Panel className="space-y-3">
      <PanelHeader
        title="Consumo por provedor"
        subtitle="Cada provedor possui métricas, limites e credenciais próprias"
        action={
          <div className="relative w-48">
            <Search className="absolute left-2.5 top-1/2 size-3 -translate-y-1/2 text-muted" />
            <input
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Buscar provedor…"
              className="w-full rounded-md border border-line bg-raised py-1.5 pl-7 pr-2.5 text-xs text-primary placeholder:text-muted focus:border-cobalt focus:outline-none"
            />
          </div>
        }
      />

      {isLoading ? (
        <div className="py-8 text-center text-xs text-muted animate-pulse">
          Carregando lista de provedores…
        </div>
      ) : filtered.length === 0 ? (
        <div className="py-8 text-center text-xs text-muted">
          Nenhum provedor encontrado para a busca informada.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-line font-mono text-[9px] uppercase tracking-[0.04em] text-muted">
                <th className="px-3 py-2">Provedor</th>
                <th className="px-3 py-2">Credencial</th>
                <th className="px-3 py-2">Consumo</th>
                <th className="px-3 py-2">Situação</th>
                <th className="px-3 py-2 text-right">Ação</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line/30 text-xs">
              {filtered.map((p) => {
                const isSelected = p.id === selectedId
                const initials = p.name ? p.name.slice(0, 2).toUpperCase() : p.id.slice(0, 2).toUpperCase()

                return (
                  <tr
                    key={p.id}
                    onClick={() => onSelect(p)}
                    className={`cursor-pointer transition-colors ${
                      isSelected
                        ? 'bg-cobalt/10 shadow-[inset_2px_0_#7896E8]'
                        : 'hover:bg-white/[0.02]'
                    }`}
                  >
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-2.5">
                        <span className="grid size-7 place-items-center rounded-lg border border-line bg-raised font-mono text-[10px] font-semibold text-accent-light">
                          {initials}
                        </span>
                        <div>
                          <strong className="block font-semibold text-primary">{p.name}</strong>
                          <small className="block font-mono text-[9px] text-muted">{p.id}</small>
                        </div>
                      </div>
                    </td>

                    <td className="px-3 py-3">
                      <span className="font-mono text-[10px] text-secondary">
                        {p.credential_configured ? 'Configurada' : 'Pendente'}
                      </span>
                    </td>

                    <td className="px-3 py-3 font-mono text-[10px] text-muted">
                      <span>Sem leitura de quota</span>
                    </td>

                    <td className="px-3 py-3">
                      <span
                        className={`inline-flex items-center gap-1.5 rounded px-2 py-0.5 font-mono text-[9px] ${
                          p.enabled && p.credential_configured
                            ? 'bg-healthy/10 text-healthy'
                            : !p.enabled
                              ? 'bg-muted/10 text-muted'
                              : 'bg-warning/10 text-warning'
                        }`}
                      >
                        <i className="size-1 rounded-full bg-current" />
                        {p.enabled && p.credential_configured
                          ? 'Normal'
                          : !p.enabled
                            ? 'Desabilitado'
                            : 'Pendente'}
                      </span>
                    </td>

                    <td className="px-3 py-3 text-right">
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation()
                          onOpenManage(p)
                        }}
                        className="rounded border border-line bg-raised px-2.5 py-1 text-[10px] text-accent-light transition-colors hover:bg-white/[0.04] hover:text-primary cursor-pointer"
                      >
                        Gerenciar
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </Panel>
  )
}

