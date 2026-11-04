import { Brain } from 'lucide-react'
import { Link } from 'react-router-dom'
import { Panel, PanelHeader } from '../../../shared/ui'
import type { CatalogModel } from '../types'

type CatalogModelDetailPanelProps = {
  model: CatalogModel | null
}

export function CatalogModelDetailPanel({ model }: CatalogModelDetailPanelProps) {
  if (!model) {
    return (
      <Panel>
        <PanelHeader title="Detalhes do modelo" subtitle="SELECIONE UM MODELO" />
        <p className="py-8 text-center text-xs text-muted">
          Clique em uma linha do catálogo para visualizar os metadados técnicos do modelo.
        </p>
      </Panel>
    )
  }

  const caps = Object.entries(model.capabilities ?? {})

  return (
    <Panel className="space-y-3.5">
      <PanelHeader title="Detalhes do modelo" subtitle="SELECIONADO" />

      <div className="border-b border-line pb-3.5">
        <div className="mb-2.5 grid size-10 place-items-center rounded-xl border border-cobalt/30 bg-accent-deep text-accent-light">
          <Brain className="size-5" />
        </div>
        <h2 className="text-sm font-semibold tracking-[-0.01em] text-primary">{model.name}</h2>
        <p className="font-mono text-[10px] text-muted">{model.model_id}</p>
      </div>

      <div className="divide-y divide-line/30 text-[10px]">
        <div className="flex justify-between py-2">
          <span className="text-muted">Provedor</span>
          <strong className="font-mono text-primary">{model.provider}</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Origem</span>
          <strong className="font-mono text-secondary">Catálogo do provedor</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Estado de listagem</span>
          <strong className="font-mono text-accent-light">{model.listing_state}</strong>
        </div>
        <div className="flex justify-between py-2">
          <span className="text-muted">Presença</span>
          <strong className="font-mono text-secondary">{model.presence}</strong>
        </div>
      </div>

      <div className="rounded-lg border border-line bg-raised p-3">
        <div className="mb-2 flex justify-between font-mono text-[8px] uppercase tracking-[0.05em] text-muted">
          <span>Metadados do modelo</span>
          <span className="text-accent-light">Quando disponíveis</span>
        </div>
        <div className="grid grid-cols-2 gap-2 text-[10px]">
          <div>
            <small className="block text-[8px] text-muted">Janela de contexto</small>
            <strong className="font-mono text-secondary">Não informada</strong>
          </div>
          <div>
            <small className="block text-[8px] text-muted">Capacidades</small>
            <strong className="font-mono text-secondary">
              {caps.length > 0 ? `${caps.length} declaradas` : 'Não informadas'}
            </strong>
          </div>
          <div>
            <small className="block text-[8px] text-muted">Preço de entrada</small>
            <strong className="font-mono text-secondary">Não informado</strong>
          </div>
          <div>
            <small className="block text-[8px] text-muted">Preço de saída</small>
            <strong className="font-mono text-secondary">Não informado</strong>
          </div>
        </div>
      </div>

      {caps.length > 0 ? (
        <div className="space-y-1.5">
          <span className="font-mono text-[8px] uppercase tracking-[0.05em] text-muted">
            Capacidades verificadas
          </span>
          <div className="flex flex-wrap gap-1">
            {caps.map(([name, cap]) => (
              <span
                key={name}
                className="rounded bg-raised border border-line px-1.5 py-0.5 font-mono text-[9px] text-secondary"
              >
                {name}: {cap.state}
              </span>
            ))}
          </div>
        </div>
      ) : null}

      <div className="pt-1">
        <Link
          to="/providers"
          className="block w-full rounded-md border border-cobalt/30 bg-accent-deep py-2 text-center text-xs font-semibold text-[#EAF0FC] transition-colors hover:bg-cobalt/30"
        >
          Ver franquias dos provedores ↗
        </Link>
      </div>

      <p className="text-[9px] leading-relaxed text-muted">
        Contexto, modalidades e preços serão apresentados quando esses metadados forem retornados
        pela API de sincronização do catálogo.
      </p>
    </Panel>
  )
}
