import { cn } from '../../../shared/ui/cn'

type DecisionStepItem = {
  readonly num: string
  readonly title: string
  readonly desc: string
  readonly state: string
  readonly future?: boolean
}

const steps: readonly DecisionStepItem[] = [
  {
    num: '01',
    title: 'Validar e projetar requisitos',
    desc: 'Validação estrutural e projeção de requisitos da requisição (understanding.go).',
    state: 'IMPLEMENTADO',
  },
  {
    num: '02',
    title: 'Verificar elegibilidade de candidatos',
    desc: 'Provedor habilitado, modelo disponível e capacidades compatíveis sem mock.',
    state: 'IMPLEMENTADO',
  },
  {
    num: '03',
    title: 'Seleção determinística & virtual auto',
    desc: 'Prioridade explícita ou ordem administrativa para aihub/auto; transparência no Gateway.',
    state: 'IMPLEMENTADO',
  },
  {
    num: '04',
    title: 'Avaliar resultados por contexto',
    desc: 'Comparar evidências de execução e qualidade antes de recomendar um modelo.',
    state: 'E10 · FUTURO',
    future: true,
  },
]

export function DecisionStackPanel() {
  return (
    <div className="rounded-[10px] border border-line bg-surface p-3">
      <div className="mb-2 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">Como a decisão acontece hoje</h2>
          <p className="mt-0.5 text-[0.625rem] text-muted">Fluxo real do roteador determinístico E08 e resolução virtual Auto.</p>
        </div>
        <span className="shrink-0 rounded-full border border-cobalt/25 bg-cobalt/10 px-2 py-0.5 font-mono text-[0.5625rem] font-medium text-cobalt">
          E08 · ACEITO
        </span>
      </div>

      <div className="grid gap-1.5">
        {steps.map((step) => (
          <div
            key={step.num}
            className={cn(
              'flex items-center gap-2.5 rounded-lg border border-transparent px-2 py-1 transition-colors',
              step.future ? 'opacity-65' : 'hover:bg-white/[0.02]'
            )}
          >
            <span className="flex size-5 shrink-0 items-center justify-center rounded border border-line bg-raised font-mono text-[0.625rem] font-medium text-muted">
              {step.num}
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-xs font-medium text-primary">{step.title}</div>
              <p className="text-[0.625rem] text-muted">{step.desc}</p>
            </div>
            <span
              className={cn(
                'shrink-0 rounded px-1.5 py-0.5 font-mono text-[0.5625rem] font-medium uppercase',
                step.future
                  ? 'border border-line bg-raised text-muted'
                  : 'border border-healthy/25 bg-healthy/10 text-healthy'
              )}
            >
              {step.state}
            </span>
          </div>
        ))}
      </div>

      <div className="mt-2 border-t border-line/60 pt-2 text-[0.625rem] leading-relaxed text-muted">
        Não existe ranking adaptativo por desempenho ou otimização de custo/latência. Uma falha de execução não dispara troca silenciosa de provedor.
      </div>
    </div>
  )
}
