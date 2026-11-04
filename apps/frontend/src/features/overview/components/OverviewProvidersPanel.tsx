import { Link } from 'react-router-dom'
import { Panel, PanelHeader } from '../../../shared/ui'
import type { ProviderDiagnostic } from '../api/queries'

type OverviewProvidersPanelProps = {
  providers?: ProviderDiagnostic[]
  isLoading?: boolean
}

export function OverviewProvidersPanel({
  providers = [],
  isLoading,
}: OverviewProvidersPanelProps) {
  return (
    <Panel>
      <PanelHeader
        title="Consumo de franquias"
        subtitle="Uso e situação por provedor conectado"
        action={
          <Link
            to="/providers"
            className="rounded-md border border-line bg-raised px-2.5 py-1 text-[10px] text-accent-light transition-colors hover:bg-white/[0.04] hover:text-primary"
          >
            Ver franquias ↗
          </Link>
        }
      />

      {isLoading ? (
        <div className="py-6 text-center text-xs text-muted animate-pulse">
          Carregando informações dos provedores…
        </div>
      ) : providers.length === 0 ? (
        <div className="py-6 text-center text-xs text-muted">
          Nenhum provedor configurado no momento.
        </div>
      ) : (
        <div className="divide-y divide-line/40">
          {providers.map((p) => {
            const initials = p.id.slice(0, 2).toUpperCase()
            const isOk = p.state === 'operational' || p.state === 'ready'
            const isDegraded = p.state === 'degraded'

            return (
              <div
                key={p.id}
                className="flex items-center justify-between py-2.5 first:pt-0 last:pb-0"
              >
                <div className="flex items-center gap-2.5">
                  <span className="grid size-7 place-items-center rounded-lg border border-line bg-raised font-mono text-[10px] font-semibold text-accent-light">
                    {initials}
                  </span>
                  <div>
                    <strong className="block text-xs font-semibold text-primary capitalize">
                      {p.id}
                    </strong>
                    <small className="block text-[10px] text-muted">
                      {p.enabled ? 'Conexão ativa' : 'Desabilitado'}
                    </small>
                  </div>
                </div>

                <div className="flex items-center gap-1.5 font-mono text-[10px]">
                  <span
                    className={`size-1.5 rounded-full ${
                      isOk ? 'bg-healthy' : isDegraded ? 'bg-warning' : 'bg-failure'
                    }`}
                  />
                  <span
                    className={
                      isOk
                        ? 'text-healthy'
                        : isDegraded
                          ? 'text-warning'
                          : 'text-failure'
                    }
                  >
                    {isOk ? 'Normal' : isDegraded ? 'Atenção' : 'Degradado'}
                  </span>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </Panel>
  )
}
