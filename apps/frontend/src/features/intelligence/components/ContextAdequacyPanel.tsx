import { useState } from 'react'
import { cn } from '../../../shared/ui/cn'

const contexts = [
  { id: 'code', name: 'Código e desenvolvimento', desc: 'Geração, revisão e depuração', glyph: '⌘' },
  { id: 'reasoning', name: 'Raciocínio e análise', desc: 'Problemas multi-etapas e síntese', glyph: '∴' },
  { id: 'extraction', name: 'Extração estruturada', desc: 'Classificação e saída em schema', glyph: '{ }' },
  { id: 'chat', name: 'Conversação e suporte', desc: 'Interação e respostas contextualizadas', glyph: '◌' },
]

export function ContextAdequacyPanel() {
  const [selectedId, setSelectedId] = useState(contexts[0].id)
  const selected = contexts.find(c => c.id === selectedId) ?? contexts[0]

  return (
    <div className="rounded-[10px] border border-line bg-surface p-3.5">
      <div className="mb-2.5 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-xs font-semibold text-primary">Mapa de adequação por contexto</h2>
          <p className="mt-0.5 text-[10px] text-muted">Como o módulo organizará evidências para comparar modelos.</p>
        </div>
        <span className="shrink-0 rounded-full border border-line bg-raised/80 px-2 py-0.5 font-mono text-[9px] text-muted">
          VISÃO CONCEITUAL
        </span>
      </div>

      <div className="grid gap-1">
        <div className="flex justify-between px-2.5 py-1 font-mono text-[9px] uppercase tracking-wider text-muted">
          <span>CONTEXTO DE USO</span>
          <span className="hidden sm:inline">EVIDÊNCIA</span>
          <span className="text-right">ADEQUAÇÃO</span>
        </div>
        {contexts.map((ctx) => (
          <button
            key={ctx.id}
            type="button"
            onClick={() => setSelectedId(ctx.id)}
            className={cn(
              'flex items-center justify-between rounded-lg px-2.5 py-2 text-left transition-colors',
              selectedId === ctx.id ? 'bg-raised/90 border border-line/80' : 'hover:bg-white/[0.03] border border-transparent'
            )}
          >
            <div className="flex items-center gap-2.5">
              <span className="flex size-6 shrink-0 items-center justify-center rounded bg-raised font-mono text-xs text-accent-light">
                {ctx.glyph}
              </span>
              <div>
                <div className="text-xs font-medium text-primary">{ctx.name}</div>
                <div className="text-[10px] text-muted">{ctx.desc}</div>
              </div>
            </div>
            <span className="hidden text-[10px] text-muted sm:inline">Sem histórico</span>
            <span className="text-[10px] font-medium text-warning">Aguardando E10</span>
          </button>
        ))}
      </div>

      <div className="mt-2.5 flex items-center justify-between border-t border-line/60 pt-2 text-[10px] text-muted">
        <p className="truncate">
          Contexto selecionado: <strong className="font-semibold text-primary">{selected.name}</strong>. Categorias ilustrativas; o runtime ainda não as classifica automaticamente.
        </p>
        <b className="shrink-0 font-mono text-[9px] font-bold tracking-wider text-primary">SEM RANKING</b>
      </div>
    </div>
  )
}

