import { Link } from 'react-router-dom'
import { Panel } from '../../../shared/ui'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import {
  providerConnectionSubtitle,
  providerInitials,
} from '../lib/providerPresentation'
import {
  availableBalanceDisplay,
  consumptionCell,
  formatQuotaResetAt,
  formatQuotaSourceName,
  formatQuotaUpdatedAt,
  franchiseSummary,
  primaryGoPercent,
  primaryGoPeriodLabel,
  providerQuotaSituation,
} from '../lib/quotaPresentation'
import type { AdminProvider } from '../types'

type ProviderDetailPanelProps = {
  provider: AdminProvider | null
  snapshot?: ProviderQuotaSnapshot
  onOpenManage: (provider: AdminProvider) => void
}

export function ProviderDetailPanel({ provider, snapshot, onOpenManage }: ProviderDetailPanelProps) {
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

  const situation = providerQuotaSituation(provider, snapshot)
  const updatedAt = formatQuotaUpdatedAt(snapshot, provider.updated_at)
  const percent = primaryGoPercent(snapshot)
  const goPeriod = primaryGoPeriodLabel(snapshot)
  const consumption = consumptionCell(snapshot)
  const franchise = franchiseSummary(snapshot)
  const balance = availableBalanceDisplay(snapshot)
  const primaryReading =
    percent != null
      ? `${percent.toLocaleString('pt-BR')}% (${goPeriod ?? 'provedor'})`
      : consumption.kind === 'usage'
        ? consumption.primary
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
              {snapshot?.workspace_name ? ` · ${snapshot.workspace_name}` : ''}
            </small>
          </div>
        </div>

        <div className="providers-quota-box" data-testid={`providers-detail-quota-${provider.id}`}>
          <div className="providers-quota-box-label">CONSUMO DA FRANQUIA</div>
          <div className="providers-quota-box-main">
            <strong>
              {percent != null
                ? `${percent.toLocaleString('pt-BR')}%`
                : consumption.kind === 'usage'
                  ? consumption.primary
                  : 'Sem leitura'}
            </strong>
            <span>utilizado</span>
          </div>
          {percent != null ? (
            <div className="providers-large-track">
              <div
                className="h-full rounded-full bg-[#5E6B7C]"
                style={{ width: `${Math.min(percent, 100)}%` }}
              />
            </div>
          ) : null}
          <div className="providers-quota-foot">
            <span>
              Consumo:{' '}
              {percent != null
                ? `${percent.toLocaleString('pt-BR')}% (${goPeriod ?? 'provedor'})`
                : consumption.kind === 'usage'
                  ? franchise.value
                  : 'não reportado'}
            </span>
            <span>
              {percent != null
                ? 'Limite: percentual do provedor'
                : balance !== '—'
                  ? `Disponível: ${balance}`
                  : 'Limite: leitura oficial'}
            </span>
          </div>
        </div>

        <div className="mt-1">
          <div className="providers-detail-row">
            <span>Leitura principal</span>
            <strong>{primaryReading}</strong>
          </div>
          <div className="providers-detail-row">
            <span>Renovação / reset</span>
            <strong>{formatQuotaResetAt(snapshot)}</strong>
          </div>
          <div className="providers-detail-row">
            <span>Fonte da medição</span>
            <strong>{formatQuotaSourceName(snapshot)}</strong>
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
