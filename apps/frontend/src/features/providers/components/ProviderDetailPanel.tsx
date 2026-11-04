import { Link } from 'react-router-dom'
import { Panel } from '../../../shared/ui'
import {
  providerConnectionSubtitle,
  providerInitials,
  providerSituation,
} from '../lib/providerPresentation'
import type { AdminProvider } from '../types'

type ProviderDetailPanelProps = {
  provider: AdminProvider | null
  onOpenManage: (provider: AdminProvider) => void
}

export function ProviderDetailPanel({ provider, onOpenManage }: ProviderDetailPanelProps) {
  if (!provider) {
    return (
      <Panel className="providers-panel min-w-0">
        <div className="providers-detail-head shrink-0">
          <strong>Detalhe da franquia</strong>
          <small>SELECIONE UM PROVEDOR</small>
        </div>
        <p className="py-6 text-center text-xs text-muted">
          Selecione um provedor na tabela para inspecionar a conexão.
        </p>
      </Panel>
    )
  }

  const situation = providerSituation(provider)
  const updatedAt = provider.updated_at
    ? new Date(provider.updated_at).toLocaleString('pt-BR')
    : '—'

  return (
    <Panel className="providers-panel min-w-0">
      <div className="providers-detail-head shrink-0">
        <strong>Detalhe da franquia</strong>
        <small>INSPEÇÃO DE GOVERNANÇA</small>
      </div>

      <div className="providers-detail-body">
        <div className="providers-provider-name mt-2 border-b border-line/70 pb-2">
          <span className="providers-provider-icon size-8 text-[10px]">{providerInitials(provider)}</span>
          <div>
            <strong className="block text-xs font-semibold text-[#E7ECF5]">{provider.name}</strong>
            <small className="mt-0.5 block text-[9px] text-muted">
              {providerConnectionSubtitle(provider)}
            </small>
          </div>
        </div>

        <div className="providers-quota-box">
          <div className="providers-quota-box-label">CONSUMO DA FRANQUIA</div>
          <div className="providers-quota-box-main">
            <strong>Sem leitura</strong>
            <span>utilizado</span>
          </div>
          <div className="providers-large-track">
            <div className="h-full w-0 rounded-full bg-[#5E6B7C]" />
          </div>
          <div className="providers-quota-foot">
            <span>Consumo: não reportado</span>
            <span>Limite: não informado</span>
          </div>
        </div>

        <div className="mt-1">
          <div className="providers-detail-row">
            <span>Saldo disponível</span>
            <strong>—</strong>
          </div>
          <div className="providers-detail-row">
            <span>Renovação / reset</span>
            <strong>Não informado</strong>
          </div>
          <div className="providers-detail-row">
            <span>Fonte da medição</span>
            <strong>Sem leitura na API</strong>
          </div>
          <div className="providers-detail-row">
            <span>Última atualização</span>
            <strong>{updatedAt}</strong>
          </div>
          <div className="providers-detail-row border-b-0">
            <span>Situação</span>
            <strong
              className={
                situation.state === 'warn'
                  ? 'text-warning'
                  : situation.state === 'ok'
                    ? 'text-healthy'
                    : 'text-muted'
              }
            >
              {situation.label}
            </strong>
          </div>
        </div>

        <p className="providers-source-note">
          Percentuais e saldos só aparecem com leitura confiável do gateway ou do provedor.
        </p>

        <div className="providers-detail-actions">
          <Link to="/resources" className="providers-detail-secondary">
            Ver políticas ↗
          </Link>
          <Link to="/catalog" className="providers-detail-ghost">
            Explorar modelos ↗
          </Link>
          <button
            type="button"
            onClick={() => onOpenManage(provider)}
            className="providers-detail-ghost cursor-pointer"
          >
            Gerenciar credenciais ↗
          </button>
        </div>
      </div>
    </Panel>
  )
}
