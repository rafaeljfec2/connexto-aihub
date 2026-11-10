import { ArrowUpRight, Brain, CircleDot, Zap } from 'lucide-react'

const cards = [
  {
    label: 'ROTEAMENTO ATUAL',
    value: 'Determinístico & Auto',
    note: 'E08 aceito · aihub/auto via prioridade administrativa',
    icon: ArrowUpRight,
  },
  {
    label: 'REQUEST UNDERSTANDING',
    value: 'Estrutural',
    note: 'understanding.go · sem classificador remoto',
    icon: Zap,
  },
  {
    label: 'AVALIAÇÃO DE RESULTADOS',
    value: 'Não ativada',
    note: 'Sem comparação adaptativa por contexto',
    icon: CircleDot,
  },
  {
    label: 'POLÍTICAS APRENDIDAS',
    value: 'Indisponíveis',
    note: 'Nenhum ranking adaptativo em produção',
    icon: Brain,
  },
] as const

export function IntelligenceStatCards() {
  return (
    <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
      {cards.map((card) => {
        const Icon = card.icon
        return (
          <div
            key={card.label}
            className="rounded-[10px] border border-line bg-surface p-2.5 transition-colors hover:border-line/80 sm:p-3"
          >
            <div className="flex items-center justify-between">
              <span className="font-mono text-[9px] uppercase tracking-[0.08em] text-muted sm:text-[10px]">
                {card.label}
              </span>
              <Icon className="size-3.5 text-muted/70" aria-hidden />
            </div>
            <div className="mt-1 font-display text-sm font-bold text-primary sm:text-base">
              {card.value}
            </div>
            <div className="mt-0.5 text-[10px] leading-tight text-muted/90">
              {card.note}
            </div>
          </div>
        )
      })}
    </div>
  )
}
