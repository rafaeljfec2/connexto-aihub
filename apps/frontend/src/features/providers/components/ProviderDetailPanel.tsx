import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Edit2 } from 'lucide-react'
import { AiBrandIcon } from '../../../shared/branding'
import { Panel } from '../../../shared/ui'
import type { ProviderQuotaSnapshot } from '../../overview/types/quota'
import { categorizeAntigravityQuotas } from '../../overview/lib/antigravityCategories'
import {
  providerConnectionSubtitle,
  providerIconSlug,
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

  const isAntigravity = snapshot?.provider_id === 'antigravity'
  const situation = providerQuotaSituation(provider, snapshot)
  const updatedAt = formatQuotaUpdatedAt(snapshot, provider.updated_at)
  const goPercent = primaryGoPercent(snapshot)
  const goPeriod = primaryGoPeriodLabel(snapshot)
  const consumption = consumptionCell(snapshot)
  const franchise = franchiseSummary(snapshot)
  const balance = availableBalanceDisplay(snapshot)

  const percent = goPercent ?? (consumption.kind === 'percent' ? consumption.percent : undefined)
  const percentPeriod =
    goPercent != null
      ? (goPeriod ?? 'provedor')
      : consumption.kind === 'percent'
        ? consumption.label
        : undefined

  const mainQuotaValue =
    percent != null
      ? `${percent.toLocaleString('pt-BR')}%`
      : consumption.kind === 'usage'
        ? consumption.primary
        : 'Sem leitura'

  const mainQuotaSuffix =
    isAntigravity
      ? 'restante'
      : percent != null || consumption.kind === 'percent'
        ? 'utilizado'
        : 'utilizado'

  const primaryReading =
    percent != null
      ? `${percent.toLocaleString('pt-BR')}% (${percentPeriod})`
      : consumption.kind === 'usage'
        ? consumption.primary
        : '—'

  const antigravityFamilies =
    isAntigravity && snapshot?.status === 'available'
      ? categorizeAntigravityQuotas(snapshot.quotas).filter((f) => f.id !== 'autocomplete_background')
      : []

  const antigravityBars = useMemo(() => {
    if (!isAntigravity || snapshot?.status !== 'available') return []
    const families = categorizeAntigravityQuotas(snapshot.quotas).filter(
      (f) => f.id !== 'autocomplete_background',
    )
    const gemini = families.find((f) => f.id === 'gemini_models')
    const claudeGpt = families.find((f) => f.id === 'claude_gpt_models')

    const bars: Array<{ id: string; label: string; percent: number }> = []

    if (gemini) {
      const weekly = gemini.limits.find((l) => l.windowType === 'weekly')
      if (weekly) {
        bars.push({
          id: 'gemini-weekly',
          label: 'Gemini (Limite semanal)',
          percent: weekly.remainingPercent,
        })
      }
      const fiveHours = gemini.limits.find((l) => l.windowType === '5h')
      if (fiveHours) {
        bars.push({
          id: 'gemini-5h',
          label: 'Gemini (Limite 5h)',
          percent: fiveHours.remainingPercent,
        })
      }
    }

    if (claudeGpt) {
      const weekly = claudeGpt.limits.find((l) => l.windowType === 'weekly') ?? claudeGpt.limits[0]
      if (weekly) {
        bars.push({
          id: 'claude-gpt-weekly',
          label: `Claude/GPT (${weekly.windowLabel})`,
          percent: weekly.remainingPercent,
        })
      }
    }

    // Complement with any other limits if fewer than 3 bars are populated
    if (bars.length < 3) {
      for (const f of families) {
        for (const limit of f.limits) {
          const id = `${f.id}-${limit.windowType}`
          if (!bars.some((b) => b.id === id || b.label.includes(limit.windowLabel))) {
            const shortTitle =
              f.id === 'claude_gpt_models' ? 'Claude/GPT' : f.id === 'gemini_models' ? 'Gemini' : f.title
            bars.push({
              id,
              label: `${shortTitle} (${limit.windowLabel})`,
              percent: limit.remainingPercent,
            })
            if (bars.length >= 3) break
          }
        }
        if (bars.length >= 3) break
      }
    }

    return bars
  }, [isAntigravity, snapshot])

  return (
    <Panel className="providers-panel min-w-0">
      <div className="providers-detail-head shrink-0">
        <strong>Detalhe da franquia</strong>
        <small>INSPEÇÃO DE GOVERNANÇA</small>
      </div>

      <div className="providers-detail-body">
        <div className="providers-provider-name mt-2 border-b border-line/70 pb-2">
          <AiBrandIcon
            slug={providerIconSlug(provider)}
            label={provider.name}
            size={32}
            className="providers-provider-icon size-8"
          />
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
            <strong>{mainQuotaValue}</strong>
            <span>{mainQuotaSuffix}</span>
          </div>

          {isAntigravity && antigravityBars.length > 0 ? (
            <div className="space-y-2.5 my-2.5" data-testid="antigravity-quota-bars">
              {antigravityBars.map((bar) => (
                <div key={bar.id} className="space-y-1">
                  <div className="flex items-center justify-between text-[11px]">
                    <span className="text-[#AEBBD0] font-medium">{bar.label}</span>
                    <span className="font-semibold text-[#E7ECF5]">
                      {bar.percent.toLocaleString('pt-BR')}% restante
                    </span>
                  </div>
                  <div className="providers-large-track">
                    <div
                      className="providers-large-fill"
                      style={{ width: `${Math.min(Math.max(bar.percent, 0), 100)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          ) : percent != null ? (
            <div className="providers-large-track">
              <div
                className="providers-large-fill"
                style={{ width: `${Math.min(percent, 100)}%` }}
              />
            </div>
          ) : consumption.kind === 'usage' &&
            consumption.barValue != null &&
            consumption.barMax != null &&
            consumption.barMax > 0 ? (
            <div className="providers-large-track">
              <div
                className="providers-large-fill"
                style={{
                  width: `${Math.min(100, Math.max(0, (consumption.barValue / consumption.barMax) * 100))}%`,
                }}
              />
            </div>
          ) : (
            <div className="providers-large-track" aria-hidden />
          )}
          <div className="providers-quota-foot">
            <span title={franchise.title}>
              <span className="text-[#8A97A8]">Consumo: </span>
              {franchise.value !== '—'
                ? franchise.value
                : percent != null
                  ? `${percent.toLocaleString('pt-BR')}% (${percentPeriod})`
                  : 'não reportado'}
            </span>
            <span>
              <span className="text-[#8A97A8]">Limite: </span>
              {isAntigravity
                ? 'pools oficiais'
                : percent != null
                  ? 'percentual do provedor'
                  : balance !== '—'
                    ? `disponível ${balance}`
                    : 'leitura oficial'}
            </span>
          </div>
        </div>

        {antigravityFamilies.length > 0 ? (
          <div
            className="mt-2 rounded-lg border border-line/60 bg-[#171E29]/90 p-2 space-y-1.5"
            data-testid="antigravity-detail-families"
          >
            <div className="flex items-center justify-between text-[9px] uppercase tracking-wider font-semibold text-[#8290A5]">
              <span>Franquias por Família</span>
              <span className="font-mono text-[8px] text-muted">
                {snapshot?.quotas.length ?? 0} modelos
              </span>
            </div>
            {antigravityFamilies.map((family) => (
              <div
                key={family.id}
                className="space-y-1 rounded bg-[#0f1722]/50 p-1.5 border border-line/30"
              >
                <div className="flex items-center justify-between text-[10px] font-medium text-[#CBD5E2]">
                  <span>{family.title}</span>
                  <span className="text-[9px] text-[#8290A5]">
                    {family.totalModelsCount} modelo(s)
                  </span>
                </div>
                {family.limits.map((limit, idx) => (
                  <div
                    key={`${limit.windowType}-${idx}`}
                    className="flex items-center justify-between text-[9px]"
                  >
                    <span className="text-[#AEBBD0]">{limit.windowLabel}</span>
                    <span className="font-semibold text-[#CBD5E2]">
                      {limit.remainingPercent.toLocaleString('pt-BR')}% restante
                    </span>
                  </div>
                ))}
              </div>
            ))}
          </div>
        ) : null}

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
          <div className="providers-detail-row border-b-0 items-center">
            <span>Situação</span>
            <strong className="flex justify-start">
              <span className={`providers-state ${situation.state}`}>
                <i aria-hidden />
                {situation.label}
              </span>
            </strong>
          </div>
        </div>

        <p className="providers-source-note">
          Percentuais e saldos só aparecem com leitura confiável do gateway ou do provedor.
        </p>

        <div className="providers-detail-actions flex flex-col gap-2 pt-1">
          <button
            type="button"
            onClick={() => onOpenManage(provider)}
            className="flex items-center justify-center gap-2 w-full rounded-md bg-cobalt px-3.5 py-2 text-xs font-semibold text-white shadow-sm hover:bg-[#345ED0] transition-colors cursor-pointer"
            aria-label={`Editar credenciais e conexão de ${provider.name}`}
          >
            <Edit2 className="size-3.5" />
            Editar credenciais e conexão
          </button>

          <div className="flex items-center gap-2 w-full">
            <Link to="/resources" className="providers-detail-secondary flex-1 text-center">
              Ver políticas ↗
            </Link>
            <Link to="/catalog" className="providers-detail-ghost flex-1 text-center">
              Explorar modelos ↗
            </Link>
          </div>
        </div>
      </div>
    </Panel>
  )
}
