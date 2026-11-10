import { cn } from '../../../shared/ui/cn'

const signals = [
  { title: 'Decisões de roteamento', sub: 'Registros e motivos do E08.5', status: 'EXISTE' },
  { title: 'Resultado e latência', sub: 'Ciclo de execução e observabilidade', status: 'EXISTE' },
  { title: 'Uso reportado pelo provedor', sub: 'Tokens e metadados de uso', status: 'EXISTE' },
  { title: 'Contexto e qualidade', sub: 'Classificação de tarefa e feedback', status: 'PENDENTE', pending: true },
]

export function SignalsPanel() {
  return (
    <div className="rounded-[10px] border border-line bg-surface p-3.5">
      <div className="mb-2.5 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">Sinais disponíveis para evoluir o E10</h2>
          <p className="mt-0.5 text-[10px] text-muted">Tokens reportados quando disponíveis; ausência não é estimada nem tratada como zero.</p>
        </div>
        <span className="shrink-0 rounded-full border border-line bg-raised/80 px-2 py-0.5 font-mono text-[9px] text-muted">
          EVIDÊNCIAS
        </span>
      </div>

      <div className="grid gap-2">
        {signals.map((s) => (
          <div key={s.title} className="flex items-center justify-between gap-3 rounded-lg px-2.5 py-1.5 transition-colors hover:bg-white/[0.02]">
            <div className="flex min-w-0 items-center gap-2.5">
              <span className={`size-1.5 shrink-0 rounded-full ${s.pending ? 'bg-warning' : 'bg-healthy'}`} />
              <div className="min-w-0">
                <div className="text-xs font-medium text-primary">{s.title}</div>
                <div className="text-[10px] text-muted">{s.sub}</div>
              </div>
            </div>
            <span
              className={cn(
                'shrink-0 rounded px-1.5 py-0.5 font-mono text-[9px] font-medium uppercase',
                s.pending
                  ? 'border border-warning/25 bg-warning/10 text-warning'
                  : 'border border-healthy/25 bg-healthy/10 text-healthy'
              )}
            >
              {s.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
