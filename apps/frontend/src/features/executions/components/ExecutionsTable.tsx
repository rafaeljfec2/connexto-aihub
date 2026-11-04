import type { ExecutionRecord } from '../types'

type ExecutionsTableProps = {
  records: ExecutionRecord[]
  search: string
  selectedId: string | null
  onSelect: (record: ExecutionRecord) => void
  isLoading: boolean
  isError: boolean
}

export function ExecutionsTable({
  records,
  search,
  selectedId,
  onSelect,
  isLoading,
  isError,
}: ExecutionsTableProps) {
  if (isLoading) {
    return (
      <div className="py-8 text-center text-xs text-muted animate-pulse">
        Carregando registros de execução…
      </div>
    )
  }

  if (isError) {
    return (
      <div className="banner banner-error" role="alert">
        Falha ao buscar registros de execução.
      </div>
    )
  }

  const query = search.toLowerCase().trim()
  const filtered = records.filter((r) => {
    if (!query) return true
    const model = (r.model_id ?? '').toLowerCase()
    const provider = (r.provider_id ?? '').toLowerCase()
    const reqId = (r.request_id ?? r.execution_id ?? '').toLowerCase()
    return model.includes(query) || provider.includes(query) || reqId.includes(query)
  })

  if (filtered.length === 0) {
    return (
      <p className="empty-state py-8 text-center text-xs text-muted">
        No execution records match the active criteria.
      </p>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left">
        <thead>
          <tr className="border-b border-line font-mono text-[9px] uppercase tracking-[0.04em] text-muted">
            <th scope="col" className="px-3 py-2.5">
              Horário
            </th>
            <th scope="col" className="px-3 py-2.5">
              Modelo / Provedor
            </th>
            <th scope="col" className="px-3 py-2.5">
              Tipo
            </th>
            <th scope="col" className="px-3 py-2.5">
              Status
            </th>
            <th scope="col" className="px-3 py-2.5 text-right">
              Duração
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line/30 text-xs">
          {filtered.map((record) => {
            const id = record.execution_id ?? record.request_id ?? ''
            const isSelected = selectedId === id
            const isSuccess = record.outcome === 'completed'

            const time = record.started_at
              ? new Date(record.started_at).toLocaleTimeString('pt-BR')
              : '—'

            const durationMs =
              record.duration_ns !== undefined
                ? Math.round(record.duration_ns / 1_000_000)
                : undefined

            const durationText = durationMs !== undefined ? `${durationMs} ms` : '—'

            return (
              <tr
                key={id}
                onClick={() => onSelect(record)}
                className={`cursor-pointer transition-colors ${
                  isSelected
                    ? 'bg-cobalt/10 shadow-[inset_2px_0_#7896E8]'
                    : 'hover:bg-white/[0.02]'
                }`}
              >
                <td className="px-3 py-3 font-mono text-muted">{time}</td>
                <td className="px-3 py-3">
                  <div className="font-semibold text-primary">
                    {record.model_id ?? '—'}
                  </div>
                  <div className="text-[10px] text-secondary">
                    {record.provider_id ?? '—'}
                  </div>
                </td>
                <td className="px-3 py-3 font-mono text-[10px] text-muted">
                  responses.create
                </td>
                <td className="px-3 py-3">
                  <span
                    className={`inline-flex items-center gap-1.5 rounded px-2 py-0.5 font-mono text-[9px] ${
                      isSuccess
                        ? 'bg-healthy/10 text-healthy'
                        : 'bg-failure/10 text-failure'
                    }`}
                  >
                    <i className="size-1 rounded-full bg-current" />
                    {isSuccess ? 'Concluída' : 'Falha'}
                  </span>
                </td>
                <td className="px-3 py-3 text-right font-mono text-muted">{durationText}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
