import { Brain } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { CatalogModel } from '../types'

type CatalogModelDetailPanelProps = {
  model: CatalogModel | null
}

function connectorDescription(model: CatalogModel): string {
  return `Modelo listado pelo conector ${model.provider}. Consulte o provedor para os metadados completos.`
}

export function CatalogModelDetailPanel({ model }: CatalogModelDetailPanelProps) {
  if (!model) {
    return (
      <aside className="catalog-detail-panel">
        <div className="catalog-detail-top">
          <strong>Detalhes do modelo</strong>
          <span>SELECIONE UM MODELO</span>
        </div>
        <p className="py-8 text-center text-xs text-muted">
          Selecione uma linha do catálogo para ver os metadados disponíveis.
        </p>
      </aside>
    )
  }

  const caps = Object.entries(model.capabilities ?? {})

  return (
    <aside className="catalog-detail-panel">
      <div className="catalog-detail-top">
        <strong>Detalhes do modelo</strong>
        <span>SELECIONADO</span>
      </div>

      <div className="catalog-detail-scroll">
        <div className="catalog-detail-hero">
          <div className="catalog-detail-glyph">
            <Brain className="size-[18px]" aria-hidden />
          </div>
          <h2>{model.name}</h2>
          <p>{model.model_id}</p>
        </div>

        <div>
          <div className="catalog-metarow">
            <span>Provedor</span>
            <strong>{model.provider}</strong>
          </div>
          <div className="catalog-metarow">
            <span>Origem</span>
            <strong>Catálogo do provedor</strong>
          </div>
          <div className="catalog-metarow">
            <span>Registro</span>
            <strong>{model.listing_state || 'Encontrado'}</strong>
          </div>
        </div>

        <div className="catalog-detail-metadata">
          <div className="catalog-detail-metadata-head">
            METADADOS DO MODELO <span>QUANDO DISPONÍVEIS</span>
          </div>
          <div className="catalog-metadata-grid">
            <div>
              <small>Janela de contexto</small>
              <strong>Não informada</strong>
            </div>
            <div>
              <small>Capacidades</small>
              <strong>{caps.length > 0 ? `${caps.length} declaradas` : 'Não informadas'}</strong>
            </div>
            <div>
              <small>Preço de entrada</small>
              <strong>Não informado</strong>
            </div>
            <div>
              <small>Preço de saída</small>
              <strong>Não informado</strong>
            </div>
          </div>
        </div>

        <div className="catalog-detail-note">{connectorDescription(model)}</div>

        <Link to="/providers" className="catalog-detail-btn">
          Ver franquias dos provedores ↗
        </Link>

        <p className="catalog-detail-foot">
          Contexto, modalidades e preços serão apresentados quando esses metadados estiverem
          disponíveis no catálogo.
        </p>
      </div>
    </aside>
  )
}
