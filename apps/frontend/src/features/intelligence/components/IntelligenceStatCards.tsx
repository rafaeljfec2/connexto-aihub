import { ArrowUpRight, Brain, CircleDot, Zap } from 'lucide-react'

export function IntelligenceStatCards() {
  const cards = [
    { label: 'ROTEAMENTO ATUAL', value: 'Determinístico', note: 'E08.1 — E08.5 · aceito', icon: ArrowUpRight },
    { label: 'CONTEXTO DA TAREFA', value: 'Não classificado', note: 'E08.6 · não autorizado no MVP', icon: Zap },
    { label: 'AVALIAÇÃO DE RESULTADOS', value: 'Não ativada', note: 'Sem avaliação comparativa por contexto', icon: CircleDot },
    { label: 'POLÍTICAS APRENDIDAS', value: 'Indisponíveis', note: 'Nenhum ranking adaptativo em produção', icon: Brain },
  ]

  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {cards.map((card) => {
        const Icon = card.icon
        return (
          <div key={card.label} className="rounded-[10px] border border-line bg-surface p-3 transition-colors hover:border-line/80">
            <div className="flex items-center justify-between">
              <span className="font-mono text-[10px] uppercase tracking-[0.08em] text-muted">{card.label}</span>
              <Icon className="size-3.5 text-muted/70" aria-hidden />
            </div>
            <div className="mt-1 font-display text-base font-bold text-primary">{card.value}</div>
            <div className="mt-0.5 text-[10px] text-muted/90">{card.note}</div>
          </div>
        )
      })}
    </div>
  )
}
