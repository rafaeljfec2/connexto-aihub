import { cn } from '../../../shared/ui/cn'

const steps = [
  { num: '01', title: 'Validar e projetar requisitos', desc: 'Validação estrutural e projeção de requisitos suportados pela requisição.', state: 'IMPLEMENTADO' },
  { num: '02', title: 'Verificar elegibilidade', desc: 'Provedor habilitado, modelo disponível e capacidades compatíveis.', state: 'IMPLEMENTADO' },
  { num: '03', title: 'Selecionar deterministicamente', desc: 'Preferência explícita primeiro; múltiplos elegíveis geram routing_ambiguous.', state: 'IMPLEMENTADO' },
  { num: '04', title: 'Avaliar resultados por contexto', desc: 'Comparar evidências de execução e qualidade antes de recomendar.', state: 'E10 · FUTURO', future: true },
]

export function DecisionStackPanel() {
  return (
    <div className="rounded-[10px] border border-line bg-surface p-3.5">
      <div className="mb-2.5 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">Como a decisão acontece hoje</h2>
          <p className="mt-0.5 text-[10px] text-muted">Fluxo real do roteador E08, sem aprendizagem adaptativa.</p>
        </div>
        <span className="shrink-0 rounded-full border border-cobalt/25 bg-cobalt/10 px-2 py-0.5 font-mono text-[9px] font-medium text-cobalt">
          E08 · ACEITO
        </span>
      </div>

      <div className="grid gap-2">
        {steps.map((step) => (
          <div
            key={step.num}
            className={cn(
              'flex items-center gap-2.5 rounded-lg border border-transparent p-1.5 transition-colors',
              step.future ? 'opacity-65' : 'hover:bg-white/[0.02]'
            )}
          >
            <span className="flex size-5 shrink-0 items-center justify-center rounded border border-line bg-raised font-mono text-[10px] font-medium text-muted">
              {step.num}
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-xs font-medium text-primary">{step.title}</div>
              <p className="text-[10px] text-muted">{step.desc}</p>
            </div>
            <span
              className={cn(
                'shrink-0 rounded px-1.5 py-0.5 font-mono text-[9px] font-medium uppercase',
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

      <div className="mt-2.5 border-t border-line/60 pt-2 text-[10px] leading-relaxed text-muted">
        Hoje não existe ranking por desempenho ou otimização de custo/latência. Uma falha de execução não dispara troca silenciosa de provedor.
      </div>
    </div>
  )
}

