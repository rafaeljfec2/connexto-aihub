import { Link } from 'react-router-dom'
import { Panel, PanelHeader } from '../../../shared/ui'
import type { ExecutionRecord } from '../../executions/types'

type OverviewRecentExecutionsPanelProps = {
  executions?: ExecutionRecord[]
  isLoading?: boolean
}

export function OverviewRecentExecutionsPanel({
  executions = [],
  isLoading,
}: OverviewRecentExecutionsPanelProps) {
  return (
    <Panel className="mt-3">
      <PanelHeader
        title="Execuções recentes"
        subtitle="Amostra das últimas chamadas processadas pelo gateway"
        action={
          <Link
            to="/executions"
            className="rounded-md border border-line bg-raised px-2.5 py-1 text-[10px] text-accent-light transition-colors hover:bg-white/[0.04] hover:text-primary"
          >
            Ver todas ↗
          </Link>
        }
      />

      {isLoading ? (
        <div className="py-6 text-center text-xs text-muted animate-pulse">
          Carregando histórico de execuções…
        </div>
      ) : executions.length === 0 ? (
        <div className="py-6 text-center text-xs text-muted">
          Nenhuma execução registrada até o momento.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-line text-[9px] font-mono uppercase tracking-[0.04em] text-muted">
                <th className="px-2 py-2">Horário</th>
                <th className="px-2 py-2">Modelo</th>
                <th className="px-2 py-2">Provedor</th>
                <th className="px-2 py-2">Operação</th>
                <th className="px-2 py-2">Status</th>
                <th className="px-2 py-2 text-right">Duração</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line/30 text-[10px]">
              {executions.slice(0, 5).map((e) => {
                const isSuccess = e.outcome === 'completed'
                const formattedTime = e.started_at
                  ? new Date(e.started_at).toLocaleTimeString('pt-BR')
                  : '—'
                const durationMs =
                  e.duration_ns !== undefined
                    ? Math.round(e.duration_ns / 1_000_000)
                    : undefined
                const durationText = durationMs !== undefined ? `${durationMs} ms` : '—'

                return (
                  <tr key={e.execution_id} className="hover:bg-white/[0.02]">
                    <td className="px-2 py-2 font-mono text-muted">{formattedTime}</td>
                    <td className="px-2 py-2 font-medium text-primary">
                      {e.model_id ?? '—'}
                    </td>
                    <td className="px-2 py-2 text-secondary">{e.provider_id ?? '—'}</td>
                    <td className="px-2 py-2 font-mono text-muted">responses.create</td>
                    <td className="px-2 py-2">
                      <span
                        className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 font-mono text-[9px] ${
                          isSuccess
                            ? 'bg-healthy/10 text-healthy'
                            : 'bg-failure/10 text-failure'
                        }`}
                      >
                        <i className="size-1 rounded-full bg-current" />
                        {isSuccess ? 'CONCLUÍDA' : 'FALHA'}
                      </span>
                    </td>
                    <td className="px-2 py-2 text-right font-mono text-muted">{durationText}</td>
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
