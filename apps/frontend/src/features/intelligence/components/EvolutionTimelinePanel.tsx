import { cn } from '../../../shared/ui/cn'

const phases = [
  { num: '01', label: 'ATUAL', title: 'Decisão determinística', desc: 'Elegibilidade, seleção e rastreabilidade E08.', active: true },
  { num: '02', label: 'E10', title: 'Avaliação por contexto', desc: 'Construir evidência comparável a partir de uso real.' },
  { num: '03', label: 'E10', title: 'Recomendação', desc: 'Propor ajustes explicáveis, sujeitos a validação.' },
]

export function EvolutionTimelinePanel() {
  return (
    <div className="rounded-[10px] border border-line bg-surface p-3.5">
      <div className="mb-2.5 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">Evolução do módulo</h2>
          <p className="mt-0.5 text-[10px] text-muted">Separação entre capacidade entregue e próxima etapa.</p>
        </div>
        <span className="shrink-0 rounded-full border border-warning/25 bg-warning/10 px-2 py-0.5 font-mono text-[9px] font-medium text-warning">
          ROADMAP
        </span>
      </div>

      <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-3">
        {phases.map((p) => (
          <div
            key={p.num}
            className={cn(
              'rounded-lg border p-2.5 transition-colors',
              p.active
                ? 'border-cobalt/35 bg-cobalt/[0.04]'
                : 'border-line/70 bg-graphite/60'
            )}
          >
            <div className="font-mono text-[9px] uppercase tracking-wider text-muted">
              {p.num} · {p.label}
            </div>
            <div className="mt-0.5 text-xs font-semibold text-primary">{p.title}</div>
            <p className="mt-0.5 text-[10px] leading-relaxed text-muted">{p.desc}</p>
          </div>
        ))}
      </div>
    </div>
  )
}

