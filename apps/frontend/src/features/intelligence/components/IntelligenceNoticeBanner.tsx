import { AlertTriangle } from 'lucide-react'

export function IntelligenceNoticeBanner() {
  return (
    <div className="flex items-center justify-between gap-3 rounded-[10px] border border-warning/20 bg-warning/[0.04] p-3">
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning">
          <AlertTriangle className="size-4" />
        </div>
        <div className="min-w-0">
          <strong className="text-xs font-semibold text-primary">O Learning Engine ainda não está ativo</strong>
          <p className="mt-0.5 text-[11px] leading-relaxed text-secondary">
            O E08 já toma decisões determinísticas e rastreáveis. Avaliação de resultados por contexto, comparação adaptativa e recomendações aprendidas pertencem ao E10 e ainda não foram implementadas.
          </p>
        </div>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1">
        <span className="font-mono text-[9px] uppercase tracking-wider text-muted">STATUS DO MÓDULO</span>
        <span className="inline-flex rounded-full border border-warning/25 bg-warning/10 px-2 py-0.5 font-mono text-[9px] font-medium text-warning">
          AGUARDANDO E10
        </span>
      </div>
    </div>
  )
}

